import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { NTable } from 'naive-ui'
import OrderTable from '@/components/business/OrderTable.vue'
import AppIcon from '@/components/ui/AppIcon.vue'
import StatusBadge from '@/components/ui/StatusBadge.vue'
import { i18n } from '@/i18n'
import type { Order } from '@/types/api'
import { useZhCN } from './helpers'

const base: Order = {
  order_no: '2026010100001',
  plan_name: '白羊座',
  period: 'month',
  amount: 10,
  discount_amount: 0,
  balance_used: 0,
  pay_amount: 10,
  coupon_code: null,
  status: 0,
  pay_method: null,
  paid_at: null,
  created_at: '2026-01-01T10:00:00+08:00',
}

function order(over: Partial<Order>): Order {
  return { ...base, ...over }
}

async function mountOrders(orders: Order[]) {
  await useZhCN()
  return mount(OrderTable, {
    props: { orders },
    global: {
      plugins: [i18n],
      components: { NTable, AppIcon, StatusBadge },
      // CopyText 内部 useMessage 需要 NMessageProvider，测试中用保留文本的 stub 替代
      stubs: {
        CopyText: { props: ['text'], template: '<span class="copy-stub">{{ text }}</span>' },
      },
    },
  })
}

describe('OrderTable 订单表格（桌面）', () => {
  it('渲染订单行：名称/单号/状态/金额', async () => {
    const w = await mountOrders([order({ status: 1, pay_amount: 8 })])
    expect(w.text()).toContain('白羊座')
    expect(w.text()).toContain('2026010100001')
    expect(w.text()).toContain('已完成')
    expect(w.text()).toContain('8.00')
  })

  it('待支付订单显示「去支付」按钮，已完成订单不显示', async () => {
    const w = await mountOrders([
      order({ order_no: 'A1', status: 0 }),
      order({ order_no: 'A2', status: 1 }),
    ])
    const payButtons = w.findAll('button').filter((b) => b.text().includes('去支付'))
    expect(payButtons).toHaveLength(1)
    // 第一个订单（待支付）有去支付，第二个没有
    const rowBtns = w.findAll('tr')
    expect(rowBtns[1].text()).toContain('去支付')
    expect(rowBtns[2].text()).not.toContain('去支付')
  })

  it('点击查看详情 emit view(order)', async () => {
    const o = order({ status: 2 })
    const w = await mountOrders([o])
    const btn = w.findAll('button').find((b) => b.text().includes('查看详情'))
    await btn!.trigger('click')
    expect(w.emitted('view')?.[0]).toEqual([o])
  })

  it('点击去支付 emit pay(order)', async () => {
    const o = order({ status: 0 })
    const w = await mountOrders([o])
    const btn = w.findAll('button').find((b) => b.text().includes('去支付'))
    await btn!.trigger('click')
    expect(w.emitted('pay')?.[0]).toEqual([o])
  })

  it('退款订单显示已退款徽章', async () => {
    const w = await mountOrders([order({ status: 3 })])
    expect(w.text()).toContain('已退款')
  })
})
