/**
 * 适配缝：既有引入路径与签名保持可用，实现归属 agent module 的传输核。
 * 新调用方只学 store 接口；本文件不新增逻辑。
 */
import { agentTransport } from '@/stores/agent'

export const apiAgent = agentTransport
