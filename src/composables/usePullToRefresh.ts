import { onBeforeUnmount, onMounted, ref, type Ref } from 'vue'

/**
 * 移动端下拉刷新（docs/frontend/pages.md §6.3 建议项）。
 * 绑定滚动容器：仅当 scrollTop === 0 且下拉超过阈值时触发 onRefresh。
 * 通过原生 addEventListener 注册 touchmove（passive:false），
 * 只在真正下拉时 preventDefault，不影响正常滚动。
 */
export function usePullToRefresh(
  el: Ref<HTMLElement | null | undefined>,
  onRefresh: () => Promise<void> | void,
  threshold = 70,
) {
  const pulling = ref(false)
  const refreshing = ref(false)
  /** 当前下拉阻尼距离（px），用于指示器位移 */
  const distance = ref(0)

  let startY = 0
  let active = false
  let cleanup: (() => void) | null = null

  function onTouchStart(e: TouchEvent) {
    const target = el.value
    if (!target || target.scrollTop > 0 || refreshing.value) return
    active = true
    startY = e.touches[0]?.clientY ?? 0
    distance.value = 0
  }

  function onTouchMove(e: TouchEvent) {
    if (!active) return
    const y = e.touches[0]?.clientY ?? startY
    const delta = y - startY
    if (delta <= 0) {
      distance.value = 0
      return
    }
    // 顶部下拉：阻止浏览器回弹/滚动
    e.preventDefault()
    distance.value = Math.min(delta * 0.5, 96)
    pulling.value = true
  }

  async function onTouchEnd() {
    if (!active) return
    active = false
    pulling.value = false
    if (distance.value >= threshold) {
      refreshing.value = true
      distance.value = 48
      try {
        await onRefresh()
      } finally {
        refreshing.value = false
        distance.value = 0
      }
    } else {
      distance.value = 0
    }
  }

  onMounted(() => {
    const target = el.value
    if (!target) return
    target.addEventListener('touchstart', onTouchStart, { passive: true })
    target.addEventListener('touchmove', onTouchMove, { passive: false })
    target.addEventListener('touchend', onTouchEnd)
    target.addEventListener('touchcancel', onTouchEnd)
    cleanup = () => {
      target.removeEventListener('touchstart', onTouchStart)
      target.removeEventListener('touchmove', onTouchMove)
      target.removeEventListener('touchend', onTouchEnd)
      target.removeEventListener('touchcancel', onTouchEnd)
    }
  })

  onBeforeUnmount(() => {
    active = false
    cleanup?.()
  })

  return { pulling, refreshing, distance, threshold }
}
