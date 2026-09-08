# [P0] 02 QRCode 首屏瘦身

Status: ready-for-agent
Type: task

## Finding

- `src/components/business/PaymentModal.vue:14` + `SharePanel.vue:12` 同步 `import QRCode from 'qrcode'`（~100KB+Buffer），均为弹窗非首屏关键。

## Task

1. 首选：改用 Naive `NQrCode`（dist 已含 QrCode 主题），删 `qrcode` 依赖；若样式/容错率不满足则退守 `await import('qrcode')` 动态加载 +  loading 态。
2. 抽 `src/composables/useQrCode.ts`（toDataURL 统一 width/容错），两处复用。
3. SharePanel 邀请卡 `canvas.toBlob png 720x940` 体积大 → 转 webp0.9 或降分辨率，`img` 加 `loading=lazy decoding=async width/height` 防 CLS。

## Done

- 首屏 vendor 减小；`pnpm lint/typecheck` + 相关 vitest 通过；弹窗二维码扫码可用。

## Comments
