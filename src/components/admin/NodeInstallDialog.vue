<script setup lang="ts">
/**
 * 节点一键部署与脚本配置弹窗(F17)。纯展示:设置读写与保存动作在 useAdminNodes,
 * 经 props/settings + emit/save 接入;组件不再触碰 API(data-layer §2 组件纯展示)。
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage } from 'naive-ui'
import { resolveApiBase } from '@/utils/http'
import type { AdminServerItem, NodeSettings } from '@/types/api'
import AppIcon from '@/components/ui/AppIcon.vue'
import { NODE_INSTALL_DEFAULTS } from '@/composables/useAdminNodes'

const props = defineProps<{
  show: boolean
  node: AdminServerItem | null
  /** 部署脚本设置(useAdminNodes 提供,含默认值兜底) */
  settings: NodeSettings
  saving?: boolean
}>()

const emit = defineEmits<{
  'update:show': [boolean]
  save: [NodeSettings]
}>()

const { t } = useI18n()
const message = useMessage()

const scriptUrl = ref(props.settings.install_script_url)
const commandTemplate = ref(props.settings.install_command_template)
const showConfig = ref(false)
const activeTab = ref<'agent' | 'v2bx'>('agent')

// 外部设置落地(打开时加载/保存成功回写)同步到编辑区;保存成功(值与编辑一致)时收起配置面板
watch(
  () => props.settings,
  (s) => {
    if (
      showConfig.value &&
      s.install_script_url === scriptUrl.value &&
      s.install_command_template === commandTemplate.value
    ) {
      showConfig.value = false
    }
    scriptUrl.value = s.install_script_url
    commandTemplate.value = s.install_command_template
  },
  { immediate: true },
)

// 安装命令运行在节点机器上,端点必须是绝对地址;基址复用 http 层 resolveApiBase
// (含运行时 apiBase 覆盖优先级,设置页改地址后命令随之变化),相对形式以面板源补全
const apiBase = computed(() => {
  const base = resolveApiBase()
  return base.startsWith('http') ? base : `${window.location.origin}${base}`
})

// 计算替换占位符后的部署命令
const renderedCommand = computed(() => {
  if (!props.node) return ''
  const tpl = commandTemplate.value || NODE_INSTALL_DEFAULTS.install_command_template
  const url = scriptUrl.value || NODE_INSTALL_DEFAULTS.install_script_url
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

// V2bX 对接参数与配置生成
const v2bxApiHost = computed(() => {
  try {
    const u = new URL(apiBase.value)
    return `${u.protocol}//${u.host}`
  } catch {
    return apiBase.value.replace(/\/api\/v1\/?$/, '')
  }
})

const v2bxNodeType = computed(() => {
  if (!props.node) return 'v2ray'
  const t = props.node.type.toLowerCase()
  if (t === 'vless' || t === 'vmess') return 'v2ray'
  if (t === 'hysteria2') return 'hysteria'
  return t
})

const v2bxCore = computed(() => {
  if (!props.node) return 'xray'
  const t = props.node.type.toLowerCase()
  if (t === 'hysteria' || t === 'hysteria2') return 'hysteria2'
  return 'xray'
})

const v2bxConfigJson = computed(() => {
  if (!props.node) return ''
  const cfg = {
    Log: {
      Level: 'warning',
      Output: '',
    },
    Nodes: [
      {
        Core: v2bxCore.value,
        ApiHost: v2bxApiHost.value,
        ApiKey: props.node.node_key || '',
        NodeID: props.node.id,
        NodeType: v2bxNodeType.value,
      },
    ],
  }
  return JSON.stringify(cfg, null, 2)
})

const v2bxInstallCommand =
  'wget -N --no-check-certificate https://raw.githubusercontent.com/v2bx/v2bx/master/install.sh && bash install.sh'

async function copyText(text: string, successMsgKey: string = 'common.copied') {
  if (!text) return
  try {
    await navigator.clipboard.writeText(text)
    message.success(t(successMsgKey))
  } catch {
    message.error(t('adminNodes.copyFailed'))
  }
}

const copyV2bXConfig = () => copyText(v2bxConfigJson.value, 'common.copied')
const copyV2bXInstallCmd = () => copyText(v2bxInstallCommand, 'common.copied')
const copyCommand = () => copyText(renderedCommand.value, 'common.copied')

function resetDefaults() {
  scriptUrl.value = NODE_INSTALL_DEFAULTS.install_script_url
  commandTemplate.value = NODE_INSTALL_DEFAULTS.install_command_template
}

function saveSettings() {
  emit('save', {
    install_script_url: scriptUrl.value,
    install_command_template: commandTemplate.value,
  })
}
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
          <template v-if="node.node_key">{{ node.node_key.slice(0, 8) }}****</template>
          <template v-else>{{ t('adminNodes.nodeKeyNotGenerated') }}</template>
        </div>
      </div>

      <!-- 协议与部署守护模式切换 -->
      <div class="flex gap-4 border-b border-[var(--c-border)] pb-2 text-13">
        <button
          class="transition-colors pb-1"
          :class="
            activeTab === 'agent'
              ? 'font-600 text-[var(--c-primary)] border-b-2 border-[var(--c-primary)]'
              : 'text-[var(--c-text-sub)] hover:text-[var(--c-text)]'
          "
          @click="activeTab = 'agent'"
        >
          {{ t('adminNodes.nodeAgentTab') }}
        </button>
        <button
          class="transition-colors pb-1 flex items-center gap-1.5"
          :class="
            activeTab === 'v2bx'
              ? 'font-600 text-[var(--c-primary)] border-b-2 border-[var(--c-primary)]'
              : 'text-[var(--c-text-sub)] hover:text-[var(--c-text)]'
          "
          @click="activeTab = 'v2bx'"
        >
          <span>{{ t('adminNodes.v2bxTab') }}</span>
          <span
            class="rounded bg-[var(--c-primary)]/10 px-1 py-0.2 text-10 text-[var(--c-primary)]"
          >
            UniProxy
          </span>
        </button>
      </div>

      <!-- Tab 1: 原生 Node Agent -->
      <template v-if="activeTab === 'agent'">
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
            class="relative rounded-[var(--r-card)] border border-[var(--c-border)] bg-slate-900 dark:bg-slate-950 p-3.5 text-slate-100 font-mono text-13 leading-relaxed break-all select-all shadow-inner"
          >
            {{ renderedCommand }}
          </div>

          <div class="mt-2.5 flex items-center justify-between text-12 text-[var(--c-text-sub)]">
            <span>{{ t('adminNodes.installOsHint') }}</span>
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
              {{ t('adminNodes.templatePlaceholders') }}:
              <code>&#123;&#123;script_url&#125;&#125;</code>,
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
      </template>

      <!-- Tab 2: V2bX 节点守护对接 -->
      <template v-else-if="activeTab === 'v2bx'">
        <!-- 一键安装 V2bX 命令 -->
        <div>
          <div class="mb-1.5 flex items-center justify-between">
            <label class="text-13 font-500 text-[var(--c-text)]">
              {{ t('adminNodes.v2bxInstallCmdLabel') }}
            </label>
            <button
              class="text-12 text-[var(--c-primary)] hover:underline flex items-center gap-1"
              @click="copyV2bXInstallCmd"
            >
              <AppIcon name="copy" :size="13" /> {{ t('adminNodes.copyCommand') }}
            </button>
          </div>
          <div
            class="rounded-[var(--r-card)] border border-[var(--c-border)] bg-slate-900 dark:bg-slate-950 p-2.5 text-slate-100 font-mono text-12 leading-relaxed break-all select-all shadow-inner"
          >
            {{ v2bxInstallCommand }}
          </div>
        </div>

        <!-- 节点配置文件 (/etc/V2bX/config.json) -->
        <div>
          <div class="mb-1.5 flex items-center justify-between">
            <label class="text-13 font-500 text-[var(--c-text)]">
              {{ t('adminNodes.v2bxConfigLabel') }}
            </label>
            <button class="btn-primary h-7 px-3 text-12" @click="copyV2bXConfig">
              <AppIcon name="copy" :size="13" /> {{ t('adminNodes.copyConfig') }}
            </button>
          </div>
          <div
            class="max-h-[220px] overflow-y-auto rounded-[var(--r-card)] border border-[var(--c-border)] bg-slate-900 dark:bg-slate-950 p-3 text-slate-100 font-mono text-12 leading-relaxed whitespace-pre select-all shadow-inner"
          >
            {{ v2bxConfigJson }}
          </div>
          <p class="mt-2 text-11 text-[var(--c-text-sub)]">
            {{ t('adminNodes.v2bxConfigPathHint') }}
          </p>
        </div>
      </template>
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
