package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// UserHandler handles user-related requests
type UserHandler struct {
	userService *service.UserService
	authService *service.AuthService
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(userService *service.UserService, authService *service.AuthService) *UserHandler {
	return &UserHandler{
		userService: userService,
		authService: authService,
	}
}

// ChangePasswordRequest represents the change password request payload
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// SetPasswordRequest represents the set password request payload (for OAuth users without a password)
type SetPasswordRequest struct {
	Email       string `json:"email" binding:"required,email"`
	VerifyCode  string `json:"verify_code" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// UpdateProfileRequest represents the update profile request payload
type UpdateProfileRequest struct {
	Username *string `json:"username"`
}

// GetProfile handles getting user profile
// GET /api/v1/users/me
func (h *UserHandler) GetProfile(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	userData, err := h.userService.GetByID(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.UserFromService(userData))
}

// ChangePassword handles changing user password
// POST /api/v1/users/me/password
func (h *UserHandler) ChangePassword(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	svcReq := service.ChangePasswordRequest{
		CurrentPassword: req.OldPassword,
		NewPassword:     req.NewPassword,
	}
	err := h.userService.ChangePassword(c.Request.Context(), subject.UserID, svcReq)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Password changed successfully"})
}

// SendSetPasswordCode sends a verification code for setting password
// POST /api/v1/user/send-set-password-code
func (h *UserHandler) SendSetPasswordCode(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	userData, err := h.userService.GetByID(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	if userData.HasPassword {
		response.ErrorFrom(c, infraerrors.BadRequest("ALREADY_HAS_PASSWORD", "user already has a password"))
		return
	}

	// 检查邮箱是否为合成邮箱（需要先绑定真实邮箱）
	email := strings.ToLower(userData.Email)
	if strings.HasSuffix(email, "@wechat-auth.invalid") || strings.HasSuffix(email, "@linuxdo.invalid") {
		response.ErrorFrom(c, infraerrors.BadRequest("BIND_EMAIL_FIRST", "please bind a real email first"))
		return
	}

	if err := h.authService.SendBindEmailCode(c.Request.Context(), userData.Email); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Verification code sent", "countdown": 60})
}

// SetPassword handles setting a password for OAuth users who don't have one
// POST /api/v1/user/set-password
func (h *UserHandler) SetPassword(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req SetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 验证用户邮箱是否匹配
	userData, err := h.userService.GetByID(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	if userData.Email != req.Email {
		response.BadRequest(c, "Email does not match your account")
		return
	}

	// 验证邮箱验证码
	if err := h.authService.VerifyEmailCode(c.Request.Context(), req.Email, req.VerifyCode); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	svcReq := service.SetPasswordRequest{
		NewPassword: req.NewPassword,
	}
	if err := h.userService.SetInitialPassword(c.Request.Context(), subject.UserID, svcReq); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Password set successfully"})
}

// UpdateProfile handles updating user profile
// PUT /api/v1/users/me
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	svcReq := service.UpdateProfileRequest{
		Username: req.Username,
	}
	updatedUser, err := h.userService.UpdateProfile(c.Request.Context(), subject.UserID, svcReq)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.UserFromService(updatedUser))
}
