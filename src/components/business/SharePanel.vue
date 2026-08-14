<script setup lang="ts">
/**
 * 分享面板(浮动 panel,居中悬浮于窗口之上):品牌邀请卡片(渐变背景 + 站点名 + 邀请码 + 二维码)
 * + 复制链接 + 下载图片(canvas 合成紫色邀请卡片为 PNG 供下载,纯前端实现)。
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

// 下载图片的画布尺寸(与面板卡片同比例放大,保证导出清晰度)
const IMG_W = 720
const IMG_H = 940
// 白色二维码块区域(内嵌二维码,白块内边距 24)
const QR_BOX = { x: 80, y: 210, size: 560, pad: 24, radius: 28 }

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

/** 加载 dataURL 图片,供 canvas 绘制 */
function loadImage(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image()
    img.onload = () => resolve(img)
    img.onerror = () => reject(new Error('image load failed'))
    img.src = src
  })
}

/** 圆角矩形路径(兼容不支持 ctx.roundRect 的旧环境) */
function roundRectPath(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  w: number,
  h: number,
  r: number,
) {
  ctx.beginPath()
  ctx.moveTo(x + r, y)
  ctx.arcTo(x + w, y, x + w, y + h, r)
  ctx.arcTo(x + w, y + h, x, y + h, r)
  ctx.arcTo(x, y + h, x, y, r)
  ctx.arcTo(x, y, x + w, y, r)
  ctx.closePath()
}

/** 画布上绘制居中文本,超出最大宽度时逐级缩小字号,仍超宽则截断加省略号 */
function drawCenteredText(
  ctx: CanvasRenderingContext2D,
  text: string,
  y: number,
  maxWidth: number,
  baseSize: number,
  weight = 400,
) {
  const font = (size: number) =>
    `${weight} ${size}px "PingFang SC","Microsoft YaHei",system-ui,sans-serif`
  let size = baseSize
  ctx.font = font(size)
  while (ctx.measureText(text).width > maxWidth && size > 14) {
    size -= 2
    ctx.font = font(size)
  }
  // 最小字号仍超宽:按可容纳宽度截断并追加省略号,避免溢出画布(最差退化为仅省略号)
  let content = text
  if (ctx.measureText(content).width > maxWidth) {
    while (content.length > 0 && ctx.measureText(`${content}…`).width > maxWidth) {
      content = content.slice(0, -1)
    }
    content += '…'
  }
  const prevAlign = ctx.textAlign
  ctx.textAlign = 'center'
  ctx.fillText(content, IMG_W / 2, y)
  ctx.textAlign = prevAlign
}

/** 用 canvas 把紫色邀请卡片(渐变背景 + 站点名 + 邀请码 + 二维码)合成为 PNG 并触发下载(纯前端) */
async function onDownloadImage() {
  if (!qrDataUrl.value) {
    message.error(t('share.downloadFailed'))
    return
  }
  const canvas = document.createElement('canvas')
  canvas.width = IMG_W
  canvas.height = IMG_H
  const ctx = canvas.getContext('2d')
  if (!ctx) {
    message.error(t('share.downloadFailed'))
    return
  }
  let qr: HTMLImageElement
  try {
    qr = await loadImage(qrDataUrl.value)
  } catch {
    message.error(t('share.downloadFailed'))
    return
  }

  // 紫色渐变背景
  const grad = ctx.createLinearGradient(0, 0, IMG_W, IMG_H)
  grad.addColorStop(0, '#6558f5')
  grad.addColorStop(1, '#8b5cf6')
  ctx.fillStyle = grad
  roundRectPath(ctx, 0, 0, IMG_W, IMG_H, 28)
  ctx.fill()

  // 站点名
  ctx.fillStyle = '#FFFFFF'
  drawCenteredText(ctx, config.siteName, 104, IMG_W - 96, 44, 600)

  // 邀请码
  if (props.code) {
    ctx.fillStyle = 'rgba(255,255,255,0.85)'
    drawCenteredText(ctx, `${t('invite.code')}: ${props.code}`, 166, IMG_W - 96, 28)
  }

  // 白色二维码块
  ctx.fillStyle = '#FFFFFF'
  roundRectPath(ctx, QR_BOX.x, QR_BOX.y, QR_BOX.size, QR_BOX.size, QR_BOX.radius)
  ctx.fill()
  ctx.drawImage(
    qr,
    QR_BOX.x + QR_BOX.pad,
    QR_BOX.y + QR_BOX.pad,
    QR_BOX.size - QR_BOX.pad * 2,
    QR_BOX.size - QR_BOX.pad * 2,
  )

  // 提示语与注册链接
  ctx.fillStyle = 'rgba(255,255,255,0.85)'
  drawCenteredText(ctx, t('share.qrTip'), 822, IMG_W - 96, 26)
  ctx.fillStyle = 'rgba(255,255,255,0.6)'
  drawCenteredText(ctx, props.text, 864, IMG_W - 96, 20)

  canvas.toBlob((blob) => {
    if (!blob) {
      message.error(t('share.downloadFailed'))
      return
    }
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `ylink-invite-${props.code ?? 'card'}.png`
    // 旧版 Safari/iOS 要求 anchor 挂载在 DOM 中才能触发下载
    document.body.appendChild(a)
    a.click()
    a.remove()
    // Safari/iOS 在下载进行中回收 URL 会中止下载,延迟释放
    setTimeout(() => URL.revokeObjectURL(url), 1000)
  }, 'image/png')
}
</script>

<template>
  <n-modal
    :show="props.show"
    preset="card"
    :style="{ width: 'min(92vw, 30rem)' }"
    @update:show="(v: boolean) => emit('update:show', v)"
  >
    <div class="w-full px-1 pb-2 pt-1">
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

      <!-- 操作按钮:复制链接 + 下载图片 -->
      <div class="mt-4 flex gap-3">
        <button class="btn-primary h-10 flex-1 text-14" @click="onCopy">
          <AppIcon name="copy" :size="16" />
          {{ t('share.copyLink') }}
        </button>
        <button class="btn-soft-neutral h-10 flex-1 text-14" @click="onDownloadImage">
          <AppIcon name="download" :size="16" />
          {{ t('share.downloadImage') }}
        </button>
      </div>
    </div>
  </n-modal>
</template>
