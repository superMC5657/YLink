import { http } from '@/utils/http'
import type {
  CaptchaSendReq,
  CaptchaSendResp,
  ChangePasswordReq,
  ProfileResp,
  ProfileUpdateReq,
  SiteConfig,
  SubscribeInfo,
  TrafficLog,
  UserStat,
} from '@/types/api'

export const apiConfig = {
  fetch: () => http.get<SiteConfig>('/config', { silent: true }),
}

export const apiCaptcha = {
  sendEmail: (body: CaptchaSendReq) => http.post<CaptchaSendResp>('/captcha/email', { body }),
}

export const apiUser = {
  stat: () => http.get<UserStat>('/user/stat'),
  profile: () => http.get<ProfileResp>('/user/profile'),
  updateProfile: (body: ProfileUpdateReq) => http.put<ProfileResp>('/user/profile', { body }),
  changePassword: (body: ChangePasswordReq) => http.post<null>('/user/password/change', { body }),
  subscribe: () => http.get<SubscribeInfo>('/user/subscribe'),
  resetSubscribe: (body: { password: string }) =>
    http.post<{ subscribe_url: string }>('/user/subscribe/reset', { body }),
  trafficLogs: (from: string, to: string) =>
    http.get<{ list: TrafficLog[] }>('/user/traffic-logs', { query: { from, to } }),
}
