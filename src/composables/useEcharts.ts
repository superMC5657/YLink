/**
 * ECharts 共享封装(动态加载 + 按需注册 + 实例管理)。
 *
 * - 动态加载:echarts/core、charts、components、renderers 全部动态 import,
 *   模块级缓存保证全应用只加载注册一次;echarts(~475KB)不进首屏,进入图表页才拉取。
 * - 实例缓存:按挂载元素缓存实例 promise,重复 getChart 同一 el 返回同一实例,天然防重复 init。
 * - 统一收口:dispose(卸载时)/ resize / cssVar 取色均由本 composable 管理,调用方不再裸操作实例。
 * 调用方:TrafficView(用户流量)/ AdminReportsView(后台报表 5 图,视口懒初始化)。
 */
import { onBeforeUnmount } from 'vue'
import type { ECharts } from 'echarts/core'

type EchartsCore = typeof import('echarts/core')

let corePromise: Promise<EchartsCore> | null = null

/** 动态加载 echarts 并注册本项目用到的图表/组件(全应用单次) */
function loadEcharts(): Promise<EchartsCore> {
  corePromise ??= Promise.all([
    import('echarts/core'),
    import('echarts/charts'),
    import('echarts/components'),
    import('echarts/renderers'),
  ]).then(([core, charts, components, renderers]) => {
    core.use([
      charts.BarChart,
      charts.LineChart,
      components.GridComponent,
      components.TooltipComponent,
      components.LegendComponent,
      renderers.CanvasRenderer,
    ])
    return core
  })
  return corePromise
}

export function useEcharts() {
  /** el → 实例 promise:同一挂载元素只 init 一次 */
  const pending = new Map<HTMLElement, Promise<ECharts>>()

  /** 获取(或创建)绑定到 el 的图表实例;失败返回 null */
  async function getChart(el: HTMLElement): Promise<ECharts | null> {
    let p = pending.get(el)
    if (!p) {
      p = loadEcharts().then((echarts) => echarts.init(el))
      pending.set(el, p)
    }
    try {
      return await p
    } catch {
      pending.delete(el)
      return null
    }
  }

  /** 运行时读取 CSS 变量取主题色(读不到回退 fallback,兼容 jsdom/SSR) */
  function cssVar(name: string, fallback: string): string {
    if (typeof document === 'undefined') return fallback
    const v = getComputedStyle(document.documentElement).getPropertyValue(name).trim()
    return v || fallback
  }

  /** 元素进入视口时触发一次 cb(用于图表懒初始化);环境不支持 IntersectionObserver 时立即触发 */
  function whenVisible(el: HTMLElement, cb: () => void): () => void {
    if (typeof IntersectionObserver === 'undefined') {
      cb()
      return () => {}
    }
    const io = new IntersectionObserver(
      (entries) => {
        if (entries.some((e) => e.isIntersecting)) {
          io.disconnect()
          cb()
        }
      },
      { rootMargin: '0px 0px 100px 0px' },
    )
    io.observe(el)
    return () => io.disconnect()
  }

  /** 重绘所有已创建实例(窗口 resize 时调用) */
  function resizeAll(): void {
    for (const p of pending.values()) void p.then((c) => c.resize()).catch(() => {})
  }

  onBeforeUnmount(() => {
    for (const p of pending.values()) void p.then((c) => c.dispose()).catch(() => {})
    pending.clear()
  })

  return { getChart, cssVar, whenVisible, resizeAll }
}
