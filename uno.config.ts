import {
  defineConfig,
  presetUno,
  presetAttributify,
  presetIcons,
  transformerDirectives,
} from 'unocss'

export default defineConfig({
  presets: [
    presetUno(),
    presetAttributify(),
    presetIcons({
      scale: 1.2,
      warn: true,
      collections: {
        // 由 @iconify/vue 运行时按需加载,此处仅声明
      },
    }),
  ],
  transformers: [transformerDirectives()],
  rules: [
    // 覆盖 presetUno 的 text-<number> 规则:默认按 4px 基准转 rem(如 text-13 → 3.25rem = 52px),
    // 本项目约定 text-N = N px(design-system.md §3 字号阶梯 12/13/14/16/18/20/24/28/32)。
    [/^text-(\d+(?:\.\d+)?)$/, ([, d]) => ({ 'font-size': `${d}px` })],
  ],
  shortcuts: {
    'card-base': 'bg-[var(--c-bg-card)] rounded-[var(--r-card)] shadow-[var(--s-card)]',
    'btn-primary':
      'whitespace-nowrap inline-flex items-center justify-center gap-1 rounded-[var(--r-control)] bg-[linear-gradient(135deg,#6558F5,#8B5CF6)] text-white font-medium transition-all duration-[var(--t-base)] hover:shadow-lg hover:brightness-105 active:scale-98 disabled:opacity-60 disabled:pointer-events-none',
    'btn-olive':
      'whitespace-nowrap inline-flex items-center justify-center gap-1 rounded-[var(--r-control)] bg-[var(--c-olive)] text-white font-medium transition-all duration-[var(--t-base)] hover:brightness-105 active:scale-98 disabled:opacity-60 disabled:pointer-events-none',
    'btn-ghost':
      'whitespace-nowrap inline-flex items-center justify-center gap-1 rounded-[var(--r-control)] border border-[var(--c-border)] bg-transparent text-[var(--c-text)] font-medium transition-all duration-[var(--t-base)] hover:bg-[var(--c-bg-hover)] active:scale-98 disabled:opacity-60 disabled:pointer-events-none',
    'btn-danger':
      'whitespace-nowrap inline-flex items-center justify-center gap-1 rounded-[var(--r-control)] bg-[var(--c-danger)] text-white font-medium transition-all duration-[var(--t-base)] hover:brightness-105 active:scale-98 disabled:opacity-60 disabled:pointer-events-none',
    // 彩色浅底按钮(表格操作列等使用;浅色主题为浅底+深色文字,暗色主题为半透明底+亮色文字)
    'btn-soft-primary':
      'whitespace-nowrap inline-flex items-center justify-center gap-1 rounded-[var(--r-control)] bg-[var(--c-primary-soft)] text-[var(--c-primary-text)] font-medium transition-all duration-[var(--t-base)] hover:brightness-105 active:scale-98 disabled:opacity-60 disabled:pointer-events-none',
    'btn-soft-blue':
      'whitespace-nowrap inline-flex items-center justify-center gap-1 rounded-[var(--r-control)] bg-[var(--c-blue-bg)] text-[var(--c-blue)] font-medium transition-all duration-[var(--t-base)] hover:brightness-105 active:scale-98 disabled:opacity-60 disabled:pointer-events-none',
    'btn-soft-success':
      'whitespace-nowrap inline-flex items-center justify-center gap-1 rounded-[var(--r-control)] bg-[var(--c-success-bg)] text-[var(--c-success)] font-medium transition-all duration-[var(--t-base)] hover:brightness-105 active:scale-98 disabled:opacity-60 disabled:pointer-events-none',
    'btn-soft-warning':
      'whitespace-nowrap inline-flex items-center justify-center gap-1 rounded-[var(--r-control)] bg-[var(--c-warning-bg)] text-[var(--c-warning)] font-medium transition-all duration-[var(--t-base)] hover:brightness-105 active:scale-98 disabled:opacity-60 disabled:pointer-events-none',
    'btn-soft-danger':
      'whitespace-nowrap inline-flex items-center justify-center gap-1 rounded-[var(--r-control)] bg-[var(--c-danger-bg)] text-[var(--c-danger)] font-medium transition-all duration-[var(--t-base)] hover:brightness-105 active:scale-98 disabled:opacity-60 disabled:pointer-events-none',
    'btn-soft-olive':
      'whitespace-nowrap inline-flex items-center justify-center gap-1 rounded-[var(--r-control)] bg-[var(--c-olive-bg)] text-[var(--c-olive)] font-medium transition-all duration-[var(--t-base)] hover:brightness-105 active:scale-98 disabled:opacity-60 disabled:pointer-events-none',
    // 中性浅底按钮(取消/关闭/刷新等次要操作;浅色主题为浅灰底+中性文字,暗色主题为半透明底)
    'btn-soft-neutral':
      'whitespace-nowrap inline-flex items-center justify-center gap-1 rounded-[var(--r-control)] bg-[var(--c-bg-hover)] text-[var(--c-text-sub)] font-medium transition-all duration-[var(--t-base)] hover:bg-[var(--c-border)] hover:text-[var(--c-text)] active:scale-98 disabled:opacity-60 disabled:pointer-events-none',
    'num-font':
      'font-family: "DIN Alternate", "Bahnschrift", "Microsoft YaHei UI", sans-serif; font-feature-settings: "tnum"',
  },
  theme: {
    breakpoints: {
      sm: '768px',
      md: '1024px',
      lg: '1280px',
    },
    colors: {
      primary: '#6558F5',
    },
  },
})
