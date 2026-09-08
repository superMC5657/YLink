/**
 * Markdown 渲染统一入口(单例 + 懒加载 + 集中 XSS 策略)。
 *
 * - 懒加载:`markdown-it`(~130KB)与 DOMPurify 均动态 import,首次调用才加载,
 *   不进首屏 bundle(dashboard 公告/套餐卡/文档详情共用同一 pipeline)。
 * - 单例:renderer 模块级缓存,多调用方共享同一 markdown-it 实例与净化配置。
 * - XSS 策略集中于此,调用方一律走 renderMarkdown,禁止各自裸调 sanitize:
 *   html:false(禁内联 HTML)/ linkify / breaks → DOMPurify 白名单净化
 *   (URI 白名单 http(s)|mailto,FORBID style/form/input)→ a 标签统一
 *   target=_blank + rel=noopener noreferrer(净化后后处理,不可被注入)。
 * 调用方:NoticePanel / PlanCard / DocsDetailView。
 */
import type MarkdownIt from 'markdown-it'

/** 与设计规范一致的代码块高亮:简单转义 + 行号容器,后续可接入 highlight.js */
function highlight(_lang: string, str: string): string {
  const escaped = str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
  return `<pre class="code-block"><code>${escaped}</code></pre>`
}

/** URI 白名单:仅 http(s)/mailto,阻断 javascript:/data: 等危险协议 */
const ALLOWED_URI_REGEXP = /^(?:https?|mailto):/i

/** DOMPurify 净化配置:额外禁 style/form/input,保留 target/rel 供链接后处理 */
const SANITIZE_CONFIG = {
  ALLOWED_URI_REGEXP,
  ADD_ATTR: ['target', 'rel'],
  FORBID_TAGS: ['style', 'form', 'input'],
}

/** 链接后处理:所有外链新窗口打开 + noopener(防 reverse tabnabbing);净化后执行,属性不可伪造 */
function anchorGuard(html: string): string {
  return html.replace(/<a /g, '<a target="_blank" rel="noopener noreferrer" ')
}

type RenderPipeline = (src: string) => string

let pipelinePromise: Promise<RenderPipeline> | null = null

function loadPipeline(): Promise<RenderPipeline> {
  pipelinePromise ??= Promise.all([import('markdown-it'), import('dompurify')]).then(
    ([MarkdownItMod, DOMPurifyMod]) => {
      const MarkdownItCtor = MarkdownItMod.default
      const DOMPurifyDefault = DOMPurifyMod.default
      const md: MarkdownIt = new MarkdownItCtor({
        html: false,
        linkify: true,
        breaks: true,
        highlight,
      })
      return (src: string): string =>
        anchorGuard(DOMPurifyDefault.sanitize(md.render(src), SANITIZE_CONFIG))
    },
  )
  return pipelinePromise
}

/** 渲染 Markdown 并集中净化,返回安全的 HTML 片段(供 v-html) */
export async function renderMarkdown(src: string): Promise<string> {
  const render = await loadPipeline()
  return render(src)
}
