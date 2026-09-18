<template>
  <div class="dash-page">
    <div class="page-head">
      <h2>仪表盘</h2>
      <p class="sub">最近一次快照的用量与配额变动 · 已隐藏账号隐私信息</p>
    </div>

    <el-space wrap>
      <el-button type="primary" :disabled="isTest" @click="runSnapshot">立即采集快照</el-button>
      <el-button type="success" :disabled="isTest" @click="sendReport">立即发送日报</el-button>
      <el-button @click="testFeishu">测试飞书连接</el-button>
      <el-button v-if="!isTest" :disabled="!date" @click="generateTest">生成测试变动</el-button>
      <el-button v-else type="danger" plain @click="removeTest">删除测试数据</el-button>
    </el-space>

    <el-alert v-if="isTest" type="warning" show-icon :closable="false" class="test-banner"
      title="TEST · 模拟数据"
      :description="`以 ${date} 的真实数据为基准模拟下一次快照；变动相对此基准计算，仅供查看涨跌效果。删除测试数据或刷新页面即可恢复，不写入历史记录或发送日报。`" />

    <el-empty v-if="!date" description="暂无快照，请先点击『立即采集快照』" :image-size="80" style="margin-top: 24px" />

    <div v-for="(sec, i) in sections" :key="i" class="group metric-panel">
      <div class="group-title">
        {{ sec.name }}
        <el-tag v-if="date" size="small" type="info">{{ date }}</el-tag>
        <el-tag v-if="isTest" size="small" type="warning">TEST 模拟</el-tag>
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
import { createTestPreview } from '../testPreview'

const date = ref<string | null>(null)
const realSections = ref<any[]>([])
const testSections = ref<any[] | null>(null)
const isTest = computed(() => testSections.value !== null)
const sections = computed(() => testSections.value ?? realSections.value)
function generateTest() { testSections.value = createTestPreview(realSections.value) }
function removeTest() {
  testSections.value = null
  ElMessage.success('测试数据已删除，已恢复真实快照')
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

.test-banner { margin-top: 16px; }

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
