/**
 * 本地通知封装(desktop-tauri.md §4「通知触发点」)。
 * Tauri 走 plugin-notification(动态 import);Web 端走 Notification API,
 * 不支持/未授权一律静默降级,不打扰用户。
 */
import { isTauri } from './platform'

export async function notify(title: string, body?: string): Promise<void> {
  try {
    if (isTauri()) {
      const { isPermissionGranted, requestPermission, sendNotification } =
        await import('@tauri-apps/plugin-notification')
      let granted = await isPermissionGranted()
      if (!granted) {
        const res = await requestPermission()
        granted = res === 'granted'
      }
      if (granted) {
        sendNotification({ title, body })
      }
      return
    }
    // Web 降级:Notification 仅在 https/localhost 可用
    if (typeof Notification === 'undefined') return
    if (Notification.permission === 'granted') {
      new Notification(title, { body })
    } else if (Notification.permission !== 'denied') {
      const res = await Notification.requestPermission()
      if (res === 'granted') new Notification(title, { body })
    }
  } catch {
    // 静默:通知不可用不打扰用户
  }
}
