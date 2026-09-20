import { describe, expect, it } from 'vitest'
import { negativeQuotaNote, negativeQuotaNoteText } from './metricNotes'

describe('negative quota notes', () => {
  it('annotates negative initial quota values by path or label', () => {
    expect(negativeQuotaNote('total_available', '-12,258,029,230')).toBe(negativeQuotaNoteText)
    expect(negativeQuotaNote('剩余额度', -100)).toBe(negativeQuotaNoteText)
  })

  it('does not annotate normal values, usage counts, or deltas', () => {
    expect(negativeQuotaNote('total_available', 100)).toBe('')
    expect(negativeQuotaNote('total_used', -100)).toBe('')
    expect(negativeQuotaNote('变动', -100)).toBe('')
  })
})
