# 代码评审 — YLink 管理端模块(0.2.0 之后增量)

- **日期:** 2026-08-11
- **范围:** 增量 — v0.2.0 评审之后的管理端提交(`7949622..a8295ce`):新管理端模块(优惠券、流量导入)、mocks、测试、缓存与端口变更
- **方法:** 评审模型对上述提交的整体评审;后端可构建、新测试通过、DTO/类型形状匹配
- **状态:** 已解决 — 2 项 P1、1 项 P3 均已修复(见下方删除线条目;对应提交 `078f541`、`294c116` 及流量导入文案修复)

## 摘要

新管理端模块、mocks、测试以及缓存/端口变更在其他方面一致(后端可构建、新测试通过、DTO/类型形状匹配),但端口上移导致随仓库交付的 docker-compose Caddy 部署不可用;并且通过新管理端 UI 创建的百分比优惠券,由于存储值与结账计算之间的单位不一致,会在结账时给订单打 100% 折扣。两项 P1 与 P3 文案问题均已修复。

## 发现

### ~~[P1] 端口 8080→8081 迁移后 Caddyfile 仍代理 api:8080 — server/docker-compose.yml:35-36~~

~~本次提交将 API 监听端口改为 8081(`configs/config.yaml` 的 `addr: ":8081"` 经 Dockerfile 固化进镜像),并在此发布 `8081:8081`,但被同一 compose 文件中 `caddy` 服务以只读方式挂载的 `server/deploy/Caddyfile` 仍然写着 `reverse_proxy api:8080`。在 compose 网络内部,Caddy 直接访问容器,因此经由生产入口的每一个请求都会失败(connection refused/502)。`docs/backend/deploy.md` 已更新为 `reverse_proxy api:8081`,但实际的 Caddyfile 被遗漏;`server/Dockerfile:14` 也仍写着 `EXPOSE 8080`,而 deploy.md 的片段已更新为 8081。严重度仅限于 docker-compose/Caddy 部署(dev-up.sh 直接运行二进制,不受影响),但该路径被本次变更完全弄坏。~~

**已修复** — 提交 `294c116`:`server/deploy/Caddyfile` 改代理 `api:8081`,`server/Dockerfile` 改 `EXPOSE 8081`。

### ~~[P1] 新管理端 UI 创建的百分比优惠券会全额免单 — server/internal/service/admin_crud.go:249-251~~

~~本次提交的契约(docs §16.1 "type=2 为百分比数值,如 10 表示 10%")、mock 数据、`AdminCouponsView.vue` 以及此处 `FenToYuan` 的展示换算,都将百分比优惠券的 `value` 视为原始百分数。然而 `CreateCoupon`/`UpdateCoupon` 对所有类型都存储 `model.YuanToFen(req.Value)`(10 → 1000),而结账计算 `discount = amount * coupon.Value / 100` 并封顶到订单金额(`order_service.go` 的 `validateCoupon`)。因此,通过新管理端页面创建或编辑的 "10%" 优惠券会得到 `amount * 10` → 封顶 → 整单免单;任何 ≥ 1 的百分数都会让订单免单。写路径的换算与结账计算早于本次提交,但本次提交引入了使百分比优惠券可触达的 UI 与文档契约,因此该端到端功能在交付状态下存在资金安全风险。修复需要统一为单一单位,例如 `type == 2` 时跳过 `YuanToFen`/`FenToYuan`(并保留结账的 `/100`),或在结账时除以 10000。~~

**已修复** — 提交 `078f541`:结账计算改为 `discount = amount * coupon.Value / 10000`(存储值为"百分比×100"的分),10% 券现在恰好折扣订单的 10%。

### ~~[P3] 流量导入提示 "累加覆盖" 与实际覆盖行为矛盾 — src/views/admin/AdminTrafficImportView.vue:128~~

~~提示称重复导入同一 user+date "将累加覆盖",但 `TrafficLogRepo.Upsert`(`server/internal/repo/admin.go`,注释为 "按 (user_id, date) 覆盖导入")是替换 `u`/`d` 而非累加。管理员若按提示理解为累加,只会导入增量部分,从而悄无声息地覆盖当天此前已记录的用量。建议改写提示,说明重复导入会覆盖当天的数值。~~

**已修复** — 提示改为 "同一用户同一天重复导入将覆盖当日已记录的上行/下行数据",与 `Upsert` 的覆盖行为一致。
