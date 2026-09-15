package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	"ylink-backend/internal/repo"
)

// ---- 审计日志（F08） ----

// AuditLogFilter 审计日志筛选参数（handler 解析后的 query）。
type AuditLogFilter struct {
	AdminID *int64
	Action  string
	Target  string
	From    string // YYYY-MM-DD，可空
	To      string // YYYY-MM-DD，可空（含当天）
}

// ListAuditLogs GET /admin/audit-logs：审计日志分页查询（只读，数据源为 audit_logs 表）。
func (s *AdminService) ListAuditLogs(ctx context.Context, f AuditLogFilter, page, pageSize int) ([]model.AdminAuditLogItem, int64, []string, error) {
	q := repo.AuditLogQuery{AdminID: f.AdminID, Action: f.Action, Target: f.Target}
	parseDay := func(s string) (time.Time, error) {
		return time.ParseInLocation("2006-01-02", s, time.Local)
	}
	if f.From != "" {
		t, err := parseDay(f.From)
		if err != nil {
			return nil, 0, nil, errs.New(40000, "参数校验失败: from 需为 YYYY-MM-DD")
		}
		q.From = &t
	}
	if f.To != "" {
		t, err := parseDay(f.To)
		if err != nil {
			return nil, 0, nil, errs.New(40000, "参数校验失败: to 需为 YYYY-MM-DD")
		}
		t = t.AddDate(0, 0, 1) // 含 to 当天
		q.To = &t
	}
	list, total, err := s.repos.Audit.ListByPage(s.db, q, page, pageSize)
	if err != nil {
		return nil, 0, nil, err
	}
	s.enrichAuditTargets(list)
	actions, err := s.repos.Audit.ListActions(s.db)
	if err != nil {
		return list, total, nil, nil // 动作列表失败不阻断主查询
	}
	return list, total, actions, nil
}

// ---- 审计日志目标可读化（F08 增强）----

// auditTargetKind 按 action 判定 target 指向的实体类型；空串表示无实体或未收录动作。
// 与各写入点的 s.audit(...) 调用保持一致（admin_service / admin_crud / admin_node_batch 等）。
func auditTargetKind(action string) string {
	switch action {
	case "ban_user", "update_role", "adjust_balance", "reset_sub_token",
		"agent_approve", "agent_reject", "withdraw_pay", "withdraw_reject":
		return "user"
	case "send_mail", "traffic_reset": // target 为摘要 "batch:<count>"（完整 ID 列表在 detail）
		return "users"
	case "reset_node_key", "copy_server":
		return "server"
	case "create_knowledge_category", "update_knowledge_category", "delete_knowledge_category":
		return "knowledge_category"
	case "close_order", "refund": // target 为订单号，本身可读
		return "order"
	case "edit_mail_template", "reset_mail_template", "test_mail_template": // target 为模板名
		return "mail_template"
	default:
		return ""
	}
}

// auditTargetIDs 解析 target 中的单个 ID（当前格式为 fmt.Sprint(userID)）；非 ID 内容返回空。
func auditTargetIDs(target string) []int64 {
	if id, err := strconv.ParseInt(target, 10, 64); err == nil {
		return []int64{id}
	}
	return nil
}

// auditDetailEmail 从审计 detail JSON 中提取 email 字段（ban_user / update_role /
// adjust_balance / reset_sub_token 写入时留痕了操作对象邮箱）。解析失败返回空串。
func auditDetailEmail(detail *string) string {
	if detail == nil || *detail == "" {
		return ""
	}
	var m map[string]any
	if json.Unmarshal([]byte(*detail), &m) != nil {
		return ""
	}
	e, _ := m["email"].(string)
	return e
}

// enrichAuditTargets 把本页日志的 target 翻译成实体类型 + 可读名称（用户邮箱/节点名/分类名）。
// 纯展示增强：任何查询失败或实体已删除都静默跳过（TargetDisplay 保持 nil，前端回退原始 target），绝不影响主查询。
func (s *AdminService) enrichAuditTargets(list []model.AdminAuditLogItem) {
	idsByKind := map[string][]int64{}
	for i := range list {
		kind := auditTargetKind(list[i].Action)
		if kind == "" || list[i].Target == nil || *list[i].Target == "" {
			continue
		}
		list[i].TargetKind = &kind
		switch kind {
		case "user", "users":
			if strings.HasPrefix(*list[i].Target, "batch:") {
				// 批量动作摘要（batch:<count>）：完整 ID 列表在 detail JSON，不反查实体，直接透出摘要
				t := *list[i].Target
				list[i].TargetDisplay = &t
				continue
			}
			idsByKind["user"] = append(idsByKind["user"], auditTargetIDs(*list[i].Target)...)
		case "server":
			idsByKind["server"] = append(idsByKind["server"], auditTargetIDs(*list[i].Target)...)
		case "knowledge_category":
			idsByKind["knowledge_category"] = append(idsByKind["knowledge_category"], auditTargetIDs(*list[i].Target)...)
		default: // order / mail_template：订单号与模板名本身可读，直接透出
			t := *list[i].Target
			list[i].TargetDisplay = &t
		}
	}
	emails := s.emailsOf(idsByKind["user"])
	servers := s.serverNamesOf(idsByKind["server"])
	cats := s.categoryNamesOf(idsByKind["knowledge_category"])
	for i := range list {
		if list[i].TargetKind == nil || list[i].TargetDisplay != nil || list[i].Target == nil {
			continue
		}
		var lookup map[int64]string
		switch *list[i].TargetKind {
		case "user", "users":
			lookup = emails
		case "server":
			lookup = servers
		case "knowledge_category":
			lookup = cats
		default:
			continue
		}
		ids := auditTargetIDs(*list[i].Target)
		names := make([]string, 0, len(ids))
		for _, id := range ids {
			if name, ok := lookup[id]; ok && name != "" {
				names = append(names, name)
			}
		}
		if len(names) == 0 {
			// 实体查不到（如用户已被删除）：单用户动作尝试用 detail 里留痕的邮箱兜底
			if *list[i].TargetKind == "user" && len(ids) == 1 {
				if e := auditDetailEmail(list[i].Detail); e != "" {
					list[i].TargetDisplay = &e
				}
			}
			continue // 仍解析不出则回退原始 target
		}
		if len(names) > 3 {
			names = append(names[:3], fmt.Sprintf("…(+%d)", len(names)-3))
		}
		d := strings.Join(names, ", ")
		list[i].TargetDisplay = &d
	}
}

// serverNamesOf 批量查节点名（id → name），查询失败返回空 map（不阻断展示）。
func (s *AdminService) serverNamesOf(ids []int64) map[int64]string {
	out := map[int64]string{}
	if len(ids) == 0 {
		return out
	}
	var servers []model.Server
	if err := s.db.Select("id, name").Where("id IN ?", ids).Find(&servers).Error; err != nil {
		return out
	}
	for _, sv := range servers {
		out[sv.ID] = sv.Name
	}
	return out
}

// categoryNamesOf 批量查知识分类名（id → name），查询失败返回空 map（不阻断展示）。
func (s *AdminService) categoryNamesOf(ids []int64) map[int64]string {
	out := map[int64]string{}
	if len(ids) == 0 {
		return out
	}
	var cats []model.KnowledgeCategory
	if err := s.db.Select("id, name").Where("id IN ?", ids).Find(&cats).Error; err != nil {
		return out
	}
	for _, c := range cats {
		out[c.ID] = c.Name
	}
	return out
}
