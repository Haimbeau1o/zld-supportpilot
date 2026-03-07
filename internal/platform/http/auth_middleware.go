package http

import (
	"strings"

	stdhttp "net/http"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
)

type tokenParser interface {
	Parse(tokenString string) (identity.IdentityClaims, error)
}

type AuthMiddleware struct {
	tokenParser tokenParser
}

func NewAuthMiddleware(tokenParser tokenParser) AuthMiddleware {
	return AuthMiddleware{tokenParser: tokenParser}
}

func (middleware AuthMiddleware) RequireIdentity(next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
		authorizationHeader := strings.TrimSpace(request.Header.Get("Authorization"))
		if !strings.HasPrefix(authorizationHeader, "Bearer ") {
			writeError(writer, stdhttp.StatusUnauthorized, "unauthorized", "缺少有效的 Bearer Token")
			return
		}

		tokenString := strings.TrimSpace(strings.TrimPrefix(authorizationHeader, "Bearer "))
		claims, err := middleware.tokenParser.Parse(tokenString)
		if err != nil {
			writeError(writer, stdhttp.StatusUnauthorized, "unauthorized", "访问令牌无效或已过期")
			return
		}

		// 中间件只从令牌恢复最小身份边界，权限集合始终由服务端根据角色重建，避免令牌里出现重复且易漂移的数据源。
		identityContext := identity.IdentityContext{
			UserID:         claims.UserID,
			OrganizationID: claims.OrganizationID,
			Role:           claims.Role,
			Permissions:    identity.RolePermissions(claims.Role),
		}

		request = request.WithContext(withIdentityContext(request.Context(), identityContext))
		next.ServeHTTP(writer, request)
	})
}
