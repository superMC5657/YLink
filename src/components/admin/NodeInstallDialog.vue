<script setup lang="ts">
/**
 * 节点一键部署与脚本配置弹窗(F17)。
 * 自动渲染当前节点的安装命令，支持一键复制；
 * 允许在中台动态更换指定部署脚本 URL 与命令模板，保存后全站生效。
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage } from 'naive-ui'
import { apiAdmin } from '@/api/admin'
import type { AdminServerItem, NodeSettings } from '@/types/api'
import AppIcon from '@/components/ui/AppIcon.vue'

const props = defineProps<{
  show: boolean
  node: AdminServerItem | null
}>()

const emit = defineEmits<{
  'update:show': [boolean]
}>()

const { t } = useI18n()
const message = useMessage()

const DEFAULT_SCRIPT_URL =
  'https://raw.githubusercontent.com/superMC5657/YLink/main/deploy/install-agent.sh'
const DEFAULT_TEMPLATE =
  'curl -fsSL {{script_url}} | bash -s -- --endpoint {{endpoint}} --key {{node_key}}'

const scriptUrl = ref(DEFAULT_SCRIPT_URL)
const commandTemplate = ref(DEFAULT_TEMPLATE)
const showConfig = ref(false)
const saving = ref(false)
const loading = ref(false)

// 面板接口 base 地址
const apiBase = computed(() => {
  const custom = import.meta.env.VITE_API_BASE_URL
  if (custom && custom.startsWith('http')) return custom
  return `${window.location.origin}/api/v1`
})

// 计算替换占位符后的部署命令
const renderedCommand = computed(() => {
  if (!props.node) return ''
  const tpl = commandTemplate.value || DEFAULT_TEMPLATE
  const url = scriptUrl.value || DEFAULT_SCRIPT_URL
  return tpl
    .split('{{script_url}}')
    .join(url)
    .split('{{endpoint}}')
    .join(apiBase.value)
    .split('{{node_key}}')
    .join(props.node.node_key || '')
    .split('{{node_id}}')
    .join(String(props.node.id))
})

async function loadSettings() {
  loading.value = true
  try {
    const res = await apiAdmin.settings()
    const nodeItem = res.list.find((x) => x.key === 'node')
    if (nodeItem && nodeItem.value) {
      try {
        const parsed: NodeSettings = JSON.parse(nodeItem.value)
        if (parsed.install_script_url) scriptUrl.value = parsed.install_script_url
        if (parsed.install_command_template) commandTemplate.value = parsed.install_command_template
      } catch {
        // 使用默认
      }
    }
  } catch {
    // 忽略加载失败，兜底用默认
  } finally {
    loading.value = false
  }
}

async function saveSettings() {
  saving.value = true
  try {
    const payload: NodeSettings = {
      install_script_url: scriptUrl.value.trim() || DEFAULT_SCRIPT_URL,
      install_command_template: commandTemplate.value.trim() || DEFAULT_TEMPLATE,
    }
    await apiAdmin.saveSetting({
      key: 'node',
      value: JSON.stringify(payload),
    })
    message.success(t('adminNodes.settingsSaved'))
    showConfig.value = false
  } finally {
    saving.value = false
  }
}

function resetDefaults() {
  scriptUrl.value = DEFAULT_SCRIPT_URL
  commandTemplate.value = DEFAULT_TEMPLATE
}

async function copyCommand() {
  if (!renderedCommand.value) return
  try {
    await navigator.clipboard.writeText(renderedCommand.value)
    message.success(t('common.copied'))
  } catch {
    message.error('复制失败，请手动选择复制')
  }
}

watch(
  () => props.show,
  (visible) => {
    if (visible) {
      void loadSettings()
    }
  },
)
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    :title="t('adminNodes.installModalTitle')"
    style="width: 620px; max-width: 95vw"
    @update:show="emit('update:show', $event)"
  >
    <div v-if="node" class="space-y-4">
      <!-- 节点基础信息 -->
      <div
        class="flex flex-wrap items-center justify-between gap-2 rounded-[var(--r-control)] bg-[var(--c-bg-card-sub)] p-3 text-14"
      >
        <div>
          <span class="font-600 text-[var(--c-text)]">{{ node.name }}</span>
          <span class="ml-2 text-[var(--c-text-sub)]">({{ node.type.toUpperCase() }})</span>
        </div>
        <div class="text-12 text-[var(--c-text-sub)]">
          ID: {{ node.id }} · NodeKey:
          {{ node.node_key ? `${node.node_key.slice(0, 8)}****` : '未生成' }}
        </div>
      </div>

      <!-- 命令预览与复制 -->
      <div>
        <div class="mb-2 flex items-center justify-between">
          <label class="text-14 font-500 text-[var(--c-text)]">
            {{ t('adminNodes.installCommandLabel') }}
          </label>
          <button
            class="text-13 text-[var(--c-primary)] hover:underline flex items-center gap-1"
            @click="showConfig = !showConfig"
          >
            <AppIcon name="settings" :size="14" />
            {{ showConfig ? t('adminNodes.hideConfig') : t('adminNodes.changeScript') }}
          </button>
        </div>

        <div
          class="relative rounded-[var(--r-card)] border border-[var(--c-border)] bg-[#1e1e2e] p-3.5 text-[#f8f8f2] font-mono text-13 leading-relaxed break-all select-all shadow-inner"
        >
          {{ renderedCommand }}
        </div>

        <div class="mt-2.5 flex items-center justify-between text-12 text-[var(--c-text-sub)]">
          <span>💡 支持 Ubuntu 20+/Debian 11+/CentOS 8+ 等主流 Linux 系统</span>
          <button class="btn-primary h-8 px-4 text-13" @click="copyCommand">
            <AppIcon name="copy" :size="14" /> {{ t('adminNodes.copyCommand') }}
          </button>
        </div>
      </div>

      <!-- 自定义部署脚本设置面板(折叠展开) -->
      <div
        v-if="showConfig"
        class="rounded-[var(--r-card)] border border-[var(--c-primary)]/30 bg-[var(--c-bg-card-sub)] p-4 space-y-3 transition-all"
      >
        <div class="flex items-center justify-between border-b border-[var(--c-border)] pb-2">
          <span class="text-14 font-600 text-[var(--c-text)]">
            {{ t('adminNodes.customScriptTitle') }}
          </span>
          <button
            class="text-12 text-[var(--c-text-sub)] hover:text-[var(--c-danger)]"
            @click="resetDefaults"
          >
            {{ t('adminNodes.resetDefault') }}
          </button>
        </div>

        <div>
          <label class="mb-1 block text-12 text-[var(--c-text-sub)]">
            {{ t('adminNodes.scriptUrlLabel') }}
          </label>
          <n-input
            v-model:value="scriptUrl"
            size="small"
            placeholder="https://.../install-agent.sh"
          />
        </div>

        <div>
          <label class="mb-1 block text-12 text-[var(--c-text-sub)]">
            {{ t('adminNodes.templateLabel') }}
          </label>
          <n-input
            v-model:value="commandTemplate"
            type="textarea"
            :rows="2"
            size="small"
            placeholder="curl -fsSL {{script_url}} | bash -s -- --endpoint {{endpoint}} --key {{node_key}}"
          />
          <span class="mt-1 block text-11 text-[var(--c-text-sub)]">
            可用占位符: <code>&#123;&#123;script_url&#125;&#125;</code>,
            <code>&#123;&#123;endpoint&#125;&#125;</code>,
            <code>&#123;&#123;node_key&#125;&#125;</code>,
            <code>&#123;&#123;node_id&#125;&#125;</code>
          </span>
        </div>

        <div class="flex justify-end gap-2 pt-1">
          <button class="btn-soft-neutral h-7 px-3 text-12" @click="showConfig = false">
            {{ t('common.cancel') }}
          </button>
          <button class="btn-primary h-7 px-3 text-12" :disabled="saving" @click="saveSettings">
            {{ t('adminNodes.saveScriptConfig') }}
          </button>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end">
        <button class="btn-soft-neutral h-9 px-4 text-14" @click="emit('update:show', false)">
          {{ t('common.close') }}
        </button>
      </div>
    </template>
  </n-modal>
</template>
