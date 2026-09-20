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
        <el-select
          v-model="visibleMetricKeys"
          class="metric-selector"
          multiple
          collapse-tags
          collapse-tags-tooltip
          placeholder="选择要显示的指标"
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
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../api'
import CollectionIssuesDialog from '../components/CollectionIssuesDialog.vue'
import type { Issue } from '../collectionIssues'
import MetricCard from '../components/MetricCard.vue'
import { negativeQuotaNote } from '../metricNotes'

const date = ref<string | null>(null)
const realSections = ref<any[]>([])
const sections = computed(() => realSections.value)
const visibleMetricKeys = ref<string[]>([])
const metricSelectionInitialized = ref(false)
const metricSelectionStorageKey = 'quick-feishu.dashboard.visible-metrics'

function restoreMetricSelection() {
  try {
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
  const visible = new Set(visibleMetricKeys.value)
  return sections.value.map((sec: any, sectionIndex: number) => ({
    ...sec,
    key: `${sectionIndex}:${sec.name}`,
    fields: (sec.fields || []).filter((field: any) => visible.has(`${sectionIndex}:${field.label}`)),
    tokens: (sec.tokens || []).map((token: any) => ({
      ...token,
      metrics: (token.metrics || []).filter((metric: any) => visible.has(`${sectionIndex}:${metric.label}`)),
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
  visibleMetricKeys.value = visibleMetricKeys.value.filter((key) => availableSet.has(key))
  saveMetricSelection()
}
const logs = ref<any[]>([])
const issues = ref<Issue[]>([])
const issuesDialog = ref(false)

async function load() {
  const res = await api.dashboard()
  logs.value = res.data.recent_logs || []
  const latest = await api.latest()
  date.value = latest.data.date
  realSections.value = latest.data.sections || []
  syncMetricSelection()
}
onMounted(load)

async function runSnapshot() {
  try {
    const res = await api.runSnapshot()
    ElMessage.success(`快照已采集: ${res.data.date}`)
    issues.value = res.data.issues || []
    if (issues.value.length) issuesDialog.value = true
    load()
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
