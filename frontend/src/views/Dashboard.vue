<template>
  <div class="dash-page">
    <div class="page-head">
      <h2>仪表盘</h2>
      <p class="sub">最近一次快照的用量与配额变动 · 已隐藏账号隐私信息</p>
    </div>

    <el-space wrap>
      <el-button type="primary" @click="runSnapshot">立即采集快照</el-button>
      <el-button type="success" @click="sendReport">立即发送日报</el-button>
      <el-button @click="testFeishu">测试飞书连接</el-button>
    </el-space>

    <el-card v-if="metricOptions.length" shadow="never" class="metric-filter-card">
      <div class="metric-filter">
        <span class="metric-filter-label">显示指标</span>
        <el-switch
          v-model="usageCurrency"
          active-text="金额"
          inactive-text="配额"
          @change="onUsageCurrencyChange"
        />
        <el-select
          :model-value="visibleMetricKeys"
          class="metric-selector"
          multiple
          collapse-tags
          collapse-tags-tooltip
          placeholder="选择要显示的指标"
          @update:model-value="updateVisibleMetricKeys"
          @change="saveMetricSelection"
        >
          <el-option
            v-for="option in metricOptions"
            :key="option.key"
            :value="option.key"
            :label="option.label"
          />
        </el-select>
      </div>
    </el-card>

    <div v-if="liveBilling" class="currency-panel metric-panel">
      <div class="group-title">
        实时美元账务
        <el-tag size="small" type="success">每 30 秒自动更新</el-tag>
        <el-tag v-if="billingUpdatedAt" size="small" type="info">更新于 {{ billingUpdatedAt }}</el-tag>
      </div>
      <div class="currency-grid">
        <div class="currency-card">
          <span>当前余额</span>
          <strong>{{ formatUSD(liveBilling.balance_usd) }}</strong>
          <small>{{ formatInteger(liveBilling.quota) }} quota</small>
        </div>
        <div class="currency-card">
          <span>历史消耗</span>
          <strong>{{ formatUSD(liveBilling.used_usd) }}</strong>
          <small>{{ formatInteger(liveBilling.used_quota) }} quota</small>
        </div>
        <div class="currency-card">
          <span>汇率</span>
          <strong>{{ formatInteger(liveBilling.quota_per_unit) }} quota</strong>
          <small>= $1.00 USD</small>
        </div>
      </div>
      <div class="currency-formula">
        <span>计算公式</span>
        <div class="formula-list">
          <code>当前额度 ÷ 单位额度 = 当前余额（美元）</code>
          <code>累计已用额度 ÷ 单位额度 = 历史消耗（美元）</code>
        </div>
      </div>
      <el-alert v-if="billingError" type="warning" :closable="false" show-icon :title="billingError" />
    </div>

    <el-empty v-if="!date" description="暂无快照，请先点击『立即采集快照』" :image-size="80" style="margin-top: 24px" />

    <div v-for="sec in visibleSections" :key="sec.key" class="group metric-panel">
      <div class="group-title">
        {{ sec.name }}
        <el-tag v-if="date" size="small" type="info">{{ date }}</el-tag>
      </div>

      <template v-if="sec.tokens && sec.tokens.length">
        <div v-for="(tok, ti) in sec.tokens" :key="ti" class="token-block">
          <div class="token-name">{{ tok.name }}</div>
          <div class="metric-grid">
            <MetricCard
              v-for="(m, mi) in tok.metrics"
              :key="mi"
              :label="m.label"
              :value="m.value"
              :delta="m.has_delta ? m.delta : ''"
              :delta-label="m.delta_label"
              :note="negativeQuotaNote(m.label, m.value)"
            />
          </div>
        </div>
      </template>

      <div v-else class="metric-grid">
        <MetricCard
          v-for="(f, fi) in sec.fields"
          :key="fi"
          :label="f.label"
          :value="f.value"
          :delta="f.is_diff ? f.delta : ''"
          :delta-label="f.delta_label"
        />
      </div>
    </div>

    <div class="group metric-panel">
      <div class="group-title">最近发送记录</div>
      <el-table :data="logs" style="width: 100%">
        <el-table-column prop="send_time" label="发送时间" width="220" />
        <el-table-column prop="date" label="日期" width="130" />
        <el-table-column label="状态" width="100">
          <template #default="scope">
            <el-tag :type="scope.row.success ? 'success' : 'danger'">{{ scope.row.success ? '成功' : '失败' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="error_msg" label="错误" />
      </el-table>
    </div>

    <CollectionIssuesDialog v-model="issuesDialog" :issues="issues" />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../api'
import CollectionIssuesDialog from '../components/CollectionIssuesDialog.vue'
import type { Issue } from '../collectionIssues'
import MetricCard from '../components/MetricCard.vue'
import { negativeQuotaNote } from '../metricNotes'
import { appendSelectedValues } from '../selectionOrder'

const date = ref<string | null>(null)
const liveBilling = ref<any | null>(null)
const billingError = ref('')
const billingUpdatedAt = ref('')
let billingTimer: ReturnType<typeof setInterval> | undefined
let billingRefreshing = false
const realSections = ref<any[]>([])
const sections = computed(() => realSections.value)
const visibleMetricKeys = ref<string[]>([])
const metricSelectionInitialized = ref(false)
const metricSelectionStorageKey = 'quick-feishu.dashboard.visible-metrics'
const usageCurrencyStorageKey = 'quick-feishu.dashboard.usage-currency'
const usageCurrency = ref(true)

const usdFormatter = new Intl.NumberFormat('en-US', {
  style: 'currency',
  currency: 'USD',
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
})
const integerFormatter = new Intl.NumberFormat('en-US', { maximumFractionDigits: 0 })
function formatUSD(value: number) { return usdFormatter.format(value) }
function formatInteger(value: number) { return integerFormatter.format(value) }

function restoreMetricSelection() {
  try {
    const savedCurrency = localStorage.getItem(usageCurrencyStorageKey)
    if (savedCurrency !== null) usageCurrency.value = savedCurrency === 'true'
    const saved = localStorage.getItem(metricSelectionStorageKey)
    if (saved === null) return
    const parsed = JSON.parse(saved)
    if (!Array.isArray(parsed) || !parsed.every((key) => typeof key === 'string')) return
    visibleMetricKeys.value = parsed
    metricSelectionInitialized.value = true
  } catch {
    // Ignore unavailable storage and malformed data; the default is to show all metrics.
  }
}

function saveMetricSelection() {
  try {
    localStorage.setItem(metricSelectionStorageKey, JSON.stringify(visibleMetricKeys.value))
  } catch {
    // Browsers may disable local storage; filtering still works for the current visit.
  }
}

function updateVisibleMetricKeys(next: string[]) {
  visibleMetricKeys.value = appendSelectedValues(visibleMetricKeys.value, next)
}

function saveUsageCurrency() {
  try {
    localStorage.setItem(usageCurrencyStorageKey, String(usageCurrency.value))
  } catch {
    // Browsers may disable local storage; the switch still works for the current visit.
  }
}

restoreMetricSelection()

const metricOptions = computed(() => {
  const options: Array<{ key: string; label: string }> = []
  sections.value.forEach((sec: any, sectionIndex: number) => {
    const labels = new Set<string>()
    ;(sec.fields || []).forEach((field: any) => labels.add(field.label))
    ;(sec.tokens || []).forEach((token: any) => {
      ;(token.metrics || []).forEach((metric: any) => labels.add(metric.label))
    })
    labels.forEach((label) => options.push({
      key: `${sectionIndex}:${label}`,
      label: `${sec.name} · ${label}`,
    }))
  })
  return options
})

const visibleSections = computed(() => {
  const selectionOrder = new Map(visibleMetricKeys.value.map((key, index) => [key, index]))
  const selectedMetrics = (metrics: any[], sectionIndex: number) => metrics
    .filter((metric: any) => selectionOrder.has(`${sectionIndex}:${metric.label}`))
    .sort((left: any, right: any) => (
      selectionOrder.get(`${sectionIndex}:${left.label}`)! - selectionOrder.get(`${sectionIndex}:${right.label}`)!
    ))

  return sections.value.map((sec: any, sectionIndex: number) => ({
    ...sec,
    key: `${sectionIndex}:${sec.name}`,
    fields: selectedMetrics(sec.fields || [], sectionIndex),
    tokens: (sec.tokens || []).map((token: any) => ({
      ...token,
      metrics: selectedMetrics(token.metrics || [], sectionIndex),
    })),
  }))
})

function syncMetricSelection() {
  const available = metricOptions.value.map((option) => option.key)
  if (!metricSelectionInitialized.value) {
    visibleMetricKeys.value = available
    metricSelectionInitialized.value = true
    saveMetricSelection()
    return
  }
  const availableSet = new Set(available)
  const validSelection = visibleMetricKeys.value.filter((key) => availableSet.has(key))
  // An empty or completely stale browser-side selection otherwise hides every
  // card while leaving section/token headings visible. This is especially easy
  // to hit after deploying under a hostname with old localStorage data or after
  // changing the report template labels.
  visibleMetricKeys.value = validSelection.length === 0 && available.length > 0
    ? available
    : validSelection
  saveMetricSelection()
}
const logs = ref<any[]>([])
const issues = ref<Issue[]>([])
const issuesDialog = ref(false)

async function load() {
  const [res, latest] = await Promise.all([
    api.dashboard(),
    api.latest(usageCurrency.value),
  ])
  logs.value = res.data.recent_logs || []
  date.value = latest.data.date
  realSections.value = latest.data.sections || []
  syncMetricSelection()
}

async function onUsageCurrencyChange() {
  saveUsageCurrency()
  await load()
}

async function refreshLiveBilling() {
  if (billingRefreshing) return
  billingRefreshing = true
  try {
    const res = await api.liveBilling()
    liveBilling.value = res.data
    billingUpdatedAt.value = new Date(res.data.updated_at).toLocaleTimeString('zh-CN', { hour12: false })
    billingError.value = ''
  } catch {
    billingError.value = '实时账务更新失败，继续显示上一次成功获取的数据'
  } finally {
    billingRefreshing = false
  }
}

onMounted(() => {
  load()
  refreshLiveBilling()
  billingTimer = setInterval(refreshLiveBilling, 30_000)
})
onBeforeUnmount(() => {
  if (billingTimer) clearInterval(billingTimer)
})

async function runSnapshot() {
  try {
    const res = await api.runSnapshot()
    const count = Number(res.data.token_count || 0)
    if (!res.data.token_list_available) {
      ElMessage.warning('账号快照已保存，但令牌列表采集失败；请查看采集提醒')
    } else {
      ElMessage.success(`快照已采集: ${res.data.date}（${count} 个令牌）`)
    }
    issues.value = res.data.issues || []
    if (issues.value.length) issuesDialog.value = true
    await Promise.all([load(), refreshLiveBilling()])
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || '采集失败')
  }
}
async function sendReport() {
  try {
    await api.sendReport()
    ElMessage.success('日报已发送')
    load()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || '发送失败')
  }
}
async function testFeishu() {
  try {
    await api.testFeishu()
    ElMessage.success('飞书连接正常')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || '飞书连接失败')
  }
}
</script>

<style scoped>
.dash-page {
  width: 100%;
  min-width: 0;
  container-type: inline-size;
}

.metric-filter-card {
  margin-top: 16px;
}

.currency-panel {
  margin-top: 16px;
}

.currency-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 240px));
  gap: 12px;
}

.currency-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px 16px;
  border: 1px solid var(--el-border-color-lighter, #ebeef5);
  border-radius: 8px;
  color: var(--el-text-color-regular, #606266);
}

.currency-card strong {
  color: var(--el-text-color-primary, #303133);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 22px;
}

.currency-card small {
  color: var(--el-text-color-secondary, #909399);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

.currency-formula {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  margin-top: 10px;
  color: var(--el-text-color-secondary, #909399);
  font-size: 12px;
}

.formula-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.currency-formula code {
  padding: 4px 8px;
  border-radius: 5px;
  background: var(--el-fill-color-light, #f5f7fa);
  color: var(--el-text-color-primary, #303133);
}

@container (max-width: 559px) {
  .currency-grid { grid-template-columns: minmax(0, 1fr); }
}

.metric-filter {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.metric-filter-label {
  color: var(--el-text-color-regular, #606266);
  font-size: 14px;
}

.metric-selector {
  width: min(420px, 100%);
}

.group {
  margin-top: 24px;
}

.group-title {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 12px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.token-block {
  margin-bottom: 14px;
}

.token-name {
  font-weight: 500;
  color: var(--el-color-primary, #409eff);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  margin-bottom: 8px;
  overflow-wrap: anywhere;
}

</style>
