<script setup lang="ts">
/**
 * 文档详情:Markdown 渲染(markdown-it + DOMPurify + 代码高亮),顶部返回。
 * 数据:GET /knowledges/{id}(docs/api/README.md §7.2)
 */
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useKnowledgeStore } from '@/stores/knowledge'
import { formatTime } from '@/utils/format'
import { useI18n } from 'vue-i18n'
import MarkdownIt from 'markdown-it'
import DOMPurify from 'dompurify'

const route = useRoute()
const router = useRouter()
const knowledge = useKnowledgeStore()
const { t } = useI18n()

const loading = ref(true)

const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true,
  highlight: (_lang: string, str: string) => {
    // 代码高亮一期使用简单转义 + 行号容器,后续可接入 highlight.js
    const escaped = str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
    return `<pre class="code-block"><code>${escaped}</code></pre>`
  },
})

const article = computed(() => knowledge.detail)

const rendered = computed(() =>
  article.value ? DOMPurify.sanitize(md.render(article.value.body)) : '',
)

onMounted(async () => {
  const id = Number(route.params.id)
  try {
    await knowledge.fetchDetail(id)
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div>
    <button
      class="mb-4 flex cursor-pointer items-center gap-1.5 text-13 text-[var(--c-text-sub)] transition-colors hover:text-[var(--c-primary-text)]"
      @click="router.back()"
    >
      <AppIcon name="arrow-left" :size="16" />
      {{ t('common.back') }}
    </button>

    <n-spin :show="loading">
      <article v-if="article" class="card-base p-5 md:p-8">
        <div class="mb-5 border-b border-[var(--c-border)] pb-4">
          <div class="mb-2 flex items-center gap-2">
            <StatusBadge type="primary">{{ article.category }}</StatusBadge>
            <StatusBadge type="neutral" :dot="false">{{ article.language }}</StatusBadge>
          </div>
          <h1 class="text-22 font-700 text-[var(--c-text)]">{{ article.title }}</h1>
          <div class="mt-2 flex items-center gap-4 text-12 text-[var(--c-text-sub)]">
            <span class="flex items-center gap-1">
              <AppIcon name="calendar" :size="14" />
              {{ t('docs.updatedAt') }}:{{ formatTime(article.updated_at) }}
            </span>
          </div>
        </div>

        <!-- eslint-disable-next-line vue/no-v-html -->
        <div class="markdown-body text-15 leading-7 text-[var(--c-text)]" v-html="rendered" />
      </article>
    </n-spin>
  </div>
</template>

<style scoped>
.markdown-body :deep(h1) {
  font-size: 24px;
  font-weight: 700;
  margin: 20px 0 10px;
}
.markdown-body :deep(h2) {
  font-size: 19px;
  font-weight: 600;
  margin: 18px 0 8px;
  padding-left: 10px;
  border-left: 3px solid var(--c-primary);
}
.markdown-body :deep(h3) {
  font-size: 16px;
  font-weight: 600;
  margin: 14px 0 6px;
}
.markdown-body :deep(p) {
  margin: 8px 0;
}
.markdown-body :deep(ul),
.markdown-body :deep(ol) {
  padding-left: 22px;
  margin: 8px 0;
}
.markdown-body :deep(ul) {
  list-style: disc;
}
.markdown-body :deep(ol) {
  list-style: decimal;
}
.markdown-body :deep(a) {
  color: var(--c-primary-text);
  text-decoration: underline;
}
.markdown-body :deep(blockquote) {
  border-left: 3px solid var(--c-border);
  padding-left: 12px;
  color: var(--c-text-sub);
  margin: 10px 0;
}
.markdown-body :deep(code) {
  background: var(--c-bg-hover);
  border-radius: 5px;
  padding: 2px 6px;
  font-size: 13px;
  font-family: 'JetBrains Mono', Consolas, monospace;
}
.markdown-body :deep(pre.code-block) {
  background: var(--c-bg-hover);
  border-radius: 10px;
  padding: 14px;
  overflow-x: auto;
  margin: 12px 0;
}
.markdown-body :deep(pre.code-block code) {
  background: transparent;
  padding: 0;
}
.markdown-body :deep(hr) {
  border: none;
  border-top: 1px solid var(--c-border);
  margin: 16px 0;
}
.markdown-body :deep(strong) {
  font-weight: 700;
  color: var(--c-marketing);
}
</style>
