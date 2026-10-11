/**
 * 适配缝：既有引入路径与签名保持可用，实现归属 order module 的传输核。
 * 新调用方只学 store 接口；本文件不新增逻辑。
 */
import { couponTransport, orderTransport } from '@/stores/order'

export const apiCoupon = couponTransport
export const apiOrder = orderTransport
