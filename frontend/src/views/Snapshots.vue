<template>
  <div>
    <h2>历史快照</h2>
    <el-table :data="snapshots" style="width:100%" @row-click="loadDetail">
      <el-table-column prop="snapshot_date" label="日期" />
      <el-table-column prop="account_quota" label="总配额" />
      <el-table-column prop="account_used" label="已用配额" />
      <el-table-column prop="request_count" label="请求次数" />
      <el-table-column prop="created_at" label="创建时间" />
    </el-table>
    <el-pagination layout="prev, pager, next" :total="total" :page-size="size" @current-change="loadList" style="margin-top:12px" />

    <template v-if="detail">
      <h3 style="margin-top:16px">快照详情 {{ detail.snapshot.snapshot_date }}</h3>
      <el-tabs>
        <el-tab-pane label="账号信息"><pre>{{ pretty(detail.snapshot.account_raw) }}</pre></el-tab-pane>
        <el-tab-pane label="令牌列表"><pre>{{ pretty(detail.snapshot.token_list_raw) }}</pre></el-tab-pane>
        <el-tab-pane label="令牌使用情况"><pre>{{ pretty(detail.snapshot.token_usage_raw) }}</pre></el-tab-pane>
      </el-tabs>
    </template>

    <h3 style="margin-top:16px">日期对比</h3>
    <el-date-picker v-model="range" type="daterange" value-format="YYYY-MM-DD" />
    <el-button @click="doCompare" :disabled="!range || range.length !== 2">对比</el-button>
    <el-descriptions v-if="compare" :column="2" border style="margin-top:12px">
      <el-descriptions-item label="起始">{{ compare.from.date }} 已用 {{ compare.from.used }}</el-descriptions-item>
      <el-descriptions-item label="结束">{{ compare.to.date }} 已用 {{ compare.to.used }}</el-descriptions-item>
      <el-descriptions-item label="已用差值">{{ compare.diff.used }}</el-descriptions-item>
      <el-descriptions-item label="请求差值">{{ compare.diff.requests }}</el-descriptions-item>
    </el-descriptions>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../api'

const snapshots = ref<any[]>([])
const total = ref(0)
const size = 20
const detail = ref<any>(null)
const range = ref<string[] | null>(null)
const compare = ref<any>(null)

async function loadList(page = 1) {
  const res = await api.snapshots(page, size)
  snapshots.value = res.data.items || []
  total.value = res.data.total || 0
}
onMounted(() => loadList(1))

async function loadDetail(row: any) {
  const res = await api.snapshot(row.id)
  detail.value = res.data
}
function pretty(raw: any) {
  if (raw === null || raw === undefined) return ''
  if (typeof raw === 'object') return JSON.stringify(raw, null, 2)
  try { return JSON.stringify(JSON.parse(raw), null, 2) } catch { return String(raw) }
}
async function doCompare() {
  if (!range.value || range.value.length !== 2) return
  try {
    const res = await api.compare(range.value[0], range.value[1])
    compare.value = res.data
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || '对比失败')
  }
}
</script>
