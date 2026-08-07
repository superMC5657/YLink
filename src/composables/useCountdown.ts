/**
 * 倒计时组合式函数:验证码 60s / 支付二维码有效期倒计时。
 */
import { onBeforeUnmount, ref } from 'vue'

export function useCountdown(initialSeconds = 60) {
  const remaining = ref(0)
  const running = ref(false)
  let timer: ReturnType<typeof setInterval> | null = null

  function start(seconds = initialSeconds) {
    stop()
    remaining.value = seconds
    running.value = true
    timer = setInterval(() => {
      remaining.value -= 1
      if (remaining.value <= 0) stop()
    }, 1000)
  }

  function stop() {
    if (timer) {
      clearInterval(timer)
      timer = null
    }
    running.value = false
  }

  onBeforeUnmount(stop)

  return { remaining, running, start, stop }
}
