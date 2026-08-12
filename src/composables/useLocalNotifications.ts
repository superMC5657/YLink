/**
 * 本地通知触发点(desktop-tauri.md §4):订阅到期(≤3 天)、工单状态变为已回复。
 * 触发时机由 MainLayout 的窗口聚焦刷新与定时轮询驱动;
 * 防打扰:到期按「到期日」只通知一次,工单按状态快照只通知变化,均用 localStorage 持久化。
 */
import { useI18n } from 'vue-i18n'
import { apiTicket } from '@/api/ticket'
import { useUserStore } from '@/stores/user'
import { notify } from '@/utils/notify'

const TICKET_KEY = 'notify:ticket-status:v1'
const EXPIRE_KEY = 'notify:expire'

export function useLocalNotifications() {
  const { t } = useI18n()

  /** 订阅到期 ≤3 天且未过期 → 通知一次(按到期日去重) */
  async function checkExpire(): Promise<void> {
    const user = useUserStore()
    const sub = user.subscribe
    if (!sub?.expired_at || sub.is_expired) return
    const days = sub.expired_days
    if (days < 0 || days > 3) return
    const key = `${EXPIRE_KEY}:${sub.expired_at.slice(0, 10)}`
    if (localStorage.getItem(key)) return
    localStorage.setItem(key, '1')
    await notify(t('notify.expireTitle'), t('notify.expireBody', { days }))
  }

  /** 工单状态变为「已回复」(0→1 或 2→1)→ 通知一次(状态快照去重) */
  async function checkTickets(): Promise<void> {
    try {
      const data = await apiTicket.fetch({ page: 1, page_size: 10 })
      const list = data?.list ?? []
      if (list.length === 0) return
      let prev: Record<number, number> = {}
      const raw = localStorage.getItem(TICKET_KEY)
      if (raw) {
        try {
          prev = JSON.parse(raw) as Record<number, number>
        } catch {
          // 忽略损坏的快照
        }
      }
      const cur: Record<number, number> = {}
      for (const tk of list) {
        cur[tk.id] = tk.status
        // 仅旧工单(快照中已存在)状态变为已回复才提醒,避免首次运行与新工单误报
        if (tk.status === 1 && prev[tk.id] !== undefined && prev[tk.id] !== 1) {
          await notify(t('notify.ticketReplied'), tk.subject)
        }
      }
      localStorage.setItem(TICKET_KEY, JSON.stringify(cur))
    } catch {
      // 网络失败静默,下次轮询再试
    }
  }

  return { checkExpire, checkTickets }
}
