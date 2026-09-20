const quotaFields = new Set([
  'total_available',
  'total_granted',
  'remain_quota',
  '可用总量',
  '授予总量',
  '剩余额度',
])

export const negativeQuotaNoteText =
  '平台返回的原始额度值。不限额度令牌的额度账本可能出现负数，不代表欠费、不可用或本次负变动。'

export function negativeQuotaNote(field: string, value: unknown) {
  if (!quotaFields.has(field)) return ''
  const numeric = Number(String(value ?? '').replaceAll(',', '').trim())
  return Number.isFinite(numeric) && numeric < 0 ? negativeQuotaNoteText : ''
}
