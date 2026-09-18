import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import Snapshots from './views/Snapshots.vue'
import { formatSnapshotTime } from './snapshotTime'

vi.mock('./api', () => ({default: {
  tokens: vi.fn().mockResolvedValue({data:{items:[]}}),
  history: vi.fn().mockResolvedValue({data:{fields:[{path:'used_quota',label:'已用配额'}],rows:Array.from({length:8},(_,i)=>({
    id:i+1,date:'2026-09-18',captured_at:`2026-09-18T10:0${i}:00+08:00`,values:{used_quota:i*100},deltas:{used_quota:'+100'},
  }))}}),
}}))

describe('snapshot history', () => {
  it('formats actual capture times in Beijing time', () => {
    expect(formatSnapshotTime('2026-09-18T02:03:04Z','2026-09-18')).toBe('2026年09月18日 10:03:04')
    expect(formatSnapshotTime(undefined,'2026-09-18')).toContain('采集时间未记录')
  })
  it('paginates independent same-day captures and changes page size', async () => {
    const wrapper=mount(Snapshots,{global:{plugins:[ElementPlus]}})
    await flushPromises()
    expect(wrapper.findAll('.day-panel')).toHaveLength(6)
    expect(wrapper.find('.day-date').text()).toContain('10:07:00')
    await wrapper.find('.btn-next').trigger('click')
    await flushPromises()
    expect(wrapper.findAll('.day-panel')).toHaveLength(2)
    expect(wrapper.find('.day-date').text()).toContain('10:01:00')
    const pager=wrapper.findComponent({name:'ElPagination'})
    pager.vm.$emit('update:page-size',1)
    pager.vm.$emit('size-change',1)
    await flushPromises()
    expect(wrapper.findAll('.day-panel')).toHaveLength(1)
    expect(wrapper.find('.day-date').text()).toContain('10:07:00')
    wrapper.unmount()
  })
})
