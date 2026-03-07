package identity

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// IdentityClaims 是在 JWT 中传递的最小身份信息集合。
// 这里只放后续模块真正需要的身份边界，避免把无关字段塞进令牌。
type IdentityClaims struct {
	UserID         string `json:"user_id"`
	OrganizationID string `json:"organization_id"`
	Role           Role   `json:"role"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	signingKey []byte
	tokenTTL   time.Duration
}

func NewTokenManager(signingKey string, tokenTTL time.Duration) TokenManager {
	return TokenManager{
		signingKey: []byte(signingKey),
		tokenTTL:   tokenTTL,
	}
}

func (manager TokenManager) Issue(identityContext IdentityContext) (string, error) {
	now := time.Now()
	claims := IdentityClaims{
		UserID:         identityContext.UserID,
		OrganizationID: identityContext.OrganizationID,
		Role:           identityContext.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(manager.tokenTTL)),
		},
	}

	// 令牌只使用单一签名密钥，当前阶段先把边界稳定下来，后续再补轮换策略。
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(manager.signingKey)
}

func (manager TokenManager) Parse(tokenString string) (IdentityClaims, error) {
	parsedToken, err := jwt.ParseWithClaims(tokenString, &IdentityClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Method.Alg())
		}

		return manager.signingKey, nil
	})
	if err != nil {
		return IdentityClaims{}, err
	}

	claims, ok := parsedToken.Claims.(*IdentityClaims)
	if !ok || !parsedToken.Valid {
		return IdentityClaims{}, fmt.Errorf("invalid token claims")
	}

	return *claims, nil
}
