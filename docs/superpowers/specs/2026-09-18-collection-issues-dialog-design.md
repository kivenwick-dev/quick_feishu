# 采集问题可关闭弹窗 — 设计文档

> 日期：2026-09-18
> 状态：已确认（用户逐项批准）
> 关联：`2026-09-18-quick-feishu-design.md`

## 1. 背景与问题

「应用配置并重启」和「立即采集快照」在采集部分失败时，前端只显示一条通用告警：

> 部分数据采集失败，请检查服务端配置或日志

真实原因被 `privacy.go` 的 `publicWarnings` 折叠掉了。实测典型场景：

- 令牌 `gpt6robodjo-zhangkunsheng`（ID 2477230）是**有限额度**令牌：`unlimited_quota=false`、`total=150000000`、`used_quota=150398630`、`remain_quota=-398630`。
- 平台（new-api/quickrouter）在令牌用量触顶后拒绝该令牌**所有**请求，连只读的 `GET /api/usage/token/` 自检也返回 `401 该令牌额度已用尽`。
- 这不是采集逻辑的 bug，而是平台侧停用了令牌；但用户从通用文案里无法得知，误以为服务端配置错误。

**目标**：把这个提示改成一个**可关闭的弹窗**，明确告知出现的问题、涉及的令牌、以及「额度已用尽」的成因与处理建议。

## 2. 已确认的决策

| 决策点 | 结论 |
| --- | --- |
| 触发范围 | 「应用配置并重启」+ 仪表盘「立即采集快照」两个手动采集入口 |
| 内容粒度 | 后端把采集错误**结构化分类**后返回；前端按类型分组展示 |
| 关闭行为 | 仅可关闭（X / ESC / 点遮罩）；无「不再提示」，下次仍弹 |
| 其他错误 | 非「额度已用尽」的采集错误也进同一个弹窗，简要展示 |

## 3. 方案

### 3.1 后端：在 collector 层结构化分类（方案 A）

把 `collector.Result.Errors []string` 改为结构化切片：

```go
type Issue struct {
    Scope     string // account | tokenlist | usage
    TokenName string // usage 时为令牌名；account/tokenlist 为空
    Kind      string // quota_exhausted | unauthorized | network | server_error | other
    Status    int    // HTTP 状态码；网络错误为 0
    Detail    string // 平台返回的 message（脱敏，绝不含 token key）
}
```

分类函数 `classify(scope, tokenName string, err error) Issue`，规则：

| 条件 | Kind |
| --- | --- |
| `*api.HTTPError` 且 `Status==401` 且 body 含「额度已用尽」 | `quota_exhausted` |
| `*api.HTTPError` 且 `Status==401/403` | `unauthorized` |
| `*api.HTTPError` 且 `Status>=500` | `server_error` |
| 非 `*api.HTTPError`（超时/网络） | `network` |
| 其余 | `other` |

`Detail` 只从 `HTTPError.Body` 解析 `{"error":{"message":"..."}}` 取 message；解析失败时退化为 `HTTP <status>`，**绝不落原始 body、绝不含令牌 key**。

### 3.2 接口变更

- `POST /api/snapshot/run`：`errors`（通用字符串数组）→ `issues`（结构化数组）。响应其余字段不变。
- `POST /api/scheduler/restart`：`snapshot_warnings` → `snapshot_issues`。`success` / `snapshot_error` / `next_runs` 不变。
- 删除 `publicWarnings`，新增 `publicIssues([]collector.Issue) []issueView` 做脱敏映射。

无外部调用方，直接替换，不做兼容。

### 3.3 前端

- 新增 `frontend/src/components/CollectionIssuesDialog.vue`：props `modelValue: boolean` + `issues: Issue[]`，**使用 Element Plus 组件**（`el-dialog` 容器、`el-alert`/`el-tag`/`el-descriptions` 展示），可 X/ESC/点遮罩关闭，无「不再提示」。
- 按 `kind` 分组：
  - `quota_exhausted` 组显示成因说明（见 3.4）+ 受影响令牌名列表。
  - 其余 kind 每组一行简述（鉴权失败 / 网络错误 / 平台服务端错误 / 其他）+ 明细。
- `Dashboard.vue` `runSnapshot`：成功 toast 保留；`res.data.issues` 非空时打开弹窗（顺带修复当前忽略 `errors` 的问题）。
- `Settings.vue` `restart`：`data.snapshot_issues` 非空 → 打开弹窗；`data.snapshot_error` → 保留原失败 toast；否则成功 toast。

### 3.4 「额度已用尽」说明文案（前端常量）

> 以下令牌是有限额度令牌，累计用量 used_quota 已达到并略微超过 total。
> 在 new-api/quickrouter 这类平台上，一旦令牌用量触顶，平台会拒绝该令牌的所有请求——包括只读的 /api/usage/token/ 自检接口，返回 401「该令牌额度已用尽」。
> 这不是采集逻辑的 bug，而是该令牌在平台侧已被停用；在额度恢复前，这些令牌的 usage 数据将一直无法采集。
> 处理方式：充值/提高 total、改为不限额度、停用或删除该令牌。

## 4. 数据流

```
定时/手动采集 → collector.Collect()
  → 记录 []Issue（含 kind）
  → collector.Save() 照常入库
  → handler 返回 issues / snapshot_issues
  → 前端 CollectionIssuesDialog 按 kind 分组展示
```

## 5. 测试策略

- Go 单元：`classify` 表驱动（5 种 kind + 参数校验）；`publicIssues` 断言含令牌名与 kind、不含 token key。
- 前端 vitest：分组/筛选纯函数、组件渲染与关闭事件。
- 验证命令：`go test ./...`、`cd frontend && npm run test`、`npm run build`、重编二进制重启后手动点击验证。

## 6. 影响范围与风险

- 影响文件：`internal/collector/collector.go`、`internal/server/privacy.go`、`internal/server/handlers.go`、`internal/server/handlers2.go`、`internal/app/app.go`（日志打印）、`frontend/src/views/Dashboard.vue`、`frontend/src/views/Settings.vue`、新增组件。
- 风险：响应字段重命名（无外部调用方）；`Detail` 脱敏需确保不泄漏 key。
- 前端资源 `go:embed`，改完必须 `go build` 重编 + 重启进程才生效。
