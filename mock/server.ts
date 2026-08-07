import type { MockMethod } from 'vite-plugin-mock'
import { verifyAccess } from './auth'
import type { ServerListResp } from '../src/types/api'

function ok(data: unknown) {
  return { code: 0, message: 'ok', data }
}

function unauthorized() {
  return { code: 40100, message: '未登录或 token 失效', data: null }
}

const servers: ServerListResp = {
  groups: [
    {
      group: '香港',
      servers: [
        {
          id: 11,
          name: '香港 01 | IPLC',
          type: 'trojan',
          rate: 1.0,
          status: 1,
          tags: ['IPLC', '流媒体'],
        },
        { id: 12, name: '香港 02 | IEPL', type: 'trojan', rate: 1.2, status: 1, tags: ['IEPL'] },
        { id: 13, name: '香港 03', type: 'vmess', rate: 0.8, status: 2, tags: [] },
      ],
    },
    {
      group: '台湾',
      servers: [
        {
          id: 21,
          name: '台湾 01 | IPLC',
          type: 'trojan',
          rate: 1.5,
          status: 1,
          tags: ['IPLC', '流媒体'],
        },
        { id: 22, name: '台湾 02', type: 'vmess', rate: 1.0, status: 3, tags: [] },
      ],
    },
    {
      group: '日本',
      servers: [
        { id: 31, name: '东京 01', type: 'shadowsocks', rate: 1.0, status: 1, tags: ['流媒体'] },
        {
          id: 32,
          name: '东京 02 | SoftBank',
          type: 'trojan',
          rate: 1.1,
          status: 2,
          tags: ['SoftBank'],
        },
      ],
    },
    {
      group: '新加坡',
      servers: [
        { id: 41, name: '新加坡 01 | IPLC', type: 'trojan', rate: 1.3, status: 1, tags: ['IPLC'] },
      ],
    },
    {
      group: '美国',
      servers: [
        {
          id: 51,
          name: '洛杉矶 01',
          type: 'vmess',
          rate: 1.6,
          status: 1,
          tags: ['流媒体', '解锁'],
        },
        { id: 52, name: '洛杉矶 02', type: 'trojan', rate: 1.6, status: 3, tags: [] },
      ],
    },
  ],
}

export default [
  {
    url: '/api/v1/servers',
    method: 'get',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAccess(headers)) return unauthorized()
      return ok(servers)
    },
  },
] as MockMethod[]
