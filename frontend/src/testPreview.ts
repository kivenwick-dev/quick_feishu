// Browser-only fixture: current snapshot is the baseline, never persisted or sent.
export function createTestPreview(sections: any[]) {
  let index = 0
  function metric(source: any, deltaFlag: string) {
    const value = Number(source.value)
    if (source.value === '' || source.value == null || !Number.isFinite(value)) return { ...source }
    const mode = index++ % 3
    let delta = mode === 0 ? 1200 : mode === 1 ? -300 : 0
    // Do not manufacture negative counts from a small non-negative baseline.
    if (value >= 0 && value + delta < 0) delta = value > 0 ? -value : 300
    return {
      ...source,
      value: String(value + delta),
      delta: delta > 0 ? `+${delta}` : String(delta),
      [deltaFlag]: true,
    }
  }
  return sections.map(section => ({
    ...section,
    fields: section.fields?.map((field: any) => metric(field, 'is_diff')),
    tokens: section.tokens?.map((token: any) => ({
      ...token,
      metrics: token.metrics?.map((item: any) => metric(item, 'has_delta')),
    })),
  }))
}
