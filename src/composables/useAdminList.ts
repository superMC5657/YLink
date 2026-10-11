import { computed, ref } from 'vue'
import { clampPage, clampPageSize } from '@/utils/adminList'
import type { PageResult } from '@/types/api'

/**
 * 管理端列表深模块(C3 transport-core 收敛,决策见 ADR-0003)。
 *
 * 收敛前各域(用户/节点/订单/工单…)各抄一份骨架:
 * loading/list/total/page/load/onPageChange + 多选 + 批量汇总,行为靠人工对齐。
 * 本模块是该骨架的唯一归属,自带三层深:
 *   1. 钳制:page/page_size 进出均按后端同形规则钳制(utils/adminList);
 *   2. 多选:行选/全选/清空 + 批量结果汇总(文案与收敛前逐字一致);
 *   3. 错误恢复与空页回退:失败保留旧表并重抛(http 层已 toast);
 *      因删空翻到无数据的页时自动退一页重取(单步有界,首页空即接受)。
 *
 * 多选默认按条目 `id` 取键;无 id 线型(如订单 order_no)如需多选请传 getId。
 * 确认弹窗等域语义留在各组合式函数,本模块不碰。
 */

/** 批量失败明细(与收敛前 useAdminSelection 同形) */
export interface AdminBatchFailedItem {
  id: number
  reason: string
}

/** 批量结果汇总(与收敛前 useAdminSelection 同形) */
export interface AdminBatchResult {
  success: number
  failed: AdminBatchFailedItem[]
}

export interface AdminListNotifier {
  success: (msg: string) => void
  warning: (msg: string) => void
}

export interface AdminListFetch<TItem, TQuery extends Record<string, unknown>> {
  (
    query: TQuery & { page: number; page_size: number },
  ): Promise<PageResult<TItem> | { list: TItem[] }>
}

export interface AdminListOptions<TItem, TQuery extends Record<string, unknown>> {
  fetch: AdminListFetch<TItem, TQuery>
  /** 首取与 load() 的筛选(不含分页键);缺省空对象 */
  initialQuery?: TQuery
  /** 默认 10;钳制规则与后端同形 */
  pageSize?: number
  /** 多选键;缺省读条目 id(非数字 id 会抛错,防静默错选) */
  getId?: (item: TItem) => number
  /** 批量汇总提示出口;缺省 reportBatch 为空操作(纯函数版照常用) */
  notify?: AdminListNotifier
  /** 失败文案词;函数形式则每次现取( locale 切换不 snapshot) */
  failedWord?: string | (() => string)
  /** 每次取数成功后的透出(如审计日志 actions 聚合),不改变主流程 */
  onResponse?: (res: PageResult<TItem> | { list: TItem[] }) => void
}

function defaultGetId(item: unknown): number {
  const id = (item as unknown as { id?: unknown } | null)?.id
  if (typeof id !== 'number') throw new Error('useAdminList: 可选项缺数字 id,请传 getId')
  return id
}

export function useAdminList<TItem, TQuery extends Record<string, unknown>>(
  options: AdminListOptions<TItem, TQuery>,
) {
  const { fetch, getId = defaultGetId, notify, failedWord = '', onResponse } = options

  const loading = ref(false)
  const list = ref<TItem[]>([])
  const total = ref(0)
  const page = ref(1)
  const pageSize = ref(clampPageSize(options.pageSize ?? 10))
  const lastQuery = ref<TQuery>({ ...((options.initialQuery ?? {}) as TQuery) })

  // ref 解包后的条目再断回 TItem,供 getId 消费(运行时同一对象)
  const keyOf = (item: unknown): number => getId(item as TItem)

  function assign(res: PageResult<TItem> | { list: TItem[] }) {
    list.value = res.list ?? []
    total.value =
      typeof (res as PageResult<TItem>).total === 'number'
        ? (res as PageResult<TItem>).total
        : list.value.length
    onResponse?.(res)
    return res
  }

  function currentQuery(): TQuery & { page: number; page_size: number } {
    return { ...lastQuery.value, page: page.value, page_size: pageSize.value }
  }

  async function fetchPage() {
    page.value = clampPage(page.value)
    loading.value = true
    try {
      let res = assign(await fetch(currentQuery()))
      if (list.value.length === 0 && page.value > 1 && total.value > 0) {
        page.value -= 1
        res = assign(await fetch(currentQuery()))
      }
      return res
    } finally {
      loading.value = false
    }
  }

  /**
   * 取数:复用上次筛选(刷新/变更后重载;模板 @click 直绑,零参)。
   * 失败保留旧表,loading 必复位,错误继续抛出(提示已由 http 层统一 toast)。
   */
  function load() {
    return fetchPage()
  }

  /** 换筛选并回首页取数(搜索/过滤入口)。 */
  function applyFilters(query: TQuery) {
    lastQuery.value = { ...query }
    page.value = 1
    return fetchPage()
  }

  function onPageChange(p: number) {
    page.value = clampPage(p)
    return fetchPage()
  }

  function setPageSize(ps: number) {
    pageSize.value = clampPageSize(ps)
    page.value = 1
    return fetchPage()
  }

  // ---- 多选(自 useAdminSelection 整体搬入,语义逐字一致) ----
  const selectedIds = ref<number[]>([])

  function isSelected(id: number): boolean {
    return selectedIds.value.includes(id)
  }

  function toggleRow(id: number, checked: boolean) {
    if (checked) {
      if (!selectedIds.value.includes(id)) selectedIds.value = [...selectedIds.value, id]
    } else {
      selectedIds.value = selectedIds.value.filter((v) => v !== id)
    }
  }

  const allChecked = computed(
    () =>
      list.value.length > 0 && list.value.every((item) => selectedIds.value.includes(keyOf(item))),
  )

  function toggleAll(checked: boolean) {
    selectedIds.value = checked ? list.value.map((item) => keyOf(item)) : []
  }

  function clearSelection() {
    selectedIds.value = []
  }

  // ---- 批量汇总(自 useAdminSelection 整体搬入,文案逐字一致) ----
  function reportBatch(title: string, resp: AdminBatchResult) {
    reportBatchResult(
      notify,
      title,
      resp,
      typeof failedWord === 'function' ? failedWord() : failedWord,
    )
  }

  return {
    loading,
    list,
    total,
    page,
    pageSize,
    load,
    applyFilters,
    onPageChange,
    setPageSize,
    selectedIds,
    isSelected,
    toggleRow,
    toggleAll,
    clearSelection,
    allChecked,
    reportBatch,
  }
}

/** 批量结果汇总提示:成功 n,失败列出原因(与收敛前两处文案同形) */
export function reportBatchResult(
  message: AdminListNotifier | undefined,
  title: string,
  resp: AdminBatchResult,
  failedWord: string,
) {
  if (!message) return
  if (resp.failed.length === 0) {
    message.success(`${title}: ${resp.success}`)
  } else {
    const detail = resp.failed.map((f) => `#${f.id} ${f.reason}`).join('; ')
    message.warning(`${title}: ${resp.success} / ${failedWord} ${resp.failed.length} — ${detail}`)
  }
}
