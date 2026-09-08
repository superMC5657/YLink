import { describe, expect, it } from 'vitest'
import { renderMarkdown } from '../markdown'

/** XSS 集中策略回归:NoticePanel/PlanCard/DocsDetailView 三处共用 renderMarkdown */
describe('renderMarkdown XSS 策略', () => {
  it('内联 HTML 被转义(html:false),script/img 不以标签形式出现', async () => {
    const html = await renderMarkdown('<script>alert(1)</script><img src=x onerror=alert(1)>**hi**')
    expect(html).toContain('<strong>hi</strong>')
    // html:false 下内联 HTML 整体转义为可见文本,不产生任何真实标签
    expect(html).not.toContain('<script')
    expect(html).not.toContain('<img')
    expect(html).toContain('&lt;script&gt;')
  })

  it('javascript:/data: 链接不产出 href,仅保留 http(s)/mailto', async () => {
    const html = await renderMarkdown(
      '[a](javascript:alert(1)) [b](data:text/html,<script>) [c](https://ok.example.com) [d](mailto:safe@example.com)',
    )
    expect(html).not.toContain('href="javascript:')
    expect(html).not.toContain('href="data:')
    expect(html).toContain('href="https://ok.example.com"')
    expect(html).toContain('href="mailto:safe@example.com"')
  })

  it('链接统一后处理 target=_blank + rel=noopener noreferrer', async () => {
    const html = await renderMarkdown('[x](https://example.com)')
    expect(html).toContain(
      '<a target="_blank" rel="noopener noreferrer" href="https://example.com"',
    )
  })

  it('forbid style/form/input 标签', async () => {
    const html = await renderMarkdown('<style>body{}</style><form><input></form>text')
    expect(html).not.toContain('<style')
    expect(html).not.toContain('<form')
    expect(html).not.toContain('<input')
  })

  it('基础渲染与 linkify 正常(误杀回归)', async () => {
    const html = await renderMarkdown('**粗体** 与 https://auto.example.com')
    expect(html).toContain('<strong>粗体</strong>')
    expect(html).toContain('href="https://auto.example.com"')
  })
})
