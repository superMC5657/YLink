<script setup lang="ts">
/**
 * 节点状态:按分组展示节点名/类型/倍率/状态点,60s 静默刷新。
 * 数据:GET /servers(契约 §14);安全约束:不展示 host/port 等连接信息。
 */
import { computed, onBeforeUnmount, onMounted } from 'vue'
import { useServerStore } from '@/stores/server'
import { serverStatusMeta } from '@/utils/format'
import { useI18n } from 'vue-i18n'

const server = useServerStore()
const { t } = useI18n()

const totalNodes = computed(() => server.groups.reduce((acc, g) => acc + g.servers.length, 0))
const healthyNodes = computed(() =>
  server.groups.reduce((acc, g) => acc + g.servers.filter((s) => s.status === 1).length, 0),
)

const updatedText = computed(() => {
  if (!server.lastUpdated) return ''
  const d = new Date(server.lastUpdated)
  return `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}:${d.getSeconds().toString().padStart(2, '0')}`
})

onMounted(() => {
  void server.fetch()
  server.startPolling()
})

onBeforeUnmount(() => server.stopPolling())
</script>

<template>
  <div>
    <div class="mb-5 flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-20 font-600 text-[var(--c-text)]">{{ t('node.title') }}</h1>
        <p class="mt-1 text-14 text-[var(--c-text-sub)]">
          {{ t('node.group') }} {{ server.groups.length }} · {{ t('node.normal') }}
          {{ healthyNodes }}/{{ totalNodes }}
        </p>
      </div>
      <span class="text-14 text-[var(--c-text-sub)]">
        {{ t('node.updatedAt', { time: updatedText || '--' }) }}
      </span>
    </div>

    <n-spin :show="server.loading">
      <div class="space-y-5">
        <div
          v-for="group in server.groups"
          :key="group.group"
          class="card-base card-hoverable p-5 md:p-6"
        >
          <div class="mb-4 flex items-center gap-2">
            <span
              class="flex h-8 w-8 items-center justify-center rounded-full"
              style="background: var(--c-primary-soft); color: var(--c-primary-text)"
            >
              <AppIcon name="server" :size="17" />
            </span>
            <h3 class="text-16 font-600 text-[var(--c-text)]">{{ group.group }}</h3>
            <span class="ml-auto text-14 text-[var(--c-text-sub)]"
              >{{ group.servers.length }} 节点</span
            >
          </div>

          <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            <div
              v-for="s in group.servers"
              :key="s.id"
              class="flex items-center gap-3 rounded-xl border border-[var(--c-border)] p-3.5 transition-all hover:border-[var(--c-primary)]"
            >
              <span class="relative flex h-2.5 w-2.5 shrink-0">
                <span
                  v-if="s.status === 1"
                  class="absolute inline-flex h-full w-full animate-ping rounded-full opacity-40"
                  :style="{ backgroundColor: serverStatusMeta(s.status).color }"
                />
                <span
                  class="relative inline-flex h-2.5 w-2.5 rounded-full"
                  :style="{ backgroundColor: serverStatusMeta(s.status).color }"
                />
              </span>

              <div class="min-w-0 flex-1">
                <div class="truncate text-14 font-500 text-[var(--c-text)]">{{ s.name }}</div>
                <div class="mt-1 flex flex-wrap items-center gap-1.5">
                  <span
                    class="rounded-full px-2 py-0.5 text-14"
                    style="background: var(--c-bg-hover); color: var(--c-text-sub)"
                  >
                    {{ s.type }}
                  </span>
                  <span
                    class="num rounded-full px-2 py-0.5 text-14"
                    style="background: var(--c-bg-hover); color: var(--c-text-sub)"
                  >
                    ×{{ s.rate }}
                  </span>
                  <span
                    v-for="tag in s.tags"
                    :key="tag"
                    class="rounded-full px-2 py-0.5 text-14"
                    style="background: var(--c-primary-soft); color: var(--c-primary-text)"
                  >
                    {{ tag }}
                  </span>
                </div>
              </div>

              <span
                class="shrink-0 text-14 font-500"
                :style="{ color: serverStatusMeta(s.status).color }"
              >
                {{ serverStatusMeta(s.status).label }}
              </span>
            </div>
          </div>
        </div>

        <EmptyState
          v-if="!server.loading && server.groups.length === 0"
          :text="t('common.empty')"
          :icon="'server'"
        />
      </div>
    </n-spin>
  </div>
</template>
