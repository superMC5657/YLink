# [P2] Send the zero baseline before advancing demo counters

Status: resolved
Type: task

## Finding

`reportOnce` 在序列化前推进随机累计值，每次新启动首轮就计费 1–50 MiB 上行与 1–500 MiB 下行。

## Resolution

先按当前计数上报（首轮为 0），建立服务端快照后再推进下一轮累计值。

## Comments

2026-08-25: 已修复 `server/cmd/node-agent/main.go`。
