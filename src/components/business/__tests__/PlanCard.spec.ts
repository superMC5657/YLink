import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import PlanCard from '@/components/business/PlanCard.vue'
import PriceText from '@/components/ui/PriceText.vue'
import StatusBadge from '@/components/ui/StatusBadge.vue'
import { i18n } from '@/i18n'
import type { Plan } from '@/types/api'
import { useZhCN } from './helpers'

const plan: Plan = {
  id: 1,
  name: '白羊座',
  prices: { month: 9.9, quarter: 26.73, year: 99 },
  traffic_gb: 100,
  speed_limit: null,
  device_limit: 3,
  content: '**高速**节点\n<script>alert(1)</script>',
  sort: 1,
}

async function mountPlan(p: Plan = plan, period?: Plan['prices'][keyof Plan['prices']]) {
  await useZhCN()
  return mount(PlanCard, {
    props: { plan: p, period: period as never },
    global: { plugins: [i18n], components: { PriceText, StatusBadge } },
  })
}

describe('PlanCard 套餐卡片', () => {
  it('渲染名称、流量、设备数与价格', async () => {
    const w = await mountPlan()
    expect(w.text()).toContain('白羊座')
    expect(w.text()).toContain('100G')
    expect(w.text()).toContain('设备数')
    // 月付默认价 ¥9.90（PriceText 拆分整数/小数渲染）
    expect(w.text()).toContain('9.90')
    expect(w.text()).toContain('月付')
  })

  it('周期切换联动价格并 emit period-change', async () => {
    const w = await mountPlan()
    const yearBtn = w.findAll('button').find((b) => b.text().includes('年付'))
    expect(yearBtn).toBeTruthy()
    await yearBtn!.trigger('click')
    expect(w.emitted('period-change')?.[0]).toEqual([plan, 'year'])
    // 价格切换为年付 ¥99
    await w.setProps({ period: 'year' })
    expect(w.text()).toContain('99.00')
  })

  it('点击立即购买 emit buy(plan)', async () => {
    const w = await mountPlan()
    const buyBtn = w.findAll('button').find((b) => b.text().includes('立即购买'))
    await buyBtn!.trigger('click')
    expect(w.emitted('buy')?.[0]).toEqual([plan])
  })

  it('Markdown 正文渲染，脚本标签不生效', async () => {
    const w = await mountPlan()
    const html = w.html()
    expect(html).toContain('<strong>高速</strong>')
    // DOMPurify 不允许 script 元素：不出现真实 <script> 标签
    expect(html).not.toContain('<script')
    expect(html).not.toContain('onerror')
  })
})
