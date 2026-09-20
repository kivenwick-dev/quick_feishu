import { beforeEach, describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import type { VueWrapper } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import Snapshots from './views/Snapshots.vue'
import { formatSnapshotTime } from './snapshotTime'
import api from './api'

vi.mock('./api', () => ({default: {
  tokens: vi.fn().mockResolvedValue({data:{items:[]}}),
  history: vi.fn().mockResolvedValue({data:{fields:[{path:'used_quota',label:'已用配额'}],rows:Array.from({length:8},(_,i)=>({
    id:i+1,date:'2026-09-18',captured_at:`2026-09-18T10:0${i}:00+08:00`,values:{used_quota:i*100},deltas:{used_quota:'+100'},
  }))}}),
}}))

describe('snapshot history', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.mocked(api.tokens).mockResolvedValue({data:{items:[]}} as any)
    vi.mocked(api.history).mockResolvedValue({data:{fields:[{path:'used_quota',label:'已用配额'}],rows:Array.from({length:8},(_,i)=>({
      id:i+1,date:'2026-09-18',captured_at:`2026-09-18T10:0${i}:00+08:00`,values:{used_quota:i*100},deltas:{used_quota:'+100'},
    }))}} as any)
  })

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

  it('keeps metric selections independent when switching sources', async () => {
    vi.mocked(api.tokens).mockResolvedValue({data:{items:[{token_id:7,token_name:'令牌 A'}]}} as any)
    vi.mocked(api.history).mockImplementation(async (source: string) => ({data: source === 'account' ? {
      fields: [{path:'used',label:'已用'}, {path:'quota',label:'总额'}],
      rows: [{id:1,date:'2026-09-20',captured_at:'2026-09-20T08:00:00+08:00',values:{used:'10',quota:'20'},deltas:{}}],
    } : {
      fields: [{path:'requests',label:'请求数'}],
      rows: [{id:1,date:'2026-09-20',captured_at:'2026-09-20T08:00:00+08:00',values:{requests:'3'},deltas:{}}],
    }} as any))
    let wrapper = mount(Snapshots,{global:{plugins:[ElementPlus]}})
    await flushPromises()
    const fieldSelector = () => wrapper.findComponent('.field-selector') as VueWrapper<any>

    let selector = fieldSelector()
    selector.vm.$emit('update:modelValue', ['used'])
    selector.vm.$emit('change', ['used'])
    await flushPromises()

    const sourcePicker = wrapper.findComponent({name:'ElRadioGroup'})
    sourcePicker.vm.$emit('update:modelValue', 'usage')
    await flushPromises()
    sourcePicker.vm.$emit('change', 'usage')
    await flushPromises()
    expect(fieldSelector().props('modelValue')).toEqual(['requests'])

    sourcePicker.vm.$emit('update:modelValue', 'account')
    await flushPromises()
    sourcePicker.vm.$emit('change', 'account')
    await flushPromises()
    selector = fieldSelector()
    expect(selector.props('modelValue')).toEqual(['used'])
    expect(wrapper.findAll('.mcard')).toHaveLength(1)

    wrapper.unmount()
    wrapper = mount(Snapshots,{global:{plugins:[ElementPlus]}})
    await flushPromises()
    selector = fieldSelector()
    expect(selector.props('modelValue')).toEqual(['used'])
    expect(wrapper.findAll('.mcard')).toHaveLength(1)

    selector.vm.$emit('update:modelValue', [])
    selector.vm.$emit('change', [])
    await flushPromises()
    expect(wrapper.findAll('.mcard')).toHaveLength(0)
    wrapper.unmount()
  })
})
