import axios from 'axios'

export const api = axios.create({ baseURL: '/api' })

export default {
  dashboard: () => api.get('/dashboard'),
  runSnapshot: () => api.post('/snapshot/run'),
  sendReport: () => api.post('/report/send'),
  snapshots: (page = 1, size = 20) => api.get('/snapshots', { params: { page, size } }),
  snapshot: (id: number) => api.get(`/snapshots/${id}`),
  compare: (from: string, to: string) => api.get('/compare', { params: { from, to } }),
  getTemplate: () => api.get('/template'),
  saveTemplate: (data: any) => api.put('/template', data),
  getDict: (source: string) => api.get(`/dict/${source}`),
  saveDict: (source: string, data: any) => api.put(`/dict/${source}`, data),
  getSettings: () => api.get('/settings'),
  saveSettings: (data: any) => api.put('/settings', data),
  testFeishu: () => api.post('/feishu/test'),
  sendLogs: (page = 1, size = 20) => api.get('/sendlogs', { params: { page, size } }),
}
