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

async function mountDialog() {
  setActivePinia(createPinia())
  await useZhCN()
  return mount(NodeInstallDialog, {
    props: { show: true, node, settings: { ...NODE_INSTALL_DEFAULTS } },
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
})
