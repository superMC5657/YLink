import { http } from '@/utils/http'
import type { ServerListResp } from '@/types/api'

export const apiServer = {
  fetch: () => http.get<ServerListResp>('/servers'),
}
