<script setup lang="ts">
/**
 * 使用文档(截图2):搜索(防抖 300ms)+ 语言切换 + 分类分组列表。
 * 数据:GET /knowledges?keyword=&language=(docs/api/README.md §7.1)
 */
import { onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useKnowledgeStore } from '@/stores/knowledge'
import { useAppStore } from '@/stores/app'
import { fromNow } from '@/utils/format'
import { useI18n } from 'vue-i18n'

const knowledge = useKnowledgeStore()
const app = useAppStore()
const router = useRouter()
const { t } = useI18n()

const keyword = ref('')
let debounceTimer: ReturnType<typeof setTimeout> | null = null

function onSearch(value: string) {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    void knowledge.fetch({ keyword: value.trim() })
  }, 300)
}

function goDetail(id: number) {
  router.push(`/docs/${id}`)
}

onMounted(() => {
  // 语言跟随网站全局语言(app.language),不提供独立选择
  void knowledge.fetch({ language: app.language })
})

// 网站语言变化时重新拉取对应语言的知识库
watch(
  () => app.language,
  (lang) => {
    void knowledge.fetch({ language: lang })
  },
)
</script>

<template>
  <div>
    <div class="mb-5 flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
      <h1 class="text-20 font-600 text-[var(--c-text)]">{{ t('docs.title') }}</h1>
      <div class="flex w-full items-center gap-2 md:w-auto">
        <!-- 搜索 -->
        <div class="relative min-w-0 flex-1 md:max-w-72">
          <span class="absolute top-1/2 left-3 -translate-y-1/2 text-[var(--c-text-sub)]">
            <AppIcon name="search" :size="16" />
          </span>
          <input
            v-model="keyword"
            type="text"
            :placeholder="t('docs.searchPlaceholder')"
            class="h-10 w-full rounded-[var(--r-control)] border border-[var(--c-border)] bg-[var(--c-bg-card)] pr-3 pl-9 text-14 text-[var(--c-text)] outline-none transition-colors placeholder:text-[var(--c-text-sub)] focus:border-[var(--c-primary)]"
            @input="onSearch(keyword)"
          />
        </div>
      </div>
    </div>

    <n-spin :show="knowledge.loading">
      <div class="space-y-5">
        <div
          v-for="group in knowledge.groups"
          :key="group.category"
          class="card-base card-hoverable p-5 md:p-6"
        >
          <div class="mb-3 flex items-center gap-2">
            <span
              class="h-4 w-1 rounded-full"
              style="background: linear-gradient(180deg, #6558f5, #8b5cf6)"
            />
            <h3 class="text-16 font-600 text-[var(--c-text)]">{{ group.category }}</h3>
            <span class="ml-auto text-14 text-[var(--c-text-sub)]"
              >{{ group.items.length }} 篇</span
            >
          </div>

          <div v-if="group.items.length" class="divide-y divide-[var(--c-border)]">
            <button
              v-for="item in group.items"
              :key="item.id"
              class="flex w-full cursor-pointer items-center justify-between gap-3 py-3 text-left transition-colors hover:bg-[var(--c-bg-hover)]"
              @click="goDetail(item.id)"
            >
              <span class="min-w-0 flex-1 truncate text-14 text-[var(--c-text)]">{{
                item.title
              }}</span>
              <span class="flex shrink-0 items-center gap-2.5">
                <span class="text-14 text-[var(--c-text-sub)]">{{ fromNow(item.updated_at) }}</span>
                <span
                  class="flex items-center gap-0.5 text-14 font-500 text-[var(--c-primary-text)]"
                >
                  {{ t('docs.read') }}
                  <AppIcon name="chevron-right" :size="14" />
                </span>
              </span>
            </button>
          </div>
          <EmptyState v-else :text="t('docs.noResult')" :icon="'book'" />
        </div>

        <EmptyState
          v-if="!knowledge.loading && (knowledge.groups?.length ?? 0) === 0"
          :text="t('docs.noResult')"
          :icon="'search'"
        />
      </div>
    </n-spin>
  </div>
</template>
