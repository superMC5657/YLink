/**
 * mock 响应信封与分页切片(各 mock 文件共用,收敛前 ok/unauthorized ×6 文件逐字重复)。
 * vite-plugin-mock 按文件独立编译,本文件只放无状态纯函数,不持模块级数据。
 */

/** 业务成功信封(envelope {code,message,data},docs/api/README.md §1) */
export function ok(data: unknown) {
  return { code: 0, message: 'ok', data }
}

/** 未登录/token 失效(40100,http 层以此触发静默刷新) */
export function unauthorized() {
  return { code: 40100, message: '未登录或 token 失效', data: null }
}

/** 非管理员调管理接口:与后端 RequireRole 一致返回 40300(而非 40100) */
export function forbidden() {
  return { code: 40300, message: '无权限操作', data: null }
}

/** 内存列表分页切片(与后端 PageOK 同形:list/total/page/page_size) */
export function paged<T>(list: T[], page: number, pageSize: number) {
  const start = (page - 1) * pageSize
  return {
    list: list.slice(start, start + pageSize),
    total: list.length,
    page,
    page_size: pageSize,
  }
}
