import { defineConfig, presetUno, presetAttributify, presetIcons, transformerDirectives } from 'unocss'

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
  shortcuts: {
    'card-base': 'bg-[var(--c-bg-card)] rounded-[var(--r-card)] shadow-[var(--s-card)]',
    'btn-primary':
      'inline-flex items-center justify-center gap-1 rounded-[var(--r-pill)] bg-[linear-gradient(135deg,#6558F5,#8B5CF6)] text-white font-medium transition-all duration-[var(--t-base)] hover:shadow-lg hover:brightness-105 active:scale-98 disabled:opacity-60 disabled:pointer-events-none',
    'btn-olive':
      'inline-flex items-center justify-center gap-1 rounded-[var(--r-pill)] bg-[var(--c-olive)] text-white font-medium transition-all duration-[var(--t-base)] hover:brightness-105 active:scale-98 disabled:opacity-60 disabled:pointer-events-none',
    'btn-ghost':
      'inline-flex items-center justify-center gap-1 rounded-[var(--r-pill)] border border-[var(--c-border)] bg-transparent text-[var(--c-text)] font-medium transition-all duration-[var(--t-base)] hover:bg-[var(--c-bg-hover)] active:scale-98 disabled:opacity-60 disabled:pointer-events-none',
    'btn-danger':
      'inline-flex items-center justify-center gap-1 rounded-[var(--r-pill)] bg-[var(--c-danger)] text-white font-medium transition-all duration-[var(--t-base)] hover:brightness-105 active:scale-98 disabled:opacity-60 disabled:pointer-events-none',
    'num-font': 'font-family: "DIN Alternate", "Bahnschrift", Inter, sans-serif; font-feature-settings: "tnum"',
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
