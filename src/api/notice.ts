import { http } from '@/utils/http'
import type {
  Notice,
  PageQuery,
  PageResult,
} from '@/types/api'

export const apiNotice = {
  fetch: (q: PageQuery = {}) =>
    http.get<PageResult<Notice>>('/notices', { query: { page: q.page, page_size: q.page_size } }),
}
