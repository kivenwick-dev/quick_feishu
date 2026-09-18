<template>
  <div class="mcard">
    <div class="line1">
      <span class="label">{{ label }}</span>
      <span class="value">{{ value === '' || value === null || value === undefined ? '-' : value }}</span>
    </div>
    <div class="line2">
      变动：<span :class="cls">{{ delta ? delta : '—' }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  label: string
  value: string | number
  delta?: string
}>()

// 国内习惯：增(+)红色，减(-)绿色
const cls = computed(() => {
  const d = props.delta || ''
  if (d.startsWith('+')) return 'up'
  if (d.startsWith('-')) return 'down'
  return 'flat'
})
</script>

<style scoped>
.mcard {
  background: #fff;
  border: 1px solid var(--el-border-color-lighter, #ebeef5);
  border-radius: 8px;
  padding: 10px 12px;
}

.line1 {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 8px;
}

.label {
  color: var(--el-text-color-secondary, #909399);
  font-size: 12px;
  white-space: nowrap;
}

.value {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-weight: 600;
  font-size: 15px;
  word-break: break-all;
  text-align: right;
}

.line2 {
  margin-top: 6px;
  font-size: 12px;
  color: var(--el-text-color-secondary, #909399);
}

.up {
  color: #f56c6c;
  font-weight: 600;
}

.down {
  color: #67c23a;
  font-weight: 600;
}

.flat {
  color: var(--el-text-color-secondary, #909399);
}
</style>
