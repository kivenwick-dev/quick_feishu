# QuickFeishu — AI 平台日报工具 设计文档

> 日期：2026-09-18
> 状态：已确认（用户逐章批准）
> 技术栈：Go + Gin + GORM(SQLite) + robfig/cron + Vue3 + Element Plus

## 1. 项目概述

调用 QuickRouter 平台三个接口获取平台信息，每日定时采集快照，通过飞书机器人 Webhook 向群内发送定制化日报。

**核心能力：**
- 采集：令牌列表、各令牌使用情况、账号信息
- 存储：每日快照 + 全量 JSON + 字段字典
- 日报：自定义模板（选日期 + 选字段 + 差值计算），飞书交互卡片
- 配置：YAML 可视化编辑 + 环境变量覆盖
- 形态：单二进制，双击启动，自动打开浏览器

## 2. 目标平台与形态

- **二进制文件**：macOS + Linux + Windows 交叉编译
- **双击启动**：自动启动 Web 服务 + 后台定时任务 + 自动打开默认浏览器
- **零命令**：所有操作（配置、手动发送、查看快照）在 Web 页面完成
- **服务器部署**：裸二进制 + `serve` 参数，命令行运行

## 3. 启动入口

```
quick-feishu serve            # 启动 Web 服务 + 定时任务（主模式）
quick-feishu run-report       # 手动：立即采集 + 立即发日报（测试）
quick-feishu snapshot         # 手动：只采集快照不发日报
```

双击二进制等价于 `serve` 模式。

## 4. 三个 QuickRouter 接口（已实测验证）

### 4.1 令牌列表 `GET /api/token/`
- **认证**：Header `new-api-user`(账户id) + `Authorization: Bearer <系统令牌>`
- **Query 参数**：`p`(页码)、`size`(每页数量)
- **关键字段**：`id, user_id, key, status, name, created_time, accessed_time, expired_time, remain_quota, unlimited_quota, model_limits_enabled, used_quota, group_ids, group`

### 4.2 令牌使用情况 `GET /api/usage/token/`
- **认证**：Header `Authorization: Bearer <令牌自身key>`（**注意：不是系统令牌**）
- **关键字段**：`expires_at, model_limits, model_limits_enabled, name, object, total_available, total_granted, total_used, unlimited_quota`

### 4.3 账号信息 `GET /api/user/self`
- **认证**：Header `new-api-user`(账户id) + `Authorization: Bearer <系统令牌>`
- **关键字段**：`id, username, display_name, role, status, quota, used_quota, request_count, group_id, group, created_at`

### 4.4 采集流程
```
令牌列表(系统令牌) → 拿到每个令牌的 key → 用 key 作为 Bearer 调 usage 逐令牌
```

### 4.5 实测账号
- user_id: 827947
- 系统令牌: H53aSL+5WayLHN7XfER8ilPuLp2jiQ==
- 两个令牌：claude (Kiro-Claude-1)、gpt (Codex-Gpt-1)

## 5. 总体架构

```
┌─────────────────────────────────────────────────┐
│                quick-feishu 工具                   │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐   │
│  │ Web 界面  │    │ 定时调度   │    │ CLI 入口  │   │
│  │ Gin+Vue  │    │ cron      │    │ (启动/调试)│   │
│  └────┬─────┘    └────┬─────┘    └────┬─────┘   │
│       ▼               ▼               ▼         │
│  ┌─────────────────────────────────────────┐    │
│  │             采集服务 (Collector)          │    │
│  │  令牌列表 → 逐个令牌 usage → 账号信息       │    │
│  └─────────────────────────────────────────┘    │
│       │                 │                       │
│       ▼                 ▼                       │
│  ┌──────────┐    ┌──────────────┐               │
│  │ 存储层     │    │ 日报引擎      │               │
│  │ GORM+     │    │ 差值计算      │               │
│  │ SQLite    │    │ 卡片生成      │               │
│  └──────────┘    └──────┬───────┘               │
│                         ▼                       │
│                ┌──────────────┐                 │
│                │ 飞书推送      │                 │
│                │ Webhook      │                 │
│                └──────────────┘                 │
└─────────────────────────────────────────────────┘
```

### 模块职责
| 模块 | 职责 | 依赖 |
| --- | --- | --- |
| 采集服务 | 调三个接口，返回结构化数据 | 无（只发 HTTP） |
| 存储层 | 快照/历史记录 读写 | GORM+SQLite |
| 日报引擎 | 取相邻快照算差值，按映射表选字段，生成卡片 JSON | 存储层+配置 |
| 飞书推送 | POST 卡片到 Webhook，重试 | 无 |
| 调度/入口 | cron 触发、CLI 命令、HTTP 路由 | 上述全部 |

## 6. 数据模型（SQLite）

### 6.1 存储策略：全量 JSON + 冗余索引字段
- 三接口响应全字段落库（JSON 原样存储）
- 冗余常用字段作为索引列（便于差值计算）
- 接口加新字段无需迁移历史数据

### 6.2 快照表 `snapshots`
```go
type Snapshot struct {
    ID            uint
    SnapshotDate  string          // "2026-09-18" 北京时间
    AccountRaw    datatypes.JSON  // /api/user/self 完整响应
    TokenListRaw  datatypes.JSON  // /api/token/ 完整响应
    TokenUsageRaw datatypes.JSON  // 各令牌 usage 完整响应集合
    AccountQuota  int64           // 冗余索引
    AccountUsed   int64           // 冗余索引
    RequestCount  int64           // 冗余索引
    CreatedAt     time.Time
}
```

### 6.3 令牌快照表 `token_snapshots`
```go
type TokenSnapshot struct {
    ID           uint
    SnapshotID   uint
    TokenID      int
    TokenName    string
    UsageRaw     datatypes.JSON   // usage 接口完整响应
    ListRaw      datatypes.JSON   // 该令牌在列表接口的完整字段
    UsedQuota    int64            // 冗余索引
    RemainQuota  int64
    TotalUsed    int64
    TotalGranted int64
}
```

### 6.4 发送日志表 `send_logs`
```go
type SendLog struct {
    ID         uint
    SendTime   time.Time
    Date       string
    Success    bool
    ErrorMsg   string
    FeishuResp string
}
```

## 7. 字段字典（内置可编辑，按接口分三张表）

映射表 = 字段字典，记录每个字段的**含义**。日报展示字段由使用者从字典挑选。

### 7.1 账号信息字段表 `dict_account`（/api/user/self）
```go
type DictAccountField struct {
    ID          uint
    FieldPath   string   // "quota"、"used_quota"...
    Label       string   // 中文含义
    FieldType   string   // string/int/bool/object...
    Description string
    IsDefault   bool     // 内置不可删
}
```

### 7.2 令牌列表字段表 `dict_token`（/api/token/）
```go
type DictTokenField struct {
    ID          uint
    FieldPath   string
    Label       string
    FieldType   string
    Description string
    IsDefault   bool
}
```

### 7.3 令牌使用情况字段表 `dict_usage`（/api/usage/token/）
```go
type DictUsageField struct {
    ID          uint
    FieldPath   string
    Label       string
    FieldType   string
    Description string
    IsDefault   bool
}
```

### 7.4 内置填充
首次启动自动填充三张字典表（基于真实响应整理全部字段及中文含义）。页面可查看、编辑含义、追加新字段。令牌列表字段（含 id、key、group_ids 等）全部内置。

### 7.5 作用链路
```
内置字典（字段含义）→ 页面勾选字段 → 组成日报模板 → 从快照取字段 → 算差值 → 生成卡片
```

## 8. 日报模板

### 8.1 三层挑选
1. **挑日期**：选择哪两天快照做差值（默认自动=最近两天，可自定义）
2. **挑字段**：从字段字典勾选要展示的字段
3. **算差值**：勾选字段展示"今日值-昨日值"或当前值

### 8.2 模板结构（YAML 存储，页面可视化编辑）
```yaml
report_template:
  title: "AI 平台日报"
  date_mode: "auto"              # auto=昨天vs前天, manual=指定两日期
  send_time: "10:30"
  sections:
    - section: "账号概况"
      source: "account"
      fields:
        - field: "used_quota"
          diff: true
        - field: "request_count"
          diff: true
    - section: "各令牌用量"
      source: "token"
      per_token: true
      fields:
        - field: "total_used"
          diff: true
        - field: "remain_quota"
          diff: false
```

### 8.3 页面操作流程
1. 选快照日期（起止，默认自动，可预览原始快照）
2. 勾选字段（三接口字典分别展示）
3. 设差值开关（差值或当前值）
4. 分组编排（分区、排序、标题）
5. 保存模板 → YAML → 每日按模板拼装

### 8.4 差值计算规则
- 取快照 A(早) 与快照 B(晚)，diff 字段计算 `B - A`
- 数值字段直接相减；字符串/布尔字段跳过差值只显示 B 值
- 快照缺失：用最近可用快照替代，卡片标注"数据截至 X 日"
- 首次运行无历史：只发当前值，标注"无历史对比"

## 9. 飞书交互卡片

- `msg_type: interactive`
- header：标题（含日期）+ 颜色
- elements：div 字段块（is_short 两两并排）+ hr 分隔线
- 差值正负颜色标记
- 卡片元素上限约 50 个 field：模板字段数量限制，令牌多时分页
- 推送失败重试：3 次指数退避（2s/4s/8s），仍失败记 send_logs 页面标红

## 10. 配置体系

### 10.1 config.yaml（默认生成在可执行文件同目录）
```yaml
app:
  port: 8080
  timezone: "Asia/Shanghai"
account:
  user_id: "827947"
  system_token: "xxx"
  api_base: "https://api.quickrouter.ai"
feishu:
  webhook_url: "https://open.feishu.cn/open-apis/bot/v2/hook/xxx"
  retry_times: 3
schedule:
  snapshot_time: "00:00"
  report_time: "10:30"
report_template:
  ...
```

### 10.2 环境变量覆盖（优先级最高）
- `QR_USER_ID` → account.user_id
- `QR_SYSTEM_TOKEN` → account.system_token
- `QR_FEISHU_WEBHOOK` → feishu.webhook_url

### 10.3 配置入口（三选一）
1. 直接编辑 YAML
2. **Web 页面设置表单（主入口）**：user_id、system_token、webhook 可视化编辑，保存写回 YAML
3. 环境变量（服务器部署，优先级最高）

环境变量已设置的项，页面显示"已由环境变量提供"并禁止修改。

### 10.4 端口处理
默认 8080，占用时自动递增（8081/8082...），打开对应浏览器地址。

## 11. Web 页面结构（4 页）

### ① 仪表盘
- 最近快照概览：总配额/已用/请求次数 + 令牌摘要
- 按钮：「立即采集快照」「立即发送日报」（手动触发测试）
- 最近发送记录（成功/失败）

### ② 历史快照
- 快照列表（按日期）
- 详情：账号 + 令牌列表 + 各令牌 usage 全量 JSON
- 对比功能：勾选两个日期显示差值

### ③ 日报模板
- 可视化编辑器：数据来源、勾选字段（字典）、差值开关、分区编排、标题
- 字段字典管理：查看/编辑含义
- 实时预览卡片

### ④ 设置
- 账号信息、飞书 webhook（含测试按钮）、发送时间、时区、端口
- 环境变量覆盖项状态显示

## 12. 项目结构

```
quick-feishu/
├── main.go                    # 入口：命令分发
├── cmd/
│   └── serve.go
├── internal/
│   ├── config/                # YAML 加载、环境变量覆盖
│   │   ├── config.go
│   │   └── default.yaml
│   ├── api/                   # QuickRouter 三接口客户端
│   │   ├── client.go
│   │   ├── account.go
│   │   ├── tokenlist.go
│   │   └── usage.go
│   ├── model/                 # GORM 模型
│   ├── db/                    # SQLite 初始化、建表、内置字典
│   ├── collector/             # 采集服务
│   ├── report/                # 日报引擎
│   ├── feishu/                # 飞书推送
│   ├── scheduler/             # cron 调度
│   ├── server/                # Gin 路由 + 静态资源
│   └── frontend/              # Vue 源码
├── web/                       # Vue 构建产物（embed）
├── config.yaml
└── data/
```

## 13. 技术栈

| 用途 | 库 |
| --- | --- |
| Web 框架 | gin-gonic/gin |
| ORM | gorm.io/gorm + gorm.io/driver/sqlite |
| 定时 | robfig/cron/v3 |
| YAML | gopkg.in/yaml.v3 |
| SQLite 驱动 | glebarez/sqlite（纯 Go 无 CGO，三平台交叉编译） |
| 前端 | Vue3 + Vite + Element Plus |

构建流程：Vue `npm run build` → 产物 web/ → Go `embed` → 交叉编译三平台。

## 14. 错误处理 & 边界场景

| 场景 | 处理 |
| --- | --- |
| 单接口失败 | 重试3次(2s/4s/8s)，仍失败标 null + 记录错误，不阻断其他接口 |
| 令牌列表为空 | 快照入库，日报令牌部分显示无数据 |
| 单个令牌 usage 失败 | 跳过该令牌记录失败，其余照常 |
| 飞书推送失败 | 重试3次，记 send_logs，页面标红可手动重发 |
| Webhook 失效 | 页面提示 + 仪表盘告警 |
| 快照缺失 | 向前找最近快照替代，标注日期 |
| 首次运行 | 只发当前值，标注无历史对比 |
| 时区 | 固定 Asia/Shanghai 计算快照日期 |
| 关机错过 0 点 | 下次启动补采缺失日期快照 |
| 启动时序 | 启动立即补采当天快照（若无） |

## 15. 测试策略

| 层 | 内容 | 工具 |
| --- | --- | --- |
| 单元 | 差值计算、卡片 JSON、模板解析、环境变量覆盖、JSON 提取 | Go testing + testify |
| 接口 Mock | 三接口客户端 httptest mock | Go 标准库 |
| 集成 | 内存 SQLite 跑采集→存储→日报全链路 | Go |
| 飞书推送 | mock Webhook 验证卡片结构与重试 | Go |
| 前端 | 模板编辑、快照查看 vitest | Vitest |

关键测试点：差值边界（数值/字符串/缺失/首次）、卡片结构（字段数量/分区/配对）、配置加载（默认/覆盖/非法值）。

验证命令：`go test ./...`、`npm run test`、端到端真实调用。

## 16. 已确认的关键决策记录

| 决策点 | 结论 |
| --- | --- |
| 部署形态 | 单二进制，双击启动 + 自动开浏览器 |
| 目标平台 | macOS + Linux + Windows 三平台 |
| 技术栈 | Go + Gin + GORM + SQLite(纯Go) + cron + Vue3/Element Plus |
| 配置 | YAML 页面可视化编辑（主入口）+ 环境变量覆盖 |
| 存储 | SQLite 全量 JSON + 冗余索引，字段字典三表 |
| 日报 | 挑日期 + 挑字段 + 差值，飞书交互卡片 |
| 调度 | 0点快照 + 10:30发日报（可自定义）+ 手动触发 + 补采 |
| 容错 | 重试退避、部分失败不阻断、快照缺失向前替代 |
| 多账号 | 单账号单群（架构预留扩展） |