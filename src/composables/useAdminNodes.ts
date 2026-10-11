import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { nodesTransport } from '@/stores/admin/nodes'
import type {
  AdminBatchServerResp,
  AdminServerGroupItem,
  AdminServerItem,
  AdminServerReq,
  ServerStatus,
} from '@/types/api'
import { useMessage, useDialog } from 'naive-ui'
import { statusBadge, statusLabel } from '@/utils/statusMeta'
import { useAdminList } from './useAdminList'

/**
 * 节点管理数据域(F09):节点/分组 CRUD、多选与批量操作、复制/排序。
 * 纯逻辑与状态;弹窗呈现拆在 components/admin/*Dialog.vue,表格留在视图。
 * 列表骨架(取数/多选/批量汇总)归 useAdminList,本域只留语义(C3 收敛)。
 */
export function useAdminNodes() {
  const { t } = useI18n()
  const message = useMessage()
  const dialog = useDialog()

  const groups = ref<AdminServerGroupItem[]>([])

  // 列表骨架:节点与分组同取(分组经闭包旁路,视图分页键不消费);多选/批量汇总经深模块
  const {
    loading,
    list: servers,
    load,
    selectedIds,
    isSelected,
    toggleRow,
    toggleAll,
    clearSelection,
    allChecked,
    reportBatch: reportListBatch,
  } = useAdminList<AdminServerItem, Record<string, unknown>>({
    fetch: async () => {
      const [s, g] = await Promise.all([nodesTransport.servers(), nodesTransport.serverGroups()])
      groups.value = g.list
      return { list: s.list }
    },
    notify: message,
    failedWord: () => t('adminNodes.batchFailed'),
  })

  // 节点新建/编辑
  const nodeModal = ref(false)
  const editingNode = ref<number | null>(null)
  const nodeSaving = ref(false)
  const nodeForm = reactive<AdminServerReq>({
    group_id: 0,
    name: '',
    type: 'shadowsocks',
    host: '',
    port: 8080,
    config: '',
    rate: 1,
    tags: [],
    status: 1,
    is_show: true,
    sort: 0,
  })

  // 分组新建/编辑
  const groupModal = ref(false)
  const editingGroup = ref<number | null>(null)
  const groupSaving = ref(false)
  const groupName = ref('')
  const groupSort = ref(0)

  // 部署命令弹窗
  const installModal = ref(false)
  const installingNode = ref<AdminServerItem | null>(null)

  function openInstallDialog(node: AdminServerItem) {
    installingNode.value = node
    installModal.value = true
  }

  function statusText(status: ServerStatus): string {
    // 状态元深模块同源,输出与收敛前逐字一致
    return statusLabel('server', status)
  }

  function statusType(status: ServerStatus): 'success' | 'warning' | 'neutral' {
    // 徽标直读表,不再从色值字符串反推(旧泄漏点已删)
    return statusBadge('server', status) as 'success' | 'warning' | 'neutral'
  }

  const groupNameOf = computed(() => {
    const map = new Map(groups.value.map((g) => [g.id, g.name]))
    return (id: number) => map.get(id) ?? t('adminNodes.groupFallback', { id })
  })

  function openNodeCreate() {
    editingNode.value = null
    Object.assign(nodeForm, {
      group_id: groups.value[0]?.id ?? 0,
      name: '',
      type: 'shadowsocks',
      host: '',
      port: 8080,
      config: '',
      rate: 1,
      tags: [],
      status: 1,
      is_show: true,
      sort: 0,
    })
    nodeModal.value = true
  }

  function openNodeEdit(srv: AdminServerItem) {
    editingNode.value = srv.id
    Object.assign(nodeForm, {
      group_id: srv.group_id,
      name: srv.name,
      type: srv.type,
      host: srv.host,
      port: srv.port,
      config: srv.config,
      rate: srv.rate,
      tags: srv.tags ?? [],
      status: srv.status,
      is_show: srv.is_show,
      sort: srv.sort,
    })
    nodeModal.value = true
  }

  async function saveNode() {
    if (!nodeForm.group_id) {
      message.warning(t('adminNodes.selectGroup'))
      return
    }
    if (!nodeForm.name.trim() || !nodeForm.host.trim() || !nodeForm.config.trim()) {
      message.warning(t('adminNodes.requiredFields'))
      return
    }
    nodeSaving.value = true
    try {
      if (editingNode.value === null) {
        await nodesTransport.createServer({ ...nodeForm })
        message.success(t('adminNodes.nodeCreated'))
      } else {
        await nodesTransport.updateServer(editingNode.value, { ...nodeForm })
        message.success(t('adminNodes.nodeUpdated'))
      }
      nodeModal.value = false
      void load()
    } finally {
      nodeSaving.value = false
    }
  }

  function removeNode(srv: AdminServerItem) {
    dialog.warning({
      title: t('adminNodes.deleteNodeTitle'),
      content: t('adminNodes.deleteNodeConfirm', { name: srv.name }),
      positiveText: t('adminNodes.delete'),
      negativeText: t('common.cancel'),
      onPositiveClick: async () => {
        await nodesTransport.deleteServer(srv.id)
        message.success(t('adminNodes.deleted'))
        void load()
      },
    })
  }

  function resetNodeKey(srv: AdminServerItem) {
    dialog.warning({
      title: t('adminNodes.resetKeyTitle'),
      content: t('adminNodes.resetKeyConfirm', { name: srv.name }),
      positiveText: t('adminNodes.resetKeyBtn'),
      negativeText: t('common.cancel'),
      onPositiveClick: async () => {
        const { node_key: key } = await nodesTransport.resetServerNodeKey(srv.id)
        message.success(t('adminNodes.resetKeySuccess', { key }))
        void load()
      },
    })
  }

  // ---- F09: 多选 / 批量操作 / 复制 / 排序 ----

  /** 批量结果汇总提示:成功 n,失败列出原因(与用户端批量同风格) */
  function reportBatch(resp: AdminBatchServerResp) {
    reportListBatch(t('adminNodes.batchDone'), resp)
  }

  const batchSaving = ref(false)

  function batchDelete() {
    const ids = [...selectedIds.value]
    dialog.warning({
      title: t('adminNodes.batchDelete'),
      content: t('adminNodes.batchDeleteConfirm', { count: ids.length }),
      positiveText: t('adminNodes.delete'),
      negativeText: t('common.cancel'),
      onPositiveClick: async () => {
        batchSaving.value = true
        try {
          const resp = await nodesTransport.batchServers({ action: 'delete', ids })
          reportBatch(resp)
          clearSelection()
          void load()
        } finally {
          batchSaving.value = false
        }
      },
    })
  }

  async function batchShow(isShow: boolean) {
    batchSaving.value = true
    try {
      const resp = await nodesTransport.batchServers({
        action: 'update',
        ids: [...selectedIds.value],
        is_show: isShow,
      })
      reportBatch(resp)
      clearSelection()
      await load()
    } finally {
      batchSaving.value = false
    }
  }

  // 批量修改公共字段弹窗
  const batchModal = ref(false)
  const batchStatus = ref<ServerStatus | null>(null)
  const batchGroup = ref<number | null>(null)
  const batchRate = ref<number | null>(null)

  function openBatchUpdate() {
    batchStatus.value = null
    batchGroup.value = null
    batchRate.value = null
    batchModal.value = true
  }

  async function saveBatchUpdate() {
    const payload: {
      action: 'update'
      ids: number[]
      status?: ServerStatus
      group_id?: number
      rate?: number
    } = {
      action: 'update',
      ids: [...selectedIds.value],
    }
    let hasAny = false
    if (batchStatus.value !== null) {
      payload.status = batchStatus.value
      hasAny = true
    }
    if (batchGroup.value !== null) {
      payload.group_id = batchGroup.value
      hasAny = true
    }
    if (batchRate.value !== null) {
      payload.rate = batchRate.value
      hasAny = true
    }
    if (!hasAny) {
      message.warning(t('adminNodes.batchUpdateEmpty'))
      return
    }
    batchSaving.value = true
    try {
      const resp = await nodesTransport.batchServers(payload)
      reportBatch(resp)
      batchModal.value = false
      clearSelection()
      await load()
    } finally {
      batchSaving.value = false
    }
  }

  function copyNode(srv: AdminServerItem) {
    dialog.warning({
      title: t('adminNodes.copyTitle'),
      content: t('adminNodes.copyConfirm', { name: srv.name }),
      positiveText: t('adminNodes.copy'),
      negativeText: t('common.cancel'),
      onPositiveClick: async () => {
        await nodesTransport.copyServer(srv.id)
        message.success(t('adminNodes.copied', { name: srv.name }))
        void load()
      },
    })
  }

  // 排序弹窗:上下移调整顺序,保存时按最终顺序写 0..n
  const sortModal = ref(false)
  const sortItems = ref<{ id: number; name: string }[]>([])
  const sortSaving = ref(false)

  function openSortModal() {
    sortItems.value = [...servers.value]
      .sort((a, b) => a.sort - b.sort || a.id - b.id)
      .map((s) => ({ id: s.id, name: s.name }))
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
      await nodesTransport.sortServers({
        items: sortItems.value.map((s, i) => ({ id: s.id, sort: i })),
      })
      message.success(t('adminNodes.sortSaved'))
      sortModal.value = false
      await load()
    } finally {
      sortSaving.value = false
    }
  }

  function openGroupCreate() {
    editingGroup.value = null
    groupName.value = ''
    groupSort.value = 0
    groupModal.value = true
  }

  function openGroupEdit(g: AdminServerGroupItem) {
    editingGroup.value = g.id
    groupName.value = g.name
    groupSort.value = g.sort
    groupModal.value = true
  }

  async function saveGroup() {
    if (!groupName.value.trim()) {
      message.warning(t('adminNodes.enterGroupName'))
      return
    }
    groupSaving.value = true
    try {
      if (editingGroup.value === null) {
        await nodesTransport.createServerGroup({ name: groupName.value, sort: groupSort.value })
        message.success(t('adminNodes.groupCreated'))
      } else {
        await nodesTransport.updateServerGroup(editingGroup.value, {
          name: groupName.value,
          sort: groupSort.value,
        })
        message.success(t('adminNodes.groupUpdated'))
      }
      groupModal.value = false
      void load()
    } finally {
      groupSaving.value = false
    }
  }

  function removeGroup(g: AdminServerGroupItem) {
    dialog.warning({
      title: t('adminNodes.deleteGroupTitle'),
      content: t('adminNodes.deleteGroupConfirm', { name: g.name }),
      positiveText: t('adminNodes.delete'),
      negativeText: t('common.cancel'),
      onPositiveClick: async () => {
        await nodesTransport.deleteServerGroup(g.id)
        message.success(t('adminNodes.deleted'))
        void load()
      },
    })
  }

  onMounted(() => void load())

  return {
    // 状态
    loading,
    servers,
    groups,
    nodeModal,
    editingNode,
    nodeSaving,
    nodeForm,
    groupModal,
    editingGroup,
    groupSaving,
    groupName,
    groupSort,
    selectedIds,
    batchSaving,
    batchModal,
    batchStatus,
    batchGroup,
    batchRate,
    sortModal,
    sortItems,
    sortSaving,
    // 计算/工具
    statusText,
    statusType,
    groupNameOf,
    allChecked,
    // 动作
    load,
    openNodeCreate,
    openNodeEdit,
    saveNode,
    removeNode,
    resetNodeKey,
    isSelected,
    toggleRow,
    toggleAll,
    clearSelection,
    batchDelete,
    batchShow,
    openBatchUpdate,
    saveBatchUpdate,
    copyNode,
    openSortModal,
    moveSortItem,
    saveSort,
    openGroupCreate,
    openGroupEdit,
    saveGroup,
    removeGroup,
    // 部署
    installModal,
    installingNode,
    openInstallDialog,
  }
}
