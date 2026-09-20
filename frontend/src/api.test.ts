import { describe, it, expect } from 'vitest'
import api, { api as instance } from './api'

describe('api module', () => {
  it('exposes all expected methods', () => {
    const methods = [
      'dashboard', 'runSnapshot', 'sendReport', 'snapshots', 'snapshot',
      'compare', 'getTemplate', 'saveTemplate', 'getDict', 'saveDict',
      'getSettings', 'saveSettings', 'testFeishu', 'sendLogs',
      'quotaRate',
      'liveBilling',
    ]
    for (const m of methods) {
      expect(typeof (api as any)[m]).toBe('function')
    }
  })

  it('uses /api baseURL', () => {
    expect((instance as any).defaults.baseURL).toBe('/api')
  })
})
