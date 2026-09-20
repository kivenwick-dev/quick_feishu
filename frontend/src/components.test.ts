import { beforeEach, describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ElementPlus from 'element-plus'

vi.mock('./api', () => ({
  default: {
    dashboard: vi.fn().mockResolvedValue({ data: { latest_snapshot: null, recent_logs: [] } }),
    latest: vi.fn().mockResolvedValue({ data: { date: null, sections: [] } }),
    runSnapshot: vi.fn(),
    sendReport: vi.fn(),
    testFeishu: vi.fn(),
  },
}))

import Dashboard from './views/Dashboard.vue'
import api from './api'

describe('Dashboard', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.mocked(api.dashboard).mockResolvedValue({ data: { latest_snapshot: null, recent_logs: [] } } as any)
    vi.mocked(api.latest).mockResolvedValue({ data: { date: null, sections: [] } } as any)
  })

  it('mounts without error', () => {
    const wrapper = mount(Dashboard, { global: { plugins: [ElementPlus] } })
    expect(wrapper.exists()).toBe(true)
  })

  it('offers a metric selector that controls dashboard cards', async () => {
    vi.mocked(api.latest).mockResolvedValue({ data: {
      date: '2026-09-20',
      sections: [{
        name: '账号信息',
        fields: [
          { label: '已用配额', value: '100', is_diff: false },
          { label: '总配额', value: '200', is_diff: false },
        ],
      }],
    } } as any)
    const wrapper = mount(Dashboard, { global: { plugins: [ElementPlus] } })
    await flushPromises()

    const selector = wrapper.findComponent({ name: 'ElSelect' })
    expect(selector.exists()).toBe(true)
    expect(selector.props('modelValue')).toEqual(['0:已用配额', '0:总配额'])
    expect(wrapper.findAll('.mcard')).toHaveLength(2)

    selector.vm.$emit('update:modelValue', ['0:已用配额'])
    selector.vm.$emit('change', ['0:已用配额'])
    await flushPromises()
    expect(wrapper.findAll('.mcard')).toHaveLength(1)
    expect(wrapper.find('.mcard').text()).toContain('已用配额')
    wrapper.unmount()

    const remounted = mount(Dashboard, { global: { plugins: [ElementPlus] } })
    await flushPromises()
    expect(remounted.findComponent({ name: 'ElSelect' }).props('modelValue')).toEqual(['0:已用配额'])
    expect(remounted.findAll('.mcard')).toHaveLength(1)
    remounted.unmount()
  })
})
