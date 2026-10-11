import { http } from '@/utils/http'
import { adminPath as ap } from './path'
import type {
  AdminBatchServerReq,
  AdminBatchServerResp,
  AdminServerGroupItem,
  AdminServerGroupReq,
  AdminServerItem,
  AdminServerReq,
  AdminSortServerReq,
} from '@/types/api'

/**
 * 节点域传输核:本域 URL 与响应形态的唯一归属
 * (C3 transport-core 收敛,决策见 ADR-0003)。
 */
export const nodesTransport = {
  servers: () => http.get<{ list: AdminServerItem[] }>(ap('servers')),
  createServer: (body: AdminServerReq) => http.post<AdminServerItem>(ap('servers'), { body }),
  updateServer: (id: number, body: AdminServerReq) => http.put<null>(ap(`servers/${id}`), { body }),
  deleteServer: (id: number) => http.delete<null>(ap(`servers/${id}`)),
  resetServerNodeKey: (id: number) =>
    http.post<{ node_key: string }>(ap(`servers/${id}/node-key/reset`)),

  // 节点批量 / 复制 / 排序（F09）
  batchServers: (body: AdminBatchServerReq) =>
    http.post<AdminBatchServerResp>(ap('servers/batch'), { body }),
  copyServer: (id: number) => http.post<AdminServerItem>(ap(`servers/${id}/copy`)),
  sortServers: (body: AdminSortServerReq) => http.post<null>(ap('servers/sort'), { body }),
  serverGroups: () => http.get<{ list: AdminServerGroupItem[] }>(ap('server-groups')),
  createServerGroup: (body: AdminServerGroupReq) =>
    http.post<AdminServerGroupItem>(ap('server-groups'), { body }),
  updateServerGroup: (id: number, body: AdminServerGroupReq) =>
    http.put<null>(ap(`server-groups/${id}`), { body }),
  deleteServerGroup: (id: number) => http.delete<null>(ap(`server-groups/${id}`)),
}
