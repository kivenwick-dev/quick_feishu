<template>
  <div>
    <h2>日报模板</h2>
    <el-form label-width="100px" style="max-width:640px">
      <el-form-item label="标题"><el-input v-model="tmpl.title" /></el-form-item>
      <el-form-item label="日期模式">
        <el-radio-group v-model="tmpl.date_mode">
          <el-radio value="auto">自动（最近两天）</el-radio>
          <el-radio value="manual">手动</el-radio>
        </el-radio-group>
      </el-form-item>
    </el-form>

    <div v-for="(sec, i) in tmpl.sections" :key="i" style="margin-bottom:12px">
      <el-card>
        <el-form label-width="90px" style="max-width:520px">
          <el-form-item label="分区标题"><el-input v-model="sec.section" /></el-form-item>
          <el-form-item label="数据来源">
            <el-select v-model="sec.source">
              <el-option value="account" label="账号信息" />
              <el-option value="token" label="令牌列表" />
              <el-option value="usage" label="令牌使用情况" />
            </el-select>
          </el-form-item>
          <el-form-item label="每令牌一行" v-if="sec.source === 'usage'">
            <el-switch v-model="sec.per_token" />
          </el-form-item>
        </el-form>
        <div v-for="(f, j) in sec.fields" :key="j" style="display:flex; gap:8px; margin-bottom:8px; align-items:center">
          <el-select v-model="f.field" filterable placeholder="选择字段" style="width:240px">
            <el-option v-for="opt in dictFor(sec.source)" :key="opt.field_path" :value="opt.field_path" :label="opt.field_path + '（' + (opt.label || '未命名') + '）'" />
          </el-select>
          <el-switch v-model="f.diff" active-text="差值" />
          <el-button type="danger" link @click="sec.fields.splice(j,1)">删除</el-button>
        </div>
        <el-button size="small" @click="sec.fields.push({ field: '', diff: false })">+ 字段</el-button>
        <el-button size="small" type="danger" link @click="tmpl.sections.splice(i,1)">删除分区</el-button>
      </el-card>
    </div>

    <el-button @click="addSection">+ 分区</el-button>
    <el-button type="primary" @click="save">保存模板</el-button>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../api'

const tmpl = ref<any>({ title: 'AI 平台日报', date_mode: 'auto', sections: [] })
const dicts = ref<Record<string, any[]>>({ account: [], token: [], usage: [] })

function dictFor(source: string) {
  return dicts.value[source] || []
}

async function load() {
  const res = await api.getTemplate()
  if (res.data && res.data.title !== undefined) {
    tmpl.value = res.data
  }
  if (!tmpl.value.sections) tmpl.value.sections = []
  for (const s of ['account', 'token', 'usage']) {
    try {
      const d = await api.getDict(s)
      dicts.value[s] = d.data.items || []
    } catch { dicts.value[s] = [] }
  }
}
onMounted(load)

function addSection() {
  tmpl.value.sections.push({ section: '', source: 'account', per_token: false, fields: [] })
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
