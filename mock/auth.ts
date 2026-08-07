import type { MockMethod } from 'vite-plugin-mock'

/**
 * 注意:vite-plugin-mock 对每个 mock 文件独立编译,模块级状态不共享。
 * 因此 token 采用「无状态」校验:只要符合 mock-token 格式即视为有效,
 * 避免跨文件 sessions Map 失效。
 */

let accessSeq = 0
let refreshSeq = 0

function issueTokens(_userId: number) {
  return {
    accessToken: `mock-access-${++accessSeq}-${Date.now()}`,
    refreshToken: `mock-refresh-${++refreshSeq}-${Date.now()}`,
  }
}

export function issueTokensFor(userId: number) {
  return issueTokens(userId)
}

export function verifyAccess(headers: Record<string, string>): number | null {
  const auth = headers?.authorization ?? headers?.Authorization ?? ''
  if (!auth.startsWith('Bearer mock-access-')) return null
  return 10086
}

/** 校验管理员 token(仅 mock 管理员登录签发 `mock-admin-access-` 前缀) */
export function verifyAdmin(headers: Record<string, string>): boolean {
  const auth = headers?.authorization ?? headers?.Authorization ?? ''
  return auth.startsWith('Bearer mock-admin-access-')
}

const REGISTERED_EMAILS = new Set<string>(['taken@example.com'])
const USER = { id: 10086, email: '2734921923@qq.com', role: 0 }
const ADMIN = { id: 1, email: 'admin@example.com', role: 1 }

export default [
  {
    url: '/api/v1/auth/login',
    method: 'post',
    response: ({ body }: { body: { email?: string; password?: string } }) => {
      if (!body?.email || !body?.password) {
        return { code: 40101, message: '邮箱或密码错误', data: null }
      }
      if (body.email === 'blocked@example.com') {
        return { code: 40300, message: '账号已被封禁,请联系客服', data: null }
      }
      // 管理员账号(角色区分 E2E 用)
      if (body.email === ADMIN.email) {
        if (body.password !== 'Admin@123456') {
          return { code: 40101, message: '邮箱或密码错误', data: null }
        }
        const t = {
          accessToken: `mock-admin-access-${++accessSeq}-${Date.now()}`,
          refreshToken: `mock-refresh-${++refreshSeq}-${Date.now()}`,
        }
        return {
          code: 0,
          message: 'ok',
          data: {
            access_token: t.accessToken,
            refresh_token: t.refreshToken,
            token_type: 'Bearer',
            expires_in: 7200,
            user: { ...ADMIN },
          },
        }
      }
      if (body.password !== 'Passw0rd' && body.password !== '123456') {
        return { code: 40101, message: '邮箱或密码错误', data: null }
      }
      const t = issueTokens(USER.id)
      return {
        code: 0,
        message: 'ok',
        data: {
          access_token: t.accessToken,
          refresh_token: t.refreshToken,
          token_type: 'Bearer',
          expires_in: 7200,
          user: { ...USER },
        },
      }
    },
  },
  {
    url: '/api/v1/auth/register',
    method: 'post',
    response: ({ body }: { body: { email?: string; email_code?: string } }) => {
      if (!body?.email_code || body.email_code.length !== 6) {
        return { code: 10002, message: '验证码错误或已过期', data: null }
      }
      if (REGISTERED_EMAILS.has(body.email ?? '')) {
        return { code: 10001, message: '该邮箱已注册', data: null }
      }
      const t = issueTokens(USER.id)
      return {
        code: 0,
        message: 'ok',
        data: {
          access_token: t.accessToken,
          refresh_token: t.refreshToken,
          token_type: 'Bearer',
          expires_in: 7200,
          user: { ...USER, email: body.email ?? '' },
        },
      }
    },
  },
  {
    url: '/api/v1/auth/refresh',
    method: 'post',
    response: ({ body }: { body: { refresh_token?: string } }) => {
      if (!body?.refresh_token?.startsWith('mock-refresh-')) {
        return { code: 40100, message: '登录已过期', data: null }
      }
      const t = issueTokens(USER.id)
      return {
        code: 0,
        message: 'ok',
        data: {
          access_token: t.accessToken,
          refresh_token: t.refreshToken,
          token_type: 'Bearer',
          expires_in: 7200,
        },
      }
    },
  },
  {
    url: '/api/v1/auth/forgot',
    method: 'post',
    response: () => ({ code: 0, message: '密码已重置,请重新登录', data: null }),
  },
  {
    url: '/api/v1/auth/logout',
    method: 'post',
    response: () => ({ code: 0, message: 'ok', data: null }),
  },
] as MockMethod[]
