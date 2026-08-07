import { http } from '@/utils/http'
import type { AuthResp, ForgotReq, LoginReq, RefreshReq, RegisterReq } from '@/types/api'

export const apiAuth = {
  login: (body: LoginReq) => http.post<AuthResp>('/auth/login', { body }),
  register: (body: RegisterReq) => http.post<AuthResp>('/auth/register', { body }),
  forgot: (body: ForgotReq) => http.post<null>('/auth/forgot', { body }),
  refresh: (body: RefreshReq) => http.post<AuthResp>('/auth/refresh', { body }),
  logout: () => http.post<null>('/auth/logout'),
}
