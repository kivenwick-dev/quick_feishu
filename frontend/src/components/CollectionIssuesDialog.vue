<template>
  <el-dialog
    :model-value="modelValue"
    title="采集提醒"
    width="640px"
    @update:model-value="(v: boolean) => emit('update:modelValue', v)"
  >
    <el-alert
      v-if="hasQuotaExhausted"
      type="warning"
      :closable="false"
      show-icon
      title="令牌额度已用尽"
      class="quota-alert"
    >
      <p v-for="(line, i) in QUOTA_EXHAUSTED_HINT" :key="i" class="hint-line">{{ line }}</p>
    </el-alert>

    <div v-for="g in groups" :key="g.kind" class="issue-group">
      <div class="group-title">{{ g.title }}</div>
      <el-descriptions :column="1" border size="small">
        <el-descriptions-item v-for="(it, i) in g.issues" :key="i" :label="scopeLabel(it.scope)">
          <span v-if="it.token_name" class="token-name">{{ it.token_name }} · </span>
          <span class="detail">{{ it.detail || `HTTP ${it.status}` }}</span>
        </el-descriptions-item>
      </el-descriptions>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { groupIssues, QUOTA_EXHAUSTED_HINT, type Issue } from '../collectionIssues'

const props = defineProps<{ modelValue: boolean; issues: Issue[] }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const groups = computed(() => groupIssues(props.issues || []))
const hasQuotaExhausted = computed(() => groups.value.some((g) => g.kind === 'quota_exhausted'))

const SCOPE_LABELS: Record<string, string> = {
  account: '账号信息',
  tokenlist: '令牌列表',
  usage: '令牌用量',
}
function scopeLabel(scope: string) {
  return SCOPE_LABELS[scope] || scope
}
</script>

<style scoped>
.quota-alert { margin-bottom: 16px; }
.hint-line { margin: 4px 0; font-size: 13px; line-height: 1.6; }
.issue-group { margin-top: 12px; }
.group-title { font-weight: 600; margin-bottom: 8px; }
.token-name { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
.detail { color: var(--el-text-color-regular, #606266); }
</style>
