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

- [x] 退守方案落地:`useQrCode.ts` 动态 `import('qrcode')` + loading/failed 态,PaymentModal/SharePanel 两处复用;`qrcode` 移入弹窗异步 chunk,首屏不再包含。
- [x] SharePanel 邀请卡导出改 WebP q0.9(不支持的浏览器按规范自动回退 PNG,文件名按 blob.type 定扩展名);两处二维码 `<img>` 补 width/height/loading=lazy/decoding=async。
- [x] `pnpm lint`/`typecheck` + vitest 23 用例(新增 useQrCode 3 例 + SharePanel webp 命名 1 例)全绿;`pnpm build:web` 产物确认 qrcode 仅存在于 PaymentModal 异步 chunk。

## Comments

- 未选 NQrCode 首选方案:SharePanel 需将二维码作为位图绘入 canvas 合成邀请卡,NQrCode 只渲染内联 SVG/组件,无法直接产出 dataURL,强行序列化 SVG 反而引入脆弱路径;故保留 `qrcode` 依赖但动态加载,两弹窗打开时才拉取。
