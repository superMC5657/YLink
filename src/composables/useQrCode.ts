/**
 * 二维码生成(懒加载 composable):`qrcode` 包(~100KB,含 Buffer polyfill)不进首屏,
 * 首次调用 toDataURL 时才动态 import,由 Vite 拆进异步 chunk(弹窗打开时加载)。
 * 统一 toDataURL 入口:宽度/留白/颜色参数收敛,调用方只传内容与可选覆盖项;
 * 失败不抛异常,返回空串并置 failed,由调用方决定重试/错误态 UI。
 * 调用方:PaymentModal(收银台 220px,主题色) / SharePanel(分享卡 512px,固定深色码)。
 */
import { ref } from 'vue'

export interface QrCodeOptions {
  /** 输出图片边长(px),默认 512(分享导出清晰度) */
  width?: number
  /** 码点颜色,默认 #1F2430(白底深色码,扫码可靠性最佳) */
  dark?: string
  /** 背景色,默认 #FFFFFF */
  light?: string
}

/** 模块级缓存动态导入结果:多实例/重复调用只加载一次 */
type QrcodeModule = typeof import('qrcode')
let qrcodePromise: Promise<QrcodeModule> | null = null

function loadQrcode(): Promise<QrcodeModule> {
  qrcodePromise ??= import('qrcode')
  return qrcodePromise
}

export function useQrCode() {
  const loading = ref(false)
  const failed = ref(false)

  /** 生成二维码 dataURL;生成失败返回空串(failed 置 true) */
  async function toDataURL(text: string, opts: QrCodeOptions = {}): Promise<string> {
    loading.value = true
    failed.value = false
    try {
      const { toDataURL: generate } = await loadQrcode()
      return await generate(text, {
        width: opts.width ?? 512,
        margin: 1,
        color: { dark: opts.dark ?? '#1F2430', light: opts.light ?? '#FFFFFF' },
      })
    } catch {
      failed.value = true
      return ''
    } finally {
      loading.value = false
    }
  }

  return { loading, failed, toDataURL }
}
