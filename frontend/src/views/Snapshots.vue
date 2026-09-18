<template>
  <div class="snap-page">
    <div class="page-head">
      <h2>历史快照</h2>
      <p class="sub">每日全字段指标卡片 · 变动为相对前一日（增红 / 减绿）</p>
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
          placeholder="选择令牌"
          style="width: 220px"
          @change="load"
        >
          <el-option v-for="tk in tokens" :key="tk.token_id" :value="tk.token_id" :label="tk.name" />
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

        <el-select
          v-model="visibleFields"
          multiple
          collapse-tags
          collapse-tags-tooltip
          placeholder="显示字段"
          style="width: 300px"
        >
          <el-option v-for="f in allFields" :key="f.path" :value="f.path" :label="f.label || f.path" />
        </el-select>
      </div>
    </el-card>

    <el-empty v-if="filteredRows.length === 0" description="暂无历史数据" :image-size="80" style="margin-top: 24px" />

    <div v-for="row in pagedRows" :key="row.id" class="day-panel">
      <div class="day-head">
        <span class="day-date">{{ row.date }}</span>
        <el-button link type="primary" size="small" @click="openDetail(row)">查看原始数据</el-button>
      </div>
      <div class="grid">
        <MetricCard
          v-for="f in shownFields"
          :key="f.path"
          :label="f.label || f.path"
          :value="row.values[f.path]"
          :delta="row.deltas[f.path]"
        />
      </div>
    </div>

    <div class="pager">
      <el-pagination
        v-model:current-page="page"
        :page-size="pageSize"
        :total="filteredRows.length"
        layout="total, sizes, prev, pager, next"
        :page-sizes="[3, 6, 12, 24]"
        background
        @size-change="onFilterChange"
      />
    </div>

    <el-drawer v-model="drawer" :title="drawerTitle" size="60%">
      <el-tabs v-if="detail">
        <el-tab-pane label="账号信息"><pre class="raw">{{ pretty(detail.snapshot.account_raw) }}</pre></el-tab-pane>
        <el-tab-pane label="令牌列表"><pre class="raw">{{ pretty(detail.snapshot.token_list_raw) }}</pre></el-tab-pane>
        <el-tab-pane label="令牌使用情况"><pre class="raw">{{ pretty(detail.snapshot.token_usage_raw) }}</pre></el-tab-pane>
      </el-tabs>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import api from '../api'
import MetricCard from '../components/MetricCard.vue'

const source = ref<'account' | 'usage'>('account')
const tokenId = ref<number | undefined>(undefined)
const tokens = ref<any[]>([])
const allFields = ref<any[]>([])
const visibleFields = ref<string[]>([])
const rows = ref<any[]>([])

const range = ref<string[] | null>(null)
const page = ref(1)
const pageSize = ref(6)

const drawer = ref(false)
const detail = ref<any>(null)
const drawerTitle = ref('')

const shownFields = computed(() => {
  if (!visibleFields.value.length) return allFields.value
  const set = new Set(visibleFields.value)
  return allFields.value.filter((f) => set.has(f.path))
})

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
  const res = await api.history(source.value, source.value === 'usage' ? tokenId.value : undefined, 180)
  allFields.value = res.data.fields || []
  rows.value = (res.data.rows || []).slice().reverse() // 日期降序
  visibleFields.value = allFields.value.map((f: any) => f.path)
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

function pretty(raw: any) {
  if (raw === null || raw === undefined) return ''
  if (typeof raw === 'object') return JSON.stringify(raw, null, 2)
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return String(raw)
  }
}

async function openDetail(row: any) {
  if (!row?.id) return
  const res = await api.snapshot(row.id)
  detail.value = res.data
  drawerTitle.value = `快照 ${row.date}`
  drawer.value = true
}
</script>

<style scoped>
.snap-page {
  max-width: 1100px;
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

.day-panel {
  margin-top: 20px;
}

.day-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  padding-left: 10px;
  border-left: 3px solid var(--el-color-primary, #409eff);
}

.day-date {
  font-size: 15px;
  font-weight: 600;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(210px, 1fr));
  gap: 10px;
}

.pager {
  display: flex;
  justify-content: flex-end;
  margin-top: 20px;
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
