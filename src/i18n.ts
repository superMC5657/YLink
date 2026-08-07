/**
 * 全局 i18n 实例:供 main.ts 挂载与 router 守卫翻译页面标题共用,
 * 避免从 main.ts 导出造成 guards → main 循环依赖。
 */
import { createI18n } from 'vue-i18n'

export const i18n = createI18n({
  legacy: false,
  locale: 'zh-CN',
  fallbackLocale: 'zh-CN',
  messages: {},
})
