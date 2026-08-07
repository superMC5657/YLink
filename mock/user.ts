import type { MockMethod } from 'vite-plugin-mock'
import { verifyAccess } from './auth'
import type {
  Notice,
  KnowledgeDetail,
  KnowledgeGroup,
  Plan,
  SubscribeInfo,
  TrafficLog,
} from '../src/types/api'
import dayjs from 'dayjs'

function ok(data: unknown) {
  return { code: 0, message: 'ok', data }
}

function unauthorized() {
  return { code: 40100, message: '未登录或 token 失效', data: null }
}

/** 当前订阅 */
const subscribe: SubscribeInfo = {
  has_subscription: true,
  plan: { id: 4, name: '猎户座' },
  expired_at: dayjs().add(15, 'day').toISOString(),
  is_expired: false,
  expired_days: 15,
  transfer_enable: 107374182400, // 100G
  u: 32212254720, // 30G
  d: 42949672960, // 40G
  remaining: 32212254720,
  used_percent: 70,
  speed_limit: 100,
  device_limit: 2,
  subscribe_url: 'https://api.example.com/api/v1/client/subscribe/9f3b6c4e-mock-token',
}

const notices: Notice[] = [
  {
    id: 12,
    title: '紧急通知:香港节点线路升级维护',
    content:
      '## 维护公告\n\n香港 01/02 节点将于 **2026-07-20 02:00-04:00** 进行线路升级,期间可能出现短暂波动。\n\n感谢理解与支持!',
    created_at: '2026-07-18T00:38:55+08:00',
  },
  {
    id: 11,
    title: '新增节点:新加坡 IPLC 专线上线',
    content: '## 新节点上线\n\n新加坡 **IPLC 专线** 节点正式上线,欢迎体验。\n\n- 低延迟\n- 不限速',
    created_at: '2026-07-10T10:00:00+08:00',
  },
  {
    id: 10,
    title: '618 年中大促:年付套餐 8 折',
    content: '## 限时活动\n\n活动期间购买 **年付套餐** 立享 8 折优惠,叠加优惠券更划算!',
    created_at: '2026-06-18T09:00:00+08:00',
  },
  {
    id: 9,
    title: '关于订阅流量统计延迟的说明',
    content: '流量统计可能存在 5-10 分钟延迟,属正常现象。',
    created_at: '2026-06-01T12:00:00+08:00',
  },
  {
    id: 8,
    title: '欢迎使用 YLink',
    content: '欢迎使用 YLink!请先阅读[使用文档](/docs)获取客户端配置教程。',
    created_at: '2026-05-01T08:00:00+08:00',
  },
]

const knowledgeZh: KnowledgeDetail[] = [
  {
    id: 31,
    category: '安卓配置教程',
    title: 'Nano (推荐使用)',
    body: '## 第一步:下载 App\n\n前往 [下载页](/dashboard) 获取最新版 Nano 客户端。\n\n## 第二步:导入订阅\n\n1. 打开 App,点击「添加订阅」\n2. 粘贴你的订阅链接\n3. 点击「一键导入」\n\n## 第三步:连接\n\n选择节点,点击连接按钮即可。',
    language: 'zh-CN',
    updated_at: '2026-08-04T23:51:53+08:00',
  },
  {
    id: 32,
    category: '安卓配置教程',
    title: 'Clash Meta (备用)',
    body: '## 安装 Clash Meta\n\n从官方渠道下载 Clash Meta for Android。\n\n## 导入订阅\n\n- 复制订阅链接\n- 在 Clash Meta 中「配置」→「新增配置」→「URL 导入」',
    language: 'zh-CN',
    updated_at: '2026-08-05T19:25:25+08:00',
  },
  {
    id: 33,
    category: '苹果配置教程',
    title: 'iOS:Shadowrocket 配置',
    body: '## 小火箭导入\n\n1. 复制订阅链接\n2. 打开 Shadowrocket,选择「添加订阅」\n3. 粘贴链接并保存',
    language: 'zh-CN',
    updated_at: '2026-08-01T10:00:00+08:00',
  },
  {
    id: 34,
    category: 'Windows 配置教程',
    title: 'Windows:v2rayN 使用指南',
    body: '## v2rayN 配置\n\n1. 下载 v2rayN\n2. 复制订阅(base64)链接\n3. 订阅分组 → 添加订阅 → 粘贴',
    language: 'zh-CN',
    updated_at: '2026-07-28T14:00:00+08:00',
  },
  {
    id: 35,
    category: '新手知识科普',
    title: '什么是订阅?如何正确理解流量?',
    body: '## 订阅\n\n订阅链接相当于你的「会员凭证」,包含你所有可用的节点信息。\n\n## 流量\n\n- 1 G = 1024 MB\n- 流量为「上行 + 下行」总量',
    language: 'zh-CN',
    updated_at: '2026-07-20T09:00:00+08:00',
  },
  {
    id: 36,
    category: '防失联',
    title: '防失联:获取最新地址',
    body: '为防止失联,请加入我们的 [Telegram 群组](https://t.me/ylink) 获取最新可用地址。',
    language: 'zh-CN',
    updated_at: '2026-07-15T11:00:00+08:00',
  },
  {
    id: 37,
    category: '新手知识科普',
    title: 'Demo 文档(测试用)',
    body: '# Demo 文档\n\n这是一篇用于测试 Markdown 渲染的示例文档,涵盖了常见的排版元素。\n\n## 标题层级\n\n### 三级标题\n\n#### 四级标题\n\n## 文本样式\n\n普通文本,**加粗**,*斜体*,~~删除线~~,`行内代码`。\n\n> 这是一段引用文本,用于测试引用块样式。\n\n## 列表\n\n- 无序列表项 A\n- 无序列表项 B\n  - 嵌套子项\n\n1. 有序列表项 1\n2. 有序列表项 2\n\n## 代码块\n\n```bash\n# 测试命令\ncurl -s https://api.example.com/ping\n```\n\n```ts\nconst greeting = (name: string) => `Hello, ${name}!`\n```\n\n## 表格\n\n| 功能 | 状态 | 说明 |\n| --- | --- | --- |\n| 订阅导入 | ✅ | 支持一键导入 |\n| 节点切换 | ✅ | 全球多节点 |\n| 流量查询 | ⏳ | 开发中 |\n\n## 链接与图片\n\n访问 [YLink 官网](https://example.com) 了解更多。\n\n## 分隔线\n\n---\n\n结束。',
    language: 'zh-CN',
    updated_at: '2026-08-09T12:00:00+08:00',
  },
]

const knowledgeEn: KnowledgeDetail[] = [
  {
    id: 131,
    category: 'Android Setup',
    title: 'Nano (Recommended)',
    body: '## Step 1: Download\n\nGet the latest Nano client.\n\n## Step 2: Import\n\nPaste your subscribe URL and tap import.',
    language: 'en-US',
    updated_at: '2026-08-04T23:51:53+08:00',
  },
  {
    id: 132,
    category: 'iOS Setup',
    title: 'Shadowrocket Guide',
    body: 'Copy the subscribe link, open Shadowrocket and add it.',
    language: 'en-US',
    updated_at: '2026-08-01T10:00:00+08:00',
  },
  {
    id: 137,
    category: 'Getting Started',
    title: 'Demo Document (for testing)',
    body: '# Demo Document\n\nThis is a sample document used to test Markdown rendering.\n\n## Headings\n\n### Level 3\n\n#### Level 4\n\n## Text Styles\n\nNormal text, **bold**, *italic*, ~~strikethrough~~, `inline code`.\n\n> This is a blockquote for testing.\n\n## Lists\n\n- Item A\n- Item B\n  - Nested item\n\n1. First\n2. Second\n\n## Code Blocks\n\n```bash\ncurl -s https://api.example.com/ping\n```\n\n## Table\n\n| Feature | Status | Note |\n| --- | --- | --- |\n| Import | ✅ | One-click |\n| Nodes | ✅ | Global |\n| Traffic | ⏳ | WIP |\n\n## Link\n\nVisit [YLink](https://example.com).\n\n---\n\nEnd.',
    language: 'en-US',
    updated_at: '2026-08-09T12:00:00+08:00',
  },
]

const plans: Plan[] = [
  {
    id: 1,
    name: '白羊座',
    prices: { month: 10.0, quarter: 27.0, year: 96.0 },
    traffic_gb: 300,
    speed_limit: 300,
    device_limit: 5,
    content:
      '购买套餐后可能需要等待5分钟左右才能连接\n支持**5台**设备同时在线\n**流量按月重置**,当月未用完流量次月作废',
    sort: 1,
  },
  {
    id: 2,
    name: '金牛座',
    prices: { month: 15.0, quarter: 40.5, year: 144.0 },
    traffic_gb: 500,
    speed_limit: 500,
    device_limit: 8,
    content: '更高性价比之选\n支持**8台**设备同时在线\n解锁全部节点,含 IPLC 专线',
    sort: 2,
  },
  {
    id: 3,
    name: '射手座',
    prices: { month: 20.0, quarter: 54.0, year: 192.0 },
    traffic_gb: 650,
    speed_limit: null,
    device_limit: 10,
    content: '旗舰套餐,畅享无限速体验\n支持**10台**设备同时在线\n全部节点 + 优先线路保障',
    sort: 3,
  },
]

const trafficLogs: TrafficLog[] = Array.from({ length: 30 }, (_, i) => {
  const date = dayjs()
    .subtract(29 - i, 'day')
    .format('YYYY-MM-DD')
  const u = Math.floor(Math.random() * 2 * 1024 * 1024 * 1024)
  const d = Math.floor(Math.random() * 4 * 1024 * 1024 * 1024)
  return { date, u, d, total: u + d }
})

export default [
  {
    url: '/api/v1/user/stat',
    method: 'get',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAccess(headers)) return unauthorized()
      return ok({
        email: '2734921923@qq.com',
        balance: 168.5,
        commission_balance: 88.6,
        pending_order_count: 0,
        open_ticket_count: 1,
        invited_count: 12,
        is_agent: true,
      })
    },
  },
  {
    url: '/api/v1/user/profile',
    method: 'put',
    response: ({ headers, body }: { headers: Record<string, string>; body: unknown }) => {
      if (!verifyAccess(headers)) return unauthorized()
      return ok(body)
    },
  },
  {
    url: '/api/v1/user/password/change',
    method: 'post',
    response: ({
      headers,
      body,
    }: {
      headers: Record<string, string>
      body: { old_password?: string }
    }) => {
      if (!verifyAccess(headers)) return unauthorized()
      if (!body?.old_password || body.old_password.length < 6) {
        return { code: 40101, message: '旧密码错误', data: null }
      }
      return ok(null)
    },
  },
  {
    url: '/api/v1/user/subscribe',
    method: 'get',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAccess(headers)) return unauthorized()
      return ok(subscribe)
    },
  },
  {
    url: '/api/v1/user/subscribe/reset',
    method: 'post',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAccess(headers)) return unauthorized()
      return ok({
        subscribe_url: 'https://api.example.com/api/v1/client/subscribe/new-token-9x2y',
      })
    },
  },
  {
    url: '/api/v1/user/traffic-logs',
    method: 'get',
    response: ({
      headers,
      query,
    }: {
      headers: Record<string, string>
      query: { from?: string; to?: string }
    }) => {
      if (!verifyAccess(headers)) return unauthorized()
      if (query.from && query.to) {
        return ok({
          list: trafficLogs.filter(
            (t) => t.date >= (query.from ?? '') && t.date <= (query.to ?? ''),
          ),
        })
      }
      return ok({ list: trafficLogs })
    },
  },
  {
    url: '/api/v1/notices',
    method: 'get',
    response: ({
      headers,
      query,
    }: {
      headers: Record<string, string>
      query: { page?: string; page_size?: string }
    }) => {
      if (!verifyAccess(headers)) return unauthorized()
      const page = Number(query.page ?? 1)
      const size = Number(query.page_size ?? 5)
      const start = (page - 1) * size
      return ok({
        total: notices.length,
        page,
        page_size: size,
        list: notices.slice(start, start + size),
      })
    },
  },
  {
    url: '/api/v1/knowledges',
    method: 'get',
    response: ({
      headers,
      query,
    }: {
      headers: Record<string, string>
      query: { language?: string; keyword?: string }
    }) => {
      if (!verifyAccess(headers)) return unauthorized()
      const lang = query.language ?? 'zh-CN'
      const kw = (query.keyword ?? '').toLowerCase()
      const source = lang === 'en-US' ? knowledgeEn : knowledgeZh
      const filtered = kw ? source.filter((k) => k.title.toLowerCase().includes(kw)) : source
      const groups: KnowledgeGroup[] = []
      const map = new Map<string, KnowledgeGroup>()
      for (const k of filtered) {
        if (!map.has(k.category)) {
          const g: KnowledgeGroup = { category: k.category, items: [] }
          map.set(k.category, g)
          groups.push(g)
        }
        map.get(k.category)!.items.push({ id: k.id, title: k.title, updated_at: k.updated_at })
      }
      return ok({ groups })
    },
  },
  {
    url: '/api/v1/knowledges/:id',
    method: 'get',
    response: ({ headers, query }: { headers: Record<string, string>; query: { id?: string } }) => {
      if (!verifyAccess(headers)) return unauthorized()
      const id = Number(query.id)
      const all = [...knowledgeZh, ...knowledgeEn]
      const item = all.find((k) => k.id === id)
      if (!item) return { code: 40400, message: '文档不存在', data: null }
      return ok(item)
    },
  },
  {
    url: '/api/v1/plans',
    method: 'get',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAccess(headers)) return unauthorized()
      return ok({ list: plans })
    },
  },
] as MockMethod[]
