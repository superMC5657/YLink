import { http } from '@/utils/http'
import { withPage } from '@/utils/adminList'
import { adminPath as ap } from './path'
import type {
  AdminAuditLogsResp,
  AdminMailTemplateItem,
  AdminMailTemplateReq,
  AdminMailTemplateTestReq,
  AdminSettingsItem,
  AdminSettingsReq,
  AdminSubscriptionTemplateItem,
  AdminSubscriptionTemplatePreviewResp,
  AdminSubscriptionTemplateReq,
  AdminTelegramWebhookSetupResp,
  AdminTrafficResetLogItem,
  AdminTrafficResetReq,
  AdminTrafficResetResp,
  AdminVersionResp,
  PageResult,
  TrafficImportReq,
} from '@/types/api'

/**
 * 系统域传输核(审计/流量/设置/模板/版本):本域 URL 与响应形态的唯一归属
 * (C3 transport-core 收敛,决策见 ADR-0003)。
 */
export const systemTransport = {
  // 审计日志(F08 只读)
  auditLogs: (query: {
    page?: number
    page_size?: number
    admin_id?: number | ''
    action?: string
    target?: string
    from?: string
    to?: string
  }) => http.get<AdminAuditLogsResp>(ap('audit-logs'), { query: withPage(query) }),

  // 流量导入(模式 B)
  importTraffic: (body: TrafficImportReq) => http.post<null>(ap('traffic/import'), { body }),

  // 流量重置(F16)
  resetTraffic: (body: AdminTrafficResetReq) =>
    http.post<AdminTrafficResetResp>(ap('traffic/reset'), { body }),
  trafficResets: (query: { page?: number; page_size?: number; user_id?: number | '' }) =>
    http.get<PageResult<AdminTrafficResetLogItem>>(ap('traffic/resets'), {
      query: withPage(query),
    }),

  // 邮件模板（F11）
  mailTemplates: () => http.get<{ list: AdminMailTemplateItem[] }>(ap('mail-templates')),
  saveMailTemplate: (name: string, body: AdminMailTemplateReq) =>
    http.put<null>(ap(`mail-templates/${name}`), { body }),
  resetMailTemplate: (name: string) => http.delete<null>(ap(`mail-templates/${name}`)),
  testMailTemplate: (name: string, body: AdminMailTemplateTestReq) =>
    http.post<null>(ap(`mail-templates/${name}/test`), { body }),

  // 订阅模板（F10）
  subscriptionTemplates: () =>
    http.get<{ list: AdminSubscriptionTemplateItem[] }>(ap('subscription-templates')),
  saveSubscriptionTemplate: (name: string, body: AdminSubscriptionTemplateReq) =>
    http.put<null>(ap(`subscription-templates/${name}`), { body }),
  resetSubscriptionTemplate: (name: string) =>
    http.delete<null>(ap(`subscription-templates/${name}`)),
  previewSubscriptionTemplate: (name: string) =>
    http.post<AdminSubscriptionTemplatePreviewResp>(ap(`subscription-templates/${name}/preview`)),

  // Telegram（F12）
  setupTelegramWebhook: () =>
    http.post<AdminTelegramWebhookSetupResp>(ap('telegram/webhook/setup')),

  // 版本检查（F20）
  version: () => http.get<AdminVersionResp>(ap('version')),

  // 站点设置
  settings: () => http.get<{ list: AdminSettingsItem[] }>(ap('settings')),
  saveSetting: (body: AdminSettingsReq) => http.put<null>(ap('settings'), { body }),
}
