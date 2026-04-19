package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSanitizeFrontendRedirectPath(t *testing.T) {
	require.Equal(t, "/dashboard", sanitizeFrontendRedirectPath("/dashboard"))
	require.Equal(t, "/dashboard", sanitizeFrontendRedirectPath(" /dashboard "))
	require.Equal(t, "", sanitizeFrontendRedirectPath("dashboard"))
	require.Equal(t, "", sanitizeFrontendRedirectPath("//evil.com"))
	require.Equal(t, "", sanitizeFrontendRedirectPath("https://evil.com"))
	require.Equal(t, "", sanitizeFrontendRedirectPath("/\nfoo"))

	long := "/" + strings.Repeat("a", linuxDoOAuthMaxRedirectLen)
	require.Equal(t, "", sanitizeFrontendRedirectPath(long))
}

func TestBuildBearerAuthorization(t *testing.T) {
	auth, err := buildBearerAuthorization("", "token123")
	require.NoError(t, err)
	require.Equal(t, "Bearer token123", auth)

	auth, err = buildBearerAuthorization("bearer", "token123")
	require.NoError(t, err)
	require.Equal(t, "Bearer token123", auth)

	_, err = buildBearerAuthorization("MAC", "token123")
	require.Error(t, err)

	_, err = buildBearerAuthorization("Bearer", "token 123")
	require.Error(t, err)
}

func TestLinuxDoParseUserInfoParsesIDAndUsername(t *testing.T) {
	cfg := config.LinuxDoConnectConfig{
		UserInfoURL: "https://connect.linux.do/api/user",
	}

	claims, err := linuxDoParseUserInfo(`{"id":123,"username":"alice","avatar_url":"https://example.com/avatar.png"}`, cfg)
	require.NoError(t, err)
	require.Equal(t, "123", claims.Subject)
	require.Equal(t, "alice", claims.Username)
	require.Equal(t, "alice", claims.DisplayName)
	require.Equal(t, "https://example.com/avatar.png", claims.AvatarURL)
	require.Equal(t, "linuxdo-123@linuxdo-connect.invalid", claims.Email)
}

func TestLinuxDoParseUserInfoDefaultsUsername(t *testing.T) {
	cfg := config.LinuxDoConnectConfig{
		UserInfoURL: "https://connect.linux.do/api/user",
	}

	claims, err := linuxDoParseUserInfo(`{"id":"123"}`, cfg)
	require.NoError(t, err)
	require.Equal(t, "123", claims.Subject)
	require.Equal(t, "linuxdo_123", claims.Username)
	require.Equal(t, "linuxdo_123", claims.DisplayName)
	require.Equal(t, "linuxdo-123@linuxdo-connect.invalid", claims.Email)
}

func TestLinuxDoParseUserInfoRejectsUnsafeSubject(t *testing.T) {
	cfg := config.LinuxDoConnectConfig{
		UserInfoURL: "https://connect.linux.do/api/user",
	}

	_, err := linuxDoParseUserInfo(`{"id":"123@456"}`, cfg)
	require.Error(t, err)

	tooLong := strings.Repeat("a", linuxDoOAuthMaxSubjectLen+1)
	_, err = linuxDoParseUserInfo(`{"id":"`+tooLong+`"}`, cfg)
	require.Error(t, err)
}

func TestParseOAuthProviderErrorJSON(t *testing.T) {
	code, desc := parseOAuthProviderError(`{"error":"invalid_client","error_description":"bad secret"}`)
	require.Equal(t, "invalid_client", code)
	require.Equal(t, "bad secret", desc)
}

func TestParseOAuthProviderErrorForm(t *testing.T) {
	code, desc := parseOAuthProviderError("error=invalid_request&error_description=Missing+code_verifier")
	require.Equal(t, "invalid_request", code)
	require.Equal(t, "Missing code_verifier", desc)
}

func TestParseLinuxDoTokenResponseJSON(t *testing.T) {
	token, ok := parseLinuxDoTokenResponse(`{"access_token":"t1","token_type":"Bearer","expires_in":3600,"scope":"user"}`)
	require.True(t, ok)
	require.Equal(t, "t1", token.AccessToken)
	require.Equal(t, "Bearer", token.TokenType)
	require.Equal(t, int64(3600), token.ExpiresIn)
	require.Equal(t, "user", token.Scope)
}

func TestParseLinuxDoTokenResponseForm(t *testing.T) {
	token, ok := parseLinuxDoTokenResponse("access_token=t2&token_type=bearer&expires_in=60")
	require.True(t, ok)
	require.Equal(t, "t2", token.AccessToken)
	require.Equal(t, "bearer", token.TokenType)
	require.Equal(t, int64(60), token.ExpiresIn)
}

func TestSingleLineStripsWhitespace(t *testing.T) {
	require.Equal(t, "hello world", singleLine("hello\r\nworld"))
	require.Equal(t, "", singleLine("\n\t\r"))
}

func TestLinuxDoOAuthCallback_BoundLoginRedirectsWithTokenPair(t *testing.T) {
	gin.SetMode(gin.TestMode)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			_, _ = w.Write([]byte(`{"access_token":"provider-token","token_type":"Bearer","expires_in":3600}`))
		case "/userinfo":
			_, _ = w.Write([]byte(`{"id":123,"username":"alice","name":"Alice Example","avatar_url":"https://example.com/avatar.png"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	handler, _, repo := newOAuthCallbackHandlerForTest(t)
	handler.cfg.LinuxDo = config.LinuxDoConnectConfig{
		Enabled:             true,
		ClientID:            "cid",
		ClientSecret:        "secret",
		AuthorizeURL:        srv.URL + "/authorize",
		TokenURL:            srv.URL + "/token",
		UserInfoURL:         srv.URL + "/userinfo",
		RedirectURL:         "https://api.example.com/api/v1/auth/oauth/linuxdo/callback",
		FrontendRedirectURL: "/auth/linuxdo/callback",
	}
	repo.bindIdentity(7, "linuxdo", "linuxdo", "123")

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/linuxdo/callback?code=ok&state=state-1", nil)
	req.AddCookie(&http.Cookie{Name: linuxDoOAuthStateCookieName, Value: encodeCookieValue("state-1")})
	req.AddCookie(&http.Cookie{Name: linuxDoOAuthRedirectCookie, Value: encodeCookieValue("/billing")})
	req.AddCookie(&http.Cookie{Name: linuxDoOAuthIntentCookieName, Value: encodeCookieValue(service.PendingAuthIntentLogin)})
	ctx.Request = req

	handler.LinuxDoOAuthCallback(ctx)

	require.Equal(t, http.StatusFound, rec.Code)
	fragment := parseRedirectFragment(t, rec.Header().Get("Location"))
	require.NotEmpty(t, fragment.Get("access_token"))
	require.NotEmpty(t, fragment.Get("refresh_token"))
	require.Equal(t, "linuxdo", fragment.Get("provider"))
	require.Equal(t, service.PendingAuthIntentLogin, fragment.Get("intent"))
	require.Equal(t, "/billing", fragment.Get("redirect"))
	require.Empty(t, fragment.Get("pending_auth_token"))
	require.Empty(t, repo.sessions)
}

func TestLinuxDoOAuthCallback_UnboundLoginRedirectsWithPendingSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			_, _ = w.Write([]byte(`{"access_token":"provider-token","token_type":"Bearer","expires_in":3600}`))
		case "/userinfo":
			_, _ = w.Write([]byte(`{"id":123,"username":"alice"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	handler, authSvc, repo := newOAuthCallbackHandlerForTest(t)
	handler.cfg.LinuxDo = config.LinuxDoConnectConfig{
		Enabled:             true,
		ClientID:            "cid",
		ClientSecret:        "secret",
		AuthorizeURL:        srv.URL + "/authorize",
		TokenURL:            srv.URL + "/token",
		UserInfoURL:         srv.URL + "/userinfo",
		RedirectURL:         "https://api.example.com/api/v1/auth/oauth/linuxdo/callback",
		FrontendRedirectURL: "/auth/linuxdo/callback",
	}

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/linuxdo/callback?code=ok&state=state-2", nil)
	req.AddCookie(&http.Cookie{Name: linuxDoOAuthStateCookieName, Value: encodeCookieValue("state-2")})
	req.AddCookie(&http.Cookie{Name: linuxDoOAuthRedirectCookie, Value: encodeCookieValue("/welcome")})
	req.AddCookie(&http.Cookie{Name: linuxDoOAuthIntentCookieName, Value: encodeCookieValue(service.PendingAuthIntentLogin)})
	ctx.Request = req

	handler.LinuxDoOAuthCallback(ctx)

	require.Equal(t, http.StatusFound, rec.Code)
	fragment := parseRedirectFragment(t, rec.Header().Get("Location"))
	require.Equal(t, "pending_session", fragment.Get("auth_result"))
	require.NotEmpty(t, fragment.Get("pending_auth_token"))
	require.NotEmpty(t, fragment.Get("pending_oauth_token"))
	require.Equal(t, "linuxdo", fragment.Get("provider"))
	require.Equal(t, service.PendingAuthIntentLogin, fragment.Get("intent"))
	require.Equal(t, "/welcome", fragment.Get("redirect"))
	require.Equal(t, "true", fragment.Get("adoption_required"))
	require.Equal(t, "alice", fragment.Get("suggested_display_name"))
	require.Empty(t, fragment.Get("suggested_avatar_url"))

	session, err := authSvc.GetPendingAuthSessionForProgress(context.Background(), fragment.Get("pending_auth_token"), nil)
	require.NoError(t, err)
	require.Equal(t, service.PendingAuthIntentLogin, session.Intent)
	require.Equal(t, "linuxdo", session.ProviderType)
	require.Equal(t, "linuxdo", session.ProviderKey)
	require.Equal(t, "123", session.ProviderSubject)
	require.Equal(t, "alice", session.Metadata["suggested_display_name"])
	_, hasSuggestedAvatar := session.Metadata["suggested_avatar_url"]
	require.False(t, hasSuggestedAvatar)
	require.Len(t, repo.sessions, 1)
}
