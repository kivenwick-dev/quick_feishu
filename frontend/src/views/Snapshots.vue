<template>
  <div class="snap-page">
    <div class="page-head">
      <h2>历史快照</h2>
      <p class="sub">每次采集独立保存 · 变动相对上次采集（增红 / 减绿） · 北京时间</p>
    </div>

    <el-card shadow="never" class="filter-card">
      <div class="toolbar">
        <el-radio-group v-model="source" @change="onSourceChange">
          <el-radio-button value="account">账号信息</el-radio-button>
          <el-radio-button value="usage">令牌使用情况</el-radio-button>
        </el-radio-group>

        <el-select
          v-if="source === 'usage'"
          v-model="tokenId"
          placeholder="选择令牌名称"
          filterable
          style="width: 220px"
          @change="load"
        >
          <el-option v-for="tk in tokens" :key="tk.token_id" :value="tk.token_id" :label="tk.token_name || '未命名令牌'" />
        </el-select>

        <el-date-picker
          v-model="range"
          type="daterange"
          value-format="YYYY-MM-DD"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          :clearable="true"
          style="width: 260px"
          @change="onFilterChange"
        />

        <el-radio-group
          v-if="source === 'usage'"
          v-model="usageCurrency"
          class="currency-mode"
          @change="saveUsageCurrency"
        >
          <el-radio-button :value="true">金额</el-radio-button>
          <el-radio-button :value="false">配额</el-radio-button>
        </el-radio-group>

        <el-select
          v-model="visibleFields"
          class="field-selector"
          multiple
          collapse-tags
          collapse-tags-tooltip
          placeholder="显示字段"
          style="width: 300px"
          @change="rememberVisibleFields"
        >
          <el-option v-for="f in fieldOptions" :key="f.path" :value="f.path" :label="f.label || f.path" />
        </el-select>
      </div>
    </el-card>

    <el-empty v-if="filteredRows.length === 0" description="暂无历史数据" :image-size="80" style="margin-top: 24px" />

    <div v-for="row in pagedRows" :key="row.id" class="day-panel metric-panel">
      <div class="day-head">
        <span class="day-date">{{ formatSnapshotTime(row.captured_at, row.date) }}</span>
        <el-tag size="small" type="info">快照 #{{ row.id }}</el-tag>
      </div>
      <div class="metric-grid">
        <MetricCard
          v-for="f in shownFields"
          :key="f.path"
          :label="f.label || f.path"
          :value="row.values[f.path]"
          :delta="row.deltas[f.path]"
          :note="source === 'usage' ? negativeQuotaNote(f.path, row.values[f.path]) : ''"
        />
      </div>
    </div>

    <div class="pager">
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="filteredRows.length"
        layout="total, sizes, prev, pager, next, jumper"
        :page-sizes="[1, 3, 6, 12, 24]"
        background
        @size-change="onFilterChange"
      />
    </div>


  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import api from '../api'
import MetricCard from '../components/MetricCard.vue'
import { formatSnapshotTime } from '../snapshotTime'
import { negativeQuotaNote } from '../metricNotes'

const source = ref<'account' | 'usage'>('account')
const tokenId = ref<number | undefined>(undefined)
const tokens = ref<any[]>([])
const allFields = ref<any[]>([])
const visibleFields = ref<string[]>([])
const fieldSelections = new Map<string, string[]>()
const fieldSelectionsStorageKey = 'quick-feishu.snapshots.visible-fields'
const usageCurrencyStorageKey = 'quick-feishu.snapshots.usage-currency'
const usageCurrency = ref(true)
const rows = ref<any[]>([])

function restoreFieldSelections() {
  try {
    const savedCurrency = localStorage.getItem(usageCurrencyStorageKey)
    if (savedCurrency !== null) usageCurrency.value = savedCurrency === 'true'
    const saved = localStorage.getItem(fieldSelectionsStorageKey)
    if (saved === null) return
    const parsed = JSON.parse(saved)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return
    for (const [key, paths] of Object.entries(parsed)) {
      if (Array.isArray(paths) && paths.every((path) => typeof path === 'string')) {
        fieldSelections.set(key, paths as string[])
      }
    }
  } catch {
    // Ignore unavailable storage and malformed data; each source will default to all fields.
  }
}

function saveFieldSelections() {
  try {
    localStorage.setItem(fieldSelectionsStorageKey, JSON.stringify(Object.fromEntries(fieldSelections)))
  } catch {
    // Browsers may disable local storage; filtering still works for the current visit.
  }
}

function saveUsageCurrency() {
  try {
    localStorage.setItem(usageCurrencyStorageKey, String(usageCurrency.value))
  } catch {
    // Browsers may disable local storage; the switch still works for the current visit.
  }
}

restoreFieldSelections()

const range = ref<string[] | null>(null)
const page = ref(1)
const pageSize = ref(6)

function selectablePath(path: string) {
  return !path.endsWith('_usd')
}

function currencyPath(path: string) {
  switch (path) {
    case 'total_available':
      return 'total_available_usd'
    case 'total_used':
      return 'total_used_usd'
    case 'total_granted':
      return 'total_granted_usd'
    default:
      return path
  }
}

const fieldOptions = computed(() => allFields.value.filter((f: any) => selectablePath(f.path)))

const shownFields = computed(() => {
  const byPath = new Map(allFields.value.map((f: any) => [f.path, f]))
  return visibleFields.value
    .map((path: string) => byPath.get(path))
    .filter((field: any) => field !== undefined)
    .map((field: any) => {
      if (source.value !== 'usage' || !usageCurrency.value) return field
      const usdPath = currencyPath(field.path)
      if (usdPath === field.path || !byPath.has(usdPath)) return field
      return byPath.get(usdPath)
    })
})

function selectionKey() {
  return source.value === 'usage' ? `usage:${tokenId.value ?? ''}` : 'account'
}

function rememberVisibleFields() {
  fieldSelections.set(selectionKey(), [...visibleFields.value])
  saveFieldSelections()
}

const filteredRows = computed(() => {
  let r = rows.value
  if (range.value && range.value.length === 2) {
    const [from, to] = range.value
    r = r.filter((x) => x.date >= from && x.date <= to)
  }
  return r
})

const pagedRows = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return filteredRows.value.slice(start, start + pageSize.value)
})

function onFilterChange() {
  page.value = 1
}

async function loadTokens() {
  try {
    const res = await api.tokens()
    tokens.value = res.data.items || []
    if (tokens.value.length && tokenId.value === undefined) {
      tokenId.value = tokens.value[0].token_id
    }
  } catch {
    tokens.value = []
  }
}

async function load() {
  if (source.value === 'usage' && tokenId.value === undefined) {
    rows.value = []
    allFields.value = []
    return
  }
  const res = await api.history(source.value, source.value === 'usage' ? tokenId.value : undefined, 0)
  allFields.value = res.data.fields || []
  rows.value = (res.data.rows || []).slice().reverse() // 按日期及采集顺序倒序
  const available = new Set(fieldOptions.value.map((f: any) => f.path))
  const saved = fieldSelections.get(selectionKey())
  if (saved === undefined) {
    visibleFields.value = fieldOptions.value.map((f: any) => f.path)
  } else {
    const selected = saved.filter((path) => available.has(path))
    visibleFields.value = selected.length === 0 && fieldOptions.value.length > 0
      ? fieldOptions.value.map((f: any) => f.path)
      : selected
  }
  if (saved === undefined) rememberVisibleFields()
  page.value = 1
}

async function onSourceChange() {
  if (source.value === 'usage') {
    await loadTokens()
  }
  await load()
}

onMounted(async () => {
  await loadTokens()
  await load()
})

</script>

<style scoped>
.snap-page {
  width: 100%;
  min-width: 0;
  container-type: inline-size;
}

.filter-card {
  border-radius: 8px;
}

.toolbar {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
}

.currency-mode {
  flex: 0 0 auto;
}

.day-panel {
  margin-top: 24px;
  border: 1px solid #cbd5e1;
  box-shadow: 0 2px 6px #0f172a06;
}

.day-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  padding: 0 0 14px 10px;
  gap: 12px;
  flex-wrap: wrap;
  border-bottom: 1px solid #dce3ec;
  border-left: 3px solid var(--el-color-primary, #409eff);
}

.day-date {
  font-size: 15px;
  font-weight: 600;
}

.pager {
  display: flex;
  justify-content: flex-end;
  margin-top: 20px;
  overflow-x: auto;
}

.raw {
  white-space: pre-wrap;
  word-break: break-all;
  font-size: 12px;
  background: var(--el-fill-color-light, #f5f7fa);
  padding: 12px;
  border-radius: 6px;
}
</style>
