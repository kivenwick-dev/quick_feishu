import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import ElementPlus from 'element-plus'
import CollectionIssuesDialog from './CollectionIssuesDialog.vue'

const issues = [
  { scope: 'usage', token_name: 'gpt6robodjo', kind: 'quota_exhausted', status: 401, detail: '该令牌额度已用尽' },
]

describe('CollectionIssuesDialog', () => {
  it('renders quota exhausted token name and explanation', async () => {
    const wrapper = mount(CollectionIssuesDialog, {
      props: { modelValue: true, issues },
      global: { plugins: [ElementPlus], stubs: { teleport: true } },
    })
    await nextTick()
    await nextTick()
    expect(wrapper.text()).toContain('gpt6robodjo')
    expect(wrapper.text()).toContain('used_quota')
  })

  it('renders nothing meaningful without issues', async () => {
    const wrapper = mount(CollectionIssuesDialog, {
      props: { modelValue: true, issues: [] },
      global: { plugins: [ElementPlus], stubs: { teleport: true } },
    })
    await nextTick()
    await nextTick()
    expect(wrapper.findAll('.issue-group').length).toBe(0)
  })
})
