import { beforeEach, describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import BannerStatCard from '@/components/business/BannerStatCard.vue'
import AppIcon from '@/components/ui/AppIcon.vue'
import { i18n } from '@/i18n'
import { useUserStore } from '@/stores/user'
import { useZhCN } from './helpers'

describe('BannerStatCard 仪表板统计卡', () => {
  beforeEach(async () => {
    setActivePinia(createPinia())
    await useZhCN()
    const user = useUserStore()
    user.stat = {
      email: 'test@example.com',
      balance: 88.5,
      commission_balance: 12.34,
      pending_order_count: 1,
      open_ticket_count: 0,
      invited_count: 3,
      is_agent: false,
    }
  })

  async function mountCard() {
    return mount(BannerStatCard, {
      global: { plugins: [i18n], components: { AppIcon } },
    })
  }

  it('渲染用户邮箱', async () => {
    const w = await mountCard()
    expect(w.text()).toContain('test@example.com')
  })

  it('渲染余额与佣金金额（¥ 千分位）', async () => {
    const w = await mountCard()
    expect(w.text()).toContain('88.50')
    expect(w.text()).toContain('12.34')
    expect(w.text()).toContain('余额')
    expect(w.text()).toContain('佣金')
  })

  it('无 stat 时金额兜底为 ¥0.00', async () => {
    const user = useUserStore()
    user.stat = null
    const w = await mountCard()
    expect(w.text()).toContain('0.00')
  })
})
