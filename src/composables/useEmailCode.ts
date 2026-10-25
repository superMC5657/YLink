import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage, type FormItemInst } from 'naive-ui'
import { useCountdown } from './useCountdown'

/**
 * 认证引导页邮箱验证码发送(注册/找回共用):发送前先校验 email 表单项、
 * 60s 倒计时防重发、成功 toast。发送通道由调用方注入——apiCaptcha 直调
 * 留在引导页视图(docs/frontend/data-layer.md §2 认证视图例外)。
 */
export function useEmailCode(options: {
  /** email 表单项实例(getter 形式:挂载前为 null,发送时现取) */
  getEmailItem: () => FormItemInst | null
  /** 当前输入的邮箱 */
  getEmail: () => string
  send: (email: string) => Promise<unknown>
}) {
  const { t } = useI18n()
  const message = useMessage()
  const { remaining, running, start } = useCountdown(60)
  const sending = ref(false)

  async function sendCode() {
    if (sending.value || running.value) return
    try {
      await options.getEmailItem()?.validate()
    } catch {
      return
    }
    sending.value = true
    try {
      await options.send(options.getEmail())
      message.success(t('auth.codeSent'))
      start(60)
    } finally {
      sending.value = false
    }
  }

  return { remaining, running, sending, sendCode }
}
