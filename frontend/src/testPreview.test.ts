import { describe, it, expect } from 'vitest'
import { createTestPreview } from './testPreview'

describe('temporary test preview', () => {
  it('creates increases, decreases and zero changes without modifying real data', () => {
    const real = [{ fields: [
      { label: '已用配额', value: '10000', delta: '+10', is_diff: true },
      { label: '剩余配额', value: '5000' },
      { label: '请求次数', value: '10' },
    ], tokens: [{ name: '测试令牌', metrics: [{ value: '100', has_delta: false }] }] }]
    const before = JSON.stringify(real)
    const result = createTestPreview(real)
    expect(result[0].fields.map((f: any) => [f.value, f.delta])).toEqual([
      ['11200', '+1200'], ['4700', '-300'], ['10', '0'],
    ])
    expect(result[0].tokens[0].metrics[0].has_delta).toBe(true)
    expect(JSON.stringify(real)).toBe(before)
  })
  it('does not generate negative counts or fabricate missing metrics', () => {
    const result = createTestPreview([{ fields: [{ value:'0' }, { value:'5' }, { value:'—' }] }])
    expect(result[0].fields[1].value).toBe('0')
    expect(result[0].fields[1].delta).toBe('-5')
    expect(result[0].fields[2]).toEqual({ value:'—' })
  })
})
