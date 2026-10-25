import { computed, type Ref } from 'vue'

/**
 * 滑动切换滑块位移(ThemeToggle/LanguageToggle 共用):
 * 三态/两态均分可用行程,首态贴左缘、末态贴右缘、中间态均布。
 * 轨道几何两处一致:w-14=56px,滑块 w-7=28px,左右对称余量 5px(border 1px + padding 4px)。
 */
export function useSlideOffset(currentIndex: Ref<number>, count: number) {
  const TRACK_W = 56
  const SLIDER_W = 28
  const EDGE_GAP = 5

  const slideOffset = computed(() => {
    const usable = TRACK_W - EDGE_GAP * 2 - SLIDER_W
    const steps = count - 1
    return ((usable / steps) * currentIndex.value).toFixed(1) + 'px'
  })

  return slideOffset
}
