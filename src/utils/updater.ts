/**
 * Tauri 桌面端自动更新封装(desktop-tauri.md §5)。
 * Web 端无更新能力,全部动态 import 保证不打包进 Web 产物、不报错。
 * 降级策略:检查/下载失败一律静默返回,不打扰用户。
 */
import { isTauri } from './platform'

export interface AppUpdateInfo {
  version: string
  body?: string
  currentVersion: string
}

/**
 * 检查更新:Web 端或失败返回 null(静默降级)。
 */
export async function checkForUpdate(): Promise<AppUpdateInfo | null> {
  if (!isTauri()) return null
  try {
    const { check } = await import('@tauri-apps/plugin-updater')
    const update = await check()
    if (!update) return null
    return {
      version: update.version,
      body: update.body,
      currentVersion: update.currentVersion,
    }
  } catch (e) {
    console.warn('[updater] check failed:', e)
    return null
  }
}

/**
 * 下载并安装更新,完成后重启应用。
 * @param onProgress 下载进度回调(receivedBytes 累计, totalBytes 有总长时提供)
 * @returns 是否成功进入安装重启流程
 */
export async function downloadAndInstall(
  onProgress?: (receivedBytes: number, totalBytes?: number) => void,
): Promise<boolean> {
  if (!isTauri()) return false
  try {
    const { check } = await import('@tauri-apps/plugin-updater')
    const { relaunch } = await import('@tauri-apps/plugin-process')
    const update = await check()
    if (!update) return false
    let downloaded = 0
    let contentLength: number | undefined
    await update.downloadAndInstall((event) => {
      if (event.event === 'Started') {
        contentLength = event.data.contentLength
      } else if (event.event === 'Progress') {
        downloaded += event.data.chunkLength
        onProgress?.(downloaded, contentLength)
      }
    })
    await relaunch()
    return true
  } catch (e) {
    console.warn('[updater] install failed:', e)
    return false
  }
}

/**
 * 请求手动检查更新(供「检查更新」入口触发,UpdateCard 监听该事件展示卡片)。
 */
export function requestCheckUpdate(): void {
  window.dispatchEvent(new CustomEvent('app:check-update'))
}
