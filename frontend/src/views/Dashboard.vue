<template>
  <div>
    <h2>仪表盘</h2>
    <el-space wrap>
      <el-button type="primary" @click="runSnapshot">立即采集快照</el-button>
      <el-button type="success" @click="sendReport">立即发送日报</el-button>
      <el-button @click="testFeishu">测试飞书连接</el-button>
    </el-space>
    <el-descriptions v-if="latest" title="最近快照" :column="3" border style="margin-top:16px">
      <el-descriptions-item label="日期">{{ latest.snapshot_date }}</el-descriptions-item>
      <el-descriptions-item label="总配额">{{ latest.account_quota }}</el-descriptions-item>
      <el-descriptions-item label="已用配额">{{ latest.account_used }}</el-descriptions-item>
      <el-descriptions-item label="请求次数">{{ latest.request_count }}</el-descriptions-item>
    </el-descriptions>
    <el-alert v-else title="暂无快照，请先点击『立即采集快照』" type="info" :closable="false" style="margin-top:16px" />
    <h3 style="margin-top:16px">最近发送记录</h3>
    <el-table :data="logs" style="width:100%">
      <el-table-column prop="send_time" label="发送时间" />
      <el-table-column prop="date" label="日期" />
      <el-table-column label="状态">
        <template #default="scope">
          <el-tag :type="scope.row.success ? 'success' : 'danger'">{{ scope.row.success ? '成功' : '失败' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="error_msg" label="错误" />
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../api'

const latest = ref<any>(null)
const logs = ref<any[]>([])

async function load() {
  const res = await api.dashboard()
  latest.value = res.data.latest_snapshot
  logs.value = res.data.recent_logs || []
}
onMounted(load)

async function runSnapshot() {
  try {
    const res = await api.runSnapshot()
    ElMessage.success(`快照已采集: ${res.data.date}`)
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
