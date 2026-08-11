import type { MockMethod } from 'vite-plugin-mock'
import type { SiteConfig } from '../src/types/api'

/** 站点配置(与 docs/api/README.md §3.1 一致) */
const config: SiteConfig = {
  site_name: 'YLink',
  site_logo: '',
  site_description: '高速稳定的网络加速服务',
  register_enabled: true,
  invite_code_required: false,
  app_downloads: {
    windows: 'https://example.com/download/windows',
    macos: 'https://example.com/download/macos',
    android: 'https://example.com/download/android',
  },
  telegram: {
    group_url: 'https://t.me/ylink',
    bot_url: 'https://t.me/ylink_bot',
  },
  customer_service_url: 'https://t.me/ylink_cs',
  free_traffic_tips: '绑定 TG 机器人每天可领取 1G 免费流量,连续 7 天额外奖励 5G。',
  agent_policy: {
    required_valid_invites: 50,
    commission_rate: 40,
    benefits: [
      '佣金比例:40%(循环)',
      '套餐福利:赠送免费的年付订阅套餐',
      '订单推送:享受 bot 订单实时推送',
      '审验周期:12个月',
    ],
    notes: [
      '点击按钮申请代理权限,审核通过后将获得以上特权。',
      '有效邀请用户指通过你的邀请码注册且完成首次购买的用户。',
      '佣金按月结算,确认中的佣金 30 天后自动到账。',
    ],
  },
  payment_methods: [
    { code: 'balance', name: '余额支付', icon: 'wallet', enabled: true },
    { code: 'epay_alipay', name: '支付宝', icon: 'alipay', enabled: true },
    { code: 'epay_wxpay', name: '微信支付', icon: 'wechat', enabled: true },
  ],
  languages: ['zh-CN', 'en-US'],
}

export default [
  {
    url: '/api/v1/config',
    method: 'get',
    response: () => ({ code: 0, message: 'ok', data: config }),
  },
  {
    url: '/api/v1/captcha/email',
    method: 'post',
    response: () => ({ code: 0, message: '发送成功', data: { expire_in: 600, resend_after: 60 } }),
  },
] as MockMethod[]
