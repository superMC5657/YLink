/**
 * 管理端 API 路径段(F22 security.admin_path)。
 * 后端经 config.yaml security.admin_path 或 APP_SECURITY_ADMIN_PATH 定制;
 * 前端部署同步设置 VITE_ADMIN_PATH 重新构建,默认 admin 保持两端一致。
 * 各域传输核共用此一处,路径定制只改这里(C3 transport-core 收敛,见 ADR-0003)。
 */
const ADMIN_PATH = (import.meta.env.VITE_ADMIN_PATH as string | undefined) || 'admin'
export const adminPath = (path: string) => `/${ADMIN_PATH}/${path}`
