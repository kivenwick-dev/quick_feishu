<template>
  <div class="dash-page">
    <div class="page-head">
      <h2>仪表盘</h2>
      <p class="sub">最近一次快照的全部指标（含变动），与飞书日报一致</p>
    </div>

    <el-space wrap>
      <el-button type="primary" @click="runSnapshot">立即采集快照</el-button>
      <el-button type="success" @click="sendReport">立即发送日报</el-button>
      <el-button @click="testFeishu">测试飞书连接</el-button>
    </el-space>

    <el-empty v-if="!date" description="暂无快照，请先点击『立即采集快照』" :image-size="80" style="margin-top: 24px" />

    <div v-for="(sec, i) in sections" :key="i" class="group">
      <div class="group-title">
        {{ sec.name }}
        <el-tag v-if="i === 0 && date" size="small" type="info">{{ date }}</el-tag>
      </div>

      <template v-if="sec.tokens && sec.tokens.length">
        <div v-for="(tok, ti) in sec.tokens" :key="ti" class="token-block">
          <div class="token-name">{{ tok.name }}</div>
          <div class="grid">
            <MetricCard
              v-for="(m, mi) in tok.metrics"
              :key="mi"
              :label="m.label"
              :value="m.value"
              :delta="m.has_delta ? m.delta : ''"
            />
          </div>
        </div>
      </template>

      <div v-else class="grid">
        <MetricCard
          v-for="(f, fi) in sec.fields"
          :key="fi"
          :label="f.label"
          :value="f.value"
          :delta="f.is_diff ? f.delta : ''"
        />
      </div>
    </div>

    <div class="group">
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
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../api'
import MetricCard from '../components/MetricCard.vue'

const date = ref<string | null>(null)
const sections = ref<any[]>([])
const logs = ref<any[]>([])

async function load() {
  const res = await api.dashboard()
  logs.value = res.data.recent_logs || []
  const latest = await api.latest()
  date.value = latest.data.date
  sections.value = latest.data.sections || []
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

<style scoped>
.dash-page {
  max-width: 1000px;
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
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(210px, 1fr));
  gap: 10px;
}
</style>
