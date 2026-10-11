package service

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/subscribe"
	"ylink-backend/internal/repo"
)

// 订阅可见性深 Module：“该用户能看到哪些节点”规则的唯一拥有者。
//
// 三复制收敛（server_service.List / subscribe.buildNodes / node_service
// planIDsContainingGroup+Users+Report）：GroupIDs 解析→按分组 listing→
// 名称映射→过滤过期/封禁/无订阅，只活一次。三条路径问同一个问题、
// 拿同一种答案形状（含空集非 nil）。按用户归因随答案一起带走：
// 每用户独立订阅凭证 users.uuid，节点据此归因流量（模式A），不再只活在注释里。
//
// Depth 所在：空分组/解析失败/过期/封禁/隐藏的统一分类与空集语义。

// subscriptionVisibility 可见性深 Module 的窄 Interface。
type subscriptionVisibility struct {
	db    *gorm.DB
	repos *repo.Repos
}

// parsePlanGroups GroupIDs 解析唯一 owner：失败/空一律返回空集（非 nil），
// 调用方不再分支 nil。三处复制此前各写一遍 unmarshal。
func parsePlanGroups(raw string) []int64 {
	var groups []int64
	if err := json.Unmarshal([]byte(raw), &groups); err != nil || len(groups) == 0 {
		return []int64{}
	}
	return groups
}

// plansContainingGroup 全量套餐中 group_ids 含指定分组的套餐 id
// （含下架：存量订阅仍有效）。node 分组→套餐方向。
func plansContainingGroup(plans []model.Plan, groupID int64) []int64 {
	ids := make([]int64, 0, len(plans))
	for _, p := range plans {
		for _, g := range parsePlanGroups(p.GroupIDs) {
			if g == groupID {
				ids = append(ids, p.ID)
				break
			}
		}
	}
	return ids
}

// userAllowedOnNode 节点侧准入：套餐在允许集内、未封禁、未过期。
// Users 与 Report 共用同一谓词，不再各写一遍 if。
func userAllowedOnNode(u *model.User, allowedPlans map[int64]struct{}, now time.Time) bool {
	if u.PlanID == nil {
		return false
	}
	if _, ok := allowedPlans[*u.PlanID]; !ok {
		return false
	}
	if u.IsBanned {
		return false
	}
	if u.ExpiredAt != nil && u.ExpiredAt.Before(now) {
		return false
	}
	return true
}

// specsFromServers 行→订阅构建规格：名称映射 + 保留分组名，解析失败的
// 行由 subscribe.BuildNodes 显式丢弃并计数（调用方记 warn）。
func specsFromServers(servers []model.Server, groupName map[int64]string) []subscribe.ServerSpec {
	specs := make([]subscribe.ServerSpec, 0, len(servers))
	for _, srv := range servers {
		specs = append(specs, subscribe.ServerSpec{
			Name:      srv.Name,
			Type:      srv.Type,
			Host:      srv.Host,
			Port:      srv.Port,
			Config:    srv.Config,
			Rate:      srv.Rate,
			GroupName: groupName[srv.GroupID],
		})
	}
	return specs
}

// groupNameMap 分组 id→名称映射唯一 owner。
func groupNameMap(groups []model.ServerGroup) map[int64]string {
	m := make(map[int64]string, len(groups))
	for _, g := range groups {
		m[g.ID] = g.Name
	}
	return m
}
