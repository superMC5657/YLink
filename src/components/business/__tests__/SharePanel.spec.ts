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
  error: vi.fn(),
}))

// 无 NMessageProvider 环境:useMessage 返回假消息对象;copyText 走假实现
vi.mock('@/utils/platform', () => ({ copyText: mocks.copyText }))
// jsdom 无 canvas 2D 实现,二维码统一返回假 dataURL,让下载逻辑走到画布分支
vi.mock('qrcode', () => ({
  default: { toDataURL: vi.fn(async () => 'data:image/png;base64,iVBORw0KGgo=') },
}))
vi.mock('naive-ui', async (importOriginal) => {
  const actual = await importOriginal<typeof import('naive-ui')>()
  return {
    ...actual,
    useMessage: () => ({ success: mocks.success, error: mocks.error, warning: vi.fn() }),
  }
})

const LINK = 'https://ylink.example.com/#/register?code=ABC123'

async function mountPanel(
  extra: Partial<{ title: string; text: string; desc: string; code: string }> = {},
) {
  setActivePinia(createPinia())
  await useZhCN()
  const w = mount(SharePanel, {
    props: {
      show: false,
      title: '邀请赚钱',
      text: LINK,
      desc: '邀请好友注册',
      code: 'ABC123',
      ...extra,
    },
    global: {
      plugins: [i18n],
      components: { AppIcon },
      stubs: { 'n-modal': { template: '<div class="modal-stub"><slot /></div>' } },
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

  it('渲染下载图片按钮;画布不可用时提示生成失败', async () => {
    mocks.error.mockClear()
    // jsdom 无 canvas 2D 上下文,模拟返回 null 走失败分支
    const getCtx = vi
      .spyOn(HTMLCanvasElement.prototype, 'getContext')
      .mockReturnValue(null as never)
    try {
      const w = await mountPanel()
      const btn = w.findAll('button').find((b) => b.text().includes('下载图片'))
      expect(btn).toBeTruthy()
      await btn!.trigger('click')
      await flushPromises()
      expect(mocks.error).toHaveBeenCalledWith('图片生成失败,请重试')
    } finally {
      getCtx.mockRestore()
    }
  })

  /** 画布上下文 stub:measureText 按字符数×20px 估算宽度,可触发字号递减与超宽截断 */
  function makeCtxStub() {
    return {
      createLinearGradient: vi.fn(() => ({ addColorStop: vi.fn() })),
      fillStyle: '',
      font: '',
      textAlign: 'start',
      beginPath: vi.fn(),
      moveTo: vi.fn(),
      arcTo: vi.fn(),
      closePath: vi.fn(),
      fill: vi.fn(),
      fillText: vi.fn(),
      measureText: vi.fn((s: string) => ({ width: s.length * 20 })),
      drawImage: vi.fn(),
    } as unknown as CanvasRenderingContext2D
  }

  it('画布可用时合成 PNG 并触发下载(文件名含邀请码)', async () => {
    // jsdom 的 Image 不触发 onload/onerror,替换为假实现(setTimeout 由 fake timers 推进)
    class FakeImage {
      onload: (() => void) | null = null
      onerror: (() => void) | null = null
      set src(_: string) {
        setTimeout(() => this.onload?.(), 0)
      }
    }
    const ctxStub = makeCtxStub()
    const createObjectURL = vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:fake')
    const revokeObjectURL = vi.spyOn(URL, 'revokeObjectURL').mockReturnValue()
    const anchorClick = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    const getCtx = vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(ctxStub)
    const toBlob = vi
      .spyOn(HTMLCanvasElement.prototype, 'toBlob')
      .mockImplementation((cb) => cb(new Blob(['png'], { type: 'image/png' })))
    vi.stubGlobal('Image', FakeImage)
    try {
      const w = await mountPanel()
      vi.useFakeTimers() // 在点击前启用,让组件内的 revoke setTimeout 进入 fake 队列
      const btn = w.findAll('button').find((b) => b.text().includes('下载图片'))
      await btn!.trigger('click')
      await vi.advanceTimersByTimeAsync(0) // 触发 FakeImage onload,完成绘制与 toBlob
      expect(createObjectURL).toHaveBeenCalledWith(expect.any(Blob))
      expect(anchorClick).toHaveBeenCalledTimes(1)
      expect((anchorClick.mock.instances[0] as HTMLAnchorElement).download).toBe(
        'ylink-invite-ABC123.png',
      )
      // revokeObjectURL 延迟 1s 回收,立即断言未调用
      expect(revokeObjectURL).not.toHaveBeenCalled()
      await vi.advanceTimersByTimeAsync(1000)
      expect(revokeObjectURL).toHaveBeenCalledWith('blob:fake')
    } finally {
      vi.useRealTimers()
      getCtx.mockRestore()
      toBlob.mockRestore()
      createObjectURL.mockRestore()
      revokeObjectURL.mockRestore()
      anchorClick.mockRestore()
      vi.unstubAllGlobals()
    }
  })

  it('超宽注册链接在画布上被截断并追加省略号', async () => {
    class FakeImage {
      onload: (() => void) | null = null
      onerror: (() => void) | null = null
      set src(_: string) {
        queueMicrotask(() => this.onload?.())
      }
    }
    const ctxStub = makeCtxStub()
    const createObjectURL = vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:fake')
    const revokeObjectURL = vi.spyOn(URL, 'revokeObjectURL').mockReturnValue()
    const anchorClick = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    const getCtx = vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(ctxStub)
    const toBlob = vi
      .spyOn(HTMLCanvasElement.prototype, 'toBlob')
      .mockImplementation((cb) => cb(new Blob(['png'], { type: 'image/png' })))
    vi.stubGlobal('Image', FakeImage)
    try {
      const longLink =
        'https://ylink.example.com/#/register?code=ABC123&utm_source=share&utm_medium=qr&utm_campaign=invite&referrer=someverylongusername12345'
      const w = await mountPanel({ text: longLink })
      const btn = w.findAll('button').find((b) => b.text().includes('下载图片'))
      await btn!.trigger('click')
      await flushPromises()
      const fillText = ctxStub.fillText as unknown as ReturnType<typeof vi.fn>
      const drawn = fillText.mock.calls.map((c) => c[0] as string)
      // 链接被截断:以省略号结尾,且不含完整原文
      expect(drawn.some((s) => s.endsWith('…'))).toBe(true)
      expect(drawn.every((s) => !s.includes('someverylongusername12345'))).toBe(true)
    } finally {
      getCtx.mockRestore()
      toBlob.mockRestore()
      createObjectURL.mockRestore()
      revokeObjectURL.mockRestore()
      anchorClick.mockRestore()
      vi.unstubAllGlobals()
    }
  })
})
