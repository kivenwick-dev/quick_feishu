<template>
  <div>
    <h2>设置</h2>
    <el-form label-width="140px" style="max-width:680px">
      <el-form-item label="账号ID">
        <el-input v-model="cfg.account.user_id" :disabled="env.user_id" />
        <span v-if="env.user_id" class="hint">已由环境变量 QR_USER_ID 提供</span>
      </el-form-item>
      <el-form-item label="系统令牌">
        <el-input v-model="cfg.account.system_token" type="password" show-password :disabled="env.system_token" />
        <span v-if="env.system_token" class="hint">已由环境变量 QR_SYSTEM_TOKEN 提供</span>
      </el-form-item>
      <el-form-item label="API 地址">
        <el-input v-model="cfg.account.api_base" />
      </el-form-item>
      <el-form-item label="飞书 Webhook">
        <el-input v-model="cfg.feishu.webhook_url" :disabled="env.webhook_url" />
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
      </el-form-item>
      <el-form-item>
        <el-button type="primary" @click="save">保存</el-button>
        <el-button @click="testFeishu">测试飞书连接</el-button>
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../api'

const cfg = ref<any>({ app: { port: 8080 }, account: {}, feishu: {}, schedule: {} })
const env = ref<Record<string, boolean>>({})
const snapshotTime = ref('00:00')
const reportTime = ref('10:30')

async function load() {
  const res = await api.getSettings()
  cfg.value = res.data.config || cfg.value
  env.value = res.data.env_overridden || {}
  snapshotTime.value = cfg.value.schedule?.snapshot_time || '00:00'
  reportTime.value = cfg.value.schedule?.report_time || '10:30'
}
onMounted(load)

async function save() {
  cfg.value.schedule = { snapshot_time: snapshotTime.value, report_time: reportTime.value }
  try {
    await api.saveSettings(cfg.value)
    ElMessage.success('设置已保存（定时任务重启后生效）')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || '保存失败')
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
.hint { color: #999; margin-left: 8px; font-size: 12px; }
</style>
