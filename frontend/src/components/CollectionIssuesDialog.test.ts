import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import ElementPlus, { ElDialog } from 'element-plus'
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

  it('emits update:modelValue false when the dialog closes', async () => {
    const wrapper = mount(CollectionIssuesDialog, {
      props: { modelValue: true, issues },
      global: { plugins: [ElementPlus], stubs: { teleport: true } },
    })
    wrapper.findComponent(ElDialog).vm.$emit('update:modelValue', false)
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([false])
  })

  it('does not render the quota alert for other kinds', async () => {
    const wrapper = mount(CollectionIssuesDialog, {
      props: { modelValue: true, issues: [{ scope: 'usage', token_name: 't', kind: 'network', status: 0, detail: 'timeout' }] },
      global: { plugins: [ElementPlus], stubs: { teleport: true } },
    })
    await nextTick()
    expect(wrapper.find('.quota-alert').exists()).toBe(false)
  })
})
