# 采集问题可关闭弹窗 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 采集部分失败时，用可关闭的 Element Plus 弹窗按类型展示结构化问题，并对「令牌额度已用尽」给出成因说明。

**Architecture:** 在 `collector` 层把错误分类为结构化 `Issue`；`server` 层脱敏映射为响应字段（`issues` / `snapshot_issues`）；前端新增纯函数分组模块 + `CollectionIssuesDialog.vue`，由仪表盘「立即采集快照」与设置页「应用配置并重启」触发。

**Tech Stack:** Go 1.2x + Gin + GORM；Vue3 + Element Plus + Vitest。

**Spec:** `docs/superpowers/specs/2026-09-18-collection-issues-dialog-design.md`

---

## File Structure

| 文件 | 职责 |
| --- | --- |
| `internal/collector/collector.go`（改） | 定义 `Issue`/分类；`Result.Issues` 取代 `Result.Errors` |
| `internal/collector/collector_test.go`（改） | 分类表驱动测试；`res.Errors`→`res.Issues` |
| `internal/server/privacy.go`（改） | 删 `publicWarnings`，加 `publicIssues` + `issueView` |
| `internal/server/privacy_test.go`（改） | 新增 `publicIssues` 脱敏断言 |
| `internal/server/handlers.go`（改） | `RunSnapshot` 返回 `issues` |
| `internal/server/handlers2.go`（改） | `RestartScheduler` 返回 `snapshot_issues` |
| `internal/app/app.go`（改） | `OnSnapshot` 日志打印改用 `Issues` |
| `frontend/src/collectionIssues.ts`（新） | Issue 类型、分组纯函数、额度耗尽文案常量 |
| `frontend/src/collectionIssues.test.ts`（新） | 分组纯函数单测 |
| `frontend/src/components/CollectionIssuesDialog.vue`（新） | Element Plus 弹窗组件 |
| `frontend/src/components/CollectionIssuesDialog.test.ts`（新） | 组件渲染/关闭单测 |
| `frontend/src/views/Dashboard.vue`（改） | 「立即采集快照」后按 `issues` 弹窗 |
| `frontend/src/views/Settings.vue`（改） | 「应用配置并重启」后按 `snapshot_issues` 弹窗 |

---

## Task 1: collector 结构化分类

**Files:**
- Modify: `internal/collector/collector.go`
- Test: `internal/collector/collector_test.go`

- [ ] **Step 1: 写失败测试**

在 `internal/collector/collector_test.go` 顶部 import 增加 `"errors"`，并在文件末尾追加：

```go
func TestClassifyIssue(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"quota exhausted", &api.HTTPError{Status: 401, Body: `{"error":{"message":"该令牌额度已用尽 (request id: x)"}}`}, KindQuotaExhausted},
		{"unauthorized", &api.HTTPError{Status: 401, Body: `{"error":{"message":"invalid token"}}`}, KindUnauthorized},
		{"forbidden", &api.HTTPError{Status: 403, Body: `{"error":{"message":"forbidden"}}`}, KindUnauthorized},
		{"server error", &api.HTTPError{Status: 502, Body: `bad gateway`}, KindServerError},
		{"network", errors.New("dial tcp: timeout"), KindNetwork},
		{"other", &api.HTTPError{Status: 400, Body: `{"error":{"message":"bad request"}}`}, KindOther},
	}
	for _, tc := range cases {
		got := classify("usage", "tok", tc.err)
		if got.Kind != tc.want {
			t.Errorf("%s: kind = %s, want %s", tc.name, got.Kind, tc.want)
		}
	}
}

func TestClassifyDoesNotLeakRawBody(t *testing.T) {
	got := classify("usage", "tok", &api.HTTPError{
		Status: 401,
		Body:   `{"error":{"message":"该令牌额度已用尽"},"secret":"LEAK"}`,
	})
	if got.Detail != "该令牌额度已用尽" {
		t.Errorf("detail = %q", got.Detail)
	}
	if strings.Contains(got.Detail, "LEAK") {
		t.Fatalf("raw body leaked: %s", got.Detail)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/collector/ -run 'TestClassify' -v`
Expected: 编译失败（`undefined: KindQuotaExhausted` / `classify`）

- [ ] **Step 3: 实现分类与 Issue 类型**

把 `internal/collector/collector.go` 的 import 块改为：

```go
import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"quick-feishu/internal/api"
	"quick-feishu/internal/model"
)
```

在 `type Result struct` 之前加入类型与常量，并把 `Result.Errors []string` 改为 `Issues []Issue`：

```go
// 采集问题分类
const (
	KindQuotaExhausted = "quota_exhausted"
	KindUnauthorized   = "unauthorized"
	KindNetwork        = "network"
	KindServerError    = "server_error"
	KindOther          = "other"
)

// Issue 描述一次采集中的单个失败项。脱敏：不含令牌 key，Detail 仅保留平台 message。
type Issue struct {
	Scope     string `json:"scope"`      // account | tokenlist | usage
	TokenName string `json:"token_name"` // usage 时为令牌名
	Kind      string `json:"kind"`
	Status    int    `json:"status"` // HTTP 状态码；网络错误为 0
	Detail    string `json:"detail"`
}

type Result struct {
	Account   *api.AccountData
	TokenList *api.TokenListData
	Usages    map[int]*api.TokenUsageData
	Issues    []Issue
}

// classify 将一次采集错误归类为 Issue。
func classify(scope, tokenName string, err error) Issue {
	issue := Issue{Scope: scope, TokenName: tokenName, Kind: KindOther}
	var he *api.HTTPError
	if !errors.As(err, &he) {
		issue.Kind = KindNetwork
		issue.Detail = err.Error()
		return issue
	}
	issue.Status = he.Status
	issue.Detail = platformMessage(he.Body)
	if issue.Detail == "" {
		issue.Detail = fmt.Sprintf("HTTP %d", he.Status)
	}
	switch {
	case he.Status == http.StatusUnauthorized && strings.Contains(he.Body, "额度已用尽"):
		issue.Kind = KindQuotaExhausted
	case he.Status == http.StatusUnauthorized || he.Status == http.StatusForbidden:
		issue.Kind = KindUnauthorized
	case he.Status >= 500:
		issue.Kind = KindServerError
	}
	return issue
}

// platformMessage 只从 {"error":{"message":"..."}} 取 message，绝不回传原始 body。
func platformMessage(body string) string {
	var envelope struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		return ""
	}
	return envelope.Error.Message
}
```

- [ ] **Step 4: 让 Collect 产生 Issues**

把 `Collect` 函数体替换为：

```go
func Collect(c *api.Client) *Result {
	res := &Result{Usages: map[int]*api.TokenUsageData{}, Issues: []Issue{}}

	account, _, err := c.GetAccount()
	if err != nil {
		res.Issues = append(res.Issues, classify("account", "", err))
	} else {
		res.Account = account
	}

	list, _, err := c.GetTokenList()
	if err != nil {
		res.Issues = append(res.Issues, classify("tokenlist", "", err))
		res.TokenList = &api.TokenListData{Items: []api.TokenItem{}}
	} else {
		res.TokenList = list
		for _, it := range list.Items {
			usage, _, uerr := c.GetTokenUsage(it.Key)
			if uerr != nil {
				res.Issues = append(res.Issues, classify("usage", it.Name, uerr))
				continue
			}
			res.Usages[it.ID] = usage
		}
	}
	return res
}
```

- [ ] **Step 5: 更新旧测试中对 Errors 的引用**

`internal/collector/collector_test.go` 中 `TestSaveStoresUnknownFields` 内：

```go
	if len(res.Errors) != 0 {
		t.Fatalf("collect errors: %v", res.Errors)
	}
```

改为：

```go
	if len(res.Issues) != 0 {
		t.Fatalf("collect issues: %v", res.Issues)
	}
```

- [ ] **Step 6: 运行 collector 测试**

Run: `go test ./internal/collector/ -v`
Expected: PASS（含新 `TestClassifyIssue`、`TestClassifyDoesNotLeakRawBody`）

- [ ] **Step 7: 提交**

```bash
git add internal/collector/collector.go internal/collector/collector_test.go
git commit -m "feat: classify collection failures into structured issues"
```

---

## Task 2: app 日志适配 + 编译修复

**Files:**
- Modify: `internal/app/app.go:156`
- Modify: `internal/scheduler/backfill_test.go:82`（如引用 `Errors`）
- Modify: `internal/scheduler/backfill.go`（如引用 `Errors`）

- [ ] **Step 1: 更新 OnSnapshot 日志**

`internal/app/app.go` 中：

```go
		for _, e := range res.Errors {
			fmt.Println("  collect warning:", e)
		}
```

改为：

```go
		for _, e := range res.Issues {
			fmt.Printf("  collect issue: kind=%s scope=%s token=%s status=%d detail=%s\n", e.Kind, e.Scope, e.TokenName, e.Status, e.Detail)
		}
```

- [ ] **Step 2: 全量编译，修复其它 Errors 引用**

Run: `go build ./...`
Expected: 若报 `res.Errors undefined`，把该处改为 `res.Issues`（`backfill.go`/`backfill_test.go` 目前只用 `collector.Result{...}` 字段初始化，无需改；若报则同步）。

- [ ] **Step 3: 运行 app/scheduler 测试**

Run: `go test ./internal/app/ ./internal/scheduler/ -v`
Expected: PASS

- [ ] **Step 4: 提交**

```bash
git add internal/app/app.go internal/scheduler/
git commit -m "chore: log structured collection issues in scheduler"
```

---

## Task 3: server 响应改为 issues

**Files:**
- Modify: `internal/server/privacy.go`
- Modify: `internal/server/handlers.go:37`
- Modify: `internal/server/handlers2.go:198-202`
- Test: `internal/server/privacy_test.go`

- [ ] **Step 1: 写失败测试**

在 `internal/server/privacy_test.go` 末尾追加（import 增加 `"quick-feishu/internal/collector"`）：

```go
func TestPublicIssuesKeepsTokenNameAndKind(t *testing.T) {
	out := publicIssues([]collector.Issue{
		{Scope: "usage", TokenName: "gpt6robodjo", Kind: collector.KindQuotaExhausted, Status: 401, Detail: "该令牌额度已用尽"},
	})
	if len(out) != 1 {
		t.Fatalf("view len = %d", len(out))
	}
	if out[0].TokenName != "gpt6robodjo" || out[0].Kind != collector.KindQuotaExhausted || out[0].Status != 401 {
		t.Fatalf("bad view: %+v", out[0])
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/server/ -run TestPublicIssues -v`
Expected: 编译失败（`undefined: publicIssues`）

- [ ] **Step 3: 实现 publicIssues，删除 publicWarnings**

`internal/server/privacy.go`：import 块加入 `"quick-feishu/internal/collector"`；删除：

```go
func publicWarnings(errors []string) []string {
	if len(errors) == 0 {
		return []string{}
	}
	return []string{"部分数据采集失败，请检查服务端配置或日志"}
}
```

替换为：

```go
type issueView struct {
	Scope     string `json:"scope"`
	TokenName string `json:"token_name"`
	Kind      string `json:"kind"`
	Status    int    `json:"status"`
	Detail    string `json:"detail"`
}

// publicIssues 输出脱敏后的采集问题；不含令牌 key。
func publicIssues(issues []collector.Issue) []issueView {
	out := []issueView{}
	for _, is := range issues {
		out = append(out, issueView{is.Scope, is.TokenName, is.Kind, is.Status, is.Detail})
	}
	return out
}
```

- [ ] **Step 4: 更新两个 handler 响应字段**

`internal/server/handlers.go` 的 `RunSnapshot`：

```go
	c.JSON(http.StatusOK, gin.H{"success": true, "date": app.Today(), "issues": publicIssues(res.Issues)})
```

`internal/server/handlers2.go` 的 `RestartScheduler`：

```go
	} else if res != nil && len(res.Issues) > 0 {
		resp["snapshot_issues"] = publicIssues(res.Issues)
	}
```

- [ ] **Step 5: 运行 server 测试**

Run: `go test ./internal/server/ -v`
Expected: PASS

- [ ] **Step 6: 全量测试 + 提交**

```bash
go test ./...
git add internal/server/
git commit -m "feat: return structured collection issues from snapshot and restart APIs"
```

---

## Task 4: 前端分组纯函数

**Files:**
- Create: `frontend/src/collectionIssues.ts`
- Test: `frontend/src/collectionIssues.test.ts`

- [ ] **Step 1: 写失败测试**

`frontend/src/collectionIssues.test.ts`：

```ts
import { describe, it, expect } from 'vitest'
import { groupIssues, QUOTA_EXHAUSTED_HINT, type Issue } from './collectionIssues'

const base: Issue = { scope: 'usage', token_name: '', kind: 'other', status: 0, detail: '' }

describe('groupIssues', () => {
  it('groups by kind and lists token names in order', () => {
    const groups = groupIssues([
      { ...base, kind: 'quota_exhausted', token_name: 'a', status: 401 },
      { ...base, kind: 'quota_exhausted', token_name: 'b', status: 401 },
      { ...base, kind: 'network' },
    ])
    expect(groups.map((g) => g.kind)).toEqual(['quota_exhausted', 'network'])
    expect(groups[0].tokens).toEqual(['a', 'b'])
    expect(groups[0].title).toBe('令牌额度已用尽')
  })

  it('handles empty input and unknown kinds', () => {
    expect(groupIssues([])).toEqual([])
    const groups = groupIssues([{ ...base, kind: 'weird' }])
    expect(groups[0].kind).toBe('weird')
  })

  it('exposes the quota exhausted explanation', () => {
    expect(QUOTA_EXHAUSTED_HINT.join('')).toContain('used_quota')
    expect(QUOTA_EXHAUSTED_HINT.join('')).toContain('额度已用尽')
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && npx vitest run src/collectionIssues.test.ts`
Expected: FAIL（无法解析 `./collectionIssues`）

- [ ] **Step 3: 实现 collectionIssues.ts**

`frontend/src/collectionIssues.ts`：

```ts
export interface Issue {
  scope: string
  token_name: string
  kind: string
  status: number
  detail: string
}

export interface IssueGroup {
  kind: string
  title: string
  tokens: string[]
  issues: Issue[]
}

export const KIND_TITLES: Record<string, string> = {
  quota_exhausted: '令牌额度已用尽',
  unauthorized: '令牌鉴权失败',
  server_error: '平台服务端错误',
  network: '网络错误',
  other: '其他错误',
}

export const QUOTA_EXHAUSTED_HINT: string[] = [
  '以下令牌是有限额度令牌，累计用量 used_quota 已达到并略微超过 total。',
  '在 new-api/quickrouter 这类平台上，一旦令牌用量触顶，平台会拒绝该令牌的所有请求——包括只读的 /api/usage/token/ 自检接口，返回 401「该令牌额度已用尽」。',
  '这不是采集逻辑的 bug，而是该令牌在平台侧已被停用；在额度恢复前，这些令牌的 usage 数据将一直无法采集。',
  '处理方式：充值/提高 total、改为不限额度、停用或删除该令牌。',
]

const KIND_ORDER = ['quota_exhausted', 'unauthorized', 'server_error', 'network', 'other']

export function groupIssues(issues: Issue[]): IssueGroup[] {
  const map = new Map<string, IssueGroup>()
  for (const it of issues) {
    const kind = it.kind || 'other'
    let group = map.get(kind)
    if (!group) {
      group = { kind, title: KIND_TITLES[kind] || kind, tokens: [], issues: [] }
      map.set(kind, group)
    }
    group.issues.push(it)
    if (it.token_name && !group.tokens.includes(it.token_name)) {
      group.tokens.push(it.token_name)
    }
  }
  const rank = (k: string) => (KIND_ORDER.includes(k) ? KIND_ORDER.indexOf(k) : KIND_ORDER.length)
  return [...map.values()].sort((a, b) => rank(a.kind) - rank(b.kind))
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd frontend && npx vitest run src/collectionIssues.test.ts`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add frontend/src/collectionIssues.ts frontend/src/collectionIssues.test.ts
git commit -m "feat: add collection issue grouping helper"
```

---

## Task 5: Element Plus 弹窗组件

**Files:**
- Create: `frontend/src/components/CollectionIssuesDialog.vue`
- Test: `frontend/src/components/CollectionIssuesDialog.test.ts`

- [ ] **Step 1: 写失败测试**

`frontend/src/components/CollectionIssuesDialog.test.ts`：

```ts
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import CollectionIssuesDialog from './CollectionIssuesDialog.vue'

const issues = [
  { scope: 'usage', token_name: 'gpt6robodjo', kind: 'quota_exhausted', status: 401, detail: '该令牌额度已用尽' },
]

describe('CollectionIssuesDialog', () => {
  it('renders quota exhausted token name and explanation', () => {
    const wrapper = mount(CollectionIssuesDialog, {
      props: { modelValue: true, issues },
      global: { plugins: [ElementPlus] },
    })
    expect(wrapper.text()).toContain('gpt6robodjo')
    expect(wrapper.text()).toContain('used_quota')
  })

  it('renders nothing meaningful without issues', () => {
    const wrapper = mount(CollectionIssuesDialog, {
      props: { modelValue: true, issues: [] },
      global: { plugins: [ElementPlus] },
    })
    expect(wrapper.findAll('.issue-group').length).toBe(0)
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && npx vitest run src/components/CollectionIssuesDialog.test.ts`
Expected: FAIL（找不到组件）

- [ ] **Step 3: 实现组件**

`frontend/src/components/CollectionIssuesDialog.vue`：

```vue
<template>
  <el-dialog
    :model-value="modelValue"
    title="采集提醒"
    width="640px"
    @update:model-value="(v: boolean) => emit('update:modelValue', v)"
  >
    <el-alert
      v-if="hasQuotaExhausted"
      type="warning"
      :closable="false"
      show-icon
      title="令牌额度已用尽"
      class="quota-alert"
    >
      <p v-for="(line, i) in QUOTA_EXHAUSTED_HINT" :key="i" class="hint-line">{{ line }}</p>
    </el-alert>

    <div v-for="g in groups" :key="g.kind" class="issue-group">
      <div class="group-title">{{ g.title }}</div>
      <el-descriptions :column="1" border size="small">
        <el-descriptions-item v-for="(it, i) in g.issues" :key="i" :label="scopeLabel(it.scope)">
          <span v-if="it.token_name" class="token-name">{{ it.token_name }} · </span>
          <span class="detail">{{ it.detail || `HTTP ${it.status}` }}</span>
        </el-descriptions-item>
      </el-descriptions>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { groupIssues, QUOTA_EXHAUSTED_HINT, type Issue } from '../collectionIssues'

const props = defineProps<{ modelValue: boolean; issues: Issue[] }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const groups = computed(() => groupIssues(props.issues || []))
const hasQuotaExhausted = computed(() => groups.value.some((g) => g.kind === 'quota_exhausted'))

const SCOPE_LABELS: Record<string, string> = {
  account: '账号信息',
  tokenlist: '令牌列表',
  usage: '令牌用量',
}
function scopeLabel(scope: string) {
  return SCOPE_LABELS[scope] || scope
}
</script>

<style scoped>
.quota-alert { margin-bottom: 16px; }
.hint-line { margin: 4px 0; font-size: 13px; line-height: 1.6; }
.issue-group { margin-top: 12px; }
.group-title { font-weight: 600; margin-bottom: 8px; }
.token-name { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
.detail { color: var(--el-text-color-regular, #606266); }
</style>
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd frontend && npx vitest run src/components/CollectionIssuesDialog.test.ts`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add frontend/src/components/CollectionIssuesDialog.vue frontend/src/components/CollectionIssuesDialog.test.ts
git commit -m "feat: add closable collection issues dialog"
```

---

## Task 6: 接线仪表盘与设置页

**Files:**
- Modify: `frontend/src/views/Dashboard.vue`
- Modify: `frontend/src/views/Settings.vue`

- [ ] **Step 1: 更新 Dashboard.vue**

`<script setup>` 中 import 增加：

```ts
import CollectionIssuesDialog from '../components/CollectionIssuesDialog.vue'
import type { Issue } from '../collectionIssues'
```

新增状态：

```ts
const issues = ref<Issue[]>([])
const issuesDialog = ref(false)
```

`runSnapshot` 中成功分支改为：

```ts
    const res = await api.runSnapshot()
    ElMessage.success(`快照已采集: ${res.data.date}`)
    issues.value = res.data.issues || []
    if (issues.value.length) issuesDialog.value = true
    load()
```

template 末尾（`</div>` 之前）加入：

```vue
    <CollectionIssuesDialog v-model="issuesDialog" :issues="issues" />
```

- [ ] **Step 2: 更新 Settings.vue**

`<script setup>` 中 import 增加：

```ts
import CollectionIssuesDialog from '../components/CollectionIssuesDialog.vue'
import type { Issue } from '../collectionIssues'
```

新增状态（放在 `restarting` 之后）：

```ts
const issues = ref<Issue[]>([])
const issuesDialog = ref(false)
```

`restart()` 中 `const res = await api.restartScheduler()` 之后改为：

```ts
    const res = await api.restartScheduler()
    const data = res.data || {}
    issues.value = data.snapshot_issues || []
    if (data.snapshot_error) {
      ElMessage.warning('定时任务已重启，但快照采集失败：' + data.snapshot_error)
    } else if (!issues.value.length) {
      ElMessage.success('已应用配置并重新采集当天快照')
    }
    if (issues.value.length) issuesDialog.value = true
    await load()
```

template 中 `</div>`（`set-page` 收尾）之前加入：

```vue
    <CollectionIssuesDialog v-model="issuesDialog" :issues="issues" />
```

- [ ] **Step 3: 前端全量测试与类型检查**

Run: `cd frontend && npm run test && npm run build`
Expected: 所有 vitest 通过；`vue-tsc` 无错误，`vite build` 输出到 `internal/server/web/`

- [ ] **Step 4: 提交**

```bash
git add frontend/src/views/Dashboard.vue frontend/src/views/Settings.vue internal/server/web/
git commit -m "feat: open collection issues dialog after snapshot and restart"
```

---

## Task 7: 端到端验证（真实环境）

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

- [ ] **Step 3: 手动验证**

浏览器打开设置页 → 点「应用配置并重启」：
- 首次保存表单再重启（验证之前的修复）；
- 若存在额度耗尽令牌，应弹出可关闭弹窗，展示令牌名与该段说明；X / ESC / 点遮罩均可关闭；
- 仪表盘点「立即采集快照」同样在有问题时弹窗。

- [ ] **Step 4: 最终提交（如有构建产物变动）**

```bash
git status --short
git add -A
git commit -m "chore: rebuild embedded frontend assets"
```

---

## Self-Review 结果

- **Spec coverage:** 结构化分类（Task 1）、接口字段（Task 3）、Element Plus 弹窗（Task 5）、两处触发（Task 6）、可关闭无「不再提示」（Task 5）、全部错误进弹窗（Task 4/5 按 kind 分组）、测试（Task 1/3/4/5/7）均有对应任务。
- **Placeholder scan:** 无 TBD/TODO；每个代码步骤含完整代码。
- **Type consistency:** `Issue` 字段（`scope/token_name/kind/status/detail`）在 collector、privacy `issueView`、前端 `collectionIssues.ts` 三处一致；`collector.Kind*` 常量与前端 `KIND_TITLES` 键一致；响应字段 `issues`/`snapshot_issues` 在 handler 与前端读取处一致。
