<script setup lang="ts">
/**
 * 分享面板(移动端优先,底部弹出):品牌邀请卡片(渐变背景 + 站点名 + 邀请码 + 二维码)
 * + 复制链接 + 系统分享(优先分享二维码图片,不支持文件时退回文本)。
 * 用途:邀请注册链接等可打开的外部链接;深链 ylink:// 已由 main.ts/deeplink.ts 处理,此处分享的是 https 可访问地址。
 * 二维码用 qrcode 库,生成 512px 保证分享图片清晰;白块内固定深色码(不随暗色主题反色)。
 */
import { ref, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import QRCode from 'qrcode'
import { copyText } from '@/utils/platform'
import { useConfigStore } from '@/stores/config'

const props = defineProps<{
  show: boolean
  /** 面板标题 */
  title: string
  /** 要分享的链接/文本 */
  text: string
  /** 面板内说明文字(可选) */
  desc?: string
  /** 邀请码(可选,卡片内展示) */
  code?: string
}>()

const emit = defineEmits<{ (e: 'update:show', v: boolean): void }>()

const message = useMessage()
const { t } = useI18n()
const config = useConfigStore()

const qrDataUrl = ref('')
const qrFailed = ref(false)
const retryTick = ref(0)

// 二维码颜色固定:白块卡片背景恒为白色,不随主题变化(暗色下用 --c-text 会反色、扫码可靠性下降)
const QR_DARK = '#1F2430'
const QR_LIGHT = '#FFFFFF'

watch(
  () => [props.show, props.text, retryTick.value],
  async () => {
    if (!props.show || !props.text) {
      qrDataUrl.value = ''
      qrFailed.value = false
      return
    }
    qrFailed.value = false
    try {
      qrDataUrl.value = await QRCode.toDataURL(props.text, {
        width: 512,
        margin: 1,
        color: { dark: QR_DARK, light: QR_LIGHT },
      })
    } catch {
      qrDataUrl.value = ''
      qrFailed.value = true
    }
  },
)

/** 二维码生成失败后重试:重新触发 watch */
function retryQr() {
  qrFailed.value = false
  retryTick.value++
}

async function onCopy() {
  const ok = await copyText(props.text)
  message.success(ok ? t('common.copied') : t('share.copyFailed'))
}

/** dataURL → File,供系统分享图片 */
function dataUrlToFile(dataUrl: string, filename: string): File {
  const [meta, b64] = dataUrl.split(',')
  const mime = meta.match(/^data:(.*?);/)?.[1] ?? 'image/png'
  const bin = atob(b64)
  const bytes = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i)
  return new File([bytes], filename, { type: mime })
}

/** 系统分享仅浏览器可用(navigator.share);Tauri WebView 一般不暴露,自动隐藏 */
const canSystemShare = typeof navigator !== 'undefined' && typeof navigator.share === 'function'

async function onSystemShare() {
  try {
    const payload: ShareData = { title: props.title, text: props.text }
    // 优先分享二维码图片(Web Share API Level 2),不支持文件时退回文本
    if (qrDataUrl.value) {
      const file = dataUrlToFile(qrDataUrl.value, 'invite-qr.png')
      const canShareFile =
        typeof navigator.canShare === 'function' && navigator.canShare({ files: [file] })
      if (canShareFile) {
        await navigator.share({ ...payload, files: [file] })
        return
      }
    }
    await navigator.share(payload)
  } catch {
    // 用户取消或分享失败,静默处理
  }
}
</script>

<template>
  <n-drawer
    :show="props.show"
    placement="bottom"
    :height="520"
    @update:show="(v: boolean) => emit('update:show', v)"
  >
    <div class="mx-auto w-full max-w-md px-5 pb-6 pt-5">
      <!-- 标题栏 -->
      <div class="mb-1 flex items-center justify-between">
        <h3 class="text-16 font-600 text-[var(--c-text)]">{{ props.title }}</h3>
        <button
          class="flex h-8 w-8 cursor-pointer items-center justify-center rounded-full text-[var(--c-text-sub)] transition-colors hover:bg-[var(--c-bg-hover)]"
          @click="emit('update:show', false)"
        >
          <AppIcon name="close" :size="18" />
        </button>
      </div>
      <p v-if="props.desc" class="mb-4 text-14 text-[var(--c-text-sub)]">{{ props.desc }}</p>

      <!-- 品牌邀请卡片:渐变背景 + 站点名 + 邀请码 + 二维码 -->
      <div
        class="rounded-2xl p-5 text-white shadow-lg"
        style="background: linear-gradient(135deg, #6558f5, #8b5cf6)"
      >
        <div class="text-18 font-700">{{ config.siteName }}</div>
        <div v-if="props.code" class="mt-1 text-14 opacity-80">
          {{ t('invite.code') }}: {{ props.code }}
        </div>

        <!-- 二维码白块 -->
        <div class="mt-4 flex justify-center">
          <div class="rounded-2xl bg-white p-3">
            <img v-if="qrDataUrl" :src="qrDataUrl" alt="qr" class="h-52 w-52" />
            <div
              v-else
              class="flex h-52 w-52 flex-col items-center justify-center gap-2 text-14 text-[var(--c-text-sub)]"
            >
              <template v-if="qrFailed">
                <span>{{ t('share.qrFailed') }}</span>
                <button class="btn-soft-neutral h-8 px-4 text-14" @click="retryQr">
                  {{ t('common.retry') }}
                </button>
              </template>
              <span v-else>{{ t('common.loading') }}</span>
            </div>
          </div>
        </div>
        <p class="mt-3 text-center text-14 opacity-80">{{ t('share.qrTip') }}</p>
      </div>

      <!-- 链接展示 -->
      <div
        class="mt-4 flex items-center gap-2 rounded-xl p-3"
        style="background-color: var(--c-bg-hover)"
      >
        <span class="num min-w-0 flex-1 truncate text-14 text-[var(--c-text)]">
          {{ props.text }}
        </span>
      </div>

      <!-- 操作按钮 -->
      <div class="mt-4 flex gap-3">
        <button class="btn-primary h-10 flex-1 text-14" @click="onCopy">
          <AppIcon name="copy" :size="16" />
          {{ t('share.copyLink') }}
        </button>
        <button
          v-if="canSystemShare"
          class="btn-soft-neutral h-10 flex-1 text-14"
          @click="onSystemShare"
        >
          <AppIcon name="send" :size="16" />
          {{ t('share.systemShare') }}
        </button>
      </div>
    </div>
  </n-drawer>
</template>
