import { describe, it, expect } from 'vitest'
import { groupIssues, QUOTA_EXHAUSTED_HINT, type Issue } from './collectionIssues'

const base: Issue = { scope: 'usage', token_name: '', kind: 'other', status: 0, detail: '' }

describe('groupIssues', () => {
  it('groups by kind and lists token names in order', () => {
    const groups = groupIssues([
      { ...base, kind: 'quota_exhausted', token_name: 'a', status: 401 },
      { ...base, kind: 'quota_exhausted', token_name: 'b', status: 401 },
      { ...base, kind: 'network' },
    ])
    expect(groups.map((g) => g.kind)).toEqual(['quota_exhausted', 'network'])
    expect(groups[0].tokens).toEqual(['a', 'b'])
    expect(groups[0].title).toBe('令牌额度已用尽')
  })

  it('handles empty input and unknown kinds', () => {
    expect(groupIssues([])).toEqual([])
    const groups = groupIssues([{ ...base, kind: 'weird' }])
    expect(groups[0].kind).toBe('weird')
  })

  it('exposes the quota exhausted explanation', () => {
    expect(QUOTA_EXHAUSTED_HINT.join('')).toContain('used_quota')
    expect(QUOTA_EXHAUSTED_HINT.join('')).toContain('额度已用尽')
  })
})
