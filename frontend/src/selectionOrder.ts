// Element Plus returns multiple-select values in option order. Preserve the
// existing order and append only newly selected values in click order.
export function appendSelectedValues(previous: string[], incoming: string[]): string[] {
  const selected = new Set(incoming)
  const retained = previous.filter((value) => selected.has(value))
  const known = new Set(retained)
  for (const value of incoming) {
    if (!known.has(value)) {
      retained.push(value)
      known.add(value)
    }
  }
  return retained
}
