import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'

vi.mock('./api', () => ({
  default: {
    dashboard: vi.fn().mockResolvedValue({ data: { latest_snapshot: null, recent_logs: [] } }),
    runSnapshot: vi.fn(),
    sendReport: vi.fn(),
    testFeishu: vi.fn(),
  },
}))

import Dashboard from './views/Dashboard.vue'

describe('Dashboard', () => {
  it('mounts without error', () => {
    const wrapper = mount(Dashboard, { global: { plugins: [ElementPlus] } })
    expect(wrapper.exists()).toBe(true)
  })
})
