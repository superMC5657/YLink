import { createI18n } from 'vue-i18n'
import { describe, expect, it } from 'vitest'
import enUS from '../en-US'
import zhCN from '../zh-CN'

/**
 * i18n 消息编译回归:文案中的字面量花括号(如邮件模板占位符 `{{.site_name}}`)
 * 会被 vue-i18n 当作插值语法解析,非法占位符在编译期抛
 * `Message compilation error: Not allowed nest placeholder`,且异常发生在渲染函数内
 * → 组件渲染中断(管理后台邮件模板编辑弹窗无法打开,见 docs/frontend/progress.md)。
 * 字面量花括号必须用 vue-i18n 字面量插值语法转义:`{'{{.site_name}}'}`。
 * 本用例遍历两份 locale 的每条消息执行 t(),任何一条编译失败即红。
 */
type MsgTree = Record<string, unknown>

/** 收集消息树全部叶子 key(点分路径) */
function leafKeys(tree: MsgTree, prefix = ''): string[] {
  return Object.entries(tree).flatMap(([k, v]) => {
    const path = prefix ? `${prefix}.${k}` : k
    return typeof v === 'string' ? [path] : leafKeys(v as MsgTree, path)
  })
}

describe('i18n 消息可编译', () => {
  const locales = [
    ['zh-CN', zhCN],
    ['en-US', enUS],
  ] as const

  for (const [locale, messages] of locales) {
    it(`${locale}: 全部消息 t() 编译执行不抛错`, () => {
      // createI18n 的 messages 是 per-locale 字典({ 'zh-CN': {...} }),
      // 必须按 locale 包一层;直接传单 locale 树会静默空实例走 missing 分支
      const instance = createI18n({
        legacy: false,
        locale,
        messages: { [locale]: messages },
      })
      const keys = leafKeys(messages as MsgTree)
      // 消息树规模下限,防止空树/结构变化导致用例空转
      expect(keys.length).toBeGreaterThan(300)

      const broken: string[] = []
      for (const key of keys) {
        try {
          instance.global.t(key)
        } catch (e) {
          broken.push(`${key}: ${(e as Error).message.split('\n')[0]}`)
        }
      }
      expect(broken).toEqual([])
    })
  }
})
