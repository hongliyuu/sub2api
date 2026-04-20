package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type wechatOAuthSettingRepoStub struct {
	values map[string]string
}

func (s *wechatOAuthSettingRepoStub) Get(context.Context, string) (*service.Setting, error) {
	panic("unexpected Get call")
}
func (s *wechatOAuthSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if v, ok := s.values[key]; ok {
		return v, nil
	}
	return "", service.ErrSettingNotFound
}
func (s *wechatOAuthSettingRepoStub) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}
func (s *wechatOAuthSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if v, ok := s.values[key]; ok {
			out[key] = v
		}
	}
	return out, nil
}
func (s *wechatOAuthSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}
func (s *wechatOAuthSettingRepoStub) GetAll(_ context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for k, v := range s.values {
		out[k] = v
	}
	return out, nil
}
func (s *wechatOAuthSettingRepoStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

func TestWeChatOAuthStart_OpenModeRedirectsToQRConnectAndPersistsIntent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _, _ := newOAuthCallbackHandlerForTest(t)
	handler.settingSvc = service.NewSettingService(&wechatOAuthSettingRepoStub{values: map[string]string{
		service.SettingKeyWeChatLoginOpenEnabled:   "true",
		service.SettingKeyWeChatLoginOpenAppID:     "wx-open-app",
		service.SettingKeyWeChatLoginOpenAppSecret: "open-secret",
	}}, &config.Config{})

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/wechat/start?mode=open&intent=bind&redirect=/profile", nil)
	req.Host = "api.example.com"
	req.Header.Set("X-Forwarded-Proto", "https")
	ctx.Request = req

	handler.WeChatOAuthStart(ctx)

	require.Equal(t, http.StatusFound, rec.Code)
	location := rec.Header().Get("Location")
	parsed, err := url.Parse(location)
	require.NoError(t, err)
	require.Equal(t, "open.weixin.qq.com", parsed.Host)
	require.Equal(t, "/connect/qrconnect", parsed.Path)
	require.Equal(t, "wx-open-app", parsed.Query().Get("appid"))
	require.Equal(t, "https://api.example.com/api/v1/auth/oauth/wechat/callback", parsed.Query().Get("redirect_uri"))
	require.Equal(t, "snsapi_login", parsed.Query().Get("scope"))
	require.Equal(t, "wechat_redirect", parsed.Fragment)
	require.Equal(t, wechatOAuthCookiePath, cookieByName(t, rec, wechatOAuthStateCookieName).Path)

	intentCookie, err := decodeCookieValue(cookieValueByName(t, rec, wechatOAuthIntentCookieName))
	require.NoError(t, err)
	require.Equal(t, service.PendingAuthIntentBindCurrentUser, intentCookie)

	modeCookie, err := decodeCookieValue(cookieValueByName(t, rec, wechatOAuthModeCookieName))
	require.NoError(t, err)
	require.Equal(t, "open", modeCookie)
}

func TestWeChatOAuthStart_MPModeRedirectsToOAuth2Authorize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _, _ := newOAuthCallbackHandlerForTest(t)
	handler.settingSvc = service.NewSettingService(&wechatOAuthSettingRepoStub{values: map[string]string{
		service.SettingKeyWeChatLoginMPEnabled:   "true",
		service.SettingKeyWeChatLoginMPAppID:     "wx-mp-app",
		service.SettingKeyWeChatLoginMPAppSecret: "mp-secret",
	}}, &config.Config{})

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/wechat/start?mode=mp&redirect=/dashboard", nil)
	req.Host = "example.com"
	ctx.Request = req

	handler.WeChatOAuthStart(ctx)

	require.Equal(t, http.StatusFound, rec.Code)
	location := rec.Header().Get("Location")
	parsed, err := url.Parse(location)
	require.NoError(t, err)
	require.Equal(t, "open.weixin.qq.com", parsed.Host)
	require.Equal(t, "/connect/oauth2/authorize", parsed.Path)
	require.Equal(t, "wx-mp-app", parsed.Query().Get("appid"))
	require.Equal(t, "snsapi_userinfo", parsed.Query().Get("scope"))
}

func TestWeChatOAuthStart_UsesAPIBaseURLForCallbackHost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _, _ := newOAuthCallbackHandlerForTest(t)
	handler.settingSvc = service.NewSettingService(&wechatOAuthSettingRepoStub{values: map[string]string{
		service.SettingKeyAPIBaseURL:               "https://gateway.example.com/api/v1",
		service.SettingKeyWeChatLoginOpenEnabled:   "true",
		service.SettingKeyWeChatLoginOpenAppID:     "wx-open-app",
		service.SettingKeyWeChatLoginOpenAppSecret: "open-secret",
	}}, &config.Config{})

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/wechat/start?mode=open&redirect=/dashboard", nil)
	req.Host = "internal.example.local"
	ctx.Request = req

	handler.WeChatOAuthStart(ctx)

	require.Equal(t, http.StatusFound, rec.Code)
	location := rec.Header().Get("Location")
	parsed, err := url.Parse(location)
	require.NoError(t, err)
	require.Equal(t, "https://gateway.example.com/api/v1/auth/oauth/wechat/callback", parsed.Query().Get("redirect_uri"))
	require.Empty(t, cookieByName(t, rec, wechatOAuthStateCookieName).Domain)
}

func TestWeChatOAuthStart_SharesCookieAcrossSiblingSubdomains(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _, _ := newOAuthCallbackHandlerForTest(t)
	handler.settingSvc = service.NewSettingService(&wechatOAuthSettingRepoStub{values: map[string]string{
		service.SettingKeyAPIBaseURL:               "https://gateway.example.com/api/v1",
		service.SettingKeyWeChatLoginOpenEnabled:   "true",
		service.SettingKeyWeChatLoginOpenAppID:     "wx-open-app",
		service.SettingKeyWeChatLoginOpenAppSecret: "open-secret",
	}}, &config.Config{})

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/wechat/start?mode=open&redirect=/dashboard", nil)
	req.Host = "api.example.com"
	ctx.Request = req

	handler.WeChatOAuthStart(ctx)

	require.Equal(t, http.StatusFound, rec.Code)
	require.Equal(t, "example.com", cookieByName(t, rec, wechatOAuthStateCookieName).Domain)
}

func TestWeChatOAuthStart_PreservesAPIBaseURLPathPrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _, _ := newOAuthCallbackHandlerForTest(t)
	handler.settingSvc = service.NewSettingService(&wechatOAuthSettingRepoStub{values: map[string]string{
		service.SettingKeyAPIBaseURL:               "https://gateway.example.com/gateway",
		service.SettingKeyWeChatLoginOpenEnabled:   "true",
		service.SettingKeyWeChatLoginOpenAppID:     "wx-open-app",
		service.SettingKeyWeChatLoginOpenAppSecret: "open-secret",
	}}, &config.Config{})

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/wechat/start?mode=open&redirect=/dashboard", nil)
	req.Host = "gateway.example.com"
	ctx.Request = req

	handler.WeChatOAuthStart(ctx)

	require.Equal(t, http.StatusFound, rec.Code)
	location := rec.Header().Get("Location")
	parsed, err := url.Parse(location)
	require.NoError(t, err)
	require.Equal(t, "https://gateway.example.com/gateway/api/v1/auth/oauth/wechat/callback", parsed.Query().Get("redirect_uri"))
}

func TestWeChatOAuthStart_DefaultsToOpenOutsideWeChatWhenModeMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _, _ := newOAuthCallbackHandlerForTest(t)
	handler.settingSvc = service.NewSettingService(&wechatOAuthSettingRepoStub{values: map[string]string{
		service.SettingKeyWeChatLoginOpenEnabled:   "true",
		service.SettingKeyWeChatLoginOpenAppID:     "wx-open-app",
		service.SettingKeyWeChatLoginOpenAppSecret: "open-secret",
		service.SettingKeyWeChatLoginMPEnabled:     "true",
		service.SettingKeyWeChatLoginMPAppID:       "wx-mp-app",
		service.SettingKeyWeChatLoginMPAppSecret:   "mp-secret",
	}}, &config.Config{})

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/wechat/start?redirect=/dashboard", nil)
	req.Host = "example.com"
	req.Header.Set("User-Agent", "Mozilla/5.0")
	ctx.Request = req

	handler.WeChatOAuthStart(ctx)

	require.Equal(t, http.StatusFound, rec.Code)
	location := rec.Header().Get("Location")
	parsed, err := url.Parse(location)
	require.NoError(t, err)
	require.Equal(t, "/connect/qrconnect", parsed.Path)
	require.Equal(t, "wx-open-app", parsed.Query().Get("appid"))

	modeCookie, err := decodeCookieValue(cookieValueByName(t, rec, wechatOAuthModeCookieName))
	require.NoError(t, err)
	require.Equal(t, "open", modeCookie)
}

func cookieByName(t *testing.T, rec *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	rawHeaders := rec.Result().Header.Values("Set-Cookie")
	t.Fatalf("cookie %q not found in headers: %s", name, strings.Join(rawHeaders, " | "))
	return nil
}

func TestWeChatOAuthStart_DefaultsToMPInsideWeChatWhenModeMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _, _ := newOAuthCallbackHandlerForTest(t)
	handler.settingSvc = service.NewSettingService(&wechatOAuthSettingRepoStub{values: map[string]string{
		service.SettingKeyWeChatLoginOpenEnabled:   "true",
		service.SettingKeyWeChatLoginOpenAppID:     "wx-open-app",
		service.SettingKeyWeChatLoginOpenAppSecret: "open-secret",
		service.SettingKeyWeChatLoginMPEnabled:     "true",
		service.SettingKeyWeChatLoginMPAppID:       "wx-mp-app",
		service.SettingKeyWeChatLoginMPAppSecret:   "mp-secret",
	}}, &config.Config{})

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/wechat/start?redirect=/dashboard", nil)
	req.Host = "example.com"
	req.Header.Set("User-Agent", "Mozilla/5.0 MicroMessenger")
	ctx.Request = req

	handler.WeChatOAuthStart(ctx)

	require.Equal(t, http.StatusFound, rec.Code)
	location := rec.Header().Get("Location")
	parsed, err := url.Parse(location)
	require.NoError(t, err)
	require.Equal(t, "/connect/oauth2/authorize", parsed.Path)
	require.Equal(t, "wx-mp-app", parsed.Query().Get("appid"))

	modeCookie, err := decodeCookieValue(cookieValueByName(t, rec, wechatOAuthModeCookieName))
	require.NoError(t, err)
	require.Equal(t, "mp", modeCookie)
}

func TestWeChatOAuthStart_ModeMissingOutsideWeChatDoesNotFallbackToMPOnlyConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _, _ := newOAuthCallbackHandlerForTest(t)
	handler.settingSvc = service.NewSettingService(&wechatOAuthSettingRepoStub{values: map[string]string{
		service.SettingKeyWeChatLoginMPEnabled:   "true",
		service.SettingKeyWeChatLoginMPAppID:     "wx-mp-app",
		service.SettingKeyWeChatLoginMPAppSecret: "mp-secret",
	}}, &config.Config{})

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/wechat/start?redirect=/dashboard", nil)
	req.Host = "example.com"
	req.Header.Set("User-Agent", "Mozilla/5.0")
	ctx.Request = req

	handler.WeChatOAuthStart(ctx)

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Empty(t, rec.Header().Get("Location"))
	for _, cookie := range rec.Result().Cookies() {
		require.NotEqual(t, wechatOAuthModeCookieName, cookie.Name)
	}
}

func TestWeChatOAuthStart_RejectsDisabledMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _, _ := newOAuthCallbackHandlerForTest(t)
	handler.settingSvc = service.NewSettingService(&wechatOAuthSettingRepoStub{values: map[string]string{}}, &config.Config{})

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/wechat/start?mode=open", nil)

	handler.WeChatOAuthStart(ctx)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestWeChatOAuthCallback_BoundLoginViaChannelRedirectsWithTokenPair(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, _, repo := newOAuthCallbackHandlerForTest(t)
	handler.settingSvc = service.NewSettingService(&wechatOAuthSettingRepoStub{values: map[string]string{
		service.SettingKeyWeChatLoginOpenEnabled:   "true",
		service.SettingKeyWeChatLoginOpenAppID:     "wx-open-app",
		service.SettingKeyWeChatLoginOpenAppSecret: "open-secret",
	}}, &config.Config{})
	identity := repo.bindIdentity(7, "wechat", wechatOAuthProviderKey, "union-1")
	repo.bindChannel(identity.ID, "wechat", wechatOAuthProviderKey, "open", "wx-open-app", "open-user-1")
	stubWeChatOAuthProvider(t,
		wechatOAuthTokenResponse{AccessToken: "wx-token", OpenID: "open-user-1"},
		wechatOAuthUserInfoResponse{OpenID: "open-user-1", UnionID: "union-1", Nickname: "wx-user"},
	)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/wechat/callback?state=state-1&code=code-1&openid=forged-openid", nil)
	req.AddCookie(&http.Cookie{Name: wechatOAuthStateCookieName, Value: encodeCookieValue("state-1")})
	req.AddCookie(&http.Cookie{Name: wechatOAuthRedirectCookieName, Value: encodeCookieValue("/profile")})
	req.AddCookie(&http.Cookie{Name: wechatOAuthIntentCookieName, Value: encodeCookieValue(service.PendingAuthIntentLogin)})
	req.AddCookie(&http.Cookie{Name: wechatOAuthModeCookieName, Value: encodeCookieValue("open")})
	ctx.Request = req

	handler.WeChatOAuthCallback(ctx)

	require.Equal(t, http.StatusFound, rec.Code)
	fragment := parseRedirectFragment(t, rec.Header().Get("Location"))
	require.NotEmpty(t, fragment.Get("access_token"))
	require.NotEmpty(t, fragment.Get("refresh_token"))
	require.Equal(t, "wechat", fragment.Get("provider"))
	require.Equal(t, service.PendingAuthIntentLogin, fragment.Get("intent"))
	require.Equal(t, "/profile", fragment.Get("redirect"))
	require.Empty(t, fragment.Get("pending_auth_token"))
	require.Empty(t, repo.sessions)
}

func TestWeChatOAuthCallback_BoundMPLoginRedirectsWithTokenPair(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, _, repo := newOAuthCallbackHandlerForTest(t)
	handler.settingSvc = service.NewSettingService(&wechatOAuthSettingRepoStub{values: map[string]string{
		service.SettingKeyWeChatLoginMPEnabled:   "true",
		service.SettingKeyWeChatLoginMPAppID:     "wx-mp-app",
		service.SettingKeyWeChatLoginMPAppSecret: "mp-secret",
	}}, &config.Config{})
	identity := repo.bindIdentity(7, "wechat", wechatOAuthProviderKey, "union-mp-1")
	repo.bindChannel(identity.ID, "wechat", wechatOAuthProviderKey, "mp", "wx-mp-app", "mp-user-1")
	stubWeChatOAuthProvider(t,
		wechatOAuthTokenResponse{AccessToken: "wx-token", OpenID: "mp-user-1"},
		wechatOAuthUserInfoResponse{OpenID: "mp-user-1", UnionID: "union-mp-1", Nickname: "wx-user"},
	)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/wechat/callback?state=state-mp-1&code=code-mp-1", nil)
	req.AddCookie(&http.Cookie{Name: wechatOAuthStateCookieName, Value: encodeCookieValue("state-mp-1")})
	req.AddCookie(&http.Cookie{Name: wechatOAuthRedirectCookieName, Value: encodeCookieValue("/profile")})
	req.AddCookie(&http.Cookie{Name: wechatOAuthIntentCookieName, Value: encodeCookieValue(service.PendingAuthIntentLogin)})
	req.AddCookie(&http.Cookie{Name: wechatOAuthModeCookieName, Value: encodeCookieValue("mp")})
	ctx.Request = req

	handler.WeChatOAuthCallback(ctx)

	require.Equal(t, http.StatusFound, rec.Code)
	fragment := parseRedirectFragment(t, rec.Header().Get("Location"))
	require.NotEmpty(t, fragment.Get("access_token"))
	require.NotEmpty(t, fragment.Get("refresh_token"))
	require.Equal(t, "wechat", fragment.Get("provider"))
	require.Equal(t, service.PendingAuthIntentLogin, fragment.Get("intent"))
	require.Equal(t, "/profile", fragment.Get("redirect"))
	require.Empty(t, fragment.Get("pending_auth_token"))
	require.Empty(t, repo.sessions)
}

func TestWeChatOAuthCallback_BoundLoginFallsBackToChannelWhenUnionIDIsNew(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, _, repo := newOAuthCallbackHandlerForTest(t)
	handler.settingSvc = service.NewSettingService(&wechatOAuthSettingRepoStub{values: map[string]string{
		service.SettingKeyWeChatLoginOpenEnabled:   "true",
		service.SettingKeyWeChatLoginOpenAppID:     "wx-open-app",
		service.SettingKeyWeChatLoginOpenAppSecret: "open-secret",
	}}, &config.Config{})
	identity := repo.bindIdentity(7, "wechat", wechatOAuthProviderKey, "open-user-1")
	repo.bindChannel(identity.ID, "wechat", wechatOAuthProviderKey, "open", "wx-open-app", "open-user-1")
	stubWeChatOAuthProvider(t,
		wechatOAuthTokenResponse{AccessToken: "wx-token", OpenID: "open-user-1"},
		wechatOAuthUserInfoResponse{OpenID: "open-user-1", UnionID: "union-1", Nickname: "wx-user"},
	)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/wechat/callback?state=state-bridge&code=code-bridge&unionid=forged-unionid", nil)
	req.AddCookie(&http.Cookie{Name: wechatOAuthStateCookieName, Value: encodeCookieValue("state-bridge")})
	req.AddCookie(&http.Cookie{Name: wechatOAuthRedirectCookieName, Value: encodeCookieValue("/profile")})
	req.AddCookie(&http.Cookie{Name: wechatOAuthIntentCookieName, Value: encodeCookieValue(service.PendingAuthIntentLogin)})
	req.AddCookie(&http.Cookie{Name: wechatOAuthModeCookieName, Value: encodeCookieValue("open")})
	ctx.Request = req

	handler.WeChatOAuthCallback(ctx)

	require.Equal(t, http.StatusFound, rec.Code)
	fragment := parseRedirectFragment(t, rec.Header().Get("Location"))
	require.NotEmpty(t, fragment.Get("access_token"))
	require.NotEmpty(t, fragment.Get("refresh_token"))
	require.Equal(t, "wechat", fragment.Get("provider"))
	require.Equal(t, service.PendingAuthIntentLogin, fragment.Get("intent"))
	require.Equal(t, "/profile", fragment.Get("redirect"))
	require.Empty(t, fragment.Get("pending_auth_token"))
	require.Empty(t, repo.sessions)
}

func TestWeChatOAuthCallback_UnboundRedirectsWithPendingSessionMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, authSvc, repo := newOAuthCallbackHandlerForTest(t)
	handler.settingSvc = service.NewSettingService(&wechatOAuthSettingRepoStub{values: map[string]string{
		service.SettingKeyWeChatLoginMPEnabled:   "true",
		service.SettingKeyWeChatLoginMPAppID:     "wx-mp-app",
		service.SettingKeyWeChatLoginMPAppSecret: "mp-secret",
	}}, &config.Config{})
	stubWeChatOAuthProvider(t,
		wechatOAuthTokenResponse{AccessToken: "wx-token", OpenID: "open-user-2"},
		wechatOAuthUserInfoResponse{OpenID: "open-user-2", UnionID: "union-2", Nickname: "wx-user"},
	)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/wechat/callback?state=state-2&code=code-2&unionid=forged-unionid&nickname=forged-name", nil)
	req.AddCookie(&http.Cookie{Name: wechatOAuthStateCookieName, Value: encodeCookieValue("state-2")})
	req.AddCookie(&http.Cookie{Name: wechatOAuthRedirectCookieName, Value: encodeCookieValue("/register")})
	req.AddCookie(&http.Cookie{Name: wechatOAuthIntentCookieName, Value: encodeCookieValue(service.PendingAuthIntentLogin)})
	req.AddCookie(&http.Cookie{Name: wechatOAuthModeCookieName, Value: encodeCookieValue("mp")})
	ctx.Request = req

	handler.WeChatOAuthCallback(ctx)

	require.Equal(t, http.StatusFound, rec.Code)
	fragment := parseRedirectFragment(t, rec.Header().Get("Location"))
	require.Equal(t, "pending_session", fragment.Get("auth_result"))
	require.NotEmpty(t, fragment.Get("pending_auth_token"))
	require.NotEmpty(t, fragment.Get("pending_oauth_token"))
	require.Equal(t, "wechat", fragment.Get("provider"))
	require.Equal(t, service.PendingAuthIntentLogin, fragment.Get("intent"))
	require.Equal(t, "/register", fragment.Get("redirect"))

	session, err := authSvc.GetPendingAuthSessionForProgress(context.Background(), fragment.Get("pending_auth_token"), nil)
	require.NoError(t, err)
	require.Equal(t, service.PendingAuthIntentLogin, session.Intent)
	require.Equal(t, "wechat", session.ProviderType)
	require.Equal(t, wechatOAuthProviderKey, session.ProviderKey)
	require.Equal(t, "union-2", session.ProviderSubject)
	require.Equal(t, "union-2", session.Metadata["unionid"])
	require.Equal(t, "open-user-2", session.Metadata["openid"])
	require.Equal(t, "mp", session.Metadata["channel"])
	require.Equal(t, "wx-mp-app", session.Metadata["appid"])
	require.Len(t, repo.sessions, 1)
}

func TestWeChatOAuthCallback_RejectsMissingOAuthModeCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, _, repo := newOAuthCallbackHandlerForTest(t)
	handler.settingSvc = service.NewSettingService(&wechatOAuthSettingRepoStub{values: map[string]string{
		service.SettingKeyWeChatLoginMPEnabled:   "true",
		service.SettingKeyWeChatLoginMPAppID:     "wx-mp-app",
		service.SettingKeyWeChatLoginMPAppSecret: "mp-secret",
	}}, &config.Config{})

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/wechat/callback?state=state-3&code=code-3", nil)
	req.AddCookie(&http.Cookie{Name: wechatOAuthStateCookieName, Value: encodeCookieValue("state-3")})
	req.AddCookie(&http.Cookie{Name: wechatOAuthRedirectCookieName, Value: encodeCookieValue("/register")})
	req.AddCookie(&http.Cookie{Name: wechatOAuthIntentCookieName, Value: encodeCookieValue(service.PendingAuthIntentLogin)})
	ctx.Request = req

	handler.WeChatOAuthCallback(ctx)

	require.Equal(t, http.StatusFound, rec.Code)
	fragment := parseRedirectFragment(t, rec.Header().Get("Location"))
	require.Equal(t, "invalid_state", fragment.Get("error"))
	require.Equal(t, "missing oauth mode", fragment.Get("error_message"))
	require.Empty(t, repo.sessions)
}

func stubWeChatOAuthProvider(t *testing.T, tokenResp wechatOAuthTokenResponse, userInfo wechatOAuthUserInfoResponse) {
	t.Helper()

	previousAccessTokenURL := wechatOAuthAccessTokenURL
	previousUserInfoURL := wechatOAuthUserInfoURL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/token":
			require.NoError(t, json.NewEncoder(w).Encode(tokenResp))
		case "/userinfo":
			require.NoError(t, json.NewEncoder(w).Encode(userInfo))
		default:
			http.NotFound(w, r)
		}
	}))
	wechatOAuthAccessTokenURL = server.URL + "/token"
	wechatOAuthUserInfoURL = server.URL + "/userinfo"
	t.Cleanup(func() {
		server.Close()
		wechatOAuthAccessTokenURL = previousAccessTokenURL
		wechatOAuthUserInfoURL = previousUserInfoURL
	})
}
