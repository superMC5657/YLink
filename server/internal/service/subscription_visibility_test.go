package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"ylink-backend/internal/model"
)

// claim ica2-4：行为变更=订阅可见性三复制收敛深 Module（解析→过滤→映射归一）；
// 必须不变=无订阅空集、分组外不可见、过期/封禁不可见、存量下架套餐仍有效；
// 最怕的失败=空与 nil 漂移、隐藏规则改一处而另两处未动、订阅凭证归因丢失。
func TestVisibilityMatrix(t *testing.T) {
	// GroupIDs 解析：失败/空一律空集非 nil
	assert.NotNil(t, parsePlanGroups(""))
	assert.Empty(t, parsePlanGroups(""))
	assert.NotNil(t, parsePlanGroups("not-json"))
	assert.Empty(t, parsePlanGroups("not-json"))
	assert.NotNil(t, parsePlanGroups("[]"))
	assert.Empty(t, parsePlanGroups("[]"))
	assert.Equal(t, []int64{1, 2}, parsePlanGroups(`[1,2]`))

	// 分组→套餐：解析失败行跳过，命中只一次
	plans := []model.Plan{
		{ID: 1, GroupIDs: `[1,2]`},
		{ID: 2, GroupIDs: `bad`},
		{ID: 3, GroupIDs: `[2]`},
	}
	assert.Equal(t, []int64{1}, plansContainingGroup(plans, 1))
	assert.Equal(t, []int64{1, 3}, plansContainingGroup(plans, 2))
	assert.NotNil(t, plansContainingGroup(plans, 99))
	assert.Empty(t, plansContainingGroup(plans, 99))

	// 准入谓词矩阵
	now := time.Now()
	pid := int64(7)
	allowed := map[int64]struct{}{7: {}}
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)
	mk := func(planID *int64, banned bool, exp *time.Time) *model.User {
		return &model.User{PlanID: planID, IsBanned: banned, ExpiredAt: exp, UUID: "cred-uuid"}
	}
	assert.True(t, userAllowedOnNode(mk(&pid, false, &future), allowed, now))
	assert.True(t, userAllowedOnNode(mk(&pid, false, nil), allowed, now)) // 不限期
	assert.False(t, userAllowedOnNode(mk(nil, false, &future), allowed, now))
	other := int64(8)
	assert.False(t, userAllowedOnNode(mk(&other, false, &future), allowed, now))
	assert.False(t, userAllowedOnNode(mk(&pid, true, &future), allowed, now))
	assert.False(t, userAllowedOnNode(mk(&pid, false, &past), allowed, now))

	// 规格组装保留分组名与订阅凭证归因字段（UUID 由调用方经 BuildNodes 传入）
	servers := []model.Server{{ID: 1, GroupID: 5, Name: "n1", Type: "vless"}}
	specs := specsFromServers(servers, map[int64]string{5: "g5"})
	assert.Len(t, specs, 1)
	assert.Equal(t, "g5", specs[0].GroupName)
	assert.Equal(t, groupNameMap([]model.ServerGroup{{ID: 5, Name: "g5"}})[5], "g5")
}
