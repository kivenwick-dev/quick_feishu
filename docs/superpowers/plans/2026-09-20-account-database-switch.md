# 账号变更切换数据库 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 每个 `account.user_id` 使用独立的 SQLite 库文件；账号变更并重启后应用自动切库、补采，历史与快照按账号隔离。

**Architecture:** `internal/db` 提供「路径计算 / 打开 / 旧库迁移」三个纯函数；`internal/app` 统一持有库生命周期（启动与重启时按 `config.user_id` 切库、关旧库、触发 Backfill），并通过 `DB()` 暴露当前库；`internal/server` 的处理器改为每次请求经 `h.App.DB()` 动态取库。

**Tech Stack:** Go 1.x + GORM + glebarez/sqlite；测试用 `net/http/httptest` 保持 hermetic。

**Spec:** `docs/superpowers/specs/2026-09-18-account-database-switch-design.md`

---

## File Structure

| 文件 | 职责 |
| --- | --- |
| `internal/db/db.go`（改） | 新增 `Path`/`Open`/`MigrateLegacy`，`Init` 复用 `Open` |
| `internal/db/db_test.go`（改） | 新增 `TestPath`、`TestMigrateLegacy*` |
| `internal/app/app.go`（改） | 私有 `db` 字段 + `dataDir`/`currentDBPath`；`New` 返回 error；`DB()` 访问器；`Restart` 切库 |
| `internal/app/app_test.go`（改） | 适配新 `New` 签名；新增切库/同账号/空 user_id 三测试 |
| `internal/server/handlers.go`（改） | 去掉 `DB` 字段与 gorm import；改用 `h.App.DB()` |
| `internal/server/handlers2.go`（改） | `h.DB` → `h.App.DB()` |
| `internal/server/handlers3.go`（改） | `h.DB` → `h.App.DB()` |
| `internal/server/handlers_test.go`（改） | `newTestHandlers` 用 `app.New` |
| `internal/server/privacy_test.go`（改） | `h.DB` → `h.App.DB()` |
| `main.go`（改） | `newApp` 调 `app.New(cfg, dataDir, cfgPath)`；Handlers 不再传 DB |

**不改动：** 前端、数据库表结构、`db.Init` 的既有调用方（report/collector/scheduler 测试）。

---

## Task 1: db 路径、打开与旧库迁移

**Files:**
- Modify: `internal/db/db.go`
- Test: `internal/db/db_test.go`

- [ ] **Step 1: 写失败测试**

在 `internal/db/db_test.go` 末尾追加：

```go
func TestPath(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")
	wantAccount := filepath.Join(dir, "accounts", "42", "quick-feishu.db")
	if got := Path(dir, "42"); got != wantAccount {
		t.Errorf("Path(42) = %s, want %s", got, wantAccount)
	}
	wantLegacy := filepath.Join(dir, "quick-feishu.db")
	if got := Path(dir, ""); got != wantLegacy {
		t.Errorf("Path(empty) = %s, want %s", got, wantLegacy)
	}
}

func TestMigrateLegacyMovesOldDb(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "quick-feishu.db")
	if err := os.WriteFile(legacy, []byte("legacy"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := MigrateLegacy(dir, "42"); err != nil {
		t.Fatal(err)
	}
	target := Path(dir, "42")
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("target missing: %v", err)
	}
	if string(got) != "legacy" {
		t.Fatalf("target content = %q", got)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatalf("legacy should be gone, stat err = %v", err)
	}
}

func TestMigrateLegacyDoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "quick-feishu.db")
	target := Path(dir, "42")
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, []byte("legacy"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("target"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := MigrateLegacy(dir, "42"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "target" {
		t.Fatalf("target overwritten: %q", got)
	}
	if _, err := os.Stat(legacy); err != nil {
		t.Fatalf("legacy should remain when target exists: %v", err)
	}
}

func TestMigrateLegacyEmptyUserIDIsNoop(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "quick-feishu.db")
	if err := os.WriteFile(legacy, []byte("legacy"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := MigrateLegacy(dir, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(legacy); err != nil {
		t.Fatalf("legacy moved unexpectedly: %v", err)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/db/ -run 'TestPath|TestMigrateLegacy' -v`
Expected: 编译失败（`undefined: Path` / `MigrateLegacy`）

- [ ] **Step 3: 实现 Path / Open / MigrateLegacy**

把 `internal/db/db.go` 全文替换为：

```go
package db

import (
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"quick-feishu/internal/model"
)

// Path 返回账号对应的数据库文件路径。userID 为空时使用旧的单库路径。
func Path(dataDir, userID string) string {
	if userID == "" {
		return filepath.Join(dataDir, "quick-feishu.db")
	}
	return filepath.Join(dataDir, "accounts", userID, "quick-feishu.db")
}

// Open 打开（必要时创建父目录与文件）指定路径的数据库并迁移表结构。
func Open(path string) (*gorm.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	gdb, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}
	if err := gdb.AutoMigrate(
		&model.Snapshot{},
		&model.TokenSnapshot{},
		&model.SendLog{},
		&model.DictAccountField{},
		&model.DictTokenField{},
		&model.DictUsageField{},
	); err != nil {
		return nil, err
	}
	return gdb, nil
}

// Init 保留旧签名，打开 dataDir 下的单库（兼容既有测试）。
func Init(dataDir string) (*gorm.DB, error) {
	return Open(filepath.Join(dataDir, "quick-feishu.db"))
}

// MigrateLegacy 首次切到某账号时，把旧单库移动到该账号目录。
// 仅当 userID 非空、旧库存在且目标库不存在时执行；不覆盖已存在的账号库。
func MigrateLegacy(dataDir, userID string) error {
	if userID == "" {
		return nil
	}
	legacy := filepath.Join(dataDir, "quick-feishu.db")
	target := Path(dataDir, userID)
	if _, err := os.Stat(legacy); err != nil {
		return nil // 旧库不存在，无需迁移
	}
	if _, err := os.Stat(target); err == nil {
		return nil // 目标库已存在，不覆盖
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	return os.Rename(legacy, target)
}
```

- [ ] **Step 4: 运行 db 测试**

Run: `go test ./internal/db/ -v`
Expected: PASS（含既有 `TestInit*` 与新增测试）

- [ ] **Step 5: 提交**

```bash
git add internal/db/db.go internal/db/db_test.go
git commit -m "feat: add per-account db path, open and legacy migration"
```

---

## Task 2: App 生命周期与切库

**Files:**
- Modify: `internal/app/app.go`
- Test: `internal/app/app_test.go`

- [ ] **Step 1: 改写结构体与构造函数**

`internal/app/app.go` 中把 `type App struct` 与 `New`：

```go
type App struct {
	mu         sync.Mutex
	DB         *gorm.DB
	Config     *config.Config
	ConfigPath string

	client *api.Client
	sched  *scheduler.Scheduler
}

func New(cfg *config.Config, gdb *gorm.DB, configPath string) *App {
	a := &App{DB: gdb, Config: cfg, ConfigPath: configPath}
	a.client = api.NewClient(cfg.Account.APIBase, cfg.Account.SystemToken, cfg.Account.UserID)
	return a
}
```

替换为：

```go
type App struct {
	mu            sync.Mutex
	db            *gorm.DB
	Config        *config.Config
	ConfigPath    string
	dataDir       string
	currentDBPath string

	client *api.Client
	sched  *scheduler.Scheduler
}

// New 按 cfg.Account.UserID 迁移（首次）并打开对应账号库，返回 App。
func New(cfg *config.Config, dataDir, configPath string) (*App, error) {
	if err := db.MigrateLegacy(dataDir, cfg.Account.UserID); err != nil {
		return nil, err
	}
	path := db.Path(dataDir, cfg.Account.UserID)
	gdb, err := db.Open(path)
	if err != nil {
		return nil, err
	}
	if err := db.SeedDicts(gdb); err != nil {
		return nil, err
	}
	a := &App{
		db:            gdb,
		Config:        cfg,
		ConfigPath:    configPath,
		dataDir:       dataDir,
		currentDBPath: path,
	}
	a.client = api.NewClient(cfg.Account.APIBase, cfg.Account.SystemToken, cfg.Account.UserID)
	return a, nil
}

// DB 返回当前账号库（切库后指向新库）。
func (a *App) DB() *gorm.DB {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.db
}
```

- [ ] **Step 2: 运行编译确认失败处**

Run: `go build ./...`
Expected: FAIL，`internal/app/app.go` 中 `a.DB`（字段）与其它包 `a.DB`/`app.New` 报错——后续步骤逐个修复。

- [ ] **Step 3: 把内部读库改为 `a.DB()`**

`internal/app/app.go` 中 `RunSnapshot`、`RunReport`、`Backfill` 内所有 `a.DB` 改为 `a.DB()`（方法调用）。改完应为：

```go
func (a *App) RunSnapshot() (*collector.Result, error) {
	res := collector.Collect(a.Client())
	if err := collector.Save(a.DB(), Today(), res); err != nil {
		return res, err
	}
	return res, nil
}

func (a *App) RunReport() (*model.SendLog, error) {
	latest, err := db.LatestSnapshot(a.DB())
	if err != nil || latest == nil {
		return nil, fmt.Errorf("no snapshots yet")
	}
	var prev *model.Snapshot
	if p, e := db.SnapshotBefore(a.DB(), addDays(latest.SnapshotDate, -1)); e == nil {
		prev = p
	}
	tmpl, _ := report.TemplateFromMap(a.Config.ReportTemplate)
	if tmpl == nil {
		tmpl = report.DefaultTemplate()
	}
	return report.ExecuteReport(a.DB(), latest, prev, tmpl, a.Config.Feishu.WebhookURL, a.Config.Feishu.RetryTimes)
}

func (a *App) Backfill() error {
	return scheduler.BackfillMissing(a.DB(), a.Client(), Today(), func() *collector.Result {
		return collector.Collect(a.Client())
	})
}
```

- [ ] **Step 4: 改造 Restart 支持切库**

把 `Restart`：

```go
func (a *App) Restart() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if cfg, err := config.Load(a.ConfigPath); err == nil {
		*a.Config = *cfg
	}
	a.client = api.NewClient(a.Config.Account.APIBase, a.Config.Account.SystemToken, a.Config.Account.UserID)
	return a.startLocked()
}
```

替换为：

```go
// Restart 重载配置、按账号切库（如有变化）、重建 API 客户端并重启定时任务。
// 若发生了切库，释放锁后对空库执行 Backfill（Backfill 会再次加锁，不能持锁调用）。
func (a *App) Restart() error {
	a.mu.Lock()
	switched, err := a.restartLocked()
	a.mu.Unlock()
	if err != nil {
		return err
	}
	if switched {
		return a.Backfill()
	}
	return nil
}

func (a *App) restartLocked() (bool, error) {
	if cfg, err := config.Load(a.ConfigPath); err == nil {
		*a.Config = *cfg
	}
	switched := false
	next := db.Path(a.dataDir, a.Config.Account.UserID)
	if next != a.currentDBPath {
		if err := db.MigrateLegacy(a.dataDir, a.Config.Account.UserID); err != nil {
			return false, err
		}
		gdb, err := db.Open(next)
		if err != nil {
			return false, err
		}
		if err := db.SeedDicts(gdb); err != nil {
			return false, err
		}
		if sqlDB, cerr := a.db.DB(); cerr == nil {
			_ = sqlDB.Close()
		}
		a.db = gdb
		a.currentDBPath = next
		switched = true
	}
	a.client = api.NewClient(a.Config.Account.APIBase, a.Config.Account.SystemToken, a.Config.Account.UserID)
	if err := a.startLocked(); err != nil {
		return switched, err
	}
	return switched, nil
}
```

- [ ] **Step 5: 改写 app_test.go**

把 `internal/app/app_test.go` 全文替换为（注意新增 `net/http`、`net/http/httptest`、`model` import）：

```go
package app

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"quick-feishu/internal/config"
	"quick-feishu/internal/db"
	"quick-feishu/internal/model"
)

func clearCredentialEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"QR_USER_ID", "QR_SYSTEM_TOKEN", "QR_FEISHU_WEBHOOK"} {
		t.Setenv(key, "")
	}
}

// stubAPIServer 对所有请求返回 200 {}，使切库后的 Backfill 保持离线。
func stubAPIServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestStartRestartAndNextRuns(t *testing.T) {
	clearCredentialEnv(t)
	cfg := config.Default()
	a, err := New(cfg, t.TempDir(), filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := a.StartScheduler(); err != nil {
		t.Fatal(err)
	}
	if got := len(a.NextRuns()); got != 2 {
		t.Fatalf("next runs = %d, want 2", got)
	}
	cfg.Schedule.SnapshotTime = "01:23"
	if err := a.Restart(); err != nil {
		t.Fatal(err)
	}
	if got := len(a.NextRuns()); got != 2 {
		t.Fatalf("after restart next runs = %d, want 2", got)
	}
}

func TestClientRebuiltOnRestart(t *testing.T) {
	clearCredentialEnv(t)
	srv := stubAPIServer(t)
	path := filepath.Join(t.TempDir(), "config.yaml")

	cfg := config.Default()
	cfg.Account.UserID = "1"
	cfg.Account.SystemToken = "a"
	cfg.Account.APIBase = srv.URL
	a, err := New(cfg, t.TempDir(), path)
	if err != nil {
		t.Fatal(err)
	}
	if a.Client().UserID != "1" {
		t.Fatalf("initial client user = %s", a.Client().UserID)
	}

	next := config.Default()
	next.Account.UserID = "2"
	next.Account.SystemToken = "b"
	next.Account.APIBase = srv.URL
	if err := a.SaveConfig(next); err != nil {
		t.Fatal(err)
	}
	if err := a.Restart(); err != nil {
		t.Fatal(err)
	}
	if got := a.Client().UserID; got != "2" {
		t.Errorf("client not rebuilt after restart: %s", got)
	}
	if got := a.Client().SystemToken; got != "b" {
		t.Errorf("client token not rebuilt: %s", got)
	}
}

func TestSaveSettingsPreservesWriteOnlyCredentials(t *testing.T) {
	clearCredentialEnv(t)
	cfg := config.Default()
	cfg.Account.UserID = "existing-account"
	cfg.Account.SystemToken = "existing-token"
	cfg.Account.APIBase = "https://example.test"
	cfg.Feishu.WebhookURL = "https://example.test/webhook"
	a, err := New(cfg, t.TempDir(), filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	expectedAccount, expectedFeishu := cfg.Account, cfg.Feishu
	incoming := *cfg
	incoming.Account = config.AccountConfig{}
	incoming.Feishu.WebhookURL = ""
	incoming.Schedule.ReportTime = "12:34"
	if err := a.SaveConfig(&incoming); err != nil {
		t.Fatal(err)
	}
	if cfg.Account != expectedAccount || cfg.Feishu != expectedFeishu {
		t.Fatal("blank settings overwrote credentials")
	}
	if cfg.Schedule.ReportTime != "12:34" {
		t.Fatal("non-secret setting not saved")
	}
	incoming = *cfg
	incoming.Account.SystemToken = "replacement-token"
	if err := a.SaveConfig(&incoming); err != nil {
		t.Fatal(err)
	}
	if cfg.Account.SystemToken != "replacement-token" {
		t.Fatal("new credential not saved")
	}
}

func TestRestartSwitchesDatabase(t *testing.T) {
	clearCredentialEnv(t)
	srv := stubAPIServer(t)
	dir := t.TempDir()
	path := filepath.Join(t.TempDir(), "config.yaml")

	cfg := config.Default()
	cfg.Account.UserID = "account-a"
	cfg.Account.APIBase = srv.URL
	a, err := New(cfg, dir, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.DB().Create(&model.Snapshot{SnapshotDate: "2026-09-18", AccountUsed: 100}).Error; err != nil {
		t.Fatal(err)
	}

	next := config.Default()
	next.Account.UserID = "account-b"
	next.Account.APIBase = srv.URL
	if err := a.SaveConfig(next); err != nil {
		t.Fatal(err)
	}
	if err := a.Restart(); err != nil {
		t.Fatal(err)
	}

	if want := db.Path(dir, "account-b"); a.currentDBPath != want {
		t.Fatalf("currentDBPath = %s, want %s", a.currentDBPath, want)
	}
	var count int64
	a.DB().Model(&model.Snapshot{}).Where("account_used = ?", 100).Count(&count)
	if count != 0 {
		t.Fatalf("account A data leaked into account B db: count = %d", count)
	}
}

func TestRestartSameUserKeepsDatabase(t *testing.T) {
	clearCredentialEnv(t)
	dir := t.TempDir()
	path := filepath.Join(t.TempDir(), "config.yaml")

	cfg := config.Default()
	cfg.Account.UserID = "same-user"
	a, err := New(cfg, dir, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.DB().Create(&model.Snapshot{SnapshotDate: "2026-09-18", AccountUsed: 100}).Error; err != nil {
		t.Fatal(err)
	}
	if err := a.Config.Save(path); err != nil {
		t.Fatal(err)
	}
	if err := a.Restart(); err != nil {
		t.Fatal(err)
	}
	var count int64
	a.DB().Model(&model.Snapshot{}).Where("account_used = ?", 100).Count(&count)
	if count != 1 {
		t.Fatalf("data lost on same-user restart: count = %d", count)
	}
}

func TestEmptyUserIDUsesLegacyPath(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	a, err := New(cfg, dir, filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "quick-feishu.db"); a.currentDBPath != want {
		t.Fatalf("currentDBPath = %s, want %s", a.currentDBPath, want)
	}
}
```

- [ ] **Step 6: 运行 app 测试**

Run: `go test ./internal/app/ -v`
Expected: PASS（6 个测试全绿；`TestRestartSwitchesDatabase` 不访问真实网络）

- [ ] **Step 7: 提交**

```bash
git add internal/app/app.go internal/app/app_test.go
git commit -m "feat: switch database by account on startup and restart"
```

---

## Task 3: server 处理器改为动态取库

**Files:**
- Modify: `internal/server/handlers.go`
- Modify: `internal/server/handlers2.go`
- Modify: `internal/server/handlers3.go`
- Test: `internal/server/handlers_test.go`
- Test: `internal/server/privacy_test.go`

- [ ] **Step 1: 改写 handlers.go**

把 `internal/server/handlers.go` 全文替换为（去掉 `gorm` import 与 `DB` 字段，改用 `h.App.DB()`）：

```go
package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"quick-feishu/internal/app"
	"quick-feishu/internal/config"
	"quick-feishu/internal/db"
	"quick-feishu/internal/model"
)

type Handlers struct {
	App        *app.App
	Config     *config.Config
	ConfigPath string
}

func (h *Handlers) Dashboard(c *gin.Context) {
	snap, err := db.LatestSnapshot(h.App.DB())
	if err != nil {
		snap = nil
	}
	var logs []model.SendLog
	h.App.DB().Order("id DESC").Limit(10).Find(&logs)
	c.JSON(http.StatusOK, gin.H{"latest_snapshot": publicSnapshot(snap), "recent_logs": publicLogs(logs)})
}

func (h *Handlers) RunSnapshot(c *gin.Context) {
	res, err := h.App.RunSnapshot()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败，请检查输入或服务端配置"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "date": app.Today(), "issues": publicIssues(res.Issues)})
}

func (h *Handlers) SendReport(c *gin.Context) {
	log, err := h.App.RunReport()
	if log == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "操作失败，请检查输入或服务端配置"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败，请检查输入或服务端配置", "log_id": log.ID})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "log_id": log.ID})
}
```

- [ ] **Step 2: 批量替换 handlers2.go 与 handlers3.go 的取库**

在 `internal/server/handlers2.go` 与 `internal/server/handlers3.go` 中，把所有 `h.DB` 替换为 `h.App.DB()`（共 18 处：handlers2.go 12 处、handlers3.go 6 处）。

Run: `go build ./...`
Expected: 编译通过（`internal/server` 不再引用 `h.DB` 字段）。

- [ ] **Step 3: 改写 newTestHandlers**

把 `internal/server/handlers_test.go` 的 import 与 `newTestHandlers`：

```go
import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"quick-feishu/internal/db"
	"quick-feishu/internal/model"
)

func newTestHandlers(t *testing.T) *Handlers {
	t.Helper()
	gin.SetMode(gin.TestMode)
	gdb, err := db.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return &Handlers{DB: gdb}
}
```

替换为：

```go
import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"quick-feishu/internal/app"
	"quick-feishu/internal/config"
	"quick-feishu/internal/db"
	"quick-feishu/internal/model"
)

func newTestHandlers(t *testing.T) *Handlers {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := config.Default()
	a, err := app.New(cfg, t.TempDir(), filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return &Handlers{App: a, Config: a.Config}
}
```

并把该文件内：
- `h.DB.Create(...)`（第 62、63 行）→ `h.App.DB().Create(...)`
- `db.SeedDicts(h.DB)`（第 80 行）→ `db.SeedDicts(h.App.DB())`

- [ ] **Step 4: 改写 privacy_test.go 取库**

在 `internal/server/privacy_test.go` 中：
- `db.SeedDicts(h.DB)`（第 29 行）→ `db.SeedDicts(h.App.DB())`
- `h.DB.Create(...)`（第 35、39、44 行）→ `h.App.DB().Create(...)`

- [ ] **Step 5: 运行 server 测试**

Run: `go test ./internal/server/ -v`
Expected: PASS（既有 Dashboard/Snapshots/Compare/Dict/Privacy/SPA 测试全绿）

- [ ] **Step 6: 提交**

```bash
git add internal/server/handlers.go internal/server/handlers2.go internal/server/handlers3.go internal/server/handlers_test.go internal/server/privacy_test.go
git commit -m "refactor: handlers read current account db via App.DB()"
```

---

## Task 4: main.go 接线

**Files:**
- Modify: `main.go`

- [ ] **Step 1: 改写 newApp**

把 `main.go` 的 import 与 `newApp`：

```go
import (
	"fmt"
	"os"
	"path/filepath"

	"quick-feishu/internal/app"
	"quick-feishu/internal/config"
	"quick-feishu/internal/db"
	"quick-feishu/internal/server"
)

func newApp() (*app.App, error) {
	dir := baseDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}
	gdb, err := db.Init(filepath.Join(dir, "data"))
	if err != nil {
		return nil, fmt.Errorf("db error: %w", err)
	}
	if err := db.SeedDicts(gdb); err != nil {
		return nil, fmt.Errorf("seed error: %w", err)
	}
	return app.New(cfg, gdb, cfgPath), nil
}
```

替换为：

```go
import (
	"fmt"
	"os"
	"path/filepath"

	"quick-feishu/internal/app"
	"quick-feishu/internal/config"
	"quick-feishu/internal/server"
)

func newApp() (*app.App, error) {
	dir := baseDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}
	a, err := app.New(cfg, filepath.Join(dir, "data"), cfgPath)
	if err != nil {
		return nil, fmt.Errorf("db error: %w", err)
	}
	return a, nil
}
```

- [ ] **Step 2: 更新 Handlers 构造**

把 `serve()` 中：

```go
	handlers := &server.Handlers{App: a, DB: a.DB, Config: a.Config, ConfigPath: a.ConfigPath}
```

改为：

```go
	handlers := &server.Handlers{App: a, Config: a.Config, ConfigPath: a.ConfigPath}
```

- [ ] **Step 3: 编译与全量测试**

Run: `go build ./... && go test ./...`
Expected: 全部 PASS

- [ ] **Step 4: 提交**

```bash
git add main.go
git commit -m "feat: wire per-account database lifecycle in main"
```

---

## Task 5: 端到端验证（真实环境）

**Files:** 无（仅验证）

- [ ] **Step 1: 后端全量测试**

Run: `go test ./...`
Expected: 全部 ok

- [ ] **Step 2: 重新构建并重启**

```bash
pkill -f 'quick-feishu serve' || true
go build -o quick-feishu .
./quick-feishu serve
```

- [ ] **Step 3: 手动验证切库与隔离**

1. 设置页记录当前账号（假设 user_id = A）；确认 `data/accounts/A/quick-feishu.db` 已存在，仪表盘有历史数据。
2. 设置页把账号改为另一个 user_id = B，点「应用配置并重启」。
3. 观察进程日志无 `backfill error`；确认 `data/accounts/B/quick-feishu.db` 被创建且自动补采了当天快照。
4. 刷新历史页：只显示 B 账号的数据，看不到 A 的数据。
5. 把账号改回 A 再重启：历史恢复为 A 的数据（验证库按 user_id 复用、未丢失）。

- [ ] **Step 4: 验证旧库迁移（可选，使用临时副本）**

将现有 `data/quick-feishu.db` 备份后，清空 `config.user_id` 启动一次（走旧库），再设置某账号并重启，确认旧库被重命名为 `data/accounts/<user_id>/quick-feishu.db` 且历史保留。

- [ ] **Step 5: 最终提交（如有构建产物变动）**

```bash
git status --short
git add -A
git commit -m "chore: rebuild binary after account database switch"
```

---

## Self-Review 结果

- **Spec coverage:**
  - `Path` / `Open` / `MigrateLegacy` / `Init` 保留 → Task 1。
  - App 新增 `dataDir`/`currentDBPath`、`New` 返回 error、`DB()`、内部改用 `a.DB()`、`Restart` 切库+关旧库+释放锁后 Backfill → Task 2。
  - Handlers 去掉 `DB` 字段、全部改 `h.App.DB()`、`main.go` 不再传 DB → Task 3、Task 4。
  - 空 `user_id` 用旧库、不迁移；目标存在不覆盖 → Task 1 测试 + Task 2 `TestEmptyUserIDUsesLegacyPath`。
  - 切库后自动 Backfill、前端不改动 → Task 2 + Task 5 手动验证。
- **Placeholder scan:** 无 TBD/TODO；每个代码步骤含完整代码。
- **Type consistency:** `db.Path(dataDir, userID)`、`db.Open(path)`、`db.MigrateLegacy(dataDir, userID)`、`app.New(cfg, dataDir, configPath)`、`(*App).DB()`、`Handlers{App, Config, ConfigPath}` 在 Task 1–4 间签名一致；`currentDBPath` 字段名在 Task 2 定义并被 Task 2 测试断言。
- **风险对应:** 设计文档第 6 节的「切库瞬间在途查询」风险未做引用计数（符合 spec 的「可接受风险」）；`Restart` 显式先解锁再 `Backfill()`，避免 `DB()`/`Client()` 二次加锁死锁。
