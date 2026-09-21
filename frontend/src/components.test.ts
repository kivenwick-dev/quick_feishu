import { beforeEach, describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ElementPlus from 'element-plus'

vi.mock('./api', () => ({
  default: {
    dashboard: vi.fn().mockResolvedValue({ data: { latest_snapshot: null, recent_logs: [] } }),
    latest: vi.fn().mockResolvedValue({ data: { date: null, sections: [] } }),
    quotaRate: vi.fn().mockResolvedValue({ data: { quota_per_unit: 500000, currency: 'USD' } }),
    liveBilling: vi.fn().mockResolvedValue({ data: {
      quota: 0, used_quota: 0, quota_per_unit: 500000,
      balance_usd: 0, used_usd: 0, updated_at: '2026-09-20T12:00:00+08:00',
    } }),
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
    vi.mocked(api.quotaRate).mockResolvedValue({ data: { quota_per_unit: 500000, currency: 'USD' } } as any)
    vi.mocked(api.liveBilling).mockResolvedValue({ data: {
      quota: 0, used_quota: 0, quota_per_unit: 500000,
      balance_usd: 0, used_usd: 0, updated_at: '2026-09-20T12:00:00+08:00',
    } } as any)
  })

  it('converts the latest balance and historical usage with the live quota rate', async () => {
    vi.mocked(api.liveBilling).mockResolvedValue({ data: {
      quota: 7526436992,
      used_quota: 24299227753,
      quota_per_unit: 500000,
      balance_usd: 15052.873984,
      used_usd: 48598.455506,
      updated_at: '2026-09-20T12:00:00+08:00',
    } } as any)
    const wrapper = mount(Dashboard, { global: { plugins: [ElementPlus] } })
    await flushPromises()

    expect(wrapper.find('.currency-panel').text()).toContain('$15,052.87')
    expect(wrapper.find('.currency-panel').text()).toContain('$48,598.46')
    expect(wrapper.find('.currency-panel').text()).toContain('500,000 quota')
    expect(wrapper.find('.currency-panel').text()).toContain('= $1.00 USD')
    expect(wrapper.find('.currency-panel').text()).toContain('当前额度 ÷ 单位额度 = 当前余额（美元）')
    expect(wrapper.find('.currency-panel').text()).toContain('累计已用额度 ÷ 单位额度 = 历史消耗（美元）')
    wrapper.unmount()
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

  it.each([
    ['an empty selection', []],
    ['a stale selection', ['0:旧指标']],
  ])('restores all dashboard cards from %s in local storage', async (_case, savedSelection) => {
    localStorage.setItem('quick-feishu.dashboard.visible-metrics', JSON.stringify(savedSelection))
    vi.mocked(api.latest).mockResolvedValue({ data: {
      date: '2026-09-21',
      sections: [{
        name: '账号概况',
        fields: [
          { label: '已用配额', value: '100', is_diff: false },
          { label: '请求次数', value: '20', is_diff: false },
        ],
      }],
    } } as any)

    const wrapper = mount(Dashboard, { global: { plugins: [ElementPlus] } })
    await flushPromises()

    expect(wrapper.findComponent({ name: 'ElSelect' }).props('modelValue')).toEqual([
      '0:已用配额',
      '0:请求次数',
    ])
    expect(wrapper.findAll('.mcard')).toHaveLength(2)
    expect(JSON.parse(localStorage.getItem('quick-feishu.dashboard.visible-metrics') || '[]')).toEqual([
      '0:已用配额',
      '0:请求次数',
    ])
    wrapper.unmount()
  })
})
