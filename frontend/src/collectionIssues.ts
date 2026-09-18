export interface Issue {
  scope: string
  token_name: string
  kind: string
  status: number
  detail: string
}

export interface IssueGroup {
  kind: string
  title: string
  tokens: string[]
  issues: Issue[]
}

export const KIND_TITLES: Record<string, string> = {
  quota_exhausted: '令牌额度已用尽',
  unauthorized: '令牌鉴权失败',
  server_error: '平台服务端错误',
  network: '网络错误',
  other: '其他错误',
}

export const QUOTA_EXHAUSTED_HINT: string[] = [
  '以下令牌是有限额度令牌，累计用量 used_quota 已达到并略微超过 total。',
  '在 new-api/quickrouter 这类平台上，一旦令牌用量触顶，平台会拒绝该令牌的所有请求——包括只读的 /api/usage/token/ 自检接口，返回 401「该令牌额度已用尽」。',
  '这不是采集逻辑的 bug，而是该令牌在平台侧已被停用；在额度恢复前，这些令牌的 usage 数据将一直无法采集。',
  '处理方式：充值/提高 total、改为不限额度、停用或删除该令牌。',
]

const KIND_ORDER = ['quota_exhausted', 'unauthorized', 'server_error', 'network', 'other']

export function groupIssues(issues: Issue[]): IssueGroup[] {
  const map = new Map<string, IssueGroup>()
  for (const it of issues) {
    const kind = it.kind || 'other'
    let group = map.get(kind)
    if (!group) {
      group = { kind, title: KIND_TITLES[kind] || kind, tokens: [], issues: [] }
      map.set(kind, group)
    }
    group.issues.push(it)
    if (it.token_name && !group.tokens.includes(it.token_name)) {
      group.tokens.push(it.token_name)
    }
  }
  const rank = (k: string) => (KIND_ORDER.includes(k) ? KIND_ORDER.indexOf(k) : KIND_ORDER.length)
  return [...map.values()].sort((a, b) => rank(a.kind) - rank(b.kind))
}
