/**
 * Vitest 全局 setup:i18n 工具函数(format/http 等)通过 i18n.global 翻译,
 * 测试默认断言中文文案,这里预加载 zh-CN 语言包并固定 locale。
 */
import { i18n } from '@/i18n'
import zhCN from '@/locales/zh-CN'

i18n.global.setLocaleMessage('zh-CN', zhCN)
i18n.global.locale.value = 'zh-CN'
