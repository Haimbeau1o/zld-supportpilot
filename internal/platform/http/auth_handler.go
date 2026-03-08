package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	stdhttp "net/http"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
)

type authService interface {
	Register(input identity.RegisterInput) (identity.User, identity.Membership, error)
	Login(input identity.LoginInput) (identity.IdentityContext, error)
}

type tokenIssuerParser interface {
	Issue(identityContext identity.IdentityContext) (string, error)
	Parse(tokenString string) (identity.IdentityClaims, error)
}

type AuthDependencies struct {
	IdentityService authService
	TokenManager    tokenIssuerParser
}

type AuthHandler struct {
	identityService authService
	tokenManager    tokenIssuerParser
}

type registerRequest struct {
	Name       string        `json:"name"`
	Email      string        `json:"email"`
	Password   string        `json:"password"`
	TenantName string        `json:"tenant_name"`
	Role       identity.Role `json:"role"`
}

type registerResponse struct {
	UserID         string        `json:"user_id"`
	OrganizationID string        `json:"organization_id"`
	Role           identity.Role `json:"role"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

type meResponse struct {
	UserID         string                `json:"user_id"`
	OrganizationID string                `json:"organization_id"`
	Role           identity.Role         `json:"role"`
	Permissions    []identity.Permission `json:"permissions"`
}

type errorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorResponse struct {
	Error errorPayload `json:"error"`
}

func NewAuthHandler(identityService authService, tokenManager tokenIssuerParser) AuthHandler {
	return AuthHandler{
		identityService: identityService,
		tokenManager:    tokenManager,
	}
}

func (handler AuthHandler) Register(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	if request.Method != stdhttp.MethodPost {
		writeError(writer, stdhttp.StatusMethodNotAllowed, "method_not_allowed", "请求方法不被支持")
		return
	}

	var input registerRequest
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		writeError(writer, stdhttp.StatusBadRequest, "invalid_request", "请求体不是合法的 JSON")
		return
	}

	user, membership, err := handler.identityService.Register(identity.RegisterInput{
		Name:        input.Name,
		Email:       input.Email,
		Password:    input.Password,
		TenantName:  input.TenantName,
		DefaultRole: input.Role,
	})
	if err != nil {
		switch {
		case errors.Is(err, identity.ErrEmailAlreadyExists):
			writeError(writer, stdhttp.StatusConflict, "email_exists", "邮箱已被注册")
		default:
			writeError(writer, stdhttp.StatusInternalServerError, "internal_error", "注册失败")
		}
		return
	}

	// 注册接口只返回建立身份边界所必需的信息，不在这里直接隐式登录，避免后续会话语义混乱。
	writeJSON(writer, stdhttp.StatusCreated, registerResponse{
		UserID:         user.ID,
		OrganizationID: membership.OrganizationID,
		Role:           membership.Role,
	})
}

func (handler AuthHandler) Login(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	if request.Method != stdhttp.MethodPost {
		writeError(writer, stdhttp.StatusMethodNotAllowed, "method_not_allowed", "请求方法不被支持")
		return
	}

	var input loginRequest
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		writeError(writer, stdhttp.StatusBadRequest, "invalid_request", "请求体不是合法的 JSON")
		return
	}

	identityContext, err := handler.identityService.Login(identity.LoginInput{
		Email:    input.Email,
		Password: input.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, identity.ErrInvalidCredentials):
			writeError(writer, stdhttp.StatusUnauthorized, "invalid_credentials", "邮箱或密码错误")
		default:
			writeError(writer, stdhttp.StatusInternalServerError, "internal_error", "登录失败")
		}
		return
	}

	accessToken, err := handler.tokenManager.Issue(identityContext)
	if err != nil {
		writeError(writer, stdhttp.StatusInternalServerError, "internal_error", "签发令牌失败")
		return
	}

	claims, err := handler.tokenManager.Parse(accessToken)
	if err != nil || claims.ExpiresAt == nil {
		writeError(writer, stdhttp.StatusInternalServerError, "internal_error", "登录响应构建失败")
		return
	}

	writeJSON(writer, stdhttp.StatusOK, loginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   maxInt64(1, int64(time.Until(claims.ExpiresAt.Time).Seconds())),
	})
}

func (handler AuthHandler) Me(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	if request.Method != stdhttp.MethodGet {
		writeError(writer, stdhttp.StatusMethodNotAllowed, "method_not_allowed", "请求方法不被支持")
		return
	}

	identityContext, ok := identityContextFromContext(request.Context())
	if !ok {
		writeError(writer, stdhttp.StatusUnauthorized, "unauthorized", "当前请求缺少身份上下文")
		return
	}

	writeJSON(writer, stdhttp.StatusOK, meResponse{
		UserID:         identityContext.UserID,
		OrganizationID: identityContext.OrganizationID,
		Role:           identityContext.Role,
		Permissions:    sortedPermissions(identityContext.Permissions),
	})
}

func writeJSON(writer stdhttp.ResponseWriter, statusCode int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	_ = json.NewEncoder(writer).Encode(payload)
}

func writeError(writer stdhttp.ResponseWriter, statusCode int, code, message string) {
	writeJSON(writer, statusCode, errorResponse{
		Error: errorPayload{
			Code:    code,
			Message: message,
		},
	})
}

func sortedPermissions(permissionSet identity.PermissionSet) []identity.Permission {
	permissions := make([]identity.Permission, 0, len(permissionSet))
	for permission := range permissionSet {
		permissions = append(permissions, permission)
	}

	sort.Slice(permissions, func(left, right int) bool {
		return permissions[left] < permissions[right]
	})

	return permissions
}

func maxInt64(left, right int64) int64 {
	if left > right {
		return left
	}

	return right
}

func registerAuthRoutes(mux *stdhttp.ServeMux, runtime routeRuntime, authDependencies AuthDependencies) error {
	if authDependencies.IdentityService == nil {
		return fmt.Errorf("identity service is required")
	}
	if authDependencies.TokenManager == nil {
		return fmt.Errorf("token manager is required")
	}

	authHandler := NewAuthHandler(authDependencies.IdentityService, authDependencies.TokenManager)
	authMiddleware := NewAuthMiddleware(authDependencies.TokenManager)

	mux.Handle("POST /api/v1/auth/register", runtime.public("POST /api/v1/auth/register", stdhttp.HandlerFunc(authHandler.Register)))
	// 登录接口属于高风险公共入口，当前阶段先按客户端地址做进程内限流，用于体现基础暴力尝试保护边界。
	mux.Handle("POST /api/v1/auth/login", runtime.publicRateLimited("POST /api/v1/auth/login", clientAddressRateLimitKey, stdhttp.HandlerFunc(authHandler.Login)))
	mux.Handle("GET /api/v1/auth/me", runtime.protected("GET /api/v1/auth/me", authMiddleware, stdhttp.HandlerFunc(authHandler.Me)))
	return nil
}
