// Package passwd 提供密码哈希（bcrypt cost 12，满足安全清单）。
package passwd

import "golang.org/x/crypto/bcrypt"

// Hash 生成密码哈希。
func Hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(b), err
}

// Verify 校验密码与哈希是否匹配。
func Verify(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
