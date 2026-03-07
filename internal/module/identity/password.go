package identity

import "golang.org/x/crypto/bcrypt"

// BcryptPasswordManager 使用 bcrypt 处理密码，适合当前 API 驱动的后端场景。
type BcryptPasswordManager struct{}

func NewPasswordManager() BcryptPasswordManager {
	return BcryptPasswordManager{}
}

func (BcryptPasswordManager) Hash(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashedPassword), nil
}

func (BcryptPasswordManager) Compare(hashedPassword string, plainPassword string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword)) == nil
}
