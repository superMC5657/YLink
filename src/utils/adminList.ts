/**
 * 管理端列表机械收敛:分页钳制 / 批量审计摘要。
 * 钳制规则唯一 owner 为后端 pkg/page（标准档缺省 10/上限 50，解析即钳制于 PageParams）；
 * 本文件是前端薄适配（防未钳制透传与响应回显），数字与后端同值，改后端档须同步改此处。
 * ADR-0001 的 PageOK 唯一保证与 groups `?? []` 回落不受影响,本文件不碰该边界。
 */

/** 后端分页钳制同形:page<1 回落 1(视图只表意图,不再各写一遍) */
export function clampPage(page: number): number {
  if (!Number.isFinite(page) || page < 1) return 1
  return Math.floor(page)
}

/** 后端分页钳制同形:pageSize<1 或 >50 回落 10 */
export function clampPageSize(pageSize: number): number {
  if (!Number.isFinite(pageSize) || pageSize < 1 || pageSize > 50) return 10
  return Math.floor(pageSize)
}

/** 分页查询归一:page/page_size 钳制后透传其余筛选(后端行为不变) */
export function withPage<T extends { page?: number; page_size?: number } & Record<string, unknown>>(
  query: T,
): T {
  return {
    ...query,
    ...(query.page !== undefined ? { page: clampPage(query.page) } : {}),
    ...(query.page_size !== undefined ? { page_size: clampPageSize(query.page_size) } : {}),
  }
}

/**
 * 审计 target 摘要同形:批量动作 target 只存 `batch:<count>`(见 CONTEXT),
 * 完整 ID 列表进 detail JSON。mock 与前端拼摘要处经此一处。
 */
export function batchAuditTarget(count: number): string {
  return `batch:${count}`
}
