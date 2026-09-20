# 账号变更切换数据库 — 设计文档

> 日期：2026-09-18
> 状态：已确认（用户逐项批准）
> 关联：`2026-09-18-quick-feishu-design.md`

## 1. 背景与问题

当前 `internal/db/db.go` 固定使用单库 `data/quick-feishu.db`，`App` 与 `server.Handlers` 都各自持有启动时创建的 `*gorm.DB`。当用户在设置页把账号（`account.user_id`）改成另一个账号并「应用配置并重启」后，采集写入的仍是同一个库，导致两个账号的数据混在一起、历史页显示的还是旧账号数据。

**目标**：每个账号使用各自的数据库文件；账号变更并重启后，应用切换到该账号的库，历史与快照各账号隔离。

## 2. 已确认的决策

| 决策点 | 结论 |
| --- | --- |
| 账号标识 | 仅 `user_id`（`api_base` 变化不换库） |
| 切换时机 | 启动时 + 「应用配置并重启」时，均按 `config.user_id` 确定库 |
| 旧单库 | 首次运行新逻辑时自动迁移到当前账号目录，历史不丢 |
| `user_id` 为空 | 继续使用 `data/quick-feishu.db`，不迁移 |
| 切库后 | 自动 `Backfill()` 补齐新账号缺失的快照 |
| 前端 | 不改动 |

## 3. 方案

采用「每账号独立 SQLite 文件 + App 统一管理库生命周期 + Handlers 通过 `App.DB()` 动态取当前库」。

### 3.1 库路径与迁移（`internal/db`）

- `Path(dataDir, userID string) string`
  - `userID != ""` → `filepath.Join(dataDir, "accounts", userID, "quick-feishu.db")`
  - `userID == ""` → `filepath.Join(dataDir, "quick-feishu.db")`
- `Open(path string) (*gorm.DB, error)`：建父目录 + `gorm.Open` + `AutoMigrate`（现有 `Init` 主体）。
- `MigrateLegacy(dataDir, userID string) error`：
  - 仅当 `userID != ""`、`dataDir/quick-feishu.db` 存在、且 `Path(dataDir, userID)` 不存在时，`os.MkdirAll` 目标目录后 `os.Rename` 旧库到目标；否则不做任何事。
- `Init(dataDir string)` 保留，实现为 `Open(filepath.Join(dataDir, "quick-feishu.db"))`，兼容既有 `db_test.go`。

### 3.2 App 生命周期（`internal/app`）

- `App` 新增字段 `dataDir string` 与 `db *gorm.DB`。
- 构造函数改为 `New(cfg *config.Config, dataDir, configPath string) (*App, error)`：
  1. `db.MigrateLegacy(dataDir, cfg.Account.UserID)`
  2. `db.Open(db.Path(dataDir, cfg.Account.UserID))`
  3. `db.SeedDicts(gdb)`
  4. 初始化 client；返回 App。
- `func (a *App) DB() *gorm.DB`：`a.mu.Lock()` 后返回 `a.db`（替代直接字段访问）。
- `RunSnapshot` / `RunReport` / `Backfill` 内部改用 `a.DB()`。
- `Restart()` 流程：
  1. 在 `a.mu` 下：重载 `config.Load`；
  2. 计算 `next := db.Path(a.dataDir, a.Config.Account.UserID)`；
  3. 若 `next != a.currentDBPath`：`MigrateLegacy` → `db.Open(next)` → `db.SeedDicts` → 关闭旧库（`sqlDB.Close()`）→ `a.db = 新库` → `a.currentDBPath = next`；标记 `switched = true`；
  4. 重建 client、`startLocked()` 重启调度器；
  5. 释放 `a.mu` 后，若 `switched` 则调用 `a.Backfill()`。
     **注意**：`Backfill()` 内部会调用 `a.DB()` / `a.Client()`（均会再次加锁），因此必须在 `Restart` 释放锁之后再调用，否则死锁。
- `currentDBPath string` 字段记录当前库路径（在 `New` 中初始化），用于判断是否需要切换。

### 3.3 Handlers 取库（`internal/server`）

- `Handlers` 去掉 `DB *gorm.DB` 字段，保留 `App *app.App`、`Config *config.Config`、`ConfigPath string`。
- `handlers.go` / `handlers2.go` / `handlers3.go` 中所有 `h.DB` 改为 `h.App.DB()`。
- `main.go` 构造 `Handlers{App: a, Config: a.Config, ConfigPath: a.ConfigPath}`（不再传 DB）。

### 3.4 数据流

```
启动: config.user_id -> db.Path -> MigrateLegacy(首次) -> db.Open -> SeedDicts -> App.db
重启: 重载 config -> user_id 变? -> 开新库+SeedDicts+关旧库+Backfill : 复用
请求: handler -> h.App.DB() -> 当前账号库
```

## 4. 影响范围

- 修改：`internal/db/db.go`、`internal/app/app.go`、`main.go`、`internal/server/handlers.go`、`internal/server/handlers2.go`、`internal/server/handlers3.go`。
- 测试修改：`internal/app/app_test.go`、`internal/server/handlers_test.go`、`internal/server/privacy_test.go`（`h.DB` → `h.App.DB()`，改用 `app.New`）。
- 不涉及数据表结构变更；每个库首次打开时 `AutoMigrate` + `SeedDicts`。

## 5. 测试策略

- `internal/db`：
  - `TestPath`：有/无 user_id 两种路径。
  - `TestMigrateLegacy`：旧库存在→移动；目标已存在→不覆盖；user_id 为空→不迁移。
- `internal/app`：
  - `TestRestartSwitchesDatabase`：写入账号 A 库后把 config 改为账号 B 并 `Restart`，断言 `a.DB()` 指向 B 库（A 的数据在 B 库不可见）。该测试的 config `APIBase` 指向 `httptest.NewServer`，使切库后的 `Backfill` 保持 hermetic、不访问真实网络。
  - `TestRestartSameUserKeepsDatabase`：user_id 不变时复用同库（不丢数据）。
  - `TestEmptyUserIDUsesLegacyPath`：空 user_id 时路径为 `data/quick-feishu.db`。
- `internal/server`：既有测试改用 `h.App.DB()` 后全绿。
- 验证命令：`go test ./...`、`go build ./...`，重编重启后手动切换账号确认历史隔离。

## 6. 边界与风险

- `user_id` 为空 → 用 `data/quick-feishu.db`，不迁移。
- 自动迁移只在「旧库存在且目标库不存在」时发生一次，避免覆盖账号库。
- 切库在 `a.mu` 下进行，`DB()` 读取受保护；本地单用户场景不做查询引用计数，切库瞬间若有在途查询可能因 SQLite 关闭连接报错，属可接受风险。
- 旧库移动后，原 `data/quick-feishu.db` 不再存在；若用户回退版本，需手动移回（文档说明）。
