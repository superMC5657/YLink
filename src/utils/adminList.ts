/**
 * 管理端列表机械收敛:分页钳制 / 批量审计摘要。
 * 钳制规则唯一 owner 为后端 pkg/page（Standard 缺省 10/上限 50，Audit 缺省 20/上限 100，
 * 解析即钳制于 PageParams/各审计 List，见 ADR-0004）；
 * 本文件是前端薄适配（防未钳制透传与响应回显），数字与后端同值，改后端档须同步改此处。
 * ADR-0001 的 PageOK 唯一保证与 groups `?? []` 回落不受影响,本文件不碰该边界。
 */

/** 分页档位:数字与后端 pkg/page 的 Standard/Audit 同值 */
export const pageTier = {
  standard: { defaultSize: 10, maxSize: 50 },
  audit: { defaultSize: 20, maxSize: 100 },
} as const

export type PageTier = keyof typeof pageTier

/** 后端分页钳制同形:page<1 回落 1(视图只表意图,不再各写一遍) */
export function clampPage(page: number): number {
  if (!Number.isFinite(page) || page < 1) return 1
  return Math.floor(page)
}

/** 按档钳制条数:<1 或越上限回落档位缺省值(与后端 Limit.ClampSize 同形) */
export function clampPageSizeFor(tier: PageTier, pageSize: number): number {
  const { defaultSize, maxSize } = pageTier[tier]
  if (!Number.isFinite(pageSize) || pageSize < 1 || pageSize > maxSize) return defaultSize
  return Math.floor(pageSize)
}

/** 标准档钳制:pageSize<1 或 >50 回落 10(与 pkg/page.Standard 同值) */
export function clampPageSize(pageSize: number): number {
  return clampPageSizeFor('standard', pageSize)
}

/** 审计档钳制:pageSize<1 或 >100 回落 20(与 pkg/page.Audit 同值,审计日志/重置记录) */
export function clampPageSizeAudit(pageSize: number): number {
  return clampPageSizeFor('audit', pageSize)
}

/** 分页查询归一(标准档):page/page_size 钳制后透传其余筛选(后端行为不变) */
export function withPage<T extends { page?: number; page_size?: number } & Record<string, unknown>>(
  query: T,
): T {
  return {
    ...query,
    ...(query.page !== undefined ? { page: clampPage(query.page) } : {}),
    ...(query.page_size !== undefined ? { page_size: clampPageSize(query.page_size) } : {}),
  }
}

/** 分页查询归一(审计档):审计日志/流量重置记录等 Audit 档列表传输核用 */
export function withPageAudit<
  T extends { page?: number; page_size?: number } & Record<string, unknown>,
>(query: T): T {
  return {
    ...query,
    ...(query.page !== undefined ? { page: clampPage(query.page) } : {}),
    ...(query.page_size !== undefined ? { page_size: clampPageSizeAudit(query.page_size) } : {}),
  }
}

/**
 * 审计 target 摘要同形:批量动作 target 只存 `batch:<count>`(见 CONTEXT),
 * 完整 ID 列表进 detail JSON。前端无写入点,唯一消费方为 mock 演示数据
 * (mock/admin.ts 审计日志条目),格式以此处为单源。
 */
export function batchAuditTarget(count: number): string {
  return `batch:${count}`
}
