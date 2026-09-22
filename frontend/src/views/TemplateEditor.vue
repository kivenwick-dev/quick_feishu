<template>
  <div class="tpl-page">
    <div class="page-head">
      <h2>日报模板</h2>
      <p class="sub">配置每日推送的分区与字段，字段可开启「差值」展示当日增减</p>
    </div>

    <div class="tpl-base">
      <el-form :model="tmpl" label-width="80px" class="base-form">
        <el-form-item label="标题">
          <el-input v-model="tmpl.title" placeholder="卡片标题" style="max-width: 320px" />
        </el-form-item>
        <el-form-item label="日期模式">
          <el-radio-group v-model="tmpl.date_mode">
            <el-radio value="auto">自动（最近两天）</el-radio>
            <el-radio value="manual">手动</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="算法说明">
          <el-input
            v-model="tmpl.algorithm_note"
            type="textarea"
            :rows="3"
            placeholder="留空则日报不展示算法说明"
            style="max-width: 720px"
          />
          <div class="form-hint">展示在日报数据底部，可自定义；清空并保存则不显示。</div>
        </el-form-item>
      </el-form>
    </div>

    <div class="tree">
      <div v-for="(sec, i) in tmpl.sections" :key="i" class="tree-node">
        <div class="section-node">
          <span class="index-badge">{{ Number(i) + 1 }}</span>
          <el-input
            v-model="sec.section"
            placeholder="分区标题，如：账号概况"
            class="sec-title"
          />
          <el-select v-model="sec.source" class="sec-source" placeholder="数据来源" @change="onSourceChange(sec)">
            <el-option value="account" label="账号信息" />
            <el-option value="usage" label="令牌使用情况" />
          </el-select>
          <el-switch
            v-if="sec.source === 'usage'"
            v-model="sec.per_token"
            active-text="每令牌一行"
            class="sec-per-token"
          />
          <el-button class="sec-del" link type="danger" @click="removeSection(Number(i))">删除分区</el-button>
        </div>

        <div class="children">
          <div v-for="(f, j) in sec.fields" :key="j" class="field-node">
            <el-select
              v-model="f.field"
              filterable
              placeholder="选择字段"
              class="field-select"
              @change="normalizeField(sec.source, f)"
            >
              <el-option
                v-for="opt in dictFor(sec.source)"
                :key="opt.field_path"
                :value="opt.field_path"
                :label="opt.field_path + '（' + (opt.label || '未命名') + '）'"
              />
            </el-select>
            <el-switch v-model="f.diff" active-text="差值" />
            <el-switch
              v-if="currencyEligible(sec.source, f.field)"
              v-model="f.currency"
              active-text="金额"
            />
            <el-button class="field-del" link type="danger" @click="sec.fields.splice(Number(j), 1)">删除</el-button>
          </div>
          <div class="add-field-row">
            <el-button text type="primary" @click="sec.fields.push({ field: '', diff: false })">
              + 添加字段
            </el-button>
          </div>
        </div>
      </div>

      <el-empty
        v-if="!tmpl.sections || tmpl.sections.length === 0"
        description="还没有分区，点击下方『添加分区』开始"
        :image-size="80"
      />
    </div>

    <div class="action-bar">
      <el-button @click="addSection">+ 添加分区</el-button>
      <el-button type="primary" @click="save">保存模板</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../api'

const defaultAlgorithmNote = '主值：日报发送时实时采集值；差值：两个 00:00 快照对比。\n余额/可用类 = 前天快照 - 昨天快照，显示为「消耗」；累计/已用/授予类 = 昨天快照 - 前天快照，显示为「变动」。'
const tmpl = ref<any>({ title: 'AI 平台日报', date_mode: 'auto', algorithm_note: defaultAlgorithmNote, sections: [] })
const dicts = ref<Record<string, any[]>>({ account: [], token: [], usage: [] })

function dictFor(source: string) {
  return dicts.value[source] || []
}

function currencyEligible(source: string, field: string) {
  if (source === 'account') return field === 'balance_usd' || field === 'used_usd'
  if (source === 'usage') return ['total_available', 'total_used', 'total_granted'].includes(field)
  return false
}

function normalizeField(source: string, field: any) {
  if (currencyEligible(source, field.field)) {
    if (field.currency === undefined) field.currency = true
    return
  }
  delete field.currency
}

function normalizeTemplate() {
  for (const sec of tmpl.value.sections || []) {
    for (const f of sec.fields || []) normalizeField(sec.source, f)
  }
}

function onSourceChange(sec: any) {
  for (const f of sec.fields || []) normalizeField(sec.source, f)
}

async function load() {
  const res = await api.getTemplate()
  if (res.data && res.data.title !== undefined) {
    tmpl.value = res.data
  }
  if (tmpl.value.algorithm_note === undefined || tmpl.value.algorithm_note === null) {
    tmpl.value.algorithm_note = defaultAlgorithmNote
  }
  if (!tmpl.value.sections) tmpl.value.sections = []
  normalizeTemplate()
  for (const s of ['account', 'token', 'usage']) {
    try {
      const d = await api.getDict(s)
      dicts.value[s] = d.data.items || []
    } catch {
      dicts.value[s] = []
    }
  }
}
onMounted(load)

function addSection() {
  if (!tmpl.value.sections) tmpl.value.sections = []
  tmpl.value.sections.push({ section: '', source: 'account', per_token: false, fields: [] })
}

function removeSection(i: number) {
  tmpl.value.sections.splice(i, 1)
}

async function save() {
  try {
    await api.saveTemplate(tmpl.value)
    ElMessage.success('模板已保存')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || '保存失败')
  }
}
</script>

<style scoped>
.tpl-page {
  max-width: 920px;
}

.tpl-base {
  background: #fff;
  border: 1px solid var(--el-border-color-lighter, #ebeef5);
  border-radius: 8px;
  padding: 16px 16px 0;
  margin-bottom: 20px;
}

.base-form :deep(.el-form-item) {
  margin-bottom: 16px;
}

.form-hint {
  width: 100%;
  margin-top: 6px;
  color: var(--el-text-color-secondary, #909399);
  font-size: 12px;
  line-height: 1.4;
}

/* 树 */
.tree {
  min-height: 80px;
}

.tree-node {
  margin-bottom: 14px;
}

/* 分区节点（父） */
.section-node {
  display: flex;
  align-items: center;
  gap: 10px;
  background: var(--el-fill-color-light, #f5f7fa);
  border: 1px solid var(--el-border-color-lighter, #ebeef5);
  border-left: 3px solid var(--el-color-primary, #409eff);
  border-radius: 8px;
  padding: 10px 12px;
}

.index-badge {
  flex: none;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: var(--el-color-primary, #409eff);
  color: #fff;
  font-size: 12px;
  line-height: 22px;
  text-align: center;
}

.sec-title {
  flex: 1 1 220px;
  max-width: 280px;
}

.sec-source {
  flex: none;
  width: 160px;
}

.sec-per-token {
  flex: none;
}

.sec-del {
  margin-left: auto;
}

/* 子节点容器 + 连接线 */
.children {
  position: relative;
  padding: 6px 0 2px 30px;
}

.children::before {
  content: '';
  position: absolute;
  left: 11px;
  top: 16px;
  bottom: 22px;
  border-left: 1px dashed var(--el-border-color, #dcdfe6);
}

/* 字段节点（子） */
.field-node,
.add-field-row {
  position: relative;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 6px 8px;
  border-radius: 6px;
}

.field-node::before,
.add-field-row::before {
  content: '';
  position: absolute;
  left: -19px;
  top: 50%;
  width: 17px;
  border-top: 1px dashed var(--el-border-color, #dcdfe6);
}

.field-node:hover {
  background: var(--el-fill-color-lighter, #fafafa);
}

.field-select {
  width: 280px;
}

.field-del {
  margin-left: auto;
}

.add-field-row {
  padding-left: 8px;
}

/* 底部操作栏 */
.action-bar {
  display: flex;
  gap: 12px;
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid var(--el-border-color-lighter, #ebeef5);
}
</style>
