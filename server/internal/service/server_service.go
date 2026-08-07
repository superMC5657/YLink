package service

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"nanocloud/internal/model"
	"nanocloud/internal/pkg/errs"
	"nanocloud/internal/repo"
)

// ServerService 节点列表：仅返回当前用户套餐可见分组（不返回连接参数）。
type ServerService struct {
	db    *gorm.DB
	rdb   *redis.Client
	repos *repo.Repos
}

func NewServerService(db *gorm.DB, rdb *redis.Client, repos *repo.Repos) *ServerService {
	return &ServerService{db: db, rdb: rdb, repos: repos}
}

// List GET /servers。
func (s *ServerService) List(ctx context.Context, userID int64) ([]model.ServerGroupResp, error) {
	user, err := s.repos.User.GetByID(s.db, userID)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	// 无订阅 → 空列表
	if user.PlanID == nil {
		return []model.ServerGroupResp{}, nil
	}
	plan, err := s.repos.Plan.GetByID(s.db, *user.PlanID)
	if err != nil {
		return []model.ServerGroupResp{}, nil
	}
	var groupIDs []int64
	if err := json.Unmarshal([]byte(plan.GroupIDs), &groupIDs); err != nil || len(groupIDs) == 0 {
		return []model.ServerGroupResp{}, nil
	}

	servers, err := s.repos.Server.ListByGroupIDs(s.db, groupIDs)
	if err != nil {
		return nil, err
	}
	groups, err := s.repos.Server.ListGroups(s.db)
	if err != nil {
		return nil, err
	}

	// group_id → 名称
	nameOf := map[int64]string{}
	for _, g := range groups {
		nameOf[g.ID] = g.Name
	}
	// 保序分组（按套餐 group_ids 顺序）
	var out []model.ServerGroupResp
	idx := map[int64]int{}
	for _, gid := range groupIDs {
		if _, ok := nameOf[gid]; !ok {
			continue
		}
		if _, ok := idx[gid]; !ok {
			out = append(out, model.ServerGroupResp{Group: nameOf[gid], Servers: []model.ServerResp{}})
			idx[gid] = len(out) - 1
		}
	}
	for _, srv := range servers {
		i, ok := idx[srv.GroupID]
		if !ok {
			continue
		}
		var tags []string
		if srv.Tags != nil {
			_ = json.Unmarshal([]byte(*srv.Tags), &tags)
		}
		out[i].Servers = append(out[i].Servers, model.ServerResp{
			ID: srv.ID, Name: srv.Name, Type: srv.Type, Rate: srv.Rate, Status: srv.Status, Tags: tags,
		})
	}
	return out, nil
}
