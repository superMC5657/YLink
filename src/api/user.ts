/**
 * 适配缝：既有引入路径与签名保持可用，实现分别归属 config / user module 的传输核。
 * 新调用方只学 store 接口；本文件不新增逻辑。
 */
import { configTransport } from '@/stores/config'
import { captchaTransport, userTransport } from '@/stores/user'

export const apiConfig = configTransport
export const apiCaptcha = captchaTransport
export const apiUser = userTransport
