import { verifyAccess, verifyAdmin } from './auth'

function ok(data: unknown) {
  return { code: 0, message: 'ok', data }
}

function unauthorized() {
  return { code: 40100, message: '未登录或 token 失效', data: null }
}

/**
 * 共享公告数据：用户端 GET /notices 与管理端 /admin/notices CRUD 读写同一个数组。
 * 与真实后端语义一致——管理后台发布的公告（含优惠码）用户端立即可见。
 * （vite-plugin-mock 按文件独立编译，跨文件不共享内存态，故必须放在同一文件内。）
 * 数组按 created_at 倒序；POST 创建 unshift 到最前，保持用户端「最新在前」。
 */
const notices = [
  {
    id: 12,
    title: '618 年中大促:年付 8 折 + 专属优惠码',
    content:
      '## 限时活动\n\n活动期间购买 **年付套餐** 立享 8 折,叠加优惠券更划算!\n\n优惠码:`618SALE`(下单弹窗输入或直接点选「可用优惠券」)',
    is_show: true,
    sort: 1,
    created_at: '2026-07-18T00:38:55+08:00',
  },
  {
    id: 11,
    title: '新增节点:新加坡 IPLC 专线上线',
    content: '## 新节点上线\n\n新加坡 **IPLC 专线** 节点正式上线,欢迎体验。\n\n- 低延迟\n- 不限速',
    is_show: true,
    sort: 2,
    created_at: '2026-07-10T10:00:00+08:00',
  },
  {
    id: 10,
    title: '紧急通知:香港节点线路升级维护',
    content:
      '## 维护公告\n\n香港 01/02 节点将于 **2026-07-20 02:00-04:00** 进行线路升级,期间可能出现短暂波动。\n\n感谢理解与支持!',
    is_show: true,
    sort: 3,
    created_at: '2026-06-18T09:00:00+08:00',
  },
  {
    id: 9,
    title: '关于订阅流量统计延迟的说明',
    content: '流量统计可能存在 5-10 分钟延迟,属正常现象。',
    is_show: true,
    sort: 4,
    created_at: '2026-06-01T12:00:00+08:00',
  },
  {
    id: 8,
    title: '欢迎使用 YLink',
    content: '欢迎使用 YLink!请先阅读[使用文档](/docs)获取客户端配置教程。',
    is_show: true,
    sort: 5,
    created_at: '2026-05-01T08:00:00+08:00',
  },
]

export default [
  // ---------- 用户端：公告列表 ----------
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
      const shown = notices.filter((n) => n.is_show)
      const start = (page - 1) * size
      return ok({
        total: shown.length,
        page,
        page_size: size,
        list: shown.slice(start, start + size),
      })
    },
  },

  // ---------- 管理端：公告 CRUD ----------
  {
    url: '/api/v1/admin/notices',
    method: 'get',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      return ok({ list: notices })
    },
  },
  {
    url: '/api/v1/admin/notices',
    method: 'post',
    response: ({
      headers,
      body,
    }: {
      headers: Record<string, string>
      body: { title?: string; content?: string; is_show?: boolean; sort?: number }
    }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      if (!body.title || !body.content) {
        return { code: 40000, message: '参数校验失败: title/content 必填', data: null }
      }
      const created = {
        id: Date.now(),
        title: body.title,
        content: body.content,
        is_show: body.is_show ?? true,
        sort: body.sort ?? 0,
        created_at: new Date().toISOString(),
      }
      notices.unshift(created)
      return ok(created)
    },
  },
  {
    url: '/api/v1/admin/notices/:id',
    method: 'put',
    response: ({
      headers,
      query,
      body,
    }: {
      headers: Record<string, string>
      query: { id?: string }
      body: { title?: string; content?: string; is_show?: boolean; sort?: number }
    }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      const n = notices.find((x) => String(x.id) === query.id)
      if (!n) return { code: 40400, message: '公告不存在', data: null }
      Object.assign(n, {
        title: body.title ?? n.title,
        content: body.content ?? n.content,
        is_show: body.is_show ?? n.is_show,
        sort: body.sort ?? n.sort,
      })
      return ok(null)
    },
  },
  {
    url: '/api/v1/admin/notices/:id',
    method: 'delete',
    response: ({ headers, query }: { headers: Record<string, string>; query: { id?: string } }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      const idx = notices.findIndex((x) => String(x.id) === query.id)
      if (idx < 0) return { code: 40400, message: '公告不存在', data: null }
      notices.splice(idx, 1)
      return ok(null)
    },
  },
]
