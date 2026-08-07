/**
 * 平台能力适配层:统一暴露 copyText / openExternal 等能力,
 * 内部按 Tauri / 浏览器自动降级,业务代码不感知平台差异。
 * 见 docs/frontend/README.md §3 与 desktop-tauri.md §8。
 */

export function isTauri(): boolean {
  return typeof window !== 'undefined' && '__TAURI_INTERNALS__' in window
}

/** 复制文本:Tauri 用剪贴板插件,浏览器用 navigator.clipboard(带降级) */
export async function copyText(text: string): Promise<boolean> {
  try {
    if (isTauri()) {
      // Tauri 分支:tauri-plugin-clipboard-manager(接入时启用)
      // const { writeText } = await import('@tauri-apps/plugin-clipboard-manager')
      // await writeText(text)
      // return true
    }
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
      return true
    }
    // 降级:textarea + execCommand
    const ta = document.createElement('textarea')
    ta.value = text
    ta.style.position = 'fixed'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.select()
    const ok = document.execCommand('copy')
    document.body.removeChild(ta)
    return ok
  } catch {
    return false
  }
}

/** 打开外部链接(URL 白名单校验,仅 https/mailto/已知客户端 scheme) */
export function openExternal(url: string): void {
  if (!url) return
  const scheme = url.split(':')[0].toLowerCase()
  const allowed = new Set([
    'https', 'http', 'mailto', 'tg', 't.me',
    'clash', 'sing-box', 'shadowrocket', 'v2rayng', 'v2rayn',
  ])
  if (!allowed.has(scheme)) {
    console.warn('[platform] blocked external url scheme:', scheme)
    return
  }
  if (isTauri()) {
    // Tauri 分支:tauri-plugin-opener(接入时启用)
    // window.open(url, '_blank')
    return
  }
  if (scheme === 'http' || scheme === 'https') {
    window.open(url, '_blank', 'noopener,noreferrer')
  } else {
    window.location.href = url
  }
}

/** 客户端标识,用于 X-Client 头 */
export function clientTag(): string {
  if (!isTauri()) return 'web'
  const ua = navigator.userAgent.toLowerCase()
  if (ua.includes('windows')) return 'tauri-windows'
  if (ua.includes('mac')) return 'tauri-macos'
  return 'tauri-linux'
}
