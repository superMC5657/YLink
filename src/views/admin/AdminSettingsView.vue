<script setup lang="ts">
/**
 * 管理后台 · 站点设置:按 key 编辑配置项 JSON,保存后后端失效缓存立即生效。
 * 数据:GET/PUT /admin/settings(docs/api/README.md §16.1)
 */
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiAdmin } from '@/api/admin'
import type { AdminSettingsItem } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import { useMessage } from 'naive-ui'

const { t } = useI18n()
const message = useMessage()

const loading = ref(false)
const saving = ref(false)
const items = ref<AdminSettingsItem[]>([])
const activeKey = ref('site')
const draft = ref('')

const KEY_META: Record<string, { labelKey: string; descKey: string }> = {
  site: { labelKey: 'adminSettings.metaSite', descKey: 'adminSettings.metaSiteDesc' },
  payment: { labelKey: 'adminSettings.metaPayment', descKey: 'adminSettings.metaPaymentDesc' },
  invite: { labelKey: 'adminSettings.metaInvite', descKey: 'adminSettings.metaInviteDesc' },
  agent: { labelKey: 'adminSettings.metaAgent', descKey: 'adminSettings.metaAgentDesc' },
  order: { labelKey: 'adminSettings.metaOrder', descKey: 'adminSettings.metaOrderDesc' },
  templates: {
    labelKey: 'adminSettings.metaTemplates',
    descKey: 'adminSettings.metaTemplatesDesc',
  },
}

const activeItem = computed(() => items.value.find((x) => x.key === activeKey.value))
const activeMeta = computed(() => {
  const meta = KEY_META[activeKey.value]
  return {
    label: meta ? t(meta.labelKey) : activeKey.value,
    desc: meta ? t(meta.descKey) : '',
  }
})

async function load() {
  loading.value = true
  try {
    const res = await apiAdmin.settings()
    items.value = res.list
    if (!res.list.some((x) => x.key === activeKey.value)) {
      activeKey.value = res.list[0]?.key ?? ''
    }
    syncDraft()
  } finally {
    loading.value = false
  }
}

function syncDraft() {
  draft.value = activeItem.value ? formatJson(activeItem.value.value) : ''
}

function formatJson(raw: string): string {
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}

function select(key: string) {
  activeKey.value = key
  syncDraft()
}

async function save() {
  if (!activeKey.value) return
  let parsed: string
  try {
    parsed = JSON.stringify(JSON.parse(draft.value))
  } catch {
    message.error(t('adminSettings.invalidJson'))
    return
  }
  saving.value = true
  try {
    await apiAdmin.saveSetting({ key: activeKey.value, value: parsed })
    message.success(t('adminSettings.saved', { label: activeMeta.value.label }))
    void load()
  } finally {
    saving.value = false
  }
}

onMounted(() => void load())
</script>

<template>
  <div>
    <PageHeader :title="t('adminSettings.pageTitle')" :subtitle="t('adminSettings.subtitle')">
      <template #actions>
        <button class="btn-soft-neutral h-9 px-3 text-14" @click="load">
          <AppIcon name="refresh" :size="15" /> {{ t('common.refresh') }}
        </button>
      </template>
    </PageHeader>

    <div class="grid gap-4 lg:grid-cols-[240px_1fr]">
      <div class="card-base">
        <div class="flex flex-col gap-1 p-2">
          <button
            v-for="(meta, key) in KEY_META"
            :key="key"
            class="flex h-10 items-center justify-between rounded-lg px-3 text-14 transition-colors"
            :class="
              activeKey === key
                ? 'bg-[var(--c-primary-soft)] font-500 text-[var(--c-primary)]'
                : 'text-[var(--c-text)] hover:bg-[var(--c-bg-hover)]'
            "
            @click="select(key)"
          >
            <span>{{ t(meta.labelKey) }}</span>
            <span class="num-font text-12 text-[var(--c-text-sub)]">{{ key }}</span>
          </button>
        </div>
      </div>

      <div class="card-base">
        <n-spin :show="loading">
          <div class="mb-3">
            <div class="font-600 text-[var(--c-text)]">{{ activeMeta.label }}</div>
            <div class="mt-1 text-13 text-[var(--c-text-sub)]">{{ activeMeta.desc }}</div>
          </div>
          <n-input
            v-model:value="draft"
            type="textarea"
            :rows="16"
            class="font-mono"
            placeholder="{}"
            :disabled="!activeKey"
          />
          <div class="mt-4 flex justify-end gap-2">
            <button
              class="btn-primary h-9 px-4 text-14"
              :disabled="saving || !activeKey"
              @click="save"
            >
              {{ t('adminSettings.saveConfig') }}
            </button>
          </div>
        </n-spin>
      </div>
    </div>
  </div>
</template>
