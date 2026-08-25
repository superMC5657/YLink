/**
 * 一键导入深链接:按客户端生成 scheme 唤起 URL。
 * 规则表见 docs/frontend/data-layer.md §6.2。
 */
import { openExternal } from '@/utils/platform'
import { i18n } from '@/i18n'

export type ClientKind =
  | 'clash'
  | 'clash-meta'
  | 'clash-verge'
  | 'sing-box'
  | 'shadowrocket'
  | 'v2rayn'
  | 'v2rayng'
  | 'quantumult-x'
  | 'surge'
  | 'loon'

export interface ClientOption {
  kind: ClientKind
  name: string
  platforms: string[]
}

export const CLIENT_OPTIONS: ClientOption[] = [
  { kind: 'clash', name: 'Clash', platforms: ['windows', 'macos', 'linux', 'android'] },
  { kind: 'clash-meta', name: 'Clash Meta', platforms: ['windows', 'macos', 'linux', 'android'] },
  { kind: 'clash-verge', name: 'Clash Verge', platforms: ['windows', 'macos', 'linux'] },
  {
    kind: 'sing-box',
    name: 'sing-box',
    platforms: ['windows', 'macos', 'linux', 'android', 'ios'],
  },
  { kind: 'shadowrocket', name: 'Shadowrocket', platforms: ['ios'] },
  { kind: 'v2rayn', name: 'v2rayN', platforms: ['windows'] },
  { kind: 'v2rayng', name: 'v2rayNG', platforms: ['android'] },
  { kind: 'quantumult-x', name: 'Quantumult X', platforms: ['ios'] },
  { kind: 'surge', name: 'Surge', platforms: ['ios', 'macos'] },
  { kind: 'loon', name: 'Loon', platforms: ['ios'] },
]

function b64encode(s: string): string {
  // 浏览器环境安全的 base64
  const bytes = new TextEncoder().encode(s)
  let bin = ''
  bytes.forEach((b) => (bin += String.fromCharCode(b)))
  return btoa(bin)
}

/**
 * 生成唤起 URL。
 * @param client 客户端
 * @param subscribeUrl 订阅链接
 * @param siteName 站点名
 * @returns scheme URL;若该客户端仅支持「复制链接 + 引导」则返回 null
 */
export function buildImportUrl(
  client: ClientKind,
  subscribeUrl: string,
  siteName: string,
): string | null {
  const enc = encodeURIComponent(subscribeUrl)
  const name = encodeURIComponent(siteName || i18n.global.t('common.defaultSubName'))
  switch (client) {
    case 'clash':
    case 'clash-meta':
    case 'clash-verge':
      return `clash://install-config?url=${enc}&name=${name}`
    case 'sing-box':
      return `sing-box://import-remote-profile?url=${enc}#${name}`
    case 'shadowrocket':
      return `shadowrocket://add/sub://${b64encode(subscribeUrl)}`
    case 'v2rayng':
      return `v2rayng://install-sub?url=${enc}`
    case 'v2rayn':
      // v2rayN 无标准 scheme,走复制链接引导
      return null
    case 'quantumult-x':
    case 'surge':
    case 'loon':
      return null
    default:
      return null
  }
}

/**
 * 一键导入统一入口:能唤起的直接唤起;不能的返回 false 由调用方展示「复制链接」兜底。
 */
export function importToClient(
  client: ClientKind,
  subscribeUrl: string,
  siteName: string,
): boolean {
  const url = buildImportUrl(client, subscribeUrl, siteName)
  if (!url) return false
  openExternal(url)
  return true
}
