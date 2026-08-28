<script setup lang="ts">
/**
 * 管理后台 · 系统更新(F20 子集):版本检查 + 变更日志展示。
 * 当前版本来自后端 app.version(部署注入);配置 update.manifest_url 时
 * 远端拉取最新版本与变更日志(3s 超时 + 10min 服务端缓存)。自动执行升级不立项。
 * 数据:GET /admin/version
 */
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiAdmin } from '@/api/admin'
import type { AdminVersionResp } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import { useMessage } from 'naive-ui'

const { t } = useI18n()
const message = useMessage()

const loading = ref(false)
const info = ref<AdminVersionResp | null>(null)

async function load(silent = false) {
  if (!silent) loading.value = true
  try {
    info.value = await apiAdmin.version()
  } finally {
    loading.value = false
  }
}

async function onCheck() {
  await load(true)
  if (info.value?.has_update) {
    message.warning(t('adminVersion.updateAvailable', { version: info.value.latest ?? '' }))
  } else if (info.value?.latest) {
    message.success(t('adminVersion.upToDate'))
  } else {
    message.info(t('adminVersion.noManifest'))
  }
}

onMounted(() => void load())
</script>

<template>
  <div>
    <PageHeader :title="t('adminVersion.title')" :subtitle="t('adminVersion.subtitle')">
      <template #actions>
        <button class="btn-primary h-9 px-4 text-14" :disabled="loading" @click="onCheck">
          <AppIcon name="refresh" :size="15" /> {{ t('adminVersion.check') }}
        </button>
      </template>
    </PageHeader>

    <div class="grid gap-4 lg:grid-cols-2">
      <!-- 当前版本 -->
      <div class="card-base p-5 md:p-6">
        <h3 class="mb-4 text-16 font-600 text-[var(--c-text)]">
          {{ t('adminVersion.current') }}
        </h3>
        <div class="flex items-center justify-between py-2 text-14">
          <span class="text-[var(--c-text-sub)]">{{ t('adminVersion.currentVersion') }}</span>
          <span class="num text-16 font-600 text-[var(--c-text)]">
            {{ info?.version || '--' }}
          </span>
        </div>
        <div class="flex items-center justify-between py-2 text-14">
          <span class="text-[var(--c-text-sub)]">{{ t('adminVersion.latestVersion') }}</span>
          <span class="num text-[var(--c-text)]">{{ info?.latest || '--' }}</span>
        </div>
        <div class="flex items-center justify-between py-2 text-14">
          <span class="text-[var(--c-text-sub)]">{{ t('adminVersion.status') }}</span>
          <StatusBadge
            v-if="info"
            :type="
              info.has_update === null || info.has_update === undefined
                ? 'neutral'
                : info.has_update
                  ? 'warning'
                  : 'success'
            "
          >
            {{
              info.has_update === null || info.has_update === undefined
                ? t('adminVersion.unknown')
                : info.has_update
                  ? t('adminVersion.hasUpdate')
                  : t('adminVersion.isLatest')
            }}
          </StatusBadge>
        </div>
        <p class="mt-4 text-13 text-[var(--c-text-sub)]">{{ t('adminVersion.hint') }}</p>
      </div>

      <!-- 变更日志 -->
      <div class="card-base p-5 md:p-6">
        <h3 class="mb-4 text-16 font-600 text-[var(--c-text)]">
          {{ t('adminVersion.changelog') }}
        </h3>
        <div
          v-if="info?.notes"
          class="max-h-80 overflow-y-auto whitespace-pre-wrap rounded-xl p-4 text-14 leading-6 text-[var(--c-text)]"
          style="background: var(--c-bg-hover)"
        >
          {{ info.notes }}
        </div>
        <EmptyState v-else :text="t('adminVersion.noNotes')" icon="book" />
      </div>
    </div>
  </div>
</template>
