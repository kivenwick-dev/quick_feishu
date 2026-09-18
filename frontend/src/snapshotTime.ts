export function formatSnapshotTime(capturedAt: string | undefined, date: string) {
  if (!capturedAt || capturedAt.startsWith('0001-')) return `${date}（采集时间未记录）`
  const timestamp = new Date(capturedAt)
  if (Number.isNaN(timestamp.getTime())) return `${date}（采集时间未记录）`
  const parts = new Intl.DateTimeFormat('zh-CN', {
    timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit', second: '2-digit', hourCycle: 'h23',
  }).formatToParts(timestamp)
  const value = (type: string) => parts.find(p => p.type === type)?.value
  return `${value('year')}年${value('month')}月${value('day')}日 ${value('hour')}:${value('minute')}:${value('second')}`
}
