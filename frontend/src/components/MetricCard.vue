<template>
  <div class="mcard">
    <div class="metric-line">
      <el-tooltip :content="label" placement="top" :show-after="200" popper-class="metric-tooltip">
        <span class="metric-label" tabindex="0">{{ label }}：</span>
      </el-tooltip>
      <el-tooltip :content="displayValue" placement="top" :show-after="200" popper-class="metric-tooltip">
        <span class="metric-value" tabindex="0">{{ displayValue }}</span>
      </el-tooltip>
      <el-tooltip v-if="note" :content="note" placement="top" :show-after="100" popper-class="metric-tooltip">
        <span class="metric-note" tabindex="0" aria-label="初始值说明">ⓘ</span>
      </el-tooltip>
    </div>
    <div class="metric-line change-line">
      <span class="change-label">{{ displayDeltaLabel }}：</span>
      <el-tooltip :content="displayDelta" placement="top" :show-after="200" popper-class="metric-tooltip">
        <span class="metric-value" :class="direction" tabindex="0">{{ displayDelta }}</span>
      </el-tooltip>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  label: string
  value?: string | number | null
  delta?: string | number | null
  deltaLabel?: string | null
  note?: string
}>()

const displayValue = computed(() =>
  props.value === '' || props.value == null ? '—' : String(props.value),
)
const normalizedDelta = computed(() => String(props.delta ?? '').trim())
const direction = computed(() => {
  const value = normalizedDelta.value
  if (/^[\-−—]\s*\d/.test(value)) return 'down'
  if (/^\+/.test(value) || /^\d/.test(value) && Number.parseFloat(value.replaceAll(',', '')) > 0) return 'up'
  return 'flat'
})
const displayDelta = computed(() => {
  const value = normalizedDelta.value
  if (!value) return '暂无对比'
  if (direction.value === 'down') return value.replace(/^[\-−—]\s*/, '—')
  if (direction.value === 'up' && !value.startsWith('+')) return `+${value}`
  return value
})
const displayDeltaLabel = computed(() => String(props.deltaLabel || '变动').trim() || '变动')
</script>

<style scoped>
.mcard {
  min-width: 0;
  height: 84px;
  padding: 15px 12px;
  background: var(--el-bg-color, #fff);
  border: 1px solid var(--el-border-color-lighter, #ebeef5);
  border-radius: 8px;
  overflow: hidden;
}
.metric-line {
  display: flex;
  align-items: center;
  gap: 3px;
  min-width: 0;
  height: 22px;
  font-size: 12px;
}
.metric-label {
  flex: 0 1 auto;
  max-width: 52%;
  color: var(--el-text-color-regular, #606266);
}
.metric-label,
.metric-value {
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.metric-note {
  flex: none;
  color: var(--el-color-warning, #e6a23c);
  cursor: help;
  font-size: 14px;
  line-height: 1;
}
.metric-value {
  flex: 1 1 0;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-variant-numeric: tabular-nums;
  font-weight: 600;
  font-size: 13px;
}
.change-line { margin-top: 8px; }
.change-label { flex: none; }
.change-line, .flat { color: var(--el-text-color-secondary, #909399); }
.up { color: #d9363e; }
.down { color: #168653; }
</style>
