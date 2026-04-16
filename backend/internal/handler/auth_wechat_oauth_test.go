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

func TestBuildWeChatPaymentAuthorizeURL(t *testing.T) {
	u, err := buildWeChatPaymentAuthorizeURL(config.WeChatConnectConfig{
		AppID:       "wx123456",
		RedirectURL: "https://example.com/api/v1/auth/oauth/wechat/callback",
	}, "snsapi_base", "state-123")
	require.NoError(t, err)
	require.Contains(t, u, "appid=wx123456")
	require.Contains(t, u, "scope=snsapi_base")
	require.Contains(t, u, "state=state-123")
	require.True(t, strings.HasSuffix(u, "#wechat_redirect"))
}

func TestWeChatPaymentOAuthCallbackCarriesOpenIDBackToFrontend(t *testing.T) {
	gin.SetMode(gin.TestMode)

	origExchange := wechatPaymentExchangeCode
	t.Cleanup(func() {
		wechatPaymentExchangeCode = origExchange
	})
	wechatPaymentExchangeCode = func(ctx context.Context, cfg config.WeChatConnectConfig, code string) (*weChatPaymentTokenResult, error) {
		require.Equal(t, "code-123", code)
		require.Equal(t, "wx123456", cfg.AppID)
		return &weChatPaymentTokenResult{
			OpenID: "openid-abc",
			Scope:  "snsapi_base",
		}, nil
	}

	settingSvc := service.NewSettingService(&settingHandlerRepoStub{values: map[string]string{}}, &config.Config{
		WeChat: config.WeChatConnectConfig{
			Enabled:             true,
			AppID:               "wx123456",
			AppSecret:           "wechat-secret",
			Scopes:              "snsapi_login snsapi_base",
			RedirectURL:         "https://example.com/api/v1/auth/oauth/wechat/callback",
			FrontendRedirectURL: "/auth/wechat/callback",
		},
	})
	authHandler := NewAuthHandler(&config.Config{}, nil, nil, settingSvc, nil, nil, nil)

	startRec := httptest.NewRecorder()
	startCtx, _ := gin.CreateTestContext(startRec)
	startCtx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/wechat/payment/start?payment_type=wxpay&amount=12.5&order_type=balance&plan_id=9&redirect=%2Fpurchase%3Ffrom%3Dwechat", nil)
	authHandler.WeChatPaymentOAuthStart(startCtx)
	require.Equal(t, http.StatusFound, startRec.Code)

	startLocation := startRec.Header().Get("Location")
	startURL, err := url.Parse(startLocation)
	require.NoError(t, err)
	state := startURL.Query().Get("state")
	require.NotEmpty(t, state)

	callbackRec := httptest.NewRecorder()
	callbackCtx, _ := gin.CreateTestContext(callbackRec)
	callbackReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/wechat/payment/callback?code=code-123&state="+url.QueryEscape(state), nil)
	for _, cookie := range startRec.Result().Cookies() {
		callbackReq.AddCookie(cookie)
	}
	callbackCtx.Request = callbackReq
	authHandler.WeChatPaymentOAuthCallback(callbackCtx)
	require.Equal(t, http.StatusFound, callbackRec.Code)

	location := callbackRec.Header().Get("Location")
	redirectURL, err := url.Parse(location)
	require.NoError(t, err)
	require.Equal(t, "/auth/wechat/callback", redirectURL.Path)

	fragment, err := url.ParseQuery(redirectURL.Fragment)
	require.NoError(t, err)
	require.Equal(t, "openid-abc", fragment.Get("openid"))
	require.Equal(t, "snsapi_base", fragment.Get("scope"))
	require.Equal(t, "wxpay", fragment.Get("payment_type"))
	require.Equal(t, "12.5", fragment.Get("amount"))
	require.Equal(t, "balance", fragment.Get("order_type"))
	require.Equal(t, "9", fragment.Get("plan_id"))
	require.Equal(t, "/purchase?from=wechat", fragment.Get("redirect"))
	require.Empty(t, fragment.Get("access_token"))
}

func TestWeChatPaymentOAuthCallbackRejectsInvalidState(t *testing.T) {
	gin.SetMode(gin.TestMode)

	settingSvc := service.NewSettingService(&settingHandlerRepoStub{values: map[string]string{}}, &config.Config{
		WeChat: config.WeChatConnectConfig{
			Enabled:             true,
			AppID:               "wx123456",
			AppSecret:           "wechat-secret",
			RedirectURL:         "https://example.com/api/v1/auth/oauth/wechat/callback",
			FrontendRedirectURL: "/auth/wechat/callback",
		},
	})
	authHandler := NewAuthHandler(&config.Config{}, nil, nil, settingSvc, nil, nil, nil)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/wechat/payment/callback?code=code-123&state=wrong", nil)
	req.AddCookie(&http.Cookie{
		Name:  weChatPaymentOAuthStateCookieName,
		Value: encodeCookieValue("expected"),
		Path:  weChatPaymentOAuthCookiePath,
	})
	c.Request = req

	authHandler.WeChatPaymentOAuthCallback(c)
	require.Equal(t, http.StatusFound, rec.Code)

	location := rec.Header().Get("Location")
	redirectURL, err := url.Parse(location)
	require.NoError(t, err)
	fragment, err := url.ParseQuery(redirectURL.Fragment)
	require.NoError(t, err)
	require.Equal(t, "invalid_state", fragment.Get("error"))
}

func TestWeChatPaymentOAuthStartStoresContextCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)

	settingSvc := service.NewSettingService(&settingHandlerRepoStub{values: map[string]string{}}, &config.Config{
		WeChat: config.WeChatConnectConfig{
			Enabled:             true,
			AppID:               "wx123456",
			AppSecret:           "wechat-secret",
			RedirectURL:         "https://example.com/api/v1/auth/oauth/wechat/callback",
			FrontendRedirectURL: "/auth/wechat/callback",
		},
	})
	authHandler := NewAuthHandler(&config.Config{}, nil, nil, settingSvc, nil, nil, nil)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/wechat/payment/start?payment_type=wxpay_direct&amount=88&order_type=subscription&plan_id=12&redirect=%2Fpurchase", nil)

	authHandler.WeChatPaymentOAuthStart(c)
	require.Equal(t, http.StatusFound, rec.Code)

	cookies := rec.Result().Cookies()
	var rawCtx string
	for _, cookie := range cookies {
		if cookie.Name == weChatPaymentOAuthContextCookieName {
			rawCtx = cookie.Value
			break
		}
	}
	require.NotEmpty(t, rawCtx)

	decoded, err := decodeCookieValue(rawCtx)
	require.NoError(t, err)

	var payload weChatPaymentOAuthContext
	require.NoError(t, json.Unmarshal([]byte(decoded), &payload))
	require.Equal(t, "wxpay_direct", payload.PaymentType)
	require.Equal(t, "88", payload.Amount)
	require.Equal(t, "subscription", payload.OrderType)
	require.EqualValues(t, 12, payload.PlanID)
}
