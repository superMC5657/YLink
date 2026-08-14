import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import SharePanel from '@/components/business/SharePanel.vue'
import AppIcon from '@/components/ui/AppIcon.vue'
import { i18n } from '@/i18n'
import { useZhCN } from './helpers'

const mocks = vi.hoisted(() => ({
  copyText: vi.fn(async () => true),
  success: vi.fn(),
}))

// 无 NMessageProvider 环境:useMessage 返回假消息对象;copyText 走假实现
vi.mock('@/utils/platform', () => ({ copyText: mocks.copyText }))
vi.mock('naive-ui', async (importOriginal) => {
  const actual = await importOriginal<typeof import('naive-ui')>()
  return {
    ...actual,
    useMessage: () => ({ success: mocks.success, error: vi.fn(), warning: vi.fn() }),
  }
})

const LINK = 'https://ylink.example.com/#/register?code=ABC123'

async function mountPanel() {
  setActivePinia(createPinia())
  await useZhCN()
  const w = mount(SharePanel, {
    props: { show: false, title: '邀请赚钱', text: LINK, desc: '邀请好友注册', code: 'ABC123' },
    global: {
      plugins: [i18n],
      components: { AppIcon },
      stubs: { 'n-drawer': { template: '<div class="drawer-stub"><slot /></div>' } },
    },
  })
  await w.setProps({ show: true })
  await flushPromises()
  return w
}

describe('SharePanel 分享面板', () => {
  it('渲染标题、说明、邀请码与链接文本', async () => {
    const w = await mountPanel()
    expect(w.text()).toContain('邀请赚钱')
    expect(w.text()).toContain('邀请好友注册')
    expect(w.text()).toContain('邀请码: ABC123')
    expect(w.text()).toContain(LINK)
    expect(w.text()).toContain('复制链接')
  })

  it('点击复制调用 copyText 并提示已复制', async () => {
    mocks.copyText.mockClear()
    mocks.success.mockClear()
    const w = await mountPanel()
    const copyBtn = w.findAll('button').find((b) => b.text().includes('复制链接'))
    expect(copyBtn).toBeTruthy()
    await copyBtn!.trigger('click')
    await flushPromises()
    expect(mocks.copyText).toHaveBeenCalledWith(LINK)
    expect(mocks.success).toHaveBeenCalledWith('已复制')
  })

  it('点击关闭 emit update:show(false)', async () => {
    const w = await mountPanel()
    const closeBtn =
      w.findAll('button').find((b) => b.text().includes('关闭')) ?? w.findAll('button')[0]
    await closeBtn!.trigger('click')
    expect(w.emitted('update:show')?.at(-1)).toEqual([false])
  })
})
