/**
 * store 列表加载骨架单源：loading 置真 → 执行取数赋值 → finally 复位。
 * 六个列表 store（notice/ticket/order/plan/knowledge/server）的 fetch 共用此形状，
 * 各自差异（分页 total、append、groups、轮询时间戳）留在 work 内表达。
 * 传 store 实例本身即可：pinia 的 this 即响应式代理，直接读写 loading。
 */
export async function withLoading(
  s: { loading: boolean },
  work: () => Promise<void>,
): Promise<void> {
  s.loading = true
  try {
    await work()
  } finally {
    s.loading = false
  }
}
