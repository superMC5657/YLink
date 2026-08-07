package repo

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"nanocloud/internal/model"
	"nanocloud/internal/pkg/passwd"
)

// EnsureAdmin 幂等初始化首个管理员：users 表无管理员且环境变量齐全时创建。
func EnsureAdmin(db *gorm.DB, email, password string) error {
	if email == "" || password == "" {
		return errors.New("ADMIN_EMAIL / ADMIN_PASSWORD 未设置")
	}
	var count int64
	if err := db.Model(&model.User{}).Where("role = ?", model.RoleAdmin).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hash, err := passwd.Hash(password)
	if err != nil {
		return err
	}
	admin := &model.User{
		Email:        email,
		PasswordHash: hash,
		Role:         model.RoleAdmin,
		SubToken:     uuid.NewString(),
	}
	return db.Create(admin).Error
}
