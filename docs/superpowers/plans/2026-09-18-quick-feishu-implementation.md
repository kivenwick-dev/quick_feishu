# QuickFeishu 实施计划（实施任务进度文档）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. This document IS the progress tracker — update checkbox states and the progress table as tasks complete.

**Goal:** 构建一个单二进制工具，调用 QuickRouter 三接口每日采集快照，通过飞书交互卡片发送定制化日报。

**Architecture:** Gin 提供 Web 服务（配置/手动触发/历史查看），cron 定时调度（0点快照+10:30发日报），采集服务编排三接口，全量 JSON 落库 SQLite，日报引擎按模板算差值生成飞书卡片。

**Tech Stack:** Go + Gin + GORM(SQLite 纯Go驱动 glebarez/sqlite) + robfig/cron/v3 + yaml.v3 + Vue3 + Vite + Element Plus

**设计文档:** `docs/superpowers/specs/2026-09-18-quick-feishu-design.md`

---

## 总方针（Agent 与后续对话必须遵守）

1. **断点续接原则**：每次对话结束时，把未完成的步骤 checkbox 保留为 `- [ ]`，更新下方进度表。新对话启动后，先读本文件找到第一个未勾选步骤继续，不重复已完成工作。
2. **TDD 纪律**：每任务按"写失败测试 → 跑通 → 实现 → 跑通 → 提交"顺序执行，不允许先实现后补测试。
3. **提交纪律**：每任务完成即 commit，commit message 用 `feat:`/`fix:`/`test:` 前缀，遵循 Go 社区惯例。未获用户明确要求不得 push。
4. **配置纪律**：敏感信息（system_token/webhook）默认走 YAML，支持 `QR_USER_ID`/`QR_SYSTEM_TOKEN`/`QR_FEISHU_WEBHOOK` 环境变量覆盖；提交代码不得包含真实密钥。
5. **存储纪律**：三接口响应全字段 JSON 原样落库，冗余索引列仅作加速，字段字典表内置但可编辑。
6. **容错纪律**：单接口失败重试3次(2s/4s/8s)不阻断其他；快照缺失向前替代并标注；飞书失败重试后记 send_logs。
7. **定位问题**：出现 bug 时，在任务清单中定位到具体任务点（Task N → Step M），优先修复该点，不跨任务重写。
8. **实现环境**：Go 需 ≥1.21，Node ≥18（本机 v24.15.0），交叉编译目标 darwin/linux/windows amd64。

---

## 进度跟踪表（Agent 维护，完成任务后立即更新）

| Phase | 任务 | 状态 | 完成日期 |
| --- | --- | --- | --- |
| 0 | Task 1 项目脚手架 | ✅ | 2026-09-18 |
| 0 | Task 2 配置模块 | ✅ | 2026-09-18 |
| 0 | Task 3 数据模型+建表 | ✅ | 2026-09-18 |
| 0 | Task 4 内置字段字典 | ✅ | 2026-09-18 |
| 1 | Task 5 HTTP客户端封装 | ✅ | 2026-09-18 |
| 1 | Task 6 三接口客户端 | ✅ | 2026-09-18 |
| 1 | Task 7 采集服务 | ✅ | 2026-09-18 |
| 2 | Task 8 快照存储 | ⬜ | |
| 3 | Task 9 日报模板解析 | ⬜ | |
| 3 | Task 10 差值计算 | ⬜ | |
| 3 | Task 11 卡片生成 | ⬜ | |
| 4 | Task 12 飞书推送 | ⬜ | |
| 5 | Task 13 定时调度 | ⬜ | |
| 5 | Task 14 补采机制 | ⬜ | |
| 6 | Task 15 Gin服务+路由 | ⬜ | |
| 6 | Task 16 API接口 | ⬜ | |
| 7 | Task 17 Vue脚手架 | ⬜ | |
| 7 | Task 18 前端4页面 | ⬜ | |
| 8 | Task 19 构建打包 | ⬜ | |
| 9 | Task 20 测试完善 | ⬜ | |
| 9 | Task 21 端到端验证 | ⬜ | |

### 实施记录 / 变更日志（Agent 维护）

| 日期 | 分支/提交 | 说明 |
| --- | --- | --- |
| 2026-09-18 | feature/quick-feishu | 分支创建，基于 main 的文档提交 |
| 2026-09-18 | e084c9b / b31207a | Task 1：脚手架 + .gitignore + 目录 .gitkeep |
| 2026-09-18 | dfc56a5 / e386e6d | Task 2：配置模块 + 读取错误处理修正 |
| 2026-09-18 | b7f3388 / b0116c1 | Task 3：GORM 模型与迁移 + go.mod tidy |
| 2026-09-18 | 583cce2 | Task 4：内置字段字典（GORM Create 需指针） |
| 2026-09-18 | 2f9a9bd | Task 5：HTTP 客户端（含 sleepFn 测试钩子） |
| 2026-09-18 | 1d18469 / 14e5709 | Task 6：三接口客户端 + 分页/鉴权测试补强 |
| 2026-09-18 | 5e8ab2b | Task 7：采集服务 |
| 2026-09-18 | **5fe9cc4** | **架构修正（见下方"重要变更：方案A 存全字段"）** |
| 2026-09-18 | 29e002d | 分页死循环防护 + Save nil 防护 |

### 重要变更：方案A「存全字段」（2026-09-18 用户确认）

**背景**：原计划把接口响应解析进结构体后再 `json.Marshal` 结构体落库，会丢失结构体未定义的字段，不满足"存全"要求。

**变更**：API 客户端改为捕获**原始 `data` 子对象**的完整 JSON（全字段），采集层直接存储原始字节：
- `AccountData.Raw` / `TokenUsageData.Raw` = 原始 `data` 对象
- `TokenItem.Raw` = 每令牌原始 JSON；`TokenListData.Raw` = 汇总 `{total, items:[raw...]}`
- `collector.Save` 存储上述原始字节，不再重新 marshal 结构体
- 字段路径保持顶层（如 `used_quota`），Task 10 的差值提取逻辑**无需改动**

**对后续任务的影响**：Task 8/10/16 读取快照时，`AccountRaw`/`TokenListRaw`/`UsageRaw` 均为全字段原始 JSON（`data` 对象），字段路径为顶层。

---

## Task 1: 项目脚手架

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `internal/` 目录结构

- [ ] **Step 1: 初始化 Go 模块与目录**

```bash
cd /Users/john/Documents/My_Work/quick_feishu
go mod init quick-feishu
mkdir -p cmd internal/{config,api,model,db,collector,report,feishu,scheduler,server,frontend} web data
```

- [ ] **Step 2: 编写入口 main.go（命令分发骨架）**

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		args = []string{"serve"}
	}
	switch args[0] {
	case "serve":
		fmt.Println("serve mode (TODO)")
	case "run-report":
		fmt.Println("run-report mode (TODO)")
	case "snapshot":
		fmt.Println("snapshot mode (TODO)")
	default:
		fmt.Println("usage: quick-feishu [serve|run-report|snapshot]")
		os.Exit(1)
	}
}
```

- [ ] **Step 3: 编译验证**

Run: `go build ./...`
Expected: 无报错，`main.go` 通过编译

- [ ] **Step 4: Commit**

```bash
git add go.mod main.go
git commit -m "feat: scaffold project structure and CLI entry"
```

---

## Task 2: 配置模块

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/default.yaml`
- Test: `internal/config/config_test.go`

- [ ] **Step 1: 定义配置结构体与默认 YAML**

`internal/config/config.go`:

```go
package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App    AppConfig    `yaml:"app"`
	Account AccountConfig `yaml:"account"`
	Feishu FeishuConfig `yaml:"feishu"`
	Schedule ScheduleConfig `yaml:"schedule"`
	ReportTemplate map[string]interface{} `yaml:"report_template"`
}

type AppConfig struct {
	Port     int    `yaml:"port"`
	Timezone string `yaml:"timezone"`
}

type AccountConfig struct {
	UserID      string `yaml:"user_id"`
	SystemToken string `yaml:"system_token"`
	APIBase     string `yaml:"api_base"`
}

type FeishuConfig struct {
	WebhookURL string `yaml:"webhook_url"`
	RetryTimes int    `yaml:"retry_times"`
}

type ScheduleConfig struct {
	SnapshotTime string `yaml:"snapshot_time"`
	ReportTime   string `yaml:"report_time"`
}

func Default() *Config {
	return &Config{
		App:    AppConfig{Port: 8080, Timezone: "Asia/Shanghai"},
		Account: AccountConfig{APIBase: "https://api.quickrouter.ai"},
		Feishu: FeishuConfig{RetryTimes: 3},
		Schedule: ScheduleConfig{SnapshotTime: "00:00", ReportTime: "10:30"},
	}
}

// Load 读取 YAML，不存在则生成默认配置，再应用环境变量覆盖
func Load(path string) (*Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	} else {
		out, _ := yaml.Marshal(cfg)
		if err := os.WriteFile(path, out, 0644); err != nil {
			return nil, err
		}
	}
	cfg.ApplyEnv()
	return cfg, nil
}

func (c *Config) ApplyEnv() {
	if v := os.Getenv("QR_USER_ID"); v != "" {
		c.Account.UserID = v
	}
	if v := os.Getenv("QR_SYSTEM_TOKEN"); v != "" {
		c.Account.SystemToken = v
	}
	if v := os.Getenv("QR_FEISHU_WEBHOOK"); v != "" {
		c.Feishu.WebhookURL = v
	}
}

// Save 写回 YAML（Web 设置页保存时调用）
func (c *Config) Save(path string) error {
	out, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0644)
}

// EnvOverridden 返回被环境变量覆盖的字段名集合
func (c *Config) EnvOverridden() map[string]bool {
	m := map[string]bool{}
	if os.Getenv("QR_USER_ID") != "" {
		m["user_id"] = true
	}
	if os.Getenv("QR_SYSTEM_TOKEN") != "" {
		m["system_token"] = true
	}
	if os.Getenv("QR_FEISHU_WEBHOOK") != "" {
		m["webhook_url"] = true
	}
	return m
}
```

- [ ] **Step 2: 添加依赖**

Run: `go get gopkg.in/yaml.v3`

- [ ] **Step 3: 编写测试**

`internal/config/config_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGeneratesDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.App.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.App.Port)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("default config should be written: %v", err)
	}
}

func TestLoadReadsExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte("account:\n  user_id: \"123\"\n"), 0644)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Account.UserID != "123" {
		t.Errorf("expected 123, got %s", cfg.Account.UserID)
	}
}

func TestEnvOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.Setenv("QR_USER_ID", "env-user")
	defer os.Unsetenv("QR_USER_ID")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Account.UserID != "env-user" {
		t.Errorf("expected env override, got %s", cfg.Account.UserID)
	}
	if !cfg.EnvOverridden()["user_id"] {
		t.Error("user_id should be marked env-overridden")
	}
}

func TestSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := Default()
	cfg.Account.UserID = "999"
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}
	reloaded, _ := Load(path)
	if reloaded.Account.UserID != "999" {
		t.Errorf("expected 999 after save, got %s", reloaded.Account.UserID)
	}
}
```

- [ ] **Step 4: 跑测试确认**

Run: `go test ./internal/config/ -v`
Expected: 4 个测试全部 PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: config module with YAML load and env override"
```

---

## Task 3: 数据模型与建表

**Files:**
- Create: `internal/model/snapshot.go`
- Create: `internal/model/dict.go`
- Create: `internal/db/db.go`
- Test: `internal/db/db_test.go`

- [ ] **Step 1: 定义 GORM 模型**

`internal/model/snapshot.go`:

```go
package model

import (
	"time"

	"gorm.io/datatypes"
)

type Snapshot struct {
	ID            uint           `gorm:"primaryKey"`
	SnapshotDate  string         `gorm:"index"`
	AccountRaw    datatypes.JSON
	TokenListRaw  datatypes.JSON
	TokenUsageRaw datatypes.JSON
	AccountQuota  int64
	AccountUsed   int64
	RequestCount  int64
	CreatedAt     time.Time
}

type TokenSnapshot struct {
	ID           uint   `gorm:"primaryKey"`
	SnapshotID   uint   `gorm:"index"`
	TokenID      int
	TokenName    string
	UsageRaw     datatypes.JSON
	ListRaw      datatypes.JSON
	UsedQuota    int64
	RemainQuota  int64
	TotalUsed    int64
	TotalGranted int64
}

type SendLog struct {
	ID         uint      `gorm:"primaryKey"`
	SendTime   time.Time
	Date       string
	Success    bool
	ErrorMsg   string
	FeishuResp string
}
```

`internal/model/dict.go`:

```go
package model

type DictAccountField struct {
	ID          uint   `gorm:"primaryKey"`
	FieldPath   string `gorm:"index"`
	Label       string
	FieldType   string
	Description string
	IsDefault   bool
}

type DictTokenField struct {
	ID          uint   `gorm:"primaryKey"`
	FieldPath   string `gorm:"index"`
	Label       string
	FieldType   string
	Description string
	IsDefault   bool
}

type DictUsageField struct {
	ID          uint   `gorm:"primaryKey"`
	FieldPath   string `gorm:"index"`
	Label       string
	FieldType   string
	Description string
	IsDefault   bool
}
```

- [ ] **Step 2: 数据库初始化**

`internal/db/db.go`:

```go
package db

import (
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"quick-feishu/internal/model"
)

func Init(dataDir string) (*gorm.DB, error) {
	path := filepath.Join(dataDir, "quick-feishu.db")
	gdb, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
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
```

- [ ] **Step 3: 添加依赖**

Run: `go get github.com/glebarez/sqlite gorm.io/gorm gorm.io/datatypes`

- [ ] **Step 4: 建表测试（内存库）**

`internal/db/db_test.go`:

```go
package db

import (
	"path/filepath"
	"testing"

	"gorm.io/gorm"
	"quick-feishu/internal/model"
)

func TestInitMigrations(t *testing.T) {
	dir := t.TempDir()
	gdb, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	var snapshots []model.Snapshot
	if err := gdb.AutoMigrate(&model.Snapshot{}); err == nil {
		// table should already exist
	}
	_ = gdb
	_ = snapshots
	if err := gdb.Exec("SELECT name FROM sqlite_master WHERE type='table' AND name='snapshots'").Error; err != nil {
		t.Fatalf("snapshots table not created: %v", err)
	}
	if _, err := gorm.Open(nil, &gorm.Config{}); err == nil {
		t.Fatal("unreachable")
	}
	_ = filepath.Join
}
```

- [ ] **Step 5: 跑测试确认**

Run: `go test ./internal/db/ -v`
Expected: PASS（snapshots 表存在）

- [ ] **Step 6: Commit**

```bash
git add internal/model/ internal/db/
git commit -m "feat: GORM models and SQLite migrations"
```

---

## Task 4: 内置字段字典

**Files:**
- Create: `internal/db/seed.go`
- Test: `internal/db/seed_test.go`

- [ ] **Step 1: 编写内置字典种子数据**

`internal/db/seed.go`:

```go
package db

import (
	"gorm.io/gorm"
	"quick-feishu/internal/model"
)

type dictSeed struct {
	path string
	typ  string
	label string
	desc string
}

var accountSeeds = []dictSeed{
	{"id", "int", "账号ID", ""},
	{"username", "string", "用户名", ""},
	{"display_name", "string", "显示名", ""},
	{"role", "int", "角色", "1=用户 2=管理员"},
	{"status", "int", "状态", "1=启用"},
	{"quota", "int", "总配额", ""},
	{"used_quota", "int", "已用配额", ""},
	{"request_count", "int", "请求次数", ""},
	{"group_id", "int", "分组ID", ""},
	{"group", "string", "分组名", ""},
	{"created_at", "int", "创建时间", "Unix时间戳"},
	{"email", "string", "邮箱", ""},
}

var tokenSeeds = []dictSeed{
	{"id", "int", "令牌ID", ""},
	{"user_id", "int", "所属用户ID", ""},
	{"key", "string", "令牌Key", "用于usage接口认证"},
	{"status", "int", "状态", "1=启用"},
	{"name", "string", "令牌名称", ""},
	{"created_time", "int", "创建时间", ""},
	{"accessed_time", "int", "最后访问时间", ""},
	{"expired_time", "int", "过期时间", "-1=永不过期"},
	{"remain_quota", "int", "剩余配额", ""},
	{"unlimited_quota", "bool", "无限配额", ""},
	{"model_limits_enabled", "bool", "模型限制开关", ""},
	{"model_limits", "string", "模型限制", ""},
	{"allow_ips", "string", "允许IP", ""},
	{"used_quota", "int", "已用配额", ""},
	{"group_ids", "array", "分组ID列表", ""},
	{"group", "string", "分组名", ""},
}

var usageSeeds = []dictSeed{
	{"expires_at", "int", "过期时间", "0=未设置"},
	{"model_limits", "object", "模型限制", ""},
	{"model_limits_enabled", "bool", "模型限制开关", ""},
	{"name", "string", "令牌名称", ""},
	{"object", "string", "对象类型", "固定 token_usage"},
	{"total_available", "int", "可用总量", ""},
	{"total_granted", "int", "授予总量", ""},
	{"total_used", "int", "累计已用", ""},
	{"unlimited_quota", "bool", "无限配额", ""},
}

// SeedDicts 仅当字典表为空时填充内置字段
func SeedDicts(gdb *gorm.DB) error {
	if err := seedIfEmpty(gdb, &model.DictAccountField{}, accountSeeds,
		func(p, t, l, d string) interface{} {
			return model.DictAccountField{FieldPath: p, FieldType: t, Label: l, Description: d, IsDefault: true}
		}); err != nil {
		return err
	}
	if err := seedIfEmpty(gdb, &model.DictTokenField{}, tokenSeeds,
		func(p, t, l, d string) interface{} {
			return model.DictTokenField{FieldPath: p, FieldType: t, Label: l, Description: d, IsDefault: true}
		}); err != nil {
		return err
	}
	if err := seedIfEmpty(gdb, &model.DictUsageField{}, usageSeeds,
		func(p, t, l, d string) interface{} {
			return model.DictUsageField{FieldPath: p, FieldType: t, Label: l, Description: d, IsDefault: true}
		}); err != nil {
		return err
	}
	return nil
}

func seedIfEmpty(gdb *gorm.DB, dest interface{}, seeds []dictSeed, make func(p, t, l, d string) interface{}) error {
	var count int64
	if err := gdb.Model(dest).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	for _, s := range seeds {
		if err := gdb.Create(make(s.path, s.typ, s.label, s.desc)).Error; err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 2: 种子测试**

`internal/db/seed_test.go`:

```go
package db

import (
	"testing"

	"quick-feishu/internal/model"
)

func TestSeedDicts(t *testing.T) {
	dir := t.TempDir()
	gdb, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := SeedDicts(gdb); err != nil {
		t.Fatal(err)
	}
	var ac []model.DictAccountField
	gdb.Find(&ac)
	if len(ac) != len(accountSeeds) {
		t.Errorf("account seeds = %d, want %d", len(ac), len(accountSeeds))
	}
	// 二次调用不重复
	if err := SeedDicts(gdb); err != nil {
		t.Fatal(err)
	}
	gdb.Find(&ac)
	if len(ac) != len(accountSeeds) {
		t.Errorf("second seed should be no-op, got %d", len(ac))
	}
}
```

- [ ] **Step 3: 跑测试确认**

Run: `go test ./internal/db/ -v`
Expected: 全部 PASS

- [ ] **Step 4: Commit**

```bash
git add internal/db/
git commit -m "feat: built-in field dictionaries for three APIs"
```

---

## Task 5: HTTP 客户端封装

**Files:**
- Create: `internal/api/client.go`
- Test: `internal/api/client_test.go`

- [ ] **Step 1: 封装 HTTP 客户端**

`internal/api/client.go`:

```go
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const DefaultBase = "https://api.quickrouter.ai"

type Client struct {
	BaseURL     string
	HTTPClient  *http.Client
	SystemToken string
	UserID      string
}

func NewClient(baseURL, systemToken, userID string) *Client {
	return &Client{
		BaseURL:     baseURL,
		SystemToken: systemToken,
		UserID:      userID,
		HTTPClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

// doGet 发起 GET 请求，baseHeaders 附加到系统令牌之后
func (c *Client) doGet(path string, query map[string]string, extraHeaders map[string]string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	for k, v := range query {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func parseJSON(body []byte, v interface{}) error {
	return json.Unmarshal(body, v)
}

// WithRetry 带退避重试执行 fn（retries 次，2s/4s/8s...）
func WithRetry(retries int, fn func() ([]byte, error)) ([]byte, error) {
	var lastErr error
	for i := 0; i <= retries; i++ {
		var body []byte
		body, lastErr = fn()
		if lastErr == nil {
			return body, nil
		}
		if i < retries {
			time.Sleep(time.Duration(1<<uint(i)) * 2 * time.Second)
		}
	}
	return nil, lastErr
}
```

- [ ] **Step 2: 测试（httptest mock）**

`internal/api/client_test.go`:

```go
package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDoGetHeaders(t *testing.T) {
	var gotAuth, gotUser string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotUser = r.Header.Get("new-api-user")
		w.Write([]byte(`{"success":true}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "sys-token", "827947")
	_, err := c.doGet("/api/user/self", nil, map[string]string{"new-api-user": c.UserID, "Authorization": "Bearer " + c.SystemToken})
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer sys-token" {
		t.Errorf("auth = %s", gotAuth)
	}
	if gotUser != "827947" {
		t.Errorf("new-api-user = %s", gotUser)
	}
}

func TestWithRetrySucceeds(t *testing.T) {
	calls := 0
	fn := func() ([]byte, error) {
		calls++
		if calls < 2 {
			return nil, fmt.Errorf("boom")
		}
		return []byte(`ok`), nil
	}
	body, err := WithRetry(3, fn)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "ok" {
		t.Errorf("body = %s", body)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
}
```

- [ ] **Step 3: 补充 import 并跑测试**

`client_test.go` 需加 `"fmt"` import，然后：

Run: `go test ./internal/api/ -v`
Expected: 2 个测试 PASS

- [ ] **Step 4: Commit**

```bash
git add internal/api/client.go internal/api/client_test.go
git commit -m "feat: HTTP client wrapper with retry"
```

---

## Task 6: 三接口客户端

> **✅ 已完成（2026-09-18）** — commit 1d18469，测试补强 14e5709。
> **⚠️ 代码已按「方案A」更新**：下方示例代码为初稿，实际实现改为捕获**原始 `data` 子对象的全字段 JSON**（见文首「重要变更：方案A」）。实际实现以 `internal/api/*.go`（commit 5fe9cc4）为准：`AccountData.Raw`/`TokenUsageData.Raw` 存原始 `data` 对象，`TokenItem.Raw` 存每项原始 JSON，`TokenListData.Raw` 存 `{total, items:[raw...]}`。

**Files:**
- Create: `internal/api/account.go`
- Create: `internal/api/tokenlist.go`
- Create: `internal/api/usage.go`
- Test: `internal/api/account_test.go`, `internal/api/tokenlist_test.go`, `internal/api/usage_test.go`

- [ ] **Step 1: 账号信息接口**

`internal/api/account.go`:

```go
package api

type AccountResp struct {
	Data    AccountData `json:"data"`
	Message string      `json:"message"`
	Success bool        `json:"success"`
}

type AccountData struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	DisplayName  string `json:"display_name"`
	Role         int    `json:"role"`
	Status       int    `json:"status"`
	Quota        int64  `json:"quota"`
	UsedQuota    int64  `json:"used_quota"`
	RequestCount int64  `json:"request_count"`
	GroupID      int    `json:"group_id"`
	Group        string `json:"group"`
	CreatedAt    int64  `json:"created_at"`
	Raw          json.RawMessage `json:"-"`
}

// GetAccount 获取账号信息，返回原始 JSON 供全量存储
func (c *Client) GetAccount() (*AccountData, []byte, error) {
	headers := map[string]string{
		"new-api-user": c.UserID,
		"Authorization": "Bearer " + c.SystemToken,
	}
	body, err := WithRetry(3, func() ([]byte, error) {
		return c.doGet("/api/user/self", nil, headers)
	})
	if err != nil {
		return nil, nil, err
	}
	var resp AccountResp
	if err := parseJSON(body, &resp); err != nil {
		return nil, body, err
	}
	resp.Data.Raw = body
	return &resp.Data, body, nil
}
```

- [ ] **Step 2: 令牌列表接口**

`internal/api/tokenlist.go`:

```go
package api

type TokenListResp struct {
	Data    TokenListData `json:"data"`
	Message string        `json:"message"`
	Success bool          `json:"success"`
}

type TokenListData struct {
	Page      int         `json:"page"`
	PageSize  int         `json:"page_size"`
	Total     int         `json:"total"`
	Items     []TokenItem `json:"items"`
}

type TokenItem struct {
	ID                 int      `json:"id"`
	UserID             int      `json:"user_id"`
	Key                string   `json:"key"`
	Status             int      `json:"status"`
	Name               string   `json:"name"`
	CreatedTime        int64    `json:"created_time"`
	AccessedTime       int64    `json:"accessed_time"`
	ExpiredTime        int64    `json:"expired_time"`
	RemainQuota        int64    `json:"remain_quota"`
	UnlimitedQuota     bool     `json:"unlimited_quota"`
	ModelLimitsEnabled bool     `json:"model_limits_enabled"`
	ModelLimits        string   `json:"model_limits"`
	AllowIPs           string   `json:"allow_ips"`
	UsedQuota          int64    `json:"used_quota"`
	GroupIDs           []int    `json:"group_ids"`
	Group              string   `json:"group"`
	Raw                json.RawMessage `json:"-"`
}

// GetTokenList 获取令牌列表（分页拉全）
func (c *Client) GetTokenList() (*TokenListData, []byte, error) {
	headers := map[string]string{
		"new-api-user": c.UserID,
		"Authorization": "Bearer " + c.SystemToken,
	}
	page := 0
	all := &TokenListData{Items: []TokenItem{}}
	var lastRaw []byte
	for {
		query := map[string]string{"p": itoa(page), "size": "100"}
		body, err := WithRetry(3, func() ([]byte, error) {
			return c.doGet("/api/token/", query, headers)
		})
		if err != nil {
			return all, lastRaw, err
		}
		var resp TokenListResp
		if err := parseJSON(body, &resp); err != nil {
			return all, body, err
		}
		all.Items = append(all.Items, resp.Data.Items...)
		all.Total = resp.Data.Total
		all.PageSize = resp.Data.PageSize
		lastRaw = body
		if len(resp.Data.Items) == 0 || page*resp.Data.PageSize >= resp.Data.Total {
			break
		}
		page++
	}
	return all, lastRaw, nil
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}
```

- [ ] **Step 3: 令牌使用情况接口**

`internal/api/usage.go`:

```go
package api

type TokenUsageResp struct {
	Data    TokenUsageData `json:"data"`
	Message string         `json:"message"`
	Success bool           `json:"success"`
}

type TokenUsageData struct {
	ExpiresAt          int64           `json:"expires_at"`
	ModelLimits        json.RawMessage `json:"model_limits"`
	ModelLimitsEnabled bool            `json:"model_limits_enabled"`
	Name               string          `json:"name"`
	Object             string          `json:"object"`
	TotalAvailable     int64           `json:"total_available"`
	TotalGranted       int64           `json:"total_granted"`
	TotalUsed          int64           `json:"total_used"`
	UnlimitedQuota     bool            `json:"unlimited_quota"`
	Raw                json.RawMessage `json:"-"`
}

// GetTokenUsage 获取单个令牌使用情况，tokenKey 是令牌自身 key
func (c *Client) GetTokenUsage(tokenKey string) (*TokenUsageData, []byte, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + tokenKey,
	}
	body, err := WithRetry(3, func() ([]byte, error) {
		return c.doGet("/api/usage/token/", nil, headers)
	})
	if err != nil {
		return nil, nil, err
	}
	var resp TokenUsageResp
	if err := parseJSON(body, &resp); err != nil {
		return nil, body, err
	}
	resp.Data.Raw = body
	return &resp.Data, body, nil
}
```

- [ ] **Step 4: 三接口 mock 测试**

`internal/api/account_test.go`:

```go
package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetAccount(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/user/self" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Write([]byte(`{"data":{"id":827947,"quota":100,"used_quota":50,"request_count":10},"message":"","success":true}`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "sys-token", "827947")
	d, raw, err := c.GetAccount()
	if err != nil {
		t.Fatal(err)
	}
	if d.Quota != 100 || d.UsedQuota != 50 {
		t.Errorf("bad data: %+v", d)
	}
	if len(raw) == 0 {
		t.Error("raw should not be empty")
	}
}
```

`internal/api/tokenlist_test.go`:

```go
package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetTokenList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":{"page":1,"page_size":2,"total":2,"items":[
			{"id":1,"key":"k1","name":"claude","used_quota":10,"group":"G1"},
			{"id":2,"key":"k2","name":"gpt","used_quota":20,"group":"G2"}
		]},"message":"","success":true}`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "sys-token", "827947")
	d, _, err := c.GetTokenList()
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Items) != 2 || d.Items[0].Key != "k1" {
		t.Errorf("bad items: %+v", d.Items)
	}
}
```

`internal/api/usage_test.go`:

```go
package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetTokenUsage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token-key" {
			t.Errorf("usage must use token key as bearer, got %s", r.Header.Get("Authorization"))
		}
		w.Write([]byte(`{"data":{"name":"claude","total_used":888,"total_granted":1000,"unlimited_quota":true},"message":"","success":true}`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "sys-token", "827947")
	d, _, err := c.GetTokenUsage("token-key")
	if err != nil {
		t.Fatal(err)
	}
	if d.TotalUsed != 888 {
		t.Errorf("total_used = %d", d.TotalUsed)
	}
}
```

- [ ] **Step 5: 补充 import（fmt/json）并跑测试**

`account.go`/`tokenlist.go`/`usage.go` 需加 `"encoding/json"`，`tokenlist.go` 加 `"fmt"`。

Run: `go test ./internal/api/ -v`
Expected: 5 个测试全部 PASS

- [ ] **Step 6: Commit**

```bash
git add internal/api/
git commit -m "feat: three QuickRouter API clients"
```

---

## Task 7: 采集服务

> **✅ 已完成（2026-09-18）** — commit 5e8ab2b，方案A 修正 5fe9cc4，健壮性加固 29e002d。
> **⚠️ 代码已按「方案A」更新**：`Save` 现直接存储客户端捕获的**全字段原始 JSON**（`res.Account.Raw`/`res.TokenList.Raw`/`it.Raw`/`u.Raw`），不再 `json.Marshal` 结构体。`Collect` 逻辑不变。实际实现以 `internal/collector/collector.go` 为准。

**Files:**
- Create: `internal/collector/collector.go`
- Test: `internal/collector/collector_test.go`

- [ ] **Step 1: 编排三接口采集**

`internal/collector/collector.go`:

```go
package collector

import (
	"encoding/json"

	"gorm.io/gorm"
	"quick-feishu/internal/api"
	"quick-feishu/internal/model"
)

type Result struct {
	Account   *api.AccountData
	TokenList *api.TokenListData
	Usages    map[int]*api.TokenUsageData // tokenID -> usage
	Errors    []string
}

// Collect 依次调用三接口，单接口失败不阻断其余
func Collect(c *api.Client) *Result {
	res := &Result{Usages: map[int]*api.TokenUsageData{}, Errors: []string{}}

	account, _, err := c.GetAccount()
	if err != nil {
		res.Errors = append(res.Errors, "account: "+err.Error())
	} else {
		res.Account = account
	}

	list, _, err := c.GetTokenList()
	if err != nil {
		res.Errors = append(res.Errors, "tokenlist: "+err.Error())
		res.TokenList = &api.TokenListData{Items: []api.TokenItem{}}
	} else {
		res.TokenList = list
		for _, it := range list.Items {
			usage, _, uerr := c.GetTokenUsage(it.Key)
			if uerr != nil {
				res.Errors = append(res.Errors, "usage("+it.Name+"): "+uerr.Error())
				continue
			}
			res.Usages[it.ID] = usage
		}
	}
	return res
}

// Save 将采集结果写入快照表
func Save(gdb *gorm.DB, date string, res *Result) error {
	accountRaw, _ := json.Marshal(res.Account)
	tokenListRaw, _ := json.Marshal(res.TokenList)
	usageMap := map[string]*api.TokenUsageData{}
	for id, u := range res.Usages {
		usageMap[itoa(id)] = u
	}
	usageRaw, _ := json.Marshal(usageMap)

	snap := &model.Snapshot{
		SnapshotDate:  date,
		AccountRaw:    accountRaw,
		TokenListRaw:  tokenListRaw,
		TokenUsageRaw: usageRaw,
	}
	if res.Account != nil {
		snap.AccountQuota = res.Account.Quota
		snap.AccountUsed = res.Account.UsedQuota
		snap.RequestCount = res.Account.RequestCount
	}
	if err := gdb.Create(snap).Error; err != nil {
		return err
	}
	for _, it := range res.TokenList.Items {
		ts := &model.TokenSnapshot{
			SnapshotID:  snap.ID,
			TokenID:     it.ID,
			TokenName:   it.Name,
			UsedQuota:   it.UsedQuota,
			RemainQuota: it.RemainQuota,
		}
		listRaw, _ := json.Marshal(it)
		ts.ListRaw = listRaw
		if u, ok := res.Usages[it.ID]; ok {
			ts.TotalUsed = u.TotalUsed
			ts.TotalGranted = u.TotalGranted
			ts.UsageRaw = u.Raw
		}
		if err := gdb.Create(ts).Error; err != nil {
			return err
		}
	}
	return nil
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}
```

- [ ] **Step 2: 采集测试（mock 服务器）**

`internal/collector/collector_test.go`:

```go
package collector

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"quick-feishu/internal/api"
	"quick-feishu/internal/db"
)

func TestCollectAndSave(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/user/self":
			w.Write([]byte(`{"data":{"id":1,"quota":1000,"used_quota":300,"request_count":5},"success":true}`))
		case "/api/token/":
			w.Write([]byte(`{"data":{"page":1,"page_size":2,"total":2,"items":[
				{"id":11,"key":"k1","name":"claude","used_quota":100,"remain_quota":0},
				{"id":12,"key":"k2","name":"gpt","used_quota":200,"remain_quota":0}
			]},"success":true}`))
		case "/api/usage/token/":
			w.Write([]byte(`{"data":{"name":"x","total_used":100,"total_granted":1000},"success":true}`))
		}
	}))
	defer srv.Close()

	c := api.NewClient(srv.URL, "sys-token", "1")
	res := Collect(c)
	if len(res.Usages) != 2 {
		t.Errorf("usages = %d, want 2", len(res.Usages))
	}
	if res.Account == nil || res.Account.Quota != 1000 {
		t.Errorf("account bad: %+v", res.Account)
	}

	dir := t.TempDir()
	gdb, err := db.Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := Save(gdb, "2026-09-18", res); err != nil {
		t.Fatal(err)
	}
	var snap model.Snapshot
	gdb.First(&snap)
	if snap.AccountUsed != 300 {
		t.Errorf("snapshot used = %d", snap.AccountUsed)
	}
	var tokens []model.TokenSnapshot
	gdb.Where("snapshot_id = ?", snap.ID).Find(&tokens)
	if len(tokens) != 2 {
		t.Errorf("tokens = %d, want 2", len(tokens))
	}
}
```

- [ ] **Step 3: 补充 import 并跑测试**

`collector.go` 加 `"fmt"`，`collector_test.go` 加 `"quick-feishu/internal/model"`。

Run: `go test ./internal/collector/ -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/collector/
git commit -m "feat: collector service orchestrating three APIs"
```

---

## Task 8: 快照存储与查询

**Files:**
- Create: `internal/db/querys.go`
- Test: `internal/db/querys_test.go`

- [ ] **Step 1: 快照查询与差值基础**

`internal/db/querys.go`:

```go
package db

import (
	"gorm.io/gorm"
	"quick-feishu/internal/model"
)

// LatestSnapshot 取最近一条快照
func LatestSnapshot(gdb *gorm.DB) (*model.Snapshot, error) {
	var s model.Snapshot
	err := gdb.Order("snapshot_date DESC, id DESC").First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// SnapshotByDate 按日期取快照
func SnapshotByDate(gdb *gorm.DB, date string) (*model.Snapshot, error) {
	var s model.Snapshot
	err := gdb.Where("snapshot_date = ?", date).First(&s).Error
	return &s, err
}

// SnapshotBefore 取指定日期前（含）最近一条快照
func SnapshotBefore(gdb *gorm.DB, date string) (*model.Snapshot, error) {
	var s model.Snapshot
	err := gdb.Where("snapshot_date <= ?", date).Order("snapshot_date DESC, id DESC").First(&s).Error
	return &s, err
}

// TokenSnapshots 取某快照下的令牌列表
func TokenSnapshots(gdb *gorm.DB, snapshotID uint) ([]model.TokenSnapshot, error) {
	var list []model.TokenSnapshot
	err := gdb.Where("snapshot_id = ?", snapshotID).Find(&list).Error
	return list, err
}

// SnapshotsInRange 取日期区间内的快照（升序）
func SnapshotsInRange(gdb *gorm.DB, from, to string) ([]model.Snapshot, error) {
	var list []model.Snapshot
	err := gdb.Where("snapshot_date >= ? AND snapshot_date <= ?", from, to).
		Order("snapshot_date ASC").Find(&list).Error
	return list, err
}

// ListSnapshots 分页列出快照
func ListSnapshots(gdb *gorm.DB, page, size int) ([]model.Snapshot, int64, error) {
	var list []model.Snapshot
	var total int64
	gdb.Model(&model.Snapshot{}).Count(&total)
	err := gdb.Order("snapshot_date DESC").Offset((page - 1) * size).Limit(size).Find(&list).Error
	return list, total, err
}

// ListSendLogs 分页列出发送日志
func ListSendLogs(gdb *gorm.DB, page, size int) ([]model.SendLog, int64, error) {
	var list []model.SendLog
	var total int64
	gdb.Model(&model.SendLog{}).Count(&total)
	err := gdb.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&list).Error
	return list, total, err
}
```

- [ ] **Step 2: 查询测试**

`internal/db/querys_test.go`:

```go
package db

import (
	"testing"

	"quick-feishu/internal/model"
)

func TestSnapshotQueries(t *testing.T) {
	dir := t.TempDir()
	gdb, _ := Init(dir)
	gdb.Create(&model.Snapshot{SnapshotDate: "2026-09-17", AccountUsed: 100})
	gdb.Create(&model.Snapshot{SnapshotDate: "2026-09-18", AccountUsed: 150})

	latest, _ := LatestSnapshot(gdb)
	if latest.SnapshotDate != "2026-09-18" {
		t.Errorf("latest = %s", latest.SnapshotDate)
	}
	before, _ := SnapshotBefore(gdb, "2026-09-18")
	if before.AccountUsed != 150 {
		t.Errorf("before 09-18 should be 150, got %d", before.AccountUsed)
	}
	before2, _ := SnapshotBefore(gdb, "2026-09-17")
	if before2.AccountUsed != 100 {
		t.Errorf("before 09-17 should be 100, got %d", before2.AccountUsed)
	}
	missing, err := SnapshotBefore(gdb, "2026-09-16")
	if err != nil {
		t.Errorf("should fallback to latest available: %v", err)
	}
	_ = missing
	inRange, _ := SnapshotsInRange(gdb, "2026-09-17", "2026-09-18")
	if len(inRange) != 2 {
		t.Errorf("range = %d, want 2", len(inRange))
	}
}
```

- [ ] **Step 3: 跑测试确认**

Run: `go test ./internal/db/ -v`
Expected: 全部 PASS（注意 `SnapshotBefore` 空集需处理——测试中 09-16 前无数据会返回错误，与设计"向前替代"由调用方处理一致）

- [ ] **Step 4: Commit**

```bash
git add internal/db/querys.go internal/db/querys_test.go
git commit -m "feat: snapshot and send log queries"
```

---

## Task 9: 日报模板解析

**Files:**
- Create: `internal/report/template.go`
- Test: `internal/report/template_test.go`

- [ ] **Step 1: 模板结构与解析**

`internal/report/template.go`:

```go
package report

import (
	"gopkg.in/yaml.v3"
)

type Template struct {
	Title     string    `yaml:"title"`
	DateMode  string    `yaml:"date_mode"` // auto | manual
	StartDate string    `yaml:"start_date,omitempty"`
	EndDate   string    `yaml:"end_date,omitempty"`
	Sections  []Section `yaml:"sections"`
}

type Section struct {
	Name     string  `yaml:"section"`
	Source   string  `yaml:"source"` // account | token | usage
	PerToken bool    `yaml:"per_token"`
	Fields   []Field `yaml:"fields"`
}

type Field struct {
	Field string `yaml:"field"`
	Diff  bool   `yaml:"diff"`
}

func ParseTemplate(data []byte) (*Template, error) {
	var t Template
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	if t.DateMode == "" {
		t.DateMode = "auto"
	}
	return &t, nil
}

func DefaultTemplate() *Template {
	return &Template{
		Title:    "AI 平台日报",
		DateMode: "auto",
		Sections: []Section{
			{
				Name:   "账号概况",
				Source: "account",
				Fields: []Field{
					{Field: "used_quota", Diff: true},
					{Field: "request_count", Diff: true},
				},
			},
			{
				Name:     "各令牌用量",
				Source:   "usage",
				PerToken: true,
				Fields: []Field{
					{Field: "total_used", Diff: true},
				},
			},
		},
	}
}
```

- [ ] **Step 2: 模板解析测试**

`internal/report/template_test.go`:

```go
package report

import "testing"

func TestParseTemplate(t *testing.T) {
	data := []byte(`
title: 测试日报
date_mode: manual
start_date: "2026-09-17"
end_date: "2026-09-18"
sections:
  - section: 概况
    source: account
    fields:
      - field: used_quota
        diff: true
`)
	tmpl, err := ParseTemplate(data)
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.DateMode != "manual" || tmpl.StartDate != "2026-09-17" {
		t.Errorf("bad template: %+v", tmpl)
	}
	if len(tmpl.Sections) != 1 || tmpl.Sections[0].Fields[0].Field != "used_quota" {
		t.Errorf("bad sections: %+v", tmpl.Sections)
	}
}

func TestDefaultTemplate(t *testing.T) {
	tmpl := DefaultTemplate()
	if tmpl.DateMode != "auto" {
		t.Errorf("default should be auto")
	}
}
```

- [ ] **Step 3: 跑测试确认**

Run: `go test ./internal/report/ -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/report/
git commit -m "feat: report template parsing"
```

---

## Task 10: 差值计算

**Files:**
- Create: `internal/report/diff.go`
- Test: `internal/report/diff_test.go`

- [ ] **Step 1: 从快照取值并计算差值**

`internal/report/diff.go`:

```go
package report

import (
	"encoding/json"
	"fmt"
	"strconv"

	"quick-feishu/internal/model"
)

// extractField 从原始 JSON 中提取字段值（支持嵌套 . 路径）
func extractField(raw []byte, path string) (interface{}, bool) {
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, false
	}
	cur := interface{}(m)
	for _, seg := range splitPath(path) {
		mm, ok := cur.(map[string]interface{})
		if !ok {
			return nil, false
		}
		cur, ok = mm[seg]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

func splitPath(p string) []string {
	var segs []string
	cur := ""
	for _, r := range p {
		if r == '.' {
			segs = append(segs, cur)
			cur = ""
		} else {
			cur += string(r)
		}
	}
	segs = append(segs, cur)
	return segs
}

// DiffResult 单字段差值结果
type DiffResult struct {
	Field string
	Label string
	Diff  bool
	Value string // 展示值（差值或当前值）
	IsDiff bool // 是否实际计算了差值
}

// computeDiff 计算字段差值。早/晚快照可为 nil。
func computeDiff(early, late *model.Snapshot, f Field, label string) (DiffResult, error) {
	// account 源从 AccountRaw 提取；此处按字段路径取 late 值
	var lateVal, earlyVal interface{}
	var ok bool
	res := DiffResult{Field: f.Field, Label: label, Diff: f.Diff}
	if late != nil {
		lateVal, ok = extractField(late.AccountRaw, f.Field)
		if !ok {
			return res, fmt.Errorf("field %s not found", f.Field)
		}
	}
	if !f.Diff {
		res.Value = formatVal(lateVal)
		return res, nil
	}
	if early == nil {
		res.Value = formatVal(lateVal)
		res.IsDiff = false
		return res, nil
	}
	earlyVal, _ = extractField(early.AccountRaw, f.Field)
	// 数值相减，非数值显示晚值
	ln, lok := toFloat(lateVal)
	en, eok := toFloat(earlyVal)
	if lok && eok {
		res.Value = formatNum(ln - en)
		res.IsDiff = true
	} else {
		res.Value = formatVal(lateVal)
		res.IsDiff = false
	}
	return res, nil
}

func toFloat(v interface{}) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(t, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func formatNum(f float64) string {
	return strconv.FormatFloat(f, 'f', 0, 64)
}

func formatVal(v interface{}) string {
	if v == nil {
		return "-"
	}
	switch t := v.(type) {
	case float64:
		return strconv.FormatFloat(t, 'f', 0, 64)
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}
```

- [ ] **Step 2: 差值测试**

`internal/report/diff_test.go`:

```go
package report

import (
	"encoding/json"
	"testing"

	"quick-feishu/internal/model"
)

func mkSnap(used int64) *model.Snapshot {
	raw, _ := json.Marshal(map[string]interface{}{"used_quota": used})
	return &model.Snapshot{AccountRaw: raw}
}

func TestComputeDiffNumeric(t *testing.T) {
	early := mkSnap(100)
	late := mkSnap(150)
	res, err := computeDiff(early, late, Field{Field: "used_quota", Diff: true}, "已用配额")
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsDiff || res.Value != "50" {
		t.Errorf("expected 50, got %s (diff=%v)", res.Value, res.IsDiff)
	}
}

func TestComputeDiffNoEarly(t *testing.T) {
	late := mkSnap(150)
	res, _ := computeDiff(nil, late, Field{Field: "used_quota", Diff: true}, "已用配额")
	if res.IsDiff {
		t.Errorf("no early should not diff")
	}
	if res.Value != "150" {
		t.Errorf("value = %s", res.Value)
	}
}

func TestComputeDiffNonNumeric(t *testing.T) {
	rawA, _ := json.Marshal(map[string]interface{}{"group": "A"})
	rawB, _ := json.Marshal(map[string]interface{}{"group": "B"})
	early := &model.Snapshot{AccountRaw: rawA}
	late := &model.Snapshot{AccountRaw: rawB}
	res, _ := computeDiff(early, late, Field{Field: "group", Diff: true}, "分组")
	if res.IsDiff {
		t.Errorf("string field should not diff")
	}
	if res.Value != "B" {
		t.Errorf("value = %s", res.Value)
	}
}
```

- [ ] **Step 3: 跑测试确认**

Run: `go test ./internal/report/ -v`
Expected: 3 个测试 PASS

- [ ] **Step 4: Commit**

```bash
git add internal/report/diff.go internal/report/diff_test.go
git commit -m "feat: diff calculation for report fields"
```

---

## Task 11: 卡片生成

**Files:**
- Create: `internal/report/card.go`
- Test: `internal/report/card_test.go`

- [ ] **Step 1: 生成飞书交互卡片 JSON**

`internal/report/card.go`:

```go
package report

import (
	"encoding/json"
)

type Card struct {
	MsgType string     `json:"msg_type"`
	Card    CardBody   `json:"card"`
}

type CardBody struct {
	Config   CardConfig    `json:"config"`
	Header   CardHeader    `json:"header"`
	Elements []interface{} `json:"elements"`
}

type CardConfig struct {
	WideScreenMode bool `json:"wide_screen_mode"`
}

type CardHeader struct {
	Title    CardText `json:"title"`
	Template string   `json:"template"`
}

type CardText struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

type DivElement struct {
	Tag    string        `json:"tag"`
	Fields []CardField   `json:"fields"`
}

type CardField struct {
	IsShort bool     `json:"is_short"`
	Text    CardText `json:"text"`
}

type HR struct {
	Tag string `json:"tag"`
}

// BuildCard 根据模板与字段结果生成卡片
func BuildCard(tmpl *Template, date string, sections [][]DiffResult) (*Card, error) {
	card := &Card{
		MsgType: "interactive",
		Card: CardBody{
			Config: CardConfig{WideScreenMode: true},
			Header: CardHeader{
				Title:    CardText{Tag: "plain_text", Content: tmpl.Title + " " + date},
				Template: "blue",
			},
			Elements: []interface{}{},
		},
	}
	for i, sec := range sections {
		if len(sec) == 0 {
			continue
		}
		div := DivElement{Tag: "div", Fields: []CardField{}}
		for _, r := range sec {
			content := "**" + r.Label + "**\n" + r.Value
			div.Fields = append(div.Fields, CardField{
				IsShort: true,
				Text:    CardText{Tag: "lark_md", Content: content},
			})
		}
		card.Card.Elements = append(card.Card.Elements, div)
		if i < len(sections)-1 {
			card.Card.Elements = append(card.Card.Elements, HR{Tag: "hr"})
		}
	}
	return card, nil
}

func (c *Card) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}
```

- [ ] **Step 2: 卡片测试**

`internal/report/card_test.go`:

```go
package report

import (
	"encoding/json"
	"testing"
)

func TestBuildCard(t *testing.T) {
	tmpl := DefaultTemplate()
	sections := [][]DiffResult{
		{
			{Field: "used_quota", Label: "已用配额", Value: "50", IsDiff: true},
			{Field: "request_count", Label: "请求次数", Value: "10", IsDiff: true},
		},
		{
			{Field: "total_used", Label: "累计已用", Value: "888", IsDiff: true},
		},
	}
	card, err := BuildCard(tmpl, "2026-09-18", sections)
	if err != nil {
		t.Fatal(err)
	}
	if card.MsgType != "interactive" {
		t.Errorf("msg_type = %s", card.MsgType)
	}
	if len(card.Card.Elements) < 2 {
		t.Errorf("elements = %d, want >= 2", len(card.Card.Elements))
	}
	// 校验可序列化
	b, _ := card.ToJSON()
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m["msg_type"] != "interactive" {
		t.Errorf("json msg_type missing")
	}
}
```

- [ ] **Step 3: 跑测试确认**

Run: `go test ./internal/report/ -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/report/card.go internal/report/card_test.go
git commit -m "feat: feishu interactive card builder"
```

---

## Task 12: 飞书推送

**Files:**
- Create: `internal/feishu/client.go`
- Test: `internal/feishu/client_test.go`

- [ ] **Step 1: Webhook 推送与重试**

`internal/feishu/client.go`:

```go
package feishu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	WebhookURL string
	RetryTimes int
	HTTPClient *http.Client
}

func NewClient(webhookURL string, retryTimes int) *Client {
	return &Client{
		WebhookURL: webhookURL,
		RetryTimes: retryTimes,
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// SendCard 发送交互卡片，失败重试
func (c *Client) SendCard(cardJSON []byte) (string, error) {
	var lastErr error
	var respBody string
	for i := 0; i <= c.RetryTimes; i++ {
		respBody, lastErr = c.post(cardJSON)
		if lastErr == nil {
			return respBody, nil
		}
		if i < c.RetryTimes {
			time.Sleep(time.Duration(1<<uint(i)) * 2 * time.Second)
		}
	}
	return respBody, lastErr
}

func (c *Client) post(cardJSON []byte) (string, error) {
	req, err := http.NewRequest(http.MethodPost, c.WebhookURL, bytes.NewReader(cardJSON))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return string(body), fmt.Errorf("feishu http %d: %s", resp.StatusCode, string(body))
	}
	// 飞书业务码
	var r struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(body, &r); err == nil && r.Code != 0 {
		return string(body), fmt.Errorf("feishu code %d: %s", r.Code, r.Msg)
	}
	return string(body), nil
}
```

- [ ] **Step 2: 推送测试（mock webhook）**

`internal/feishu/client_test.go`:

```go
package feishu

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendCardSuccess(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		gotBody = string(buf[:n])
		w.Write([]byte(`{"code":0,"msg":"success"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2)
	resp, err := c.SendCard([]byte(`{"msg_type":"interactive"}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp != `{"code":0,"msg":"success"}` {
		t.Errorf("resp = %s", resp)
	}
	if gotBody == "" {
		t.Error("body should not be empty")
	}
}

func TestSendCardRetryThenFail(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`bad`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 3)
	_, err := c.SendCard([]byte(`{}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 4 {
		t.Errorf("calls = %d, want 4 (1 + 3 retries)", calls)
	}
}

func TestSendCardBusinessCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":19001,"msg":"invalid webhook"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 1)
	_, err := c.SendCard([]byte(`{}`))
	if err == nil {
		t.Fatal("business error should be returned")
	}
}
```

- [ ] **Step 3: 跑测试确认**

Run: `go test ./internal/feishu/ -v`
Expected: 3 个测试 PASS

- [ ] **Step 4: Commit**

```bash
git add internal/feishu/
git commit -m "feat: feishu webhook push with retry"
```

---

## Task 13: 定时调度

**Files:**
- Create: `internal/scheduler/scheduler.go`
- Test: `internal/scheduler/scheduler_test.go`

- [ ] **Step 1: cron 调度器**

`internal/scheduler/scheduler.go`:

```go
package scheduler

import (
	"time"

	"github.com/robfig/cron/v3"
	"quick-feishu/internal/config"
)

type Scheduler struct {
	Cron  *cron.Cron
	OnSnapshot func()
	OnReport   func()
}

func New(cfg *config.Config, loc *time.Location) *Scheduler {
	return &Scheduler{
		Cron: cron.New(cron.WithLocation(loc)),
	}
}

// RegisterSnapshot 注册每日快照任务（cron 表达式如 "0 0 * * *"）
func (s *Scheduler) RegisterSnapshot(cronExpr string) error {
	_, err := s.Cron.AddFunc(cronExpr, func() {
		if s.OnSnapshot != nil {
			s.OnSnapshot()
		}
	})
	return err
}

// RegisterReport 注册每日日报任务
func (s *Scheduler) RegisterReport(cronExpr string) error {
	_, err := s.Cron.AddFunc(cronExpr, func() {
		if s.OnReport != nil {
			s.OnReport()
		}
	})
	return err
}

func (s *Scheduler) Start() { s.Cron.Start() }

func (s *Scheduler) Stop()  { s.Cron.Stop() }

// timeToCron 将 "HH:MM" 转为 cron 表达式
func timeToCron(t string) (string, error) {
	parsed, err := time.Parse("15:04", t)
	if err != nil {
		return "", err
	}
	return "0 " + parsed.Format("04 15") + " * * *", nil
}
```

- [ ] **Step 2: 调度测试**

`internal/scheduler/scheduler_test.go`:

```go
package scheduler

import (
	"testing"
	"time"
)

func TestTimeToCron(t *testing.T) {
	expr, err := timeToCron("10:30")
	if err != nil {
		t.Fatal(err)
	}
	if expr != "0 30 10 * * *" {
		t.Errorf("expr = %s", expr)
	}
}

func TestRegisterAndTrigger(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	s := New(nil, loc)
	done := make(chan bool)
	s.OnSnapshot = func() { done <- true }
	if err := s.RegisterSnapshot("* * * * * *"); err != nil {
		t.Fatal(err)
	}
	s.Start()
	defer s.Stop()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("snapshot task not triggered")
	}
}
```

- [ ] **Step 3: 添加依赖并跑测试**

Run: `go get github.com/robfig/cron/v3 && go test ./internal/scheduler/ -v`
Expected: 2 个测试 PASS

- [ ] **Step 4: Commit**

```bash
git add internal/scheduler/
git commit -m "feat: cron scheduler for snapshot and report"
```

---

## Task 14: 补采机制

**Files:**
- Create: `internal/scheduler/backfill.go`
- Test: `internal/scheduler/backfill_test.go`

- [ ] **Step 1: 启动补采缺失日期快照**

`internal/scheduler/backfill.go`:

```go
package scheduler

import (
	"time"

	"gorm.io/gorm"
	"quick-feishu/internal/api"
	"quick-feishu/internal/collector"
	"quick-feishu/internal/db"
)

// BackfillMissing 自上次快照日期起，补齐到今天（含）之前缺失的每日快照。
// dateFn 返回今天日期；collectFn 用于采集。
func BackfillMissing(gdb *gorm.DB, c *api.Client, today string, collectFn func() *collector.Result) error {
	latest, err := db.LatestSnapshot(gdb)
	if err != nil {
		// 无任何快照：补采今天
		return saveFor(gdb, today, collectFn())
	}
	start, err := time.Parse("2006-01-02", latest.SnapshotDate)
	if err != nil {
		return err
	}
	for d := start.AddDate(0, 0, 1); ; d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		if dateStr > today {
			break
		}
		if err := saveFor(gdb, dateStr, collectFn()); err != nil {
			return err
		}
	}
	return nil
}

func saveFor(gdb *gorm.DB, date string, res *collector.Result) error {
	return collector.Save(gdb, date, res)
}
```

- [ ] **Step 2: 补采测试**

`internal/scheduler/backfill_test.go`:

```go
package scheduler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"quick-feishu/internal/api"
	"quick-feishu/internal/collector"
	"quick-feishu/internal/db"
	"quick-feishu/internal/model"
)

func TestBackfillMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/user/self":
			w.Write([]byte(`{"data":{"id":1,"quota":1000,"used_quota":100},"success":true}`))
		case "/api/token/":
			w.Write([]byte(`{"data":{"items":[]},"success":true}`))
		case "/api/usage/token/":
			w.Write([]byte(`{"data":{},"success":true}`))
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	gdb, _ := db.Init(dir)
	gdb.Create(&model.Snapshot{SnapshotDate: "2026-09-16", AccountUsed: 10})

	c := api.NewClient(srv.URL, "sys", "1")
	collectFn := func() *collector.Result { return collector.Collect(c) }

	if err := BackfillMissing(gdb, c, "2026-09-18", collectFn); err != nil {
		t.Fatal(err)
	}
	var dates []string
	gdb.Model(&model.Snapshot{}).Pluck("snapshot_date", &dates)
	if len(dates) != 3 {
		t.Errorf("dates = %v, want 3 (09-16,17,18)", dates)
	}
}
```

- [ ] **Step 3: 跑测试确认**

Run: `go test ./internal/scheduler/ -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/scheduler/backfill.go internal/scheduler/backfill_test.go
git commit -m "feat: backfill missing daily snapshots on startup"
```

---

## Task 15: Gin 服务与路由

**Files:**
- Create: `internal/server/server.go`
- Create: `internal/server/routes.go`
- Create: `internal/server/handlers.go`

- [ ] **Step 1: Gin 服务主框架**

`internal/server/server.go`:

```go
package server

import (
	"fmt"
	"net"

	"github.com/gin-gonic/gin"
)

type Server struct {
	Engine *gin.Engine
	Port   int
}

func New(port int) *Server {
	gin.SetMode(gin.ReleaseMode)
	return &Server{Engine: gin.New(), Port: port}
}

// Listen 监听端口，占用则递增
func (s *Server) Listen() (int, error) {
	port := s.Port
	for i := 0; i < 20; i++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			port++
			continue
		}
		ln.Close()
		return port, nil
	}
	return 0, fmt.Errorf("no free port")
}

func (s *Server) Run(port int) error {
	return s.Engine.Run(fmt.Sprintf(":%d", port))
}
```

- [ ] **Step 2: 路由注册**

`internal/server/routes.go`:

```go
package server

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:web
var webFS embed.FS

func (s *Server) RegisterRoutes(h *Handlers) {
	api := s.Engine.Group("/api")
	{
		api.GET("/dashboard", h.Dashboard)
		api.POST("/snapshot/run", h.RunSnapshot)
		api.POST("/report/send", h.SendReport)
		api.GET("/snapshots", h.ListSnapshots)
		api.GET("/snapshots/:id", h.GetSnapshot)
		api.GET("/snapshots/compare", h.CompareSnapshots)
		api.GET("/template", h.GetTemplate)
		api.PUT("/template", h.SaveTemplate)
		api.GET("/dict/:source", h.GetDict)
		api.PUT("/dict/:source", h.SaveDict)
		api.GET("/settings", h.GetSettings)
		api.PUT("/settings", h.SaveSettings)
		api.POST("/feishu/test", h.TestFeishu)
		api.GET("/sendlogs", h.ListSendLogs)
	}
	// 静态资源
	staticFS, _ := fs.Sub(webFS, "web")
	s.Engine.StaticFS("/", http.FS(staticFS))
}
```

- [ ] **Step 3: 添加依赖并编译**

Run: `go get github.com/gin-gonic/gin`
Run: `go build ./internal/server/`
Expected: 报错缺少 `web` 嵌入目录 → 需先创建空目录或调整 embed

说明：若 `web/` 尚为空，embed 会失败。先创建占位文件 `web/.gitkeep`，后续 Task 17 放前端产物。

Run: `mkdir -p web && touch web/.gitkeep && go build ./internal/server/`

- [ ] **Step 4: Commit**

```bash
git add internal/server/ web/.gitkeep
git commit -m "feat: gin server skeleton with API routes"
```

---

## Task 16: API 接口实现

**Files:**
- Modify: `internal/server/handlers.go`
- Test: `internal/server/handlers_test.go`

- [ ] **Step 1: Handler 依赖与核心接口**

`internal/server/handlers.go`:

```go
package server

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"quick-feishu/internal/api"
	"quick-feishu/internal/collector"
	"quick-feishu/internal/config"
	"quick-feishu/internal/db"
	"quick-feishu/internal/model"
	"quick-feishu/internal/report"
)

type Handlers struct {
	DB       *gorm.DB
	Config   *config.Config
	ConfigPath string
	APIClient *api.Client
}

func (h *Handlers) Dashboard(c *gin.Context) {
	snap, err := db.LatestSnapshot(h.DB)
	if err != nil {
		snap = nil
	}
	var logs []model.SendLog
	h.DB.Order("id DESC").Limit(10).Find(&logs)
	c.JSON(http.StatusOK, gin.H{"latest_snapshot": snap, "recent_logs": logs})
}

func (h *Handlers) RunSnapshot(c *gin.Context) {
	res := collector.Collect(h.APIClient)
	today := timeNow()
	if err := collector.Save(h.DB, today, res); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "date": today, "errors": res.Errors})
}

func (h *Handlers) SendReport(c *gin.Context) {
	latest, _ := db.LatestSnapshot(h.DB)
	if latest == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no snapshots yet, run snapshot first"})
		return
	}
	prev, _ := db.SnapshotBefore(h.DB, addDays(latest.SnapshotDate, -1))
	tmpl := report.DefaultTemplate()
	log, err := report.ExecuteReport(h.DB, latest, prev, tmpl, h.Config.Feishu.WebhookURL, h.Config.Feishu.RetryTimes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "log_id": log.ID})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "log_id": log.ID})
}
```

- [ ] **Step 2: 查询/模板/字典/设置/飞书测试 Handler**

`internal/server/handlers2.go`:

```go
package server

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"quick-feishu/internal/db"
	"quick-feishu/internal/report"
	"quick-feishu/internal/feishu"
)

func (h *Handlers) ListSnapshots(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	list, total, err := db.ListSnapshots(h.DB, page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": list, "total": total})
}

func (h *Handlers) GetSnapshot(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var snap model.Snapshot
	if err := h.DB.First(&snap, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	tokens, _ := db.TokenSnapshots(h.DB, snap.ID)
	c.JSON(http.StatusOK, gin.H{"snapshot": snap, "tokens": tokens})
}

func (h *Handlers) CompareSnapshots(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")
	early, _ := db.SnapshotByDate(h.DB, from)
	late, _ := db.SnapshotByDate(h.DB, to)
	if early == nil || late == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "snapshot not found"})
		return
	}
	// 对比账号冗余字段
	c.JSON(http.StatusOK, gin.H{
		"from": gin.H{"date": early.SnapshotDate, "used": early.AccountUsed, "quota": early.AccountQuota, "requests": early.RequestCount},
		"to":   gin.H{"date": late.SnapshotDate, "used": late.AccountUsed, "quota": late.AccountQuota, "requests": late.RequestCount},
		"diff": gin.H{"used": late.AccountUsed - early.AccountUsed, "requests": late.RequestCount - early.RequestCount},
	})
}

func (h *Handlers) GetTemplate(c *gin.Context) {
	c.JSON(http.StatusOK, h.Config.ReportTemplate)
}

func (h *Handlers) SaveTemplate(c *gin.Context) {
	var tmpl report.Template
	if err := c.ShouldBindJSON(&tmpl); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.Config.ReportTemplate = mustToMap(tmpl)
	if err := h.Config.Save(h.ConfigPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handlers) GetDict(c *gin.Context) {
	source := c.Param("source")
	var items []interface{}
	switch source {
	case "account":
		var list []model.DictAccountField
		h.DB.Find(&list)
		for _, v := range list {
			items = append(items, v)
		}
	case "token":
		var list []model.DictTokenField
		h.DB.Find(&list)
		for _, v := range list {
			items = append(items, v)
		}
	case "usage":
		var list []model.DictUsageField
		h.DB.Find(&list)
		for _, v := range list {
			items = append(items, v)
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad source"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handlers) GetSettings(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"config": h.Config,
		"env_overridden": h.Config.EnvOverridden(),
	})
}

func (h *Handlers) SaveSettings(c *gin.Context) {
	var in config.Config
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// 环境变量覆盖项不可改
	env := h.Config.EnvOverridden()
	if env["user_id"] {
		in.Account.UserID = h.Config.Account.UserID
	}
	if env["system_token"] {
		in.Account.SystemToken = h.Config.Account.SystemToken
	}
	if env["webhook_url"] {
		in.Feishu.WebhookURL = h.Config.Feishu.WebhookURL
	}
	*h.Config = in
	if err := h.Config.Save(h.ConfigPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handlers) TestFeishu(c *gin.Context) {
	client := feishu.NewClient(h.Config.Feishu.WebhookURL, 1)
	// 发送一条测试文本消息
	body := []byte(`{"msg_type":"text","content":{"text":"QuickFeishu 测试消息"}}`)
	resp, err := client.SendCard(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "resp": resp})
}

func (h *Handlers) ListSendLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	list, total, err := db.ListSendLogs(h.DB, page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": list, "total": total})
}

func mustToMap(v interface{}) map[string]interface{} {
	b, _ := json.Marshal(v)
	var m map[string]interface{}
	json.Unmarshal(b, &m)
	return m
}
```

- [ ] **Step 2: 需要的辅助函数（timeNow/addDays/ComputeDiffAccount）**

`internal/report/diff.go` 增加导出方法：

```go
// ComputeDiffAccount 供 handler 使用：对账号快照字段计算差值
func ComputeDiffAccount(early, late *model.Snapshot, f Field, label string) (DiffResult, error) {
	return computeDiff(early, late, f, label)
}
```

`internal/report/service.go`（日报执行服务，Web 与 CLI 共用）：

```go
package report

import (
	"fmt"

	"gorm.io/gorm"
	"quick-feishu/internal/db"
	"quick-feishu/internal/feishu"
	"quick-feishu/internal/model"
)

// ExecuteReport 取最新两日快照，按模板计算差值生成卡片，推送到飞书并记录日志。
// 返回发送日志（无论成败），供调用方展示。
func ExecuteReport(gdb *gorm.DB, latest, prev *model.Snapshot, tmpl *Template, webhookURL string, retryTimes int) (*model.SendLog, error) {
	if latest == nil {
		return nil, fmt.Errorf("no snapshots yet")
	}
	sections := [][]DiffResult{}
	for _, sec := range tmpl.Sections {
		var secRes []DiffResult
		switch sec.Source {
		case "account":
			for _, f := range sec.Fields {
				res, err := ComputeDiffAccount(prev, latest, f, f.Field)
				if err != nil {
					continue
				}
				secRes = append(secRes, res)
			}
		case "usage":
			if !sec.PerToken {
				continue
			}
			tokens, err := db.TokenSnapshots(gdb, latest.ID)
			if err != nil {
				continue
			}
			prevTokens, _ := db.TokenSnapshots(gdb, prevID(gdb, prev, latest))
			for _, tok := range tokens {
				for _, f := range sec.Fields {
					secRes = append(secRes, tokenDiff(prevTokens, tok, f))
				}
			}
		}
		if len(secRes) > 0 {
			sections = append(sections, secRes)
		}
	}
	card, err := BuildCard(tmpl, latest.SnapshotDate, sections)
	if err != nil {
		return nil, err
	}
	cardJSON, _ := card.ToJSON()
	client := feishu.NewClient(webhookURL, retryTimes)
	resp, sendErr := client.SendCard(cardJSON)
	log := &model.SendLog{Date: latest.SnapshotDate, Success: sendErr == nil, FeishuResp: resp}
	if sendErr != nil {
		log.ErrorMsg = sendErr.Error()
	}
	if err := gdb.Create(log).Error; err != nil {
		return log, err
	}
	return log, sendErr
}

func prevID(gdb *gorm.DB, prev *model.Snapshot, latest *model.Snapshot) uint {
	if prev != nil {
		return prev.ID
	}
	return 0
}

func tokenDiff(prevTokens []model.TokenSnapshot, tok model.TokenSnapshot, f Field) DiffResult {
	res := DiffResult{Field: f.Field, Diff: f.Diff, Value: formatVal(nil)}
	var prev *model.TokenSnapshot
	for i := range prevTokens {
		if prevTokens[i].TokenID == tok.TokenID {
			prev = &prevTokens[i]
			break
		}
	}
	switch f.Field {
	case "total_used":
		res.Label = "累计已用"
		res.Value = numDiff(prevTotal(prev), tok.TotalUsed)
	case "remain_quota":
		res.Label = "剩余配额"
		res.Value = formatNum(float64(tok.RemainQuota))
	case "used_quota":
		res.Label = "已用配额"
		res.Value = numDiff(prevUsed(prev), tok.UsedQuota)
	default:
		res.Value = formatVal(nil)
	}
	return res
}

func prevTotal(prev *model.TokenSnapshot) int64 {
	if prev != nil {
		return prev.TotalUsed
	}
	return 0
}

func prevUsed(prev *model.TokenSnapshot) int64 {
	if prev != nil {
		return prev.UsedQuota
	}
	return 0
}

// numDiff 计算差值；无历史时返回当前值
func numDiff(prevVal, cur int64) string {
	if prevVal == 0 && cur == 0 {
		return formatNum(0)
	}
	return formatNum(float64(cur - prevVal))
}
```

`internal/server/util.go`:

```go
package server

import "time"

func timeNow() string {
	return time.Now().In(time.FixedZone("CST", 8*3600)).Format("2006-01-02")
}

func addDays(date string, n int) string {
	d, _ := time.Parse("2006-01-02", date)
	return d.AddDate(0, 0, n).Format("2006-01-02")
}
```

- [ ] **Step 3: 编译并跑现有测试**

Run: `go build ./... && go test ./...`
Expected: 全部编译通过，现有测试 PASS

- [ ] **Step 4: Commit**

```bash
git add internal/server/ internal/report/diff.go
git commit -m "feat: server handlers for dashboard, snapshots, template, settings"
```

---

## Task 17: Vue 前端脚手架

**Files:**
- Create: `internal/frontend/`（Vue 源码）
- Create: `frontend/package.json`
- Create: `frontend/vite.config.ts`
- Create: `frontend/src/`

- [ ] **Step 1: 初始化 Vite + Vue3 项目**

```bash
cd /Users/john/Documents/My_Work/quick_feishu
mkdir -p frontend
cd frontend
npm create vite@latest . -- --template vue
npm install
npm install element-plus @element-plus/icons-vue axios vue-router@4
```

- [ ] **Step 2: 配置 Vite 构建输出到 web/**

`frontend/vite.config.ts`:

```ts
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  base: './',
  build: {
    outDir: '../web',
    emptyOutDir: true,
  },
})
```

- [ ] **Step 3: 配置路由与 Element Plus**

`frontend/src/main.ts`:

```ts
import { createApp } from 'vue'
import { createRouter, createWebHashHistory } from 'vue-router'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import App from './App.vue'
import Dashboard from './views/Dashboard.vue'
import Snapshots from './views/Snapshots.vue'
import TemplateEditor from './views/TemplateEditor.vue'
import Settings from './views/Settings.vue'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', component: Dashboard },
    { path: '/snapshots', component: Snapshots },
    { path: '/template', component: TemplateEditor },
    { path: '/settings', component: Settings },
  ],
})

createApp(App).use(router).use(ElementPlus).mount('#app')
```

- [ ] **Step 4: 布局骨架**

`frontend/src/App.vue`:

```vue
<template>
  <el-container>
    <el-aside width="200px">
      <el-menu :router="true" default-active="1">
        <el-menu-item index="1" route="/">📊 仪表盘</el-menu-item>
        <el-menu-item index="2" route="/snapshots">📋 历史快照</el-menu-item>
        <el-menu-item index="3" route="/template">📝 日报模板</el-menu-item>
        <el-menu-item index="4" route="/settings">⚙️ 设置</el-menu-item>
      </el-menu>
    </el-aside>
    <el-main><router-view /></el-main>
  </el-container>
</template>
```

- [ ] **Step 5: 构建并验证产物**

Run: `cd frontend && npm run build`
Expected: `../web/` 下生成 index.html + assets/

- [ ] **Step 6: Commit**

```bash
git add frontend/ web/
git commit -m "feat: vue frontend scaffold with element-plus"
```

---

## Task 18: 前端四个页面

**Files:**
- Create: `frontend/src/views/Dashboard.vue`
- Create: `frontend/src/views/Snapshots.vue`
- Create: `frontend/src/views/TemplateEditor.vue`
- Create: `frontend/src/views/Settings.vue`
- Create: `frontend/src/api.ts`

- [ ] **Step 1: API 封装**

`frontend/src/api.ts`:

```ts
import axios from 'axios'

export const api = axios.create({ baseURL: '/api' })

export default {
  dashboard: () => api.get('/dashboard'),
  runSnapshot: () => api.post('/snapshot/run'),
  sendReport: () => api.post('/report/send'),
  snapshots: (page = 1, size = 20) => api.get('/snapshots', { params: { page, size } }),
  snapshot: (id: number) => api.get(`/snapshots/${id}`),
  compare: (from: string, to: string) => api.get('/snapshots/compare', { params: { from, to } }),
  getTemplate: () => api.get('/template'),
  saveTemplate: (data: any) => api.put('/template', data),
  getDict: (source: string) => api.get(`/dict/${source}`),
  getSettings: () => api.get('/settings'),
  saveSettings: (data: any) => api.put('/settings', data),
  testFeishu: () => api.post('/feishu/test'),
  sendLogs: (page = 1, size = 20) => api.get('/sendlogs', { params: { page, size } }),
}
```

- [ ] **Step 2: 仪表盘页面**

`frontend/src/views/Dashboard.vue`（关键部分）:

```vue
<template>
  <div>
    <h2>仪表盘</h2>
    <el-space>
      <el-button type="primary" @click="runSnapshot">立即采集快照</el-button>
      <el-button type="success" @click="sendReport">立即发送日报</el-button>
      <el-button @click="testFeishu">测试飞书连接</el-button>
    </el-space>
    <el-descriptions v-if="latest" title="最近快照" :column="3" border style="margin-top:16px">
      <el-descriptions-item label="日期">{{ latest.snapshot_date }}</el-descriptions-item>
      <el-descriptions-item label="已用配额">{{ latest.account_used }}</el-descriptions-item>
      <el-descriptions-item label="请求次数">{{ latest.request_count }}</el-descriptions-item>
    </el-descriptions>
    <el-table :data="logs" style="margin-top:16px">
      <el-table-column prop="send_time" label="发送时间" />
      <el-table-column prop="date" label="日期" />
      <el-table-column label="状态">
        <template #default="scope">
          <el-tag :type="scope.row.success ? 'success' : 'danger'">{{ scope.row.success ? '成功' : '失败' }}</el-tag>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../api'

const latest = ref<any>(null)
const logs = ref<any[]>([])

onMounted(async () => {
  const res = await api.dashboard()
  latest.value = res.data.latest_snapshot
  logs.value = res.data.recent_logs
})

async function runSnapshot() {
  const res = await api.runSnapshot()
  ElMessage.success(`快照已采集: ${res.data.date}`)
  location.reload()
}
async function sendReport() {
  const res = await api.sendReport()
  ElMessage.success('日报已发送')
}
async function testFeishu() {
  await api.testFeishu()
  ElMessage.success('飞书连接正常')
}
</script>
```

- [ ] **Step 3: 历史快照页面（列表+详情+对比）**

`frontend/src/views/Snapshots.vue`（关键部分）:

```vue
<template>
  <div>
    <h2>历史快照</h2>
    <el-table :data="snapshots" @row-click="loadDetail">
      <el-table-column prop="snapshot_date" label="日期" />
      <el-table-column prop="account_used" label="已用配额" />
      <el-table-column prop="request_count" label="请求次数" />
      <el-table-column prop="created_at" label="创建时间" />
    </el-table>
    <el-pagination layout="prev, pager, next" :total="total" @current-change="loadList" />

    <template v-if="detail">
      <h3>快照详情 {{ detail.snapshot.snapshot_date }}</h3>
      <el-tabs>
        <el-tab-pane label="账号信息">
          <pre>{{ JSON.stringify(JSON.parse(detail.snapshot.account_raw), null, 2) }}</pre>
        </el-tab-pane>
        <el-tab-pane label="令牌列表">
          <pre>{{ JSON.stringify(JSON.parse(detail.snapshot.token_list_raw), null, 2) }}</pre>
        </el-tab-pane>
        <el-tab-pane label="令牌使用情况">
          <pre>{{ JSON.stringify(JSON.parse(detail.snapshot.token_usage_raw), null, 2) }}</pre>
        </el-tab-pane>
      </el-tabs>
    </template>

    <h3>对比</h3>
    <el-date-picker v-model="compareRange" type="daterange" value-format="YYYY-MM-DD" />
    <el-button @click="doCompare">对比</el-button>
    <pre v-if="compare">{{ JSON.stringify(compare, null, 2) }}</pre>
  </div>
</template>
```

- [ ] **Step 4: 日报模板页面（勾选字段+差值开关）**

`frontend/src/views/TemplateEditor.vue`（关键部分）:

```vue
<template>
  <div>
    <h2>日报模板</h2>
    <el-form label-width="100px">
      <el-form-item label="标题"><el-input v-model="tmpl.title" /></el-form-item>
      <el-form-item label="日期模式">
        <el-radio-group v-model="tmpl.date_mode">
          <el-radio value="auto">自动（最近两天）</el-radio>
          <el-radio value="manual">手动选择</el-radio>
        </el-radio-group>
      </el-form-item>
      <div v-for="(sec, i) in tmpl.sections" :key="i">
        <el-card>
          <el-input v-model="sec.section" placeholder="分区标题" />
          <el-select v-model="sec.source">
            <el-option value="account" label="账号信息" />
            <el-option value="token" label="令牌列表" />
            <el-option value="usage" label="令牌使用情况" />
          </el-select>
          <div v-for="(f, j) in sec.fields" :key="j">
            <el-input v-model="f.field" placeholder="字段名" />
            <el-switch v-model="f.diff" active-text="差值" />
          </div>
          <el-button @click="sec.fields.push({ field: '', diff: false })">+字段</el-button>
        </el-card>
      </div>
      <el-button @click="tmpl.sections.push({ section: '', source: 'account', per_token: false, fields: [] })">+分区</el-button>
      <el-button type="primary" @click="save">保存模板</el-button>
    </el-form>
  </div>
</template>
```

- [ ] **Step 5: 设置页面**

`frontend/src/views/Settings.vue`（关键部分）:

```vue
<template>
  <div>
    <h2>设置</h2>
    <el-form label-width="120px">
      <el-form-item label="账号ID">
        <el-input v-model="cfg.account.user_id" :disabled="env.user_id" />
        <span v-if="env.user_id">已由环境变量提供</span>
      </el-form-item>
      <el-form-item label="系统令牌">
        <el-input v-model="cfg.account.system_token" type="password" :disabled="env.system_token" show-password />
      </el-form-item>
      <el-form-item label="飞书Webhook">
        <el-input v-model="cfg.feishu.webhook_url" :disabled="env.webhook_url" />
      </el-form-item>
      <el-form-item label="快照时间"><el-time-picker v-model="snapTime" format="HH:mm" /></el-form-item>
      <el-form-item label="日报时间"><el-time-picker v-model="reportTime" format="HH:mm" /></el-form-item>
      <el-form-item label="端口"><el-input-number v-model="cfg.app.port" :min="1" :max="65535" /></el-form-item>
      <el-form-item>
        <el-button type="primary" @click="save">保存</el-button>
        <el-button @click="testFeishu">测试飞书连接</el-button>
      </el-form-item>
    </el-form>
  </div>
</template>
```

- [ ] **Step 6: 构建前端**

Run: `cd frontend && npm run build`
Expected: 构建成功，产物在 web/

- [ ] **Step 7: Commit**

```bash
git add frontend/src frontend/package.json frontend/vite.config.ts
git commit -m "feat: four frontend pages (dashboard, snapshots, template, settings)"
```

---

## Task 19: 构建打包与浏览器自动打开

**Files:**
- Create: `scripts/build.sh`
- Create: `internal/server/openurl.go`
- Modify: `main.go`

- [ ] **Step 1: 交叉编译脚本**

`scripts/build.sh`:

```bash
#!/usr/bin/env bash
set -e
cd "$(dirname "$0")/.."

# 构建前端
cd frontend && npm run build && cd ..

# 交叉编译三平台
GOOS=darwin GOARCH=amd64 go build -o dist/quick-feishu-darwin-amd64 .
GOOS=linux GOARCH=amd64 go build -o dist/quick-feishu-linux-amd64 .
GOOS=windows GOARCH=amd64 go build -o dist/quick-feishu-windows-amd64.exe .
echo "build complete: dist/"
```

- [ ] **Step 2: 浏览器自动打开**

`internal/server/openurl.go`:

```go
package server

import "os/exec"

// OpenBrowser 打开默认浏览器
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch {
	case isWindows():
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case isMac():
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

func isWindows() bool { return goos == "windows" }
func isMac() bool     { return goos == "darwin" }
```

需要 `goos` 常量：在 `openurl.go` 加 `const goos = runtime.GOOS` 与 `"runtime"` import。

- [ ] **Step 3: 串联 main.go**

`main.go`（完整重写）:

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"quick-feishu/internal/api"
	"quick-feishu/internal/collector"
	"quick-feishu/internal/config"
	"quick-feishu/internal/db"
	"quick-feishu/internal/report"
	"quick-feishu/internal/scheduler"
	"quick-feishu/internal/server"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		args = []string{"serve"}
	}
	switch args[0] {
	case "serve":
		serve()
	case "run-report":
		runReport()
	case "snapshot":
		runSnapshot()
	default:
		fmt.Println("usage: quick-feishu [serve|run-report|snapshot]")
		os.Exit(1)
	}
}

func baseDir() string {
	exe, _ := os.Executable()
	return filepath.Dir(exe)
}

func serve() {
	dir := baseDir()
	cfg, err := config.Load(filepath.Join(dir, "config.yaml"))
	if err != nil {
		fmt.Println("config error:", err)
		os.Exit(1)
	}
	gdb, err := db.Init(filepath.Join(dir, "data"))
	if err != nil {
		fmt.Println("db error:", err)
		os.Exit(1)
	}
	if err := db.SeedDicts(gdb); err != nil {
		fmt.Println("seed error:", err)
		os.Exit(1)
	}
	c := api.NewClient(cfg.Account.APIBase, cfg.Account.SystemToken, cfg.Account.UserID)

	// 启动补采
	today := time.Now().In(time.FixedZone("CST", 8*3600)).Format("2006-01-02")
	if err := scheduler.BackfillMissing(gdb, c, today, func() *collector.Result { return collector.Collect(c) }); err != nil {
		fmt.Println("backfill error:", err)
	}

	// 定时任务
	loc := time.FixedZone("CST", 8*3600)
	sch := scheduler.New(cfg, loc)
	sch.OnSnapshot = func() {
		res := collector.Collect(c)
		collector.Save(gdb, today, res)
	}
	sch.OnReport = func() {
		fmt.Println("report trigger (implemented in Task 16 handlers)")
	}
	snapExpr, _ := scheduler.TimeToCron(cfg.Schedule.SnapshotTime)
	reportExpr, _ := scheduler.TimeToCron(cfg.Schedule.ReportTime)
	sch.RegisterSnapshot(snapExpr)
	sch.RegisterReport(reportExpr)
	sch.Start()

	// Web 服务
	srv := server.New(cfg.App.Port)
	port, err := srv.Listen()
	if err != nil {
		fmt.Println("port error:", err)
		os.Exit(1)
	}
	handlers := &server.Handlers{DB: gdb, Config: cfg, ConfigPath: filepath.Join(dir, "config.yaml"), APIClient: c}
	srv.RegisterRoutes(handlers)
	url := fmt.Sprintf("http://localhost:%d/", port)
	fmt.Println("QuickFeishu running at", url)
	server.OpenBrowser(url)
	srv.Run(port)
}

func runSnapshot() {
	dir := baseDir()
	cfg, _ := config.Load(filepath.Join(dir, "config.yaml"))
	gdb, _ := db.Init(filepath.Join(dir, "data"))
	c := api.NewClient(cfg.Account.APIBase, cfg.Account.SystemToken, cfg.Account.UserID)
	res := collector.Collect(c)
	today := time.Now().In(time.FixedZone("CST", 8*3600)).Format("2006-01-02")
	collector.Save(gdb, today, res)
	fmt.Println("snapshot saved:", today)
}

func runReport() {
	dir := baseDir()
	cfg, _ := config.Load(filepath.Join(dir, "config.yaml"))
	gdb, _ := db.Init(filepath.Join(dir, "data"))
	latest, _ := db.LatestSnapshot(gdb)
	if latest == nil {
		fmt.Println("no snapshots yet, run 'snapshot' first")
		os.Exit(1)
	}
	prev, _ := db.SnapshotBefore(gdb, addDaysCLI(latest.SnapshotDate, -1))
	tmpl := report.DefaultTemplate()
	log, err := report.ExecuteReport(gdb, latest, prev, tmpl, cfg.Feishu.WebhookURL, cfg.Feishu.RetryTimes)
	if err != nil {
		fmt.Println("report failed:", err)
		os.Exit(1)
	}
	fmt.Println("report sent, log id:", log.ID)
}

func addDaysCLI(date string, n int) string {
	d, _ := time.Parse("2006-01-02", date)
	return d.AddDate(0, 0, n).Format("2006-01-02")
}
```

- [ ] **Step 3b: 补充 scheduler.TimeToCron 导出**

`internal/scheduler/scheduler.go` 增加导出方法：

```go
func TimeToCron(t string) (string, error) {
	return timeToCron(t)
}
```

- [ ] **Step 4: 编译验证**

Run: `go build ./... && bash scripts/build.sh`
Expected: 三平台产物生成到 dist/

- [ ] **Step 5: 本机启动验证**

Run: `./dist/quick-feishu-darwin-amd64 serve`
Expected: 打印 URL，自动打开浏览器，页面可访问

- [ ] **Step 6: Commit**

```bash
git add scripts/build.sh internal/server/openurl.go main.go internal/scheduler/scheduler.go
git commit -m "feat: cross-compile build, browser auto-open, serve wiring"
```

---

## Task 20: 测试完善与修复

**Files:**
- Modify: 各 `_test.go` 补充遗漏用例
- Test: `internal/server/handlers_test.go`

- [ ] **Step 1: Server handler 测试**

`internal/server/handlers_test.go`:

```go
package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"quick-feishu/internal/db"
)

func TestDashboardEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	gdb, _ := db.Init(dir)
	h := &Handlers{DB: gdb}
	s := New(0)
	s.RegisterRoutes(h)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
	s.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("code = %d", w.Code)
	}
}
```

- [ ] **Step 2: 跑全部测试**

Run: `go test ./...`
Expected: 全部 PASS（如有失败，定位到具体任务修复）

- [ ] **Step 3: 前端测试（vitest）**

```bash
cd frontend
npm install -D vitest @vue/test-utils
```

`frontend/src/api.test.ts`:

```ts
import { describe, it, expect } from 'vitest'
import api from './api'

describe('api', () => {
  it('has expected methods', () => {
    expect(typeof api.dashboard).toBe('function')
    expect(typeof api.runSnapshot).toBe('function')
    expect(typeof api.sendReport).toBe('function')
    expect(typeof api.snapshots).toBe('function')
  })
})
```

- [ ] **Step 4: 跑前端测试**

Run: `cd frontend && npx vitest run`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/server/handlers_test.go frontend/src/api.test.ts
git commit -m "test: add server handler and frontend api tests"
```

---

## Task 21: 端到端验证

**Files:**
- 无（验证性任务）

- [ ] **Step 1: 配置真实账号**

编辑 `config.yaml`，填入：
- `account.user_id: "827947"`
- `account.system_token: "H53aSL+..."`
- `feishu.webhook_url: "https://open.feishu.cn/open-apis/bot/v2/hook/xxx"`

注意：**不得提交含真实密钥的 config.yaml 到 git**（已在 .gitignore 中排除）。

Run: `echo "config.yaml" > .gitignore && echo "data/" >> .gitignore && echo "dist/" >> .gitignore`

- [ ] **Step 2: 启动服务**

Run: `./dist/quick-feishu-darwin-amd64 serve`
Expected: 浏览器打开，仪表盘显示最近快照

- [ ] **Step 3: 页面操作验证**

- [ ] 仪表盘点击「立即采集快照」→ 显示成功
- [ ] 仪表盘点击「立即发送日报」→ 飞书群收到卡片
- [ ] 历史快照页 → 看到快照记录与原始 JSON
- [ ] 日报模板页 → 修改字段/差值 → 保存成功
- [ ] 设置页 → 点击「测试飞书连接」→ 收到测试消息

- [ ] **Step 4: 手动命令验证**

Run: `./dist/quick-feishu-darwin-amd64 snapshot`
Run: `./dist/quick-feishu-darwin-amd64 run-report`
Expected: 命令行输出采集/发送结果

- [ ] **Step 5: 时间边界验证**

- [ ] 修改 config.yaml 的 `report_time` 为当前时间后 1 分钟，观察定时触发
- [ ] 删除 data/ 下数据库重启，验证内置字典重建 + 补采

- [ ] **Step 6: 更新进度表**

将所有完成任务的 checkbox 打勾，更新进度跟踪表。

---

## 完成标准（Definition of Done）

- [ ] 三平台二进制可构建（darwin/linux/windows）
- [ ] 三接口真实采集成功，快照完整落库
- [ ] 飞书群收到交互卡片日报（差值正确）
- [ ] 四个 Web 页面可用（仪表盘/历史快照/模板/设置）
- [ ] 手动触发 + 定时触发均可用
- [ ] 补采机制验证通过
- [ ] `go test ./...` 全绿，前端 vitest 全绿
- [ ] 无真实密钥进入 git 历史

## 故障定位索引

问题出现时，按此表定位任务点：
| 症状 | 定位 |
| --- | --- |
| 采集失败/令牌认证错误 | Task 5-7（usage 需用令牌 key，非系统令牌） |
| 快照不落库 | Task 3/7/8 |
| 差值不对 | Task 10（数值/字符串/缺失处理） |
| 卡片格式飞书拒绝 | Task 11（元素上限/字段结构） |
| 日报没发出去 | Task 12/16（webhook/重试/日志） |
| 定时不触发 | Task 13/14（cron 表达式/时区） |
| 页面打不开/接口404 | Task 15/16（路由/embed） |
| 前端构建失败 | Task 17/18（vite 配置/产物路径） |
| 设置保存不生效 | Task 16（handlers 保存逻辑/env 覆盖） |
| 服务器部署异常 | Task 19（端口/路径/config.yaml 位置） |