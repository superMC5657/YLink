import { computed, ref, type Ref } from 'vue'

/**
 * 管理端多选与批量结果汇总共享逻辑(机械收敛:useAdminUsers/useAdminNodes 同源)。
 * 只收敛逻辑与文案组装,不碰视图与样式;输出字符串与收敛前逐字一致。
 */
export interface AdminBatchFailedItem {
  id: number
  reason: string
}

export interface AdminBatchResult {
  success: number
  failed: AdminBatchFailedItem[]
}

type Notifier = {
  success: (msg: string) => void
  warning: (msg: string) => void
}

export function useAdminSelection<T extends { id: number }>(list: Ref<readonly T[]>) {
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
    () => list.value.length > 0 && list.value.every((item) => selectedIds.value.includes(item.id)),
  )

  function toggleAll(checked: boolean) {
    selectedIds.value = checked ? list.value.map((item) => item.id) : []
  }

  function clearSelection() {
    selectedIds.value = []
  }

  return { selectedIds, isSelected, toggleRow, toggleAll, clearSelection, allChecked }
}

/** 批量结果汇总提示:成功 n,失败列出原因(与收敛前两处文案同形) */
export function reportBatchResult(
  message: Notifier,
  title: string,
  resp: AdminBatchResult,
  failedWord: string,
) {
  if (resp.failed.length === 0) {
    message.success(`${title}: ${resp.success}`)
  } else {
    const detail = resp.failed.map((f) => `#${f.id} ${f.reason}`).join('; ')
    message.warning(`${title}: ${resp.success} / ${failedWord} ${resp.failed.length} — ${detail}`)
  }
}
