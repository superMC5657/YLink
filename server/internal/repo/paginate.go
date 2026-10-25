package repo

import "gorm.io/gorm"

// listPage 分页读取唯一形状：先 Count 总数，再按 orderBy（缺省 id 倒序）取当页。
// 收敛前 11 处手写 Count + Order + Offset/Limit 的重复；查询语义与各处原实现一致。
func listPage[T any](q *gorm.DB, page, pageSize int, orderBy string) ([]T, int64, error) {
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if orderBy == "" {
		orderBy = "id DESC"
	}
	var list []T
	err := q.Order(orderBy).Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}
