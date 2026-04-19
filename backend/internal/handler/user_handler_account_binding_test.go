package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newUserBindingHandlerForTest(t *testing.T) (*UserHandler, *service.AuthService, *pendingAuthHandlerUserRepoStub, string) {
	t.Helper()

	repo := newPendingAuthHandlerUserRepoStub()
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:                   "user-binding-secret",
			ExpireHour:               1,
			AccessTokenExpireMinutes: 60,
			RefreshTokenExpireDays:   7,
		},
	}
	user := &service.User{
		ID:          7,
		Email:       "owner@example.com",
		Role:        service.RoleUser,
		Status:      service.StatusActive,
		Concurrency: 3,
	}
	repo.users[user.ID] = user
	repo.usersByMail[user.Email] = user

	authSvc := service.NewAuthService(nil, repo, nil, pendingAuthRefreshCacheStub{}, cfg, nil, nil, nil, nil, nil, nil)
	userSvc := service.NewUserService(repo, nil, nil, nil)
	handler := NewUserHandler(userSvc, authSvc, nil, nil)

	pendingToken, err := authSvc.CreatePendingAuthSession(context.Background(), service.PendingAuthSessionInput{
		Intent:          service.PendingAuthIntentBindCurrentUser,
		ProviderType:    "wechat",
		ProviderKey:     "wechat-main",
		ProviderSubject: "union-1",
		TargetUserID:    &user.ID,
		Metadata: map[string]any{
			"unionid": "union-1",
		},
	})
	require.NoError(t, err)

	session, err := authSvc.GetPendingAuthSessionForProgress(context.Background(), pendingToken, nil)
	require.NoError(t, err)
	now := time.Now()
	session.EmailVerifiedAt = &now
	session.PasswordVerifiedAt = &now
	require.NoError(t, repo.UpdatePendingAuthSession(context.Background(), session))

	return handler, authSvc, repo, pendingToken
}

func TestUserHandler_ConfirmAccountBinding_BindsCurrentUserWithPendingAuthToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, authSvc, _, pendingToken := newUserBindingHandlerForTest(t)

	body := bytes.NewBufferString(`{"pending_auth_token":"` + pendingToken + `"}`)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})
	ctx.Params = gin.Params{{Key: "provider", Value: "wechat"}}
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/user/account-bindings/wechat", body)
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler.ConfirmAccountBinding(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			User struct {
				ID int64 `json:"id"`
			} `json:"user"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, int64(7), resp.Data.User.ID)

	_, err := authSvc.GetPendingAuthSessionForProgress(context.Background(), pendingToken, nil)
	require.ErrorIs(t, err, service.ErrInvalidToken)
}

func TestUserHandler_ConfirmAccountBinding_AcceptsLegacyPendingOAuthTokenField(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, authSvc, _, pendingToken := newUserBindingHandlerForTest(t)

	body := bytes.NewBufferString(`{"pending_oauth_token":"` + pendingToken + `"}`)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})
	ctx.Params = gin.Params{{Key: "provider", Value: "wechat"}}
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/user/account-bindings/wechat", body)
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler.ConfirmAccountBinding(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	_, err := authSvc.GetPendingAuthSessionForProgress(context.Background(), pendingToken, nil)
	require.ErrorIs(t, err, service.ErrInvalidToken)
}

func TestUserHandler_CompleteIdentityAdoptionDecision_StoresChoiceOnPendingSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, authSvc, _, pendingToken := newUserBindingHandlerForTest(t)

	body := bytes.NewBufferString(`{"pending_auth_token":"` + pendingToken + `","adopt_display_name":true,"adopt_avatar":false}`)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})
	ctx.Params = gin.Params{{Key: "provider", Value: "wechat"}}
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/user/account-bindings/wechat/adoption-decision", body)
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler.CompleteIdentityAdoptionDecision(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	session, err := authSvc.GetPendingAuthSessionForProgress(context.Background(), pendingToken, nil)
	require.NoError(t, err)
	require.Equal(t, true, session.Metadata["adopt_display_name"])
	require.Equal(t, false, session.Metadata["adopt_avatar"])
}

func TestUserHandler_ConfirmAccountBinding_RequiresCompleted2FAForTotpUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, authSvc, repo, pendingToken := newUserBindingHandlerForTest(t)
	repo.users[7].TotpEnabled = true

	body := bytes.NewBufferString(`{"pending_auth_token":"` + pendingToken + `"}`)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})
	ctx.Params = gin.Params{{Key: "provider", Value: "wechat"}}
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/user/account-bindings/wechat", body)
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler.ConfirmAccountBinding(ctx)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	session, err := authSvc.GetPendingAuthSessionForProgress(context.Background(), pendingToken, nil)
	require.NoError(t, err)
	require.Nil(t, session.TOTPVerifiedAt)
	require.Nil(t, session.ConsumedAt)
}

func TestUserHandler_CompleteIdentityAdoptionDecision_RejectsTargetMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, authSvc, _, pendingToken := newUserBindingHandlerForTest(t)

	body := bytes.NewBufferString(`{"pending_auth_token":"` + pendingToken + `","adopt_display_name":true,"adopt_avatar":true}`)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 8})
	ctx.Params = gin.Params{{Key: "provider", Value: "wechat"}}
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/user/account-bindings/wechat/adoption-decision", body)
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler.CompleteIdentityAdoptionDecision(ctx)

	require.Equal(t, http.StatusForbidden, rec.Code)
	session, err := authSvc.GetPendingAuthSessionForProgress(context.Background(), pendingToken, nil)
	require.NoError(t, err)
	_, hasDisplayName := session.Metadata["adopt_display_name"]
	_, hasAvatar := session.Metadata["adopt_avatar"]
	require.False(t, hasDisplayName)
	require.False(t, hasAvatar)
}

func TestUserHandler_ConfirmAccountBinding_BindsEmailThroughVerificationPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _, repo, _ := newUserBindingHandlerForTest(t)

	body := bytes.NewBufferString(`{"email":"bound@example.com","verify_code":"123456","password":"secret-123"}`)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})
	ctx.Params = gin.Params{{Key: "provider", Value: "email"}}
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/user/account-bindings/email", body)
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler.ConfirmAccountBinding(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	updatedUser, err := repo.GetByID(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, "bound@example.com", updatedUser.Email)
	require.NotEmpty(t, updatedUser.PasswordHash)
}

func TestUserHandler_GetProfile_ReportsProviderValueWithoutTreatingSyntheticEmailAsBound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _, repo, _ := newUserBindingHandlerForTest(t)
	repo.users[7] = &service.User{
		ID:           7,
		Email:        "wechat-union-abc@wechat-connect.invalid",
		PasswordHash: "synthetic-password-hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		Concurrency:  3,
	}

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/user/profile", nil)

	handler.GetProfile(ctx)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			AccountBindings map[string]struct {
				Bound           bool   `json:"bound"`
				ProviderSubject string `json:"provider_subject"`
				DisplayName     string `json:"display_name"`
			} `json:"account_bindings"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.False(t, resp.Data.AccountBindings["email"].Bound)
	require.Empty(t, resp.Data.AccountBindings["email"].ProviderSubject)
	require.Equal(t, "union-abc", resp.Data.AccountBindings["wechat"].ProviderSubject)
	require.Equal(t, "union-abc", resp.Data.AccountBindings["wechat"].DisplayName)
}
