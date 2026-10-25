import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage } from 'naive-ui'

/**
 * 列表排序弹窗数据域(节点/公告/知识库共用):打开时按 sort 升序(同值按 id)
 * 拷贝条目,上下移交换,保存时按最终顺序写 0..n,成功后重载列表。
 * 呈现层统一用 components/admin/SortDialog.vue;文案传 i18n key(本模块统一 t,
 * 避免构造期快照 locale)。
 */
export function useSortDialog<TItem extends { id: number; sort: number }>(options: {
  /** 待排序条目源(取当前筛选后的列表;每次打开弹窗时现取) */
  source: () => TItem[]
  /** 条目展示名(公告/知识库为 title,节点为 name) */
  nameOf: (item: TItem) => string
  /** 排序提交传输 */
  save: (items: { id: number; sort: number }[]) => Promise<unknown>
  /** 成功提示的 i18n key */
  savedKey: string
  /** 保存成功后重载列表 */
  reload: () => Promise<unknown> | void
}) {
  const { t } = useI18n()
  const message = useMessage()

  const sortModal = ref(false)
  const sortItems = ref<{ id: number; name: string }[]>([])
  const sortSaving = ref(false)

  function openSortModal() {
    sortItems.value = [...options.source()]
      .sort((a, b) => a.sort - b.sort || a.id - b.id)
      .map((item) => ({ id: item.id, name: options.nameOf(item) }))
    sortModal.value = true
  }

  function moveSortItem(index: number, delta: number) {
    const target = index + delta
    if (target < 0 || target >= sortItems.value.length) return
    const items = [...sortItems.value]
    ;[items[index], items[target]] = [items[target], items[index]]
    sortItems.value = items
  }

  async function saveSort() {
    sortSaving.value = true
    try {
      await options.save(sortItems.value.map((n, i) => ({ id: n.id, sort: i })))
      message.success(t(options.savedKey))
      sortModal.value = false
      await options.reload()
    } finally {
      sortSaving.value = false
    }
  }

  return { sortModal, sortItems, sortSaving, openSortModal, moveSortItem, saveSort }
}
