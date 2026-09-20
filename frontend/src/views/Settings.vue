<template>
  <div class="set-page">
    <div class="page-head">
      <h2>设置</h2>
      <p class="sub">账号、飞书与定时任务配置</p>
    </div>

    <div class="panel">
      <el-form label-width="140px" style="max-width: 680px">
        <el-form-item label="账号ID">
          <el-input v-model="cfg.account.user_id" :placeholder="configured.user_id ? '已配置，留空保持不变' : '未配置'" :disabled="env.user_id" />
          <span v-if="env.user_id" class="hint">已由环境变量 QR_USER_ID 提供</span>
        </el-form-item>
        <el-form-item label="系统令牌">
          <el-input v-model="cfg.account.system_token" :placeholder="configured.system_token ? '已配置，留空保持不变' : '未配置'" type="password" show-password :disabled="env.system_token" />
          <span v-if="env.system_token" class="hint">已由环境变量 QR_SYSTEM_TOKEN 提供</span>
        </el-form-item>
        <el-form-item label="API 地址">
          <el-input v-model="cfg.account.api_base" :placeholder="configured.api_base ? '已配置，留空保持不变' : '未配置'" />
        </el-form-item>
        <el-form-item label="飞书 Webhook">
          <el-input v-model="cfg.feishu.webhook_url" :placeholder="configured.webhook_url ? '已配置，留空保持不变' : '未配置'" :disabled="env.webhook_url" />
          <span v-if="env.webhook_url" class="hint">已由环境变量 QR_FEISHU_WEBHOOK 提供</span>
        </el-form-item>
        <el-form-item label="快照时间">
          <el-time-picker v-model="snapshotTime" format="HH:mm" value-format="HH:mm" />
        </el-form-item>
        <el-form-item label="日报时间">
          <el-time-picker v-model="reportTime" format="HH:mm" value-format="HH:mm" />
        </el-form-item>
        <el-form-item label="端口">
          <el-input-number v-model="cfg.app.port" :min="1" :max="65535" />
          <span class="hint">端口修改需重启进程</span>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="restarting" @click="restart">保存并应用</el-button>
          <el-button @click="testFeishu">测试飞书连接</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="panel">
      <div class="panel-title">定时任务</div>
      <el-descriptions :column="2" border>
        <el-descriptions-item label="快照时间">{{ status.snapshot_time || '-' }}</el-descriptions-item>
        <el-descriptions-item label="日报时间">{{ status.report_time || '-' }}</el-descriptions-item>
        <el-descriptions-item label="下次快照">{{ fmt(status.next_runs?.[0]) }}</el-descriptions-item>
        <el-descriptions-item label="下次日报">{{ fmt(status.next_runs?.[1]) }}</el-descriptions-item>
      </el-descriptions>
      <div class="panel-actions">
        <el-button type="warning" :loading="restarting" @click="restart">应用配置并重启</el-button>
        <span class="hint">重启定时任务并重新采集当天快照，使新账号/令牌/时间立即生效</span>
      </div>
    </div>

    <CollectionIssuesDialog v-model="issuesDialog" :issues="issues" />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../api'
import CollectionIssuesDialog from '../components/CollectionIssuesDialog.vue'
import type { Issue } from '../collectionIssues'

const cfg = ref<any>({ app: { port: 8080 }, account: {}, feishu: {}, schedule: {} })
const configured = ref<Record<string, boolean>>({})
const env = ref<Record<string, boolean>>({})
const snapshotTime = ref('00:00')
const reportTime = ref('10:30')
const status = ref<any>({})
const restarting = ref(false)
const issues = ref<Issue[]>([])
const issuesDialog = ref(false)

async function load() {
  const res = await api.getSettings()
  cfg.value = res.data.config || cfg.value
  env.value = res.data.env_overridden || {}
  configured.value = res.data.configured || {}
  snapshotTime.value = cfg.value.schedule?.snapshot_time || '00:00'
  reportTime.value = cfg.value.schedule?.report_time || '10:30'
  await loadStatus()
}

async function loadStatus() {
  try {
    const res = await api.schedulerStatus()
    status.value = res.data || {}
  } catch {
    status.value = {}
  }
}
onMounted(load)

function fmt(t?: string) {
  if (!t) return '-'
  const d = new Date(t)
  if (isNaN(d.getTime())) return t
  return d.toLocaleString('zh-CN', { hour12: false })
}

// persist 将表单写入 config.yaml，供保存与「应用配置并重启」共用，
// 确保重启时 App.Restart 从磁盘重载的是当前页面上的配置。
async function persist() {
  cfg.value.schedule = { snapshot_time: snapshotTime.value, report_time: reportTime.value }
  await api.saveSettings(cfg.value)
}

async function restart() {
  restarting.value = true
  try {
    await persist()
    const res = await api.restartScheduler()
    const data = res.data || {}
    issues.value = data.snapshot_issues || []
    if (data.snapshot_error) {
      ElMessage.warning('定时任务已重启，但快照采集失败：' + data.snapshot_error)
    } else if (!issues.value.length) {
      ElMessage.success('配置已应用，账号数据已切换并更新')
    }
    if (issues.value.length) issuesDialog.value = true
    await load()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || '重启失败')
  } finally {
    restarting.value = false
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
.set-page {
  max-width: 920px;
}

.panel {
  background: #fff;
  border: 1px solid var(--el-border-color-lighter, #ebeef5);
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 20px;
}

.panel-title {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 14px;
}

.panel-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 16px;
}

.hint {
  color: var(--el-text-color-secondary, #909399);
  margin-left: 8px;
  font-size: 12px;
}
</style>
