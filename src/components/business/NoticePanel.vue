<script setup lang="ts">
/**
 * 公告面板:手风琴列表(图标/标题/时间/展开),正文走 utils/markdown renderMarkdown(懒加载+XSS 净化)。
 * 数据:GET /notices?page_size=10(docs/api/README.md §6)
 * 优惠码约定:正文中用反引号包裹的大写「字母+数字」串(如 `618SALE`)渲染为高亮 chip,
 * 点击一键复制;与管理端发布公告共享同一数据源(mock/notices.ts,真实后端天然一致)。
 */
import { onMounted, ref } from 'vue'
import { useNoticeStore } from '@/stores/notice'
import { fromNow } from '@/utils/format'
import { renderMarkdown } from '@/utils/markdown'
import { copyText } from '@/utils/platform'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'

const notice = useNoticeStore()
const message = useMessage()
const { t } = useI18n()
const expandedId = ref<number | null>(null)
/** 展开项渲染后的安全 HTML(单开手风琴,一次只渲染一条;markdown-it 懒加载异步产出) */
const expandedHtml = ref('')

/**
 * 优惠码 chip 白名单:仅大写字母/数字/连字符 4-20 位(不含引号/尖括号等注入字符),
 * 先过白名单再拼 HTML,属性注入不可能成立;不合白名单保留原 <code> 渲染。
 */
const COUPON_RE = /<code>((?=[A-Z0-9-]{4,20})(?=.*[A-Z])(?=.*\d)[A-Z0-9-]+)<\/code>/g
const CODE_WHITELIST_RE = /^[A-Z0-9-]{4,20}$/

/** 渲染 + 白名单校验后追加优惠码 chip(在净化后执行,chip 内容已被白名单约束) */
function decorateCouponChips(html: string): string {
  const title = t('plan.couponTapToCopy').replace(/"/g, '&quot;')
  return html.replace(COUPON_RE, (matched, code: string) => {
    if (!CODE_WHITELIST_RE.test(code)) return matched
    return `<code class="coupon-code" data-code="${code}" title="${title}">${code}</code>`
  })
}

/** 展开时异步渲染公告正文(统一走 renderMarkdown 单例,含 XSS 净化) */
async function expand(id: number, content: string) {
  expandedId.value = id
  expandedHtml.value = ''
  const html = await renderMarkdown(content)
  // 展开项可能在等待期间被切换,过期结果不写回
  if (expandedId.value === id) expandedHtml.value = decorateCouponChips(html)
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

function toggle(id: number, content: string) {
  if (expandedId.value === id) {
    expandedId.value = null
    expandedHtml.value = ''
  } else {
    void expand(id, content)
  }
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
            @click="toggle(item.id, item.content)"
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
              v-html="expandedHtml"
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
