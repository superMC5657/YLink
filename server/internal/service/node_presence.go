package service

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"ylink-backend/internal/model"
)

// 节点在线状态深 presence Module：Key 形状 + TTL + 在线/负载语义的唯一拥有者。
//
// 收敛前：写端（node_service.Report 手拼 "node:heartbeat:%d"/"node:stats:%d"）与
// 读端（admin_crud.ListAllServers 手拼同 Key + MGet + 解析）在 Seam 处以裸字符串
// 耦合，TTL/形状一改两端加 mock 全要动；mock 的 system 形状已漂移。
// 收敛后：写端只报原始心跳，读端只问“在线否、负载如何”，Key 形状与 TTL 改动
// 收进唯一 owner，上报、列表与 mock 对齐同一形状（model.NodeSystemStats）。
//
// Depth 所在：在线判定（心跳存在即在线）+ 负载富化（stats JSON 解析降级）+
// TTL 语义（心跳 90s / 快照 180s）。

const (
	// PresenceHeartbeatTTL 心跳存活：90s 无上报即离线。
	PresenceHeartbeatTTL = 90 * time.Second
	// PresenceStatsTTL 负载快照保留：180s。
	PresenceStatsTTL = 180 * time.Second
)

// presenceHeartbeatKey 心跳 Key 形状 owner：`node:heartbeat:{id}`，值为 unix 秒。
func presenceHeartbeatKey(serverID int64) string {
	return "node:heartbeat:" + strconv.FormatInt(serverID, 10)
}

// presenceStatsKey 负载快照 Key 形状 owner：`node:stats:{id}`，值为 NodeSystemStats JSON。
func presenceStatsKey(serverID int64) string {
	return "node:stats:" + strconv.FormatInt(serverID, 10)
}

// presenceReport 写 seam：探针心跳与系统负载写入缓存（心跳 TTL 90s，负载 180s）。
// 不刷 PostgreSQL；rdb 为空时 no-op。
func presenceReport(ctx context.Context, rdb *redis.Client, serverID int64, system *model.NodeSystemStats) {
	if rdb == nil {
		return
	}
	now := time.Now().Unix()
	pipe := rdb.Pipeline()
	pipe.Set(ctx, presenceHeartbeatKey(serverID), strconv.FormatInt(now, 10), PresenceHeartbeatTTL)
	if system != nil {
		if b, err := json.Marshal(system); err == nil {
			pipe.Set(ctx, presenceStatsKey(serverID), string(b), PresenceStatsTTL)
		}
	}
	_, _ = pipe.Exec(ctx)
}

// presenceEnrich 读 seam：按 id 批量富化在线状态与负载。
// 心跳存在即在线（lastHeartbeat 为 unix 秒）；stats 解析失败降级为 nil，
// 不阻断列表主流程。答案形状与写端同一 owner，改 Key/TTL 只改一处。
func presenceEnrich(ctx context.Context, rdb *redis.Client, views []model.AdminServerView) {
	if rdb == nil || len(views) == 0 {
		return
	}
	hbKeys := make([]string, len(views))
	statKeys := make([]string, len(views))
	for i, v := range views {
		hbKeys[i] = presenceHeartbeatKey(v.ID)
		statKeys[i] = presenceStatsKey(v.ID)
	}
	hbVals, _ := rdb.MGet(ctx, hbKeys...).Result()
	statVals, _ := rdb.MGet(ctx, statKeys...).Result()

	for i := range views {
		if i < len(hbVals) && hbVals[i] != nil {
			if tsStr, ok := hbVals[i].(string); ok {
				if ts, err := strconv.ParseInt(tsStr, 10, 64); err == nil {
					views[i].IsOnline = true
					views[i].LastHeartbeatAt = &ts
				}
			}
		}
		if i < len(statVals) && statVals[i] != nil {
			if statStr, ok := statVals[i].(string); ok {
				var sys model.NodeSystemStats
				if err := json.Unmarshal([]byte(statStr), &sys); err == nil {
					views[i].System = &sys
				}
			}
		}
	}
}
