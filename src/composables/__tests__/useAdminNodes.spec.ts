import { beforeEach, describe, expect, it, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAdminNodes } from '../useAdminNodes'
import { nodesTransport } from '@/stores/admin/nodes'
import type { AdminServerGroupItem, AdminServerItem } from '@/types/api'

vi.mock('@/stores/admin/nodes', () => ({
  nodesTransport: {
    servers: vi.fn(),
    createServer: vi.fn(),
    updateServer: vi.fn(),
    deleteServer: vi.fn(),
    resetServerNodeKey: vi.fn(),
    batchServers: vi.fn(),
    copyServer: vi.fn(),
    sortServers: vi.fn(),
    serverGroups: vi.fn(),
    createServerGroup: vi.fn(),
    updateServerGroup: vi.fn(),
    deleteServerGroup: vi.fn(),
  },
}))

const ui = vi.hoisted(() => ({
  success: vi.fn(),
  warning: vi.fn(),
  dialogWarning: vi.fn(),
  dialogInfo: vi.fn(),
}))

vi.mock('naive-ui', () => ({
  useMessage: () => ({ success: ui.success, warning: ui.warning }),
  useDialog: () => ({ warning: ui.dialogWarning, info: ui.dialogInfo }),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (k: string) => k, te: () => false }),
}))

function server(over: Partial<AdminServerItem> = {}): AdminServerItem {
  return {
    id: 1,
    group_id: 1,
    name: '香港 01',
    type: 'shadowsocks',
    host: 'hk1.example.com',
    port: 443,
    config: '{}',
    rate: 1,
    tags: [],
    status: 1,
    is_show: true,
    sort: 1,
    node_key: 'k',
    is_online: true,
    last_heartbeat_at: null,
    system: null,
    ...over,
  }
}

function group(over: Partial<AdminServerGroupItem> = {}): AdminServerGroupItem {
  return { id: 1, name: '香港', sort: 1, ...over }
}

type DialogOpts = { onPositiveClick?: () => unknown }
let captured: DialogOpts[]

function confirmLastDialog() {
  const opts = captured[captured.length - 1]
  return opts.onPositiveClick?.()
}

// claim:节点域组合(双取/校验/批量/排序映射)走传输核;groupSaving 复位是组合层回归点。
// 最怕的失败:分组不同步致表单分组错位、排序映射错位、saving 卡死。
describe('useAdminNodes', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    captured = []
    ui.dialogWarning.mockImplementation((opts: DialogOpts) => {
      captured.push(opts)
    })
    vi.mocked(nodesTransport.servers).mockResolvedValue({ list: [server()] })
    vi.mocked(nodesTransport.serverGroups).mockResolvedValue({ list: [group()] })
  })

  it('load 双取写入 servers/groups', async () => {
    const n = useAdminNodes()
    await n.load()
    expect(nodesTransport.servers).toHaveBeenCalledTimes(1)
    expect(nodesTransport.serverGroups).toHaveBeenCalledTimes(1)
    expect(n.servers.value).toHaveLength(1)
    expect(n.groups.value).toEqual([group()])
    expect(n.loading.value).toBe(false)
  })

  it('saveNode 无分组:拦截,无创建调用', async () => {
    const n = useAdminNodes()
    await n.load()
    n.openNodeCreate()
    // 无分组时 group_id 回落 0
    vi.mocked(nodesTransport.serverGroups).mockResolvedValue({ list: [] })
    await n.load()
    n.openNodeCreate()
    await n.saveNode()
    expect(ui.warning).toHaveBeenCalledWith('adminNodes.selectGroup')
    expect(nodesTransport.createServer).not.toHaveBeenCalled()
  })

  it('saveNode 缺必填:拦截,无调用', async () => {
    const n = useAdminNodes()
    await n.load()
    n.openNodeCreate()
    await n.saveNode()
    expect(ui.warning).toHaveBeenCalledWith('adminNodes.requiredFields')
    expect(nodesTransport.createServer).not.toHaveBeenCalled()
    expect(nodesTransport.updateServer).not.toHaveBeenCalled()
  })

  it('saveNode 新建:表单透传,关弹窗并重取', async () => {
    vi.mocked(nodesTransport.createServer).mockResolvedValue(server())
    const n = useAdminNodes()
    await n.load()
    n.openNodeCreate()
    n.nodeForm.name = 'n2'
    n.nodeForm.host = 'h2'
    n.nodeForm.config = '{"a":1}'
    await n.saveNode()
    expect(nodesTransport.createServer).toHaveBeenCalledWith(
      expect.objectContaining({ group_id: 1, name: 'n2', host: 'h2' }),
    )
    expect(n.nodeModal.value).toBe(false)
    await vi.waitFor(() => {
      expect(nodesTransport.servers).toHaveBeenCalledTimes(2)
    })
  })

  it('batchDelete 确认:批量删除+汇总+清选中+重取', async () => {
    vi.mocked(nodesTransport.batchServers).mockResolvedValue({ success: 1, failed: [] })
    const n = useAdminNodes()
    await n.load()
    n.toggleRow(1, true)
    n.batchDelete()
    await confirmLastDialog()
    expect(nodesTransport.batchServers).toHaveBeenCalledWith({ action: 'delete', ids: [1] })
    expect(ui.success).toHaveBeenCalledWith('adminNodes.batchDone: 1')
    expect(n.selectedIds.value).toEqual([])
    await vi.waitFor(() => {
      expect(nodesTransport.servers).toHaveBeenCalledTimes(2)
    })
  })

  it('saveBatchUpdate 空字段:拦截,无调用', async () => {
    const n = useAdminNodes()
    await n.load()
    n.toggleRow(1, true)
    n.openBatchUpdate()
    await n.saveBatchUpdate()
    expect(ui.warning).toHaveBeenCalledWith('adminNodes.batchUpdateEmpty')
    expect(nodesTransport.batchServers).not.toHaveBeenCalled()
  })

  it('saveBatchUpdate 组装公共字段', async () => {
    vi.mocked(nodesTransport.batchServers).mockResolvedValue({ success: 1, failed: [] })
    const n = useAdminNodes()
    await n.load()
    n.toggleRow(1, true)
    n.openBatchUpdate()
    n.batchStatus.value = 2
    n.batchRate.value = 1.5
    await n.saveBatchUpdate()
    expect(nodesTransport.batchServers).toHaveBeenCalledWith({
      action: 'update',
      ids: [1],
      status: 2,
      rate: 1.5,
    })
    expect(n.batchModal.value).toBe(false)
  })

  it('saveSort 按展示顺序写 0..n', async () => {
    vi.mocked(nodesTransport.sortServers).mockResolvedValue(null)
    vi.mocked(nodesTransport.servers).mockResolvedValue({
      list: [server({ id: 1, sort: 5 }), server({ id: 2, sort: 3 })],
    })
    const n = useAdminNodes()
    await n.load()
    n.openSortModal()
    // sort=3 的 id=2 排前
    expect(n.sortItems.value.map((s) => s.id)).toEqual([2, 1])
    n.moveSortItem(0, 1)
    await n.saveSort()
    expect(nodesTransport.sortServers).toHaveBeenCalledWith({
      items: [
        { id: 1, sort: 0 },
        { id: 2, sort: 1 },
      ],
    })
    expect(n.sortModal.value).toBe(false)
  })

  it('saveGroup 成功:groupSaving 复位(回归),关弹窗并重取', async () => {
    vi.mocked(nodesTransport.createServerGroup).mockResolvedValue(group({ id: 9 }))
    const n = useAdminNodes()
    await n.load()
    n.openGroupCreate()
    n.groupName.value = 'g9'
    await n.saveGroup()
    expect(nodesTransport.createServerGroup).toHaveBeenCalledWith({ name: 'g9', sort: 0 })
    expect(n.groupSaving.value).toBe(false)
    expect(n.groupModal.value).toBe(false)
  })

  it('取数失败:loading 复位', async () => {
    vi.mocked(nodesTransport.servers).mockRejectedValueOnce(new Error('boom'))
    const n = useAdminNodes()
    await expect(n.load()).rejects.toThrow('boom')
    expect(n.loading.value).toBe(false)
  })

  it('copyNode 确认后重取', async () => {
    vi.mocked(nodesTransport.copyServer).mockResolvedValue(server({ id: 3 }))
    const n = useAdminNodes()
    await n.load()
    n.copyNode(server())
    await confirmLastDialog()
    expect(nodesTransport.copyServer).toHaveBeenCalledWith(1)
    await vi.waitFor(() => {
      expect(nodesTransport.servers).toHaveBeenCalledTimes(2)
    })
  })
})
