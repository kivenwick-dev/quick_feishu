import { describe, expect, it } from 'vitest'
import { appendSelectedValues } from './selectionOrder'

describe('appendSelectedValues', () => {
  it('keeps earlier selections in place and appends each new selection', () => {
    let selected = ['当前余额', '历史消耗']
    selected = appendSelectedValues(selected, ['当前余额', '总配额', '历史消耗'])
    selected = appendSelectedValues(selected, ['当前余额', '总配额', '已用配额', '历史消耗'])

    expect(selected).toEqual(['当前余额', '历史消耗', '总配额', '已用配额'])
  })

  it('removes unchecked values without changing the remaining order', () => {
    expect(appendSelectedValues(
      ['当前余额', '历史消耗', '总配额'],
      ['当前余额', '总配额'],
    )).toEqual(['当前余额', '总配额'])
  })
})
