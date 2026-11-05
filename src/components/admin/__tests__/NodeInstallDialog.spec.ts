import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import NodeInstallDialog from '@/components/admin/NodeInstallDialog.vue'
import AppIcon from '@/components/ui/AppIcon.vue'
import { i18n } from '@/i18n'
import { useZhCN } from '../../business/__tests__/helpers'
import { NODE_INSTALL_DEFAULTS } from '@/composables/useAdminNodes'
import type { AdminServerItem } from '@/types/api'

// 缝测试:安装命令端点必须复用 http 层 resolveApiBase(含运行时 apiBase 覆盖优先级)。
// 最怕的失败=组件自算基址忽略持久化覆盖——设置页改过后端地址后,弹窗命令指向旧端点。
vi.mock('naive-ui', async (importOriginal) => {
  const actual = await importOriginal<typeof import('naive-ui')>()
  return {
    ...actual,
    useMessage: () => ({ success: vi.fn(), error: vi.fn(), warning: vi.fn() }),
  }
})

const node = {
  id: 7,
  node_key: 'a1b2c3d4e5f6a7b8',
  name: 'HK-01',
  type: 'shadowsocks',
} as AdminServerItem

async function mountDialog(customNode?: AdminServerItem) {
  setActivePinia(createPinia())
  await useZhCN()
  return mount(NodeInstallDialog, {
    props: { show: true, node: customNode ?? node, settings: { ...NODE_INSTALL_DEFAULTS } },
    global: {
      plugins: [i18n],
      components: { AppIcon },
      stubs: { 'n-modal': { template: '<div class="modal-stub"><slot /></div>' } },
    },
  })
}

function commandText(w: Awaited<ReturnType<typeof mountDialog>>): string {
  return w.get('.modal-stub .font-mono').text()
}

describe('NodeInstallDialog', () => {
  it('运行时 apiBase 覆盖优先:安装命令端点随之变化', async () => {
    localStorage.setItem('app:apiBase', 'https://runtime.example.com/api/v1')
    const w = await mountDialog()
    expect(commandText(w)).toContain('--endpoint https://runtime.example.com/api/v1')
    expect(commandText(w)).toContain('--key a1b2c3d4e5f6a7b8')
    localStorage.removeItem('app:apiBase')
  })

  it('相对基址以面板源补全(安装命令需绝对地址)', async () => {
    localStorage.setItem('app:apiBase', '/custom/api')
    const w = await mountDialog()
    expect(commandText(w)).toContain(`--endpoint ${window.location.origin}/custom/api`)
    localStorage.removeItem('app:apiBase')
  })

  it('无覆盖时回落同源 /api/v1', async () => {
    localStorage.removeItem('app:apiBase')
    const w = await mountDialog()
    expect(commandText(w)).toContain(`--endpoint ${window.location.origin}/api/v1`)
  })

  it('切换到 V2bX Tab 展示 /etc/V2bX/config.json 配置', async () => {
    localStorage.setItem('app:apiBase', 'https://ylink.example.com/api/v1')
    const w = await mountDialog()
    const tabs = w.findAll('.modal-stub button')
    const v2bxTab = tabs.find((b) => b.text().includes('V2bX'))
    expect(v2bxTab).toBeDefined()
    await v2bxTab!.trigger('click')

    const codeBlocks = w.findAll('.modal-stub .font-mono')
    const configBlock = codeBlocks.find((b) => b.text().includes('"Nodes"'))
    expect(configBlock).toBeDefined()
    const content = configBlock!.text()
    expect(content).toContain('"ApiHost": "https://ylink.example.com"')
    expect(content).toContain('"ApiKey": "a1b2c3d4e5f6a7b8"')
    expect(content).toContain('"NodeID": 7')
    expect(content).toContain('"NodeType": "shadowsocks"')
    expect(content).toContain('"Core": "xray"')
    localStorage.removeItem('app:apiBase')
  })

  it('V2bX 配置生成: hysteria2 节点 Core 为 hysteria2', async () => {
    const hysteriaNode = {
      id: 8,
      node_key: 'hysteria2-key',
      name: 'US-Hy2',
      type: 'hysteria2',
    } as AdminServerItem

    const w = await mountDialog(hysteriaNode)
    const tabs = w.findAll('.modal-stub button')
    const v2bxTab = tabs.find((b) => b.text().includes('V2bX'))
    expect(v2bxTab).toBeDefined()
    await v2bxTab!.trigger('click')

    const codeBlocks = w.findAll('.modal-stub .font-mono')
    const configBlock = codeBlocks.find((b) => b.text().includes('"Nodes"'))
    expect(configBlock).toBeDefined()
    const content = configBlock!.text()
    expect(content).toContain('"Core": "hysteria2"')
    expect(content).toContain('"NodeType": "hysteria"')
    expect(content).toContain('"ApiKey": "hysteria2-key"')
    expect(content).toContain('"NodeID": 8')
  })

  it('V2bX 配置生成: hysteria 节点 Core 亦为 hysteria2', async () => {
    const hysteriaNode = {
      id: 9,
      node_key: 'hysteria-key',
      name: 'US-Hy',
      type: 'hysteria',
    } as AdminServerItem

    const w = await mountDialog(hysteriaNode)
    const tabs = w.findAll('.modal-stub button')
    const v2bxTab = tabs.find((b) => b.text().includes('V2bX'))
    expect(v2bxTab).toBeDefined()
    await v2bxTab!.trigger('click')

    const codeBlocks = w.findAll('.modal-stub .font-mono')
    const configBlock = codeBlocks.find((b) => b.text().includes('"Nodes"'))
    expect(configBlock).toBeDefined()
    const content = configBlock!.text()
    expect(content).toContain('"Core": "hysteria2"')
    expect(content).toContain('"NodeType": "hysteria"')
  })

  it('点击复制按钮调用统一的 copyText 剪贴板写入', async () => {
    const writeTextSpy = vi.fn().mockResolvedValue(undefined)
    Object.assign(navigator, {
      clipboard: {
        writeText: writeTextSpy,
      },
    })

    const w = await mountDialog()
    const copyBtn = w.findAll('.modal-stub button').find((b) => b.text().includes('复制命令'))
    expect(copyBtn).toBeDefined()
    await copyBtn!.trigger('click')
    expect(writeTextSpy).toHaveBeenCalled()
  })
})
