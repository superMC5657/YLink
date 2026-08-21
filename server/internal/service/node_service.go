package service

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	redispkg "ylink-backend/internal/pkg/redis"
	"ylink-backend/internal/repo"
)

// NodeService 流量模式 A:节点上报域(用户同步 / 流量上报,见 core-flows.md §8)。
type NodeService struct {
	db    *gorm.DB
	rdb   *redis.Client
	repos *repo.Repos
}

func NewNodeService(db *gorm.DB, rdb *redis.Client, repos *repo.Repos) *NodeService {
	return &NodeService{db: db, rdb: rdb, repos: repos}
}

// ServerIDByKey X-Node-Key → 节点 id(缓存由 NodeAuth 中间件负责,本方法纯 DB 查询)。
func (s *NodeService) ServerIDByKey(ctx context.Context, key string) (int64, error) {
	srv, err := s.repos.Server.GetByNodeKey(s.db, key)
	if err != nil {
		return 0, err
	}
	return srv.ID, nil
}

// ---- GET /node/users ----

// NodeUserItem 节点侧可见用户(配置 inbound 凭证 + 本地掐断参考)。
type NodeUserItem struct {
	UUID           string `json:"uuid"`
	U              int64  `json:"u"`
	D              int64  `json:"d"`
	TransferEnable int64  `json:"transfer_enable"`
	ExpiredAt      *int64 `json:"expired_at"` // unix 秒,nil=不限期
}

// NodeUsersResp GET /node/users。
type NodeUsersResp struct {
	Rate  float64        `json:"rate"`
	Users []NodeUserItem `json:"users"`
}

// Users 返回该节点分组下有效订阅且未封禁的用户(套餐 group_ids 含本节点分组、未过期)。
func (s *NodeService) Users(ctx context.Context, serverID int64) (*NodeUsersResp, error) {
	srv, err := s.repos.Server.GetByID(s.db, serverID)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	planIDs, err := s.planIDsContainingGroup(srv.GroupID)
	if err != nil {
		return nil, err
	}
	resp := &NodeUsersResp{Rate: srv.Rate, Users: []NodeUserItem{}}
	if len(planIDs) == 0 {
		return resp, nil
	}
	users, err := s.repos.User.ListActiveByPlanIDs(s.db, planIDs, time.Now())
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		item := NodeUserItem{
			UUID:           u.UUID,
			U:              u.U,
			D:              u.D,
			TransferEnable: u.TransferEnable,
		}
		if u.ExpiredAt != nil {
			unix := u.ExpiredAt.Unix()
			item.ExpiredAt = &unix
		}
		resp.Users = append(resp.Users, item)
	}
	return resp, nil
}

// planIDsContainingGroup 全量套餐中 group_ids 含指定分组的套餐 id(含下架:存量订阅仍有效)。
func (s *NodeService) planIDsContainingGroup(groupID int64) ([]int64, error) {
	plans, err := s.repos.Plan.ListAll(s.db)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(plans))
	for _, p := range plans {
		var groups []int64
		if json.Unmarshal([]byte(p.GroupIDs), &groups) != nil {
			continue
		}
		for _, g := range groups {
			if g == groupID {
				ids = append(ids, p.ID)
				break
			}
		}
	}
	return ids, nil
}

// ---- POST /node/report ----

// NodeReportItem 上报项:u/d 为自 agent 启动起的累计值(字节)。
type NodeReportItem struct {
	UUID string `json:"uuid" binding:"required"`
	U    int64  `json:"u" binding:"min=0"`
	D    int64  `json:"d" binding:"min=0"`
}

// NodeReportReq POST /node/report 请求。
type NodeReportReq struct {
	Data []NodeReportItem `json:"data" binding:"required,min=1,max=1000,dive"`
}

// NodeSkipItem 跳过项及原因(unknown_user / not_subscribed)。
type NodeSkipItem struct {
	UUID   string `json:"uuid"`
	Reason string `json:"reason"`
}

// NodeReportResp POST /node/report 响应。
type NodeReportResp struct {
	Accepted int            `json:"accepted"`
	Skipped  []NodeSkipItem `json:"skipped"`
}

// Report 流量上报:快照差分(幂等)→ 乘倍率 → 累加 users.u/d + traffic_logs 日聚合 → 清 userinfo 缓存。
func (s *NodeService) Report(ctx context.Context, serverID int64, req *NodeReportReq) (*NodeReportResp, error) {
	srv, err := s.repos.Server.GetByID(s.db, serverID)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	uuids := make([]string, 0, len(req.Data))
	for _, it := range req.Data {
		uuids = append(uuids, it.UUID)
	}
	users, err := s.repos.User.ListByUUIDs(s.db, uuids)
	if err != nil {
		return nil, err
	}
	byUUID := make(map[string]*model.User, len(users))
	for i := range users {
		byUUID[users[i].UUID] = &users[i]
	}

	now := time.Now()
	today := now.Format("2006-01-02")
	resp := &NodeReportResp{Skipped: []NodeSkipItem{}}
	var affected []model.User

	err = repo.WithTx(s.db, func(tx *gorm.DB) error {
		for _, it := range req.Data {
			u, ok := byUUID[it.UUID]
			if !ok {
				resp.Skipped = append(resp.Skipped, NodeSkipItem{UUID: it.UUID, Reason: "unknown_user"})
				continue
			}
			if u.PlanID == nil || u.IsBanned || (u.ExpiredAt != nil && u.ExpiredAt.Before(now)) {
				resp.Skipped = append(resp.Skipped, NodeSkipItem{UUID: it.UUID, Reason: "not_subscribed"})
				continue
			}
			st, err := s.repos.NodeStat.GetForUpdate(tx, serverID, u.ID)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			var lastU, lastD int64
			if st != nil {
				lastU, lastD = st.LastU, st.LastD
			}
			du, dd := cumDelta(it.U, lastU), cumDelta(it.D, lastD)
			bu, bd := scaleRate(du, srv.Rate), scaleRate(dd, srv.Rate)
			if bu > 0 || bd > 0 {
				if err := s.repos.User.IncrTraffic(tx, u.ID, bu, bd); err != nil {
					return err
				}
				if err := s.repos.Traffic.AdditiveUpsert(tx, u.ID, today, bu, bd); err != nil {
					return err
				}
				affected = append(affected, *u)
			}
			if err := s.repos.NodeStat.Upsert(tx, serverID, u.ID, it.U, it.D); err != nil {
				return err
			}
			resp.Accepted++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// 受影响用户 userinfo 缓存立即失效,客户端下次拉订阅即见新用量
	for _, u := range affected {
		s.rdb.Del(ctx, redispkg.Key("sub", "userinfo", u.SubToken))
	}
	return resp, nil
}

// cumDelta 累计值差分:cur < last 视为节点计数器重启,增量取当前值(对侧不变仍走差分)。
func cumDelta(cur, last int64) int64 {
	if cur < last {
		return cur
	}
	return cur - last
}

// scaleRate 增量乘节点倍率,四舍五入。
func scaleRate(v int64, rate float64) int64 {
	if v <= 0 || rate <= 0 {
		return 0
	}
	return int64(math.Round(float64(v) * rate))
}
