<script setup lang="ts">
/**
 * 公告面板:手风琴列表(图标/标题/时间/展开),markdown-it 渲染 + DOMPurify 过滤。
 * 数据:GET /notices?page_size=10(docs/api/README.md §6)
 * 优惠码约定:正文中用反引号包裹的大写「字母+数字」串(如 `618SALE`)渲染为高亮 chip,
 * 点击一键复制;与管理端发布公告共享同一数据源(mock/notices.ts,真实后端天然一致)。
 */
import { onMounted, ref } from 'vue'
import { useNoticeStore } from '@/stores/notice'
import { fromNow } from '@/utils/format'
import { copyText } from '@/utils/platform'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import MarkdownIt from 'markdown-it'
import DOMPurify from 'dompurify'

const notice = useNoticeStore()
const message = useMessage()
const { t } = useI18n()
const expandedId = ref<number | null>(null)

const md = new MarkdownIt({ html: false, linkify: true, breaks: true })

/** 匹配反引号包裹、同时含字母和数字的 4-20 位大写 token(优惠码形态) */
const COUPON_RE = /<code>((?=[A-Z0-9-]{4,20})(?=.*[A-Z])(?=.*\d)[A-Z0-9-]+)<\/code>/g

function renderHtml(content: string): string {
  let html = md.render(content)
  html = html.replace(
    COUPON_RE,
    (_, code: string) =>
      `<code class="coupon-code" data-code="${code}" title="${t('plan.couponTapToCopy')}">${code}</code>`,
  )
  return DOMPurify.sanitize(html)
}

/** 点击优惠码 chip → 复制 */
function onContentClick(e: MouseEvent) {
  const el = (e.target as HTMLElement).closest<HTMLElement>('.coupon-code')
  if (!el) return
  const code = el.dataset.code
  if (!code) return
  void copyText(code).then((ok) => {
    if (ok) message.success(t('common.copied'))
  })
}

function toggle(id: number) {
  expandedId.value = expandedId.value === id ? null : id
}

onMounted(() => {
  void notice.fetch(10)
})
</script>

<template>
  <div class="card-base p-5 md:p-6">
    <div class="mb-4 flex items-center gap-2">
      <span
        class="flex h-8 w-8 items-center justify-center rounded-full text-white"
        style="background: linear-gradient(135deg, #f5a524, #f7c948)"
      >
        <AppIcon name="bell" :size="17" />
      </span>
      <h3 class="text-16 font-600 text-[var(--c-text)]">{{ t('dashboard.notices') }}</h3>
    </div>

    <n-spin :show="notice.loading">
      <div class="divide-y divide-[var(--c-border)]">
        <div v-for="item in notice.list" :key="item.id" class="py-3.5">
          <button
            class="flex w-full cursor-pointer items-center justify-between gap-3 text-left"
            @click="toggle(item.id)"
          >
            <span class="min-w-0 flex-1 truncate text-14 font-500 text-[var(--c-text)]">
              {{ item.title }}
            </span>
            <span class="flex shrink-0 items-center gap-2">
              <span class="text-14 text-[var(--c-text-sub)]">{{ fromNow(item.created_at) }}</span>
              <AppIcon
                :name="'chevron-down'"
                :size="16"
                class="text-[var(--c-text-sub)] transition-transform duration-300"
                :style="{ transform: expandedId === item.id ? 'rotate(180deg)' : '' }"
              />
            </span>
          </button>
          <div
            v-if="expandedId === item.id"
            class="mt-2.5 rounded-xl p-3.5"
            style="background-color: var(--c-bg-hover)"
          >
            <!-- eslint-disable-next-line vue/no-v-html -->
            <div
              class="markdown-body text-14 text-[var(--c-text)]"
              @click="onContentClick"
              v-html="renderHtml(item.content)"
            />
          </div>
        </div>
        <EmptyState
          v-if="!notice.loading && notice.list.length === 0"
          :text="t('common.empty')"
          :icon="'bell'"
        />
      </div>
    </n-spin>
  </div>
</template>

<style scoped>
.markdown-body :deep(h1),
.markdown-body :deep(h2),
.markdown-body :deep(h3) {
  font-size: 15px;
  font-weight: 600;
  margin: 8px 0 4px;
}
.markdown-body :deep(p) {
  margin: 4px 0;
}
.markdown-body :deep(a) {
  color: var(--c-primary-text);
}
.markdown-body :deep(ul) {
  padding-left: 18px;
  list-style: disc;
}
.markdown-body :deep(code) {
  background: var(--c-bg-hover);
  border-radius: 4px;
  padding: 1px 4px;
  font-size: 12px;
}
/* 优惠码 chip:高亮 + 可点击复制 */
.markdown-body :deep(code.coupon-code) {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin: 0 2px;
  padding: 2px 8px;
  color: var(--c-primary-text);
  background: var(--c-primary-soft);
  border: 1px solid var(--c-primary);
  border-radius: var(--r-pill);
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;
}
.markdown-body :deep(code.coupon-code:hover) {
  background: var(--c-primary);
  color: #fff;
}
</style>
