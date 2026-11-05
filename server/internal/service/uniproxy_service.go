package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	"ylink-backend/internal/pkg/logger"
	redispkg "ylink-backend/internal/pkg/redis"
	"ylink-backend/internal/repo"
)

// UniProxyService 提供 V2Board / UniProxy 协议兼容的核心业务逻辑（配置下发、用户分发、增量计费）。
type UniProxyService struct {
	db    *gorm.DB
	rdb   *redis.Client
	repos *repo.Repos
}

// NewUniProxyService 创建 UniProxy 业务服务。
func NewUniProxyService(db *gorm.DB, rdb *redis.Client, repos *repo.Repos) *UniProxyService {
	return &UniProxyService{db: db, rdb: rdb, repos: repos}
}

// ServerByID 根据节点 ID 获取节点（供 UniProxyAuth 中间件回源查询）。
func (s *UniProxyService) ServerByID(ctx context.Context, serverID int64) (*model.Server, error) {
	return s.repos.Server.GetByID(s.db, serverID)
}

// isNodeTypeCompatible 校验 UniProxy 客户端请求的 node_type 是否与节点实际类型兼容。
func isNodeTypeCompatible(nodeType, serverType string) bool {
	req := strings.ToLower(strings.TrimSpace(nodeType))
	srv := strings.ToLower(strings.TrimSpace(serverType))
	if req == "" {
		return true
	}
	if req == srv {
		return true
	}
	switch req {
	case "v2ray":
		return srv == "vless" || srv == "vmess" || srv == "v2ray"
	case "ss", "shadowsocks":
		return srv == "ss" || srv == "shadowsocks"
	case "hysteria", "hysteria2", "hy2":
		return srv == "hysteria" || srv == "hysteria2"
	case "trojan":
		return srv == "trojan"
	case "tuic":
		return srv == "tuic"
	}
	return false
}

// GetConfig 节点配置下发：根据节点 ID 和可选协议生成 UniProxy 配置。
func (s *UniProxyService) GetConfig(ctx context.Context, serverID int64, nodeType string) (*model.UniProxyConfigResp, error) {
	srv, err := s.repos.Server.GetByID(s.db, serverID)
	if err != nil {
		return nil, errs.ErrNotFound
	}

	if nodeType != "" && !isNodeTypeCompatible(nodeType, srv.Type) {
		return nil, errs.New(40000, fmt.Sprintf("incompatible node_type '%s' for server type '%s'", nodeType, srv.Type))
	}

	cfgData, err := model.ParseUniProxyConfig(srv)
	if err != nil {
		return nil, err
	}

	if nodeType != "" && cfgData.Protocol == "" {
		cfgData.Protocol = nodeType
	}

	resp := model.NewUniProxyConfigResp(*cfgData)
	return &resp, nil
}

// GetUsers 有效用户列表下发：根据节点分组筛选有效未封禁未过期用户。
func (s *UniProxyService) GetUsers(ctx context.Context, serverID int64) (*model.UniProxyUserResp, error) {
	srv, err := s.repos.Server.GetByID(s.db, serverID)
	if err != nil {
		return nil, errs.ErrNotFound
	}

	planIDs, err := s.planIDsContainingGroup(srv.GroupID)
	if err != nil {
		return nil, err
	}

	if len(planIDs) == 0 {
		resp := model.NewUniProxyUserResp(nil)
		return &resp, nil
	}

	users, err := s.repos.User.ListActiveByPlanIDs(s.db, planIDs, time.Now())
	if err != nil {
		return nil, err
	}

	items := make([]model.UniProxyUserItem, 0, len(users))
	for _, u := range users {
		var speedLimit int64
		if u.SpeedLimit != nil {
			speedLimit = int64(*u.SpeedLimit)
		}
		var deviceLimit int
		if u.DeviceLimit != nil {
			deviceLimit = *u.DeviceLimit
		}

		items = append(items, model.UniProxyUserItem{
			ID:          u.ID,
			UUID:        u.UUID,
			SpeedLimit:  speedLimit,
			DeviceLimit: deviceLimit,
		})
	}

	resp := model.NewUniProxyUserResp(items)
	return &resp, nil
}

// PushTraffic 节点增量流量上报并计费。
// 注意：UniProxy 上报的 u 与 d 为单周期内的增量字节（Delta），需要直接与节点倍率相乘后入账。
func (s *UniProxyService) PushTraffic(ctx context.Context, serverID int64, items []model.UniProxyPushItem) (int, error) {
	srv, err := s.repos.Server.GetByID(s.db, serverID)
	if err != nil {
		return 0, errs.ErrNotFound
	}

	// 探针心跳与系统负载写入缓存：利用在线状态深 Module 的写 seam
	presenceReport(ctx, s.rdb, serverID, nil)

	if len(items) == 0 {
		return 0, nil
	}

	// 聚合多条记录（针对同批次出现重复 user_id 进行合并增量）
	type deltaTraffic struct {
		u int64
		d int64
	}
	userDeltas := make(map[int64]*deltaTraffic, len(items))
	userIDs := make([]int64, 0, len(items))
	for _, it := range items {
		if it.UserID <= 0 || (it.U <= 0 && it.D <= 0) {
			continue
		}
		if d, ok := userDeltas[it.UserID]; ok {
			d.u += it.U
			d.d += it.D
		} else {
			userDeltas[it.UserID] = &deltaTraffic{u: it.U, d: it.D}
			userIDs = append(userIDs, it.UserID)
		}
	}

	if len(userIDs) == 0 {
		return 0, nil
	}

	users, err := s.repos.User.ListByIDs(s.db, userIDs)
	if err != nil {
		return 0, err
	}
	byID := make(map[int64]*model.User, len(users))
	for i := range users {
		byID[users[i].ID] = &users[i]
	}

	planIDs, err := s.planIDsContainingGroup(srv.GroupID)
	if err != nil {
		return 0, err
	}
	allowedPlans := make(map[int64]struct{}, len(planIDs))
	for _, id := range planIDs {
		allowedPlans[id] = struct{}{}
	}

	rate := srv.Rate
	if rate <= 0 {
		logger.L().Warn("uniproxy node rate non-positive, fallback to 1",
			zapS("server_id", fmt.Sprint(serverID)),
			zapS("rate", fmt.Sprint(srv.Rate)),
		)
		rate = 1
	}

	now := time.Now()
	today := now.Format("2006-01-02")
	var affectedTokens []string
	accepted := 0

	err = repo.WithTx(s.db, func(tx *gorm.DB) error {
		for uid, d := range userDeltas {
			u, ok := byID[uid]
			if !ok {
				continue
			}
			if !userAllowedOnNode(u, allowedPlans, now) {
				continue
			}

			bu := scaleRate(d.u, rate)
			bd := scaleRate(d.d, rate)
			if bu > 0 || bd > 0 {
				if err := s.repos.User.IncrTraffic(tx, u.ID, bu, bd); err != nil {
					return err
				}
				if err := s.repos.Traffic.AdditiveUpsert(tx, u.ID, today, bu, bd); err != nil {
					return err
				}
				affectedTokens = append(affectedTokens, u.SubToken)
			}
			accepted++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}

	// 受影响用户 userinfo 缓存立即失效
	for _, token := range affectedTokens {
		s.rdb.Del(ctx, redispkg.Key("sub", "userinfo", token))
	}

	return accepted, nil
}

// planIDsContainingGroup 查询分组归属套餐 ID 集合（复用现有可见性 Module）。
func (s *UniProxyService) planIDsContainingGroup(groupID int64) ([]int64, error) {
	plans, err := s.repos.Plan.ListAll(s.db)
	if err != nil {
		return nil, err
	}
	return plansContainingGroup(plans, groupID), nil
}
