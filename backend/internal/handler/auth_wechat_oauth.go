package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/oauth"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/imroc/req/v3"
)

const (
	weChatPaymentOAuthCookiePath              = "/api/v1/auth/oauth/wechat"
	weChatPaymentOAuthStateCookieName         = "wechat_payment_oauth_state"
	weChatPaymentOAuthRedirectCookieName      = "wechat_payment_oauth_redirect"
	weChatPaymentOAuthContextCookieName       = "wechat_payment_oauth_context"
	weChatPaymentOAuthScopeCookieName         = "wechat_payment_oauth_scope"
	weChatPaymentOAuthCookieMaxAgeSec         = 10 * 60
	weChatPaymentOAuthDefaultRedirectTo       = "/purchase"
	weChatPaymentOAuthDefaultFrontendCallback = "/auth/wechat/callback"
	weChatPaymentOAuthAuthorizeURL            = "https://open.weixin.qq.com/connect/oauth2/authorize"
	weChatPaymentOAuthTokenURL                = "https://api.weixin.qq.com/sns/oauth2/access_token"
)

type weChatOAuthExchangeError struct {
	StatusCode int
	ErrCode    string
	ErrMsg     string
	Body       string
}

func (e *weChatOAuthExchangeError) Error() string {
	if e == nil {
		return ""
	}
	parts := []string{fmt.Sprintf("openid exchange status=%d", e.StatusCode)}
	if strings.TrimSpace(e.ErrCode) != "" {
		parts = append(parts, "errcode="+strings.TrimSpace(e.ErrCode))
	}
	if strings.TrimSpace(e.ErrMsg) != "" {
		parts = append(parts, "errmsg="+strings.TrimSpace(e.ErrMsg))
	}
	return strings.Join(parts, " ")
}

type weChatPaymentTokenResult struct {
	OpenID string
	Scope  string
}

type weChatPaymentOAuthContext struct {
	PaymentType string `json:"payment_type"`
	Amount      string `json:"amount,omitempty"`
	OrderType   string `json:"order_type,omitempty"`
	PlanID      int64  `json:"plan_id,omitempty"`
}

// WeChatPaymentOAuthStart starts the payment-purpose WeChat OAuth flow and redirects
// the browser to WeChat so the callback can obtain an OpenID.
// GET /api/v1/auth/oauth/wechat/payment/start?payment_type=wxpay&redirect=/payment
func (h *AuthHandler) WeChatPaymentOAuthStart(c *gin.Context) {
	cfg, err := h.getWeChatOAuthConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	paymentType := normalizeWeChatPaymentType(c.Query("payment_type"))
	if paymentType == "" {
		response.BadRequest(c, "Invalid payment type")
		return
	}

	state, err := oauth.GenerateState()
	if err != nil {
		response.ErrorFrom(c, infraerrors.InternalServer("OAUTH_STATE_GEN_FAILED", "failed to generate oauth state").WithCause(err))
		return
	}

	redirectTo := normalizeWeChatPaymentRedirectPath(sanitizeFrontendRedirectPath(c.Query("redirect")))
	if redirectTo == "" {
		redirectTo = weChatPaymentOAuthDefaultRedirectTo
	}
	scope := normalizeWeChatPaymentScope(c.Query("scope"), cfg.Scopes)
	ctxCookie, err := encodeWeChatPaymentOAuthContext(weChatPaymentOAuthContext{
		PaymentType: paymentType,
		Amount:      strings.TrimSpace(c.Query("amount")),
		OrderType:   strings.TrimSpace(c.Query("order_type")),
		PlanID:      parseInt64Default(c.Query("plan_id"), 0),
	})
	if err != nil {
		response.ErrorFrom(c, infraerrors.InternalServer("OAUTH_CONTEXT_ENCODE_FAILED", "failed to encode oauth context").WithCause(err))
		return
	}

	secureCookie := isRequestHTTPS(c)
	weChatPaymentSetCookie(c, weChatPaymentOAuthStateCookieName, encodeCookieValue(state), weChatPaymentOAuthCookieMaxAgeSec, secureCookie)
	weChatPaymentSetCookie(c, weChatPaymentOAuthRedirectCookieName, encodeCookieValue(redirectTo), weChatPaymentOAuthCookieMaxAgeSec, secureCookie)
	weChatPaymentSetCookie(c, weChatPaymentOAuthContextCookieName, encodeCookieValue(ctxCookie), weChatPaymentOAuthCookieMaxAgeSec, secureCookie)
	weChatPaymentSetCookie(c, weChatPaymentOAuthScopeCookieName, encodeCookieValue(scope), weChatPaymentOAuthCookieMaxAgeSec, secureCookie)

	authURL, err := buildWeChatPaymentAuthorizeURL(cfg, scope, state)
	if err != nil {
		response.ErrorFrom(c, infraerrors.InternalServer("OAUTH_BUILD_URL_FAILED", "failed to build oauth authorization url").WithCause(err))
		return
	}

	c.Redirect(http.StatusFound, authURL)
}

func (h *AuthHandler) WeChatOAuthStart(c *gin.Context) {
	h.WeChatPaymentOAuthStart(c)
}

// WeChatPaymentOAuthCallback handles the payment-purpose WeChat OAuth callback. It
// exchanges the code for an OpenID and redirects back to the frontend callback
// route with only the fields required to resume create-order.
// GET /api/v1/auth/oauth/wechat/callback?code=...&state=...
func (h *AuthHandler) WeChatPaymentOAuthCallback(c *gin.Context) {
	cfg, cfgErr := h.getWeChatOAuthConfig(c.Request.Context())
	if cfgErr != nil {
		response.ErrorFrom(c, cfgErr)
		return
	}

	frontendCallback := strings.TrimSpace(cfg.FrontendRedirectURL)
	if frontendCallback == "" {
		frontendCallback = weChatPaymentOAuthDefaultFrontendCallback
	}

	if providerErr := strings.TrimSpace(c.Query("error")); providerErr != "" {
		redirectOAuthError(c, frontendCallback, "provider_error", providerErr, c.Query("error_description"))
		return
	}

	code := strings.TrimSpace(c.Query("code"))
	state := strings.TrimSpace(c.Query("state"))
	if code == "" || state == "" {
		redirectOAuthError(c, frontendCallback, "missing_params", "missing code/state", "")
		return
	}

	secureCookie := isRequestHTTPS(c)
	defer func() {
		weChatPaymentClearCookie(c, weChatPaymentOAuthStateCookieName, secureCookie)
		weChatPaymentClearCookie(c, weChatPaymentOAuthRedirectCookieName, secureCookie)
		weChatPaymentClearCookie(c, weChatPaymentOAuthContextCookieName, secureCookie)
		weChatPaymentClearCookie(c, weChatPaymentOAuthScopeCookieName, secureCookie)
	}()

	expectedState, err := readCookieDecoded(c, weChatPaymentOAuthStateCookieName)
	if err != nil || expectedState == "" || state != expectedState {
		redirectOAuthError(c, frontendCallback, "invalid_state", "invalid oauth state", "")
		return
	}

	redirectTo, _ := readCookieDecoded(c, weChatPaymentOAuthRedirectCookieName)
	redirectTo = normalizeWeChatPaymentRedirectPath(sanitizeFrontendRedirectPath(redirectTo))
	if redirectTo == "" {
		redirectTo = weChatPaymentOAuthDefaultRedirectTo
	}

	rawContext, _ := readCookieDecoded(c, weChatPaymentOAuthContextCookieName)
	oauthContext, err := decodeWeChatPaymentOAuthContext(rawContext)
	if err != nil {
		redirectOAuthError(c, frontendCallback, "invalid_context", "invalid oauth context", "")
		return
	}
	if oauthContext.PaymentType == "" {
		oauthContext.PaymentType = payment.TypeWxpay
	}

	scope, _ := readCookieDecoded(c, weChatPaymentOAuthScopeCookieName)
	scope = normalizeWeChatPaymentScope(scope, cfg.Scopes)

	tokenResult, err := wechatPaymentExchangeCode(c.Request.Context(), cfg, code)
	if err != nil {
		description := err.Error()
		var exchangeErr *weChatOAuthExchangeError
		if errors.As(err, &exchangeErr) && exchangeErr != nil {
			log.Printf(
				"[WeChat OAuth] openid exchange failed: status=%d errcode=%q errmsg=%q body=%s",
				exchangeErr.StatusCode,
				exchangeErr.ErrCode,
				exchangeErr.ErrMsg,
				truncateLogValue(exchangeErr.Body, 2048),
			)
		} else {
			log.Printf("[WeChat OAuth] openid exchange failed: %v", err)
		}
		redirectOAuthError(c, frontendCallback, "token_exchange_failed", "failed to exchange oauth code", singleLine(description))
		return
	}
	if tokenResult != nil && strings.TrimSpace(tokenResult.Scope) != "" {
		scope = strings.TrimSpace(tokenResult.Scope)
	}
	if tokenResult == nil || strings.TrimSpace(tokenResult.OpenID) == "" {
		redirectOAuthError(c, frontendCallback, "missing_openid", "missing openid", "")
		return
	}

	fragment := url.Values{}
	fragment.Set("openid", strings.TrimSpace(tokenResult.OpenID))
	fragment.Set("payment_type", oauthContext.PaymentType)
	if oauthContext.Amount != "" {
		fragment.Set("amount", oauthContext.Amount)
	}
	if oauthContext.OrderType != "" {
		fragment.Set("order_type", oauthContext.OrderType)
	}
	if oauthContext.PlanID > 0 {
		fragment.Set("plan_id", strconv.FormatInt(oauthContext.PlanID, 10))
	}
	fragment.Set("redirect", redirectTo)
	fragment.Set("scope", scope)
	redirectWithFragment(c, frontendCallback, fragment)
}

func (h *AuthHandler) WeChatOAuthCallback(c *gin.Context) {
	h.WeChatPaymentOAuthCallback(c)
}

func (h *AuthHandler) getWeChatOAuthConfig(ctx context.Context) (config.WeChatConnectConfig, error) {
	if h != nil && h.settingSvc != nil {
		return h.settingSvc.GetWeChatConnectOAuthConfig(ctx)
	}
	if h == nil || h.cfg == nil {
		return config.WeChatConnectConfig{}, infraerrors.ServiceUnavailable("CONFIG_NOT_READY", "config not loaded")
	}
	if !h.cfg.WeChat.Enabled {
		return config.WeChatConnectConfig{}, infraerrors.NotFound("OAUTH_DISABLED", "oauth login is disabled")
	}
	return h.cfg.WeChat, nil
}

func buildWeChatPaymentAuthorizeURL(cfg config.WeChatConnectConfig, scope, state string) (string, error) {
	u, err := url.Parse(weChatPaymentOAuthAuthorizeURL)
	if err != nil {
		return "", fmt.Errorf("parse authorize url: %w", err)
	}
	q := u.Query()
	q.Set("appid", strings.TrimSpace(cfg.AppID))
	q.Set("redirect_uri", strings.TrimSpace(cfg.RedirectURL))
	q.Set("response_type", "code")
	q.Set("scope", normalizeWeChatPaymentScope(scope, cfg.Scopes))
	q.Set("state", strings.TrimSpace(state))
	u.RawQuery = q.Encode()
	u.Fragment = "wechat_redirect"
	return u.String(), nil
}

func normalizeWeChatPaymentType(raw string) string {
	raw = strings.TrimSpace(raw)
	switch raw {
	case payment.TypeWxpay, payment.TypeWxpayDirect:
		return raw
	default:
		return ""
	}
}

func normalizeWeChatPaymentScope(raw, fallback string) string {
	combined := strings.TrimSpace(firstNonEmpty(raw, fallback))
	if combined == "" {
		return "snsapi_base"
	}
	for _, part := range strings.FieldsFunc(combined, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	}) {
		switch strings.TrimSpace(part) {
		case "snsapi_base":
			return "snsapi_base"
		case "snsapi_userinfo":
			return "snsapi_userinfo"
		}
	}
	return "snsapi_base"
}

var wechatPaymentExchangeCode = func(ctx context.Context, cfg config.WeChatConnectConfig, code string) (*weChatPaymentTokenResult, error) {
	client := req.C().SetTimeout(30 * time.Second)
	resp, err := client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json").
		SetQueryParam("appid", strings.TrimSpace(cfg.AppID)).
		SetQueryParam("secret", strings.TrimSpace(cfg.AppSecret)).
		SetQueryParam("code", strings.TrimSpace(code)).
		SetQueryParam("grant_type", "authorization_code").
		Get(weChatPaymentOAuthTokenURL)
	if err != nil {
		return nil, fmt.Errorf("request openid: %w", err)
	}

	body := strings.TrimSpace(resp.String())
	if !resp.IsSuccessState() {
		return nil, &weChatOAuthExchangeError{
			StatusCode: resp.StatusCode,
			ErrCode:    strings.TrimSpace(getGJSON(body, "errcode")),
			ErrMsg:     strings.TrimSpace(firstNonEmpty(getGJSON(body, "errmsg"), getGJSON(body, "error_description"))),
			Body:       body,
		}
	}

	if errCode := strings.TrimSpace(getGJSON(body, "errcode")); errCode != "" && errCode != "0" {
		return nil, &weChatOAuthExchangeError{
			StatusCode: resp.StatusCode,
			ErrCode:    errCode,
			ErrMsg:     strings.TrimSpace(getGJSON(body, "errmsg")),
			Body:       body,
		}
	}

	openID := strings.TrimSpace(getGJSON(body, "openid"))
	if openID == "" {
		return nil, &weChatOAuthExchangeError{StatusCode: resp.StatusCode, Body: body}
	}
	return &weChatPaymentTokenResult{
		OpenID: openID,
		Scope:  strings.TrimSpace(getGJSON(body, "scope")),
	}, nil
}

func weChatPaymentSetCookie(c *gin.Context, name, value string, maxAgeSec int, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     weChatPaymentOAuthCookiePath,
		MaxAge:   maxAgeSec,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func weChatPaymentClearCookie(c *gin.Context, name string, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     weChatPaymentOAuthCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func encodeWeChatPaymentOAuthContext(payload weChatPaymentOAuthContext) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func decodeWeChatPaymentOAuthContext(raw string) (weChatPaymentOAuthContext, error) {
	var payload weChatPaymentOAuthContext
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return payload, nil
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return weChatPaymentOAuthContext{}, err
	}
	payload.PaymentType = normalizeWeChatPaymentType(payload.PaymentType)
	return payload, nil
}

func parseInt64Default(raw string, fallback int64) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return value
}

func normalizeWeChatPaymentRedirectPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if path == "/payment" {
		return weChatPaymentOAuthDefaultRedirectTo
	}
	if strings.HasPrefix(path, "/payment?") {
		return weChatPaymentOAuthDefaultRedirectTo + strings.TrimPrefix(path, "/payment")
	}
	return path
}
