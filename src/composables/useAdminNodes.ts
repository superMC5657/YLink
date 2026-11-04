import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { nodesTransport } from '@/stores/admin/nodes'
import { systemTransport } from '@/stores/admin/system'
import type {
  AdminBatchServerResp,
  AdminServerGroupItem,
  AdminServerItem,
  AdminServerReq,
  NodeSettings,
  ServerStatus,
} from '@/types/api'
import { useMessage, useDialog } from 'naive-ui'
import { statusBadge, statusLabel } from '@/utils/statusMeta'
import { useAdminList } from './useAdminList'
import { useSortDialog } from './useSortDialog'

/**
 * 节点管理数据域(F09):节点/分组 CRUD、多选与批量操作、复制/排序。
 * 纯逻辑与状态;弹窗呈现拆在 components/admin/*Dialog.vue,表格留在视图。
 * 列表骨架(取数/多选/批量汇总)归 useAdminList,本域只留语义(C3 收敛)。
 */

/** 部署脚本默认值:与 0009 迁移种子同形(节点域唯一来源,弹窗/composable 共用) */
export const NODE_INSTALL_DEFAULTS: NodeSettings = {
  install_script_url:
    'https://raw.githubusercontent.com/superMC5657/YLink/main/deploy/install-agent.sh',
  install_command_template:
    'curl -fsSL {{script_url}} | bash -s -- --endpoint {{endpoint}} --key {{node_key}}',
}

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
    reload,
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

  // ---- 部署脚本设置(F17):读写收口在本域,NodeInstallDialog 纯展示 ----
  const nodeSettings = ref<NodeSettings>({ ...NODE_INSTALL_DEFAULTS })
  const nodeSettingsLoading = ref(false)
  const nodeSettingsSaving = ref(false)

  async function loadNodeSettings() {
    nodeSettingsLoading.value = true
    try {
      const res = await systemTransport.settings()
      const item = res.list.find((x) => x.key === 'node')
      if (item?.value) {
        try {
          const parsed: NodeSettings = JSON.parse(item.value)
          nodeSettings.value = {
            install_script_url:
              parsed.install_script_url || NODE_INSTALL_DEFAULTS.install_script_url,
            install_command_template:
              parsed.install_command_template || NODE_INSTALL_DEFAULTS.install_command_template,
          }
        } catch {
          // 坏 JSON 兜底默认
        }
      }
    } catch {
      // 加载失败兜底默认(http 层已 toast)
    } finally {
      nodeSettingsLoading.value = false
    }
  }

  async function saveNodeSettings(s: NodeSettings): Promise<boolean> {
    nodeSettingsSaving.value = true
    try {
      await systemTransport.saveSetting({
        key: 'node',
        value: JSON.stringify({
          install_script_url:
            s.install_script_url.trim() || NODE_INSTALL_DEFAULTS.install_script_url,
          install_command_template:
            s.install_command_template.trim() || NODE_INSTALL_DEFAULTS.install_command_template,
        }),
      })
      nodeSettings.value = s
      message.success(t('adminNodes.settingsSaved'))
      return true
    } catch {
      return false
    } finally {
      nodeSettingsSaving.value = false
    }
  }

  function openInstallDialog(node: AdminServerItem) {
    installingNode.value = node
    installModal.value = true
    void loadNodeSettings()
  }

  function statusText(status: ServerStatus): string {
    // 状态元深模块同源,输出与收敛前逐字一致
    return statusLabel('server', status)
  }

  function statusType(status: ServerStatus): 'success' | 'warning' | 'neutral' {
    // 徽标直读表,不从色值字符串反推
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
      void reload()
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
        void reload()
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
        void reload()
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
          void reload()
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
        void reload()
      },
    })
  }

  // 排序弹窗:上下移调整顺序,保存时按最终顺序写 0..n(数据域归 useSortDialog,呈现归 SortDialog)
  const { sortModal, sortItems, sortSaving, openSortModal, moveSortItem, saveSort } =
    useSortDialog<AdminServerItem>({
      source: () => servers.value,
      nameOf: (s) => s.name,
      save: (items) => nodesTransport.sortServers({ items }),
      savedKey: 'adminNodes.sortSaved',
      reload: load,
    })

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
      void reload()
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
        void reload()
      },
    })
  }

  onMounted(() => void reload())

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
    reload,
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
    // 部署(脚本设置读写收口本域,弹窗纯展示)
    installModal,
    installingNode,
    openInstallDialog,
    nodeSettings,
    nodeSettingsLoading,
    nodeSettingsSaving,
    saveNodeSettings,
  }
}
