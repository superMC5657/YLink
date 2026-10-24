# [P0] cron 分布式锁：错误混淆与无属主删除

Status: ready-for-agent
Type: task

## Finding

`server/internal/service/cron_schedule.go:43-49` 两个缺陷：

1. `SetNX` 的 error 与「锁被他人持有」混在 `err != nil || !ok → skipped` 一条路径（Info 级日志）：Redis 故障时所有 cron 任务被静默跳过，且日志看不出是基础设施故障。
2. `defer rdb.Del(key)` 不校验属主：任务执行超过锁 TTL（15min）后，锁已过期并被继任实例持有，原实例结束时把继任者的锁删掉 → 下一轮双实例并发执行同一任务。

## Plan

1. 分路径：`SetNX` 返回 err → Error 日志 + 按 cron 语义决定跳过（保持现状语义但可观测）；`!ok` → Info「锁被持有，跳过」。
2. 锁值写随机 token（如 `uuid.NewString()`），释放时用 Lua `if GET == token then DEL`（或 `GET` 比对后 `DEL`，接受极小窗口；首选 Lua 单原子）。
3. 补测试：miniredis 模拟 SetNX 错误路径、token 不匹配不删除。

## Acceptance

- Redis 故障路径产生 Error 级日志且不 panic；正常锁竞争维持 skip。
- 释放只删除自己持有的锁（token 断言）。
- `go test ./internal/service/... -run Cron` 全绿；core-flows.md 若描述了锁机制则同步更新。

## Comments
