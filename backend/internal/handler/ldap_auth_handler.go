package handler

import (
	"log/slog"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// LdapLoginRequest represents the LDAP login request payload
type LdapLoginRequest struct {
	Username       string `json:"username" binding:"required"`
	Password       string `json:"password" binding:"required"`
	TurnstileToken string `json:"turnstile_token"`
}

// LdapAuthHandler 处理 LDAP 登录请求
type LdapAuthHandler struct {
	ldapService    *service.LdapService
	authService    *service.AuthService
	totpService    *service.TotpService
	settingService *service.SettingService
}

// NewLdapAuthHandler 创建 LDAP 认证 Handler
func NewLdapAuthHandler(
	ldapService *service.LdapService,
	authService *service.AuthService,
	totpService *service.TotpService,
	settingService *service.SettingService,
) *LdapAuthHandler {
	return &LdapAuthHandler{
		ldapService:    ldapService,
		authService:    authService,
		totpService:    totpService,
		settingService: settingService,
	}
}

// Login handles LDAP login requests
func (h *LdapAuthHandler) Login(c *gin.Context) {
	var req LdapLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_REQUEST", err.Error()))
		return
	}

	if err := h.authService.VerifyTurnstile(c.Request.Context(), req.TurnstileToken, ip.GetClientIP(c)); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	user, err := h.ldapService.Authenticate(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	if !user.IsActive() {
		response.ErrorFrom(c, service.ErrUserNotActive)
		return
	}

	// Check if TOTP 2FA is enabled for this user
	if h.totpService != nil && h.settingService.IsTotpEnabled(c.Request.Context()) && user.TotpEnabled {
		// Create a temporary login session for 2FA
		tempToken, err := h.totpService.CreateLoginSession(c.Request.Context(), user.ID, user.Email)
		if err != nil {
			response.ErrorFrom(c, infraerrors.InternalServer("2FA_SESSION_FAILED", "failed to create 2FA session"))
			return
		}

		response.Success(c, TotpLoginResponse{
			Requires2FA:     true,
			TempToken:       tempToken,
			UserEmailMasked: service.MaskEmail(user.Email),
		})
		return
	}

	tokenPair, err := h.authService.GenerateTokenPair(c.Request.Context(), user, "")
	if err != nil {
		slog.Error("ldap_generate_token_pair_failed", "error", err, "user_id", user.ID)
		accessToken, tokenErr := h.authService.GenerateToken(user)
		if tokenErr != nil {
			response.ErrorFrom(c, tokenErr)
			return
		}

		response.Success(c, AuthResponse{
			AccessToken: accessToken,
			TokenType:   "Bearer",
			User:        dto.UserFromService(user),
		})
		return
	}

	response.Success(c, AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
		TokenType:    "Bearer",
		User:         dto.UserFromService(user),
	})
}
