package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/oauth"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

const (
	wechatOAuthCookiePath         = "/api/v1/auth/oauth/wechat"
	wechatOAuthCookieMaxAgeSec    = 10 * 60
	wechatOAuthStateCookieName    = "wechat_oauth_state"
	wechatOAuthRedirectCookieName = "wechat_oauth_redirect"
	wechatOAuthIntentCookieName   = "wechat_oauth_intent"
	wechatOAuthModeCookieName     = "wechat_oauth_mode"
	wechatOAuthDefaultRedirectTo  = "/dashboard"
	wechatOAuthDefaultFrontendCB  = "/auth/wechat/callback"
	wechatOAuthProviderKey        = "wechat-main"
)

var (
	wechatOAuthAccessTokenURL = "https://api.weixin.qq.com/sns/oauth2/access_token"
	wechatOAuthUserInfoURL    = "https://api.weixin.qq.com/sns/userinfo"
)

type wechatOAuthConfig struct {
	mode         string
	appID        string
	appSecret    string
	redirectURI  string
	authorizeURL string
	scope        string
	frontendPath string
}

type wechatOAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenID       string `json:"openid"`
	Scope        string `json:"scope"`
	UnionID      string `json:"unionid"`
	ErrCode      int64  `json:"errcode"`
	ErrMsg       string `json:"errmsg"`
}

type wechatOAuthUserInfoResponse struct {
	OpenID     string `json:"openid"`
	Nickname   string `json:"nickname"`
	HeadImgURL string `json:"headimgurl"`
	UnionID    string `json:"unionid"`
	ErrCode    int64  `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

// WeChatOAuthStart starts the WeChat OAuth flow for either open or mp mode.
// GET /api/v1/auth/oauth/wechat/start?mode=open|mp&redirect=/dashboard&intent=bind
func (h *AuthHandler) WeChatOAuthStart(c *gin.Context) {
	cfg, err := h.getWeChatOAuthConfig(c.Request.Context(), c.Query("mode"), c)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	state, err := oauth.GenerateState()
	if err != nil {
		response.ErrorFrom(c, infraerrors.InternalServer("OAUTH_STATE_GEN_FAILED", "failed to generate oauth state").WithCause(err))
		return
	}

	redirectTo := sanitizeFrontendRedirectPath(c.Query("redirect"))
	if redirectTo == "" {
		redirectTo = wechatOAuthDefaultRedirectTo
	}
	intent := normalizeOAuthIntent(c.Query("intent"))

	secureCookie := isRequestHTTPS(c)
	setCookie(c, wechatOAuthStateCookieName, encodeCookieValue(state), wechatOAuthCookieMaxAgeSec, secureCookie)
	setCookie(c, wechatOAuthRedirectCookieName, encodeCookieValue(redirectTo), wechatOAuthCookieMaxAgeSec, secureCookie)
	setCookie(c, wechatOAuthIntentCookieName, encodeCookieValue(intent), wechatOAuthCookieMaxAgeSec, secureCookie)
	setCookie(c, wechatOAuthModeCookieName, encodeCookieValue(cfg.mode), wechatOAuthCookieMaxAgeSec, secureCookie)

	authURL, err := buildWeChatAuthorizeURL(cfg, state)
	if err != nil {
		response.ErrorFrom(c, infraerrors.InternalServer("OAUTH_BUILD_URL_FAILED", "failed to build oauth authorization url").WithCause(err))
		return
	}

	c.Redirect(http.StatusFound, authURL)
}

// WeChatOAuthCallback exchanges the provider code server-side and bridges the
// resolved identity into the pending-auth-session flow.
func (h *AuthHandler) WeChatOAuthCallback(c *gin.Context) {
	frontendCallback := wechatOAuthDefaultFrontendCB

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
		wechatClearCookie(c, wechatOAuthStateCookieName, secureCookie)
		wechatClearCookie(c, wechatOAuthRedirectCookieName, secureCookie)
		wechatClearCookie(c, wechatOAuthIntentCookieName, secureCookie)
		wechatClearCookie(c, wechatOAuthModeCookieName, secureCookie)
	}()

	expectedState, err := readCookieDecoded(c, wechatOAuthStateCookieName)
	if err != nil || expectedState == "" || state != expectedState {
		redirectOAuthError(c, frontendCallback, "invalid_state", "invalid oauth state", "")
		return
	}

	redirectTo, _ := readCookieDecoded(c, wechatOAuthRedirectCookieName)
	redirectTo = sanitizeFrontendRedirectPath(redirectTo)
	if redirectTo == "" {
		redirectTo = wechatOAuthDefaultRedirectTo
	}
	intent := normalizedOAuthIntentFromCookie(c, wechatOAuthIntentCookieName)
	mode, err := readCookieDecoded(c, wechatOAuthModeCookieName)
	if err != nil || strings.TrimSpace(mode) == "" {
		redirectOAuthError(c, frontendCallback, "invalid_state", "missing oauth mode", "")
		return
	}

	cfg, err := h.getWeChatOAuthConfig(c.Request.Context(), mode, c)
	if err != nil {
		redirectOAuthError(c, frontendCallback, "provider_error", infraerrors.Reason(err), infraerrors.Message(err))
		return
	}
	tokenResp, userInfo, err := fetchWeChatOAuthIdentity(c.Request.Context(), cfg, code)
	if err != nil {
		redirectOAuthError(c, frontendCallback, "provider_error", "wechat_identity_fetch_failed", err.Error())
		return
	}

	unionid := strings.TrimSpace(firstNonEmpty(userInfo.UnionID, tokenResp.UnionID))
	openid := strings.TrimSpace(firstNonEmpty(userInfo.OpenID, tokenResp.OpenID))
	channel := cfg.mode
	appid := cfg.appID
	nickname := strings.TrimSpace(userInfo.Nickname)
	avatarURL := strings.TrimSpace(userInfo.HeadImgURL)

	providerSubject := firstNonEmpty(unionid, openid)
	if providerSubject == "" {
		redirectOAuthError(c, frontendCallback, "provider_error", "wechat_missing_subject", "")
		return
	}

	metadata := make(map[string]any, 4)
	if unionid != "" {
		metadata["unionid"] = unionid
	}
	if openid != "" {
		metadata["openid"] = openid
	}
	if channel != "" {
		metadata["channel"] = channel
	}
	if appid != "" {
		metadata["appid"] = appid
	}

	if normalizeOAuthIntent(intent) == service.PendingAuthIntentLogin {
		boundUserID, err := h.resolveBoundWeChatUserIDForCallback(c.Request.Context(), wechatOAuthProviderKey, unionid, openid, channel, appid)
		if err != nil {
			redirectOAuthError(c, frontendCallback, "login_failed", "identity_lookup_failed", "")
			return
		}
		if boundUserID != nil {
			h.redirectOAuthLoginSuccess(c, frontendCallback, redirectTo, "wechat", service.PendingAuthIntentLogin, *boundUserID)
			return
		}
	}

	h.completeOAuthCallback(c, frontendCallback, redirectTo, oauthCallbackIdentity{
		Provider:             "wechat",
		Intent:               intent,
		ProviderType:         "wechat",
		ProviderKey:          wechatOAuthProviderKey,
		ProviderSubject:      providerSubject,
		CompatEmail:          wechatSyntheticEmail(providerSubject),
		CompatUsername:       firstNonEmpty(nickname, wechatFallbackUsername(providerSubject)),
		Metadata:             metadata,
		SuggestedDisplayName: nickname,
		SuggestedAvatarURL:   avatarURL,
	})
}

func (h *AuthHandler) resolveBoundWeChatUserIDForCallback(
	ctx context.Context,
	providerKey string,
	unionid string,
	openid string,
	channel string,
	appid string,
) (*int64, error) {
	repo := extractAuthServiceUserRepo(h.authService)
	if repo == nil {
		return nil, nil
	}

	if unionid != "" {
		userID, err := lookupBoundIdentityUserID(ctx, repo, "wechat", providerKey, unionid)
		if userID != nil || err != nil {
			return userID, err
		}
	}

	if openid == "" || channel == "" || appid == "" {
		return nil, nil
	}

	channelFinder, ok := repo.(authIdentityChannelFinder)
	if !ok {
		return nil, nil
	}
	channelRecord, err := channelFinder.FindAuthIdentityChannel(ctx, "wechat", providerKey, channel, appid, openid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if channelRecord == nil {
		return nil, nil
	}

	identityGetter, ok := repo.(authIdentityByIDGetter)
	if !ok {
		return nil, nil
	}
	identity, err := identityGetter.GetAuthIdentityByID(ctx, channelRecord.IdentityID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if identity == nil {
		return nil, nil
	}

	userID := identity.UserID
	return &userID, nil
}

func wechatSyntheticEmail(subject string) string {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return ""
	}
	return "wechat-" + subject + service.WeChatConnectSyntheticEmailDomain
}

func wechatFallbackUsername(subject string) string {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return "wechat_user"
	}
	return "wechat_" + truncateFragmentValue(subject)
}

func wechatClearCookie(c *gin.Context, name string, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     wechatOAuthCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *AuthHandler) getWeChatOAuthConfig(ctx context.Context, rawMode string, c *gin.Context) (wechatOAuthConfig, error) {
	if h == nil || h.settingSvc == nil {
		return wechatOAuthConfig{}, infraerrors.ServiceUnavailable("CONFIG_NOT_READY", "wechat oauth config not loaded")
	}

	settings, err := h.settingSvc.GetAllSettings(ctx)
	if err != nil {
		return wechatOAuthConfig{}, infraerrors.ServiceUnavailable("CONFIG_NOT_READY", "failed to load wechat oauth settings").WithCause(err)
	}
	mode, err := resolveWeChatOAuthMode(rawMode, c)
	if err != nil {
		return wechatOAuthConfig{}, err
	}

	redirectURI := buildWeChatOAuthCallbackURL(c)
	cfg := wechatOAuthConfig{
		mode:         mode,
		redirectURI:  redirectURI,
		frontendPath: wechatOAuthDefaultFrontendCB,
	}

	switch mode {
	case "open":
		if !settings.WeChatLoginOpenEnabled {
			return wechatOAuthConfig{}, infraerrors.NotFound("OAUTH_DISABLED", "wechat open login is disabled")
		}
		if strings.TrimSpace(settings.WeChatLoginOpenAppID) == "" || strings.TrimSpace(settings.WeChatLoginOpenAppSecret) == "" {
			return wechatOAuthConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "wechat open app credentials are not configured")
		}
		cfg.appID = strings.TrimSpace(settings.WeChatLoginOpenAppID)
		cfg.appSecret = strings.TrimSpace(settings.WeChatLoginOpenAppSecret)
		cfg.authorizeURL = "https://open.weixin.qq.com/connect/qrconnect"
		cfg.scope = "snsapi_login"
	case "mp":
		if !settings.WeChatLoginMPEnabled {
			return wechatOAuthConfig{}, infraerrors.NotFound("OAUTH_DISABLED", "wechat mp login is disabled")
		}
		if strings.TrimSpace(settings.WeChatLoginMPAppID) == "" || strings.TrimSpace(settings.WeChatLoginMPAppSecret) == "" {
			return wechatOAuthConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "wechat mp app credentials are not configured")
		}
		cfg.appID = strings.TrimSpace(settings.WeChatLoginMPAppID)
		cfg.appSecret = strings.TrimSpace(settings.WeChatLoginMPAppSecret)
		cfg.authorizeURL = "https://open.weixin.qq.com/connect/oauth2/authorize"
		cfg.scope = "snsapi_userinfo"
	}

	return cfg, nil
}

func resolveWeChatOAuthMode(rawMode string, c *gin.Context) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(rawMode))
	if mode == "" {
		if isWeChatBrowserRequest(c) {
			mode = "mp"
		} else {
			mode = "open"
		}
	}
	if mode != "open" && mode != "mp" {
		return "", infraerrors.BadRequest("INVALID_MODE", "wechat oauth mode must be open or mp")
	}
	return mode, nil
}

func isWeChatBrowserRequest(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}
	return strings.Contains(strings.ToLower(strings.TrimSpace(c.GetHeader("User-Agent"))), "micromessenger")
}

func buildWeChatAuthorizeURL(cfg wechatOAuthConfig, state string) (string, error) {
	u, err := url.Parse(cfg.authorizeURL)
	if err != nil {
		return "", fmt.Errorf("parse authorize url: %w", err)
	}

	query := u.Query()
	query.Set("appid", cfg.appID)
	query.Set("redirect_uri", cfg.redirectURI)
	query.Set("response_type", "code")
	query.Set("scope", cfg.scope)
	query.Set("state", state)
	u.RawQuery = query.Encode()
	u.Fragment = "wechat_redirect"
	return u.String(), nil
}

func buildWeChatOAuthCallbackURL(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}

	scheme := "http"
	if isRequestHTTPS(c) {
		scheme = "https"
	}
	host := strings.TrimSpace(c.Request.Host)
	if forwardedHost := strings.TrimSpace(c.GetHeader("X-Forwarded-Host")); forwardedHost != "" {
		host = forwardedHost
	}
	if host == "" {
		return ""
	}
	return scheme + "://" + host + "/api/v1/auth/oauth/wechat/callback"
}

func (h *AuthHandler) validateWeChatCallbackIdentity(ctx context.Context, channel, appid string) error {
	channel = strings.ToLower(strings.TrimSpace(channel))
	appid = strings.TrimSpace(appid)
	if channel == "" && appid == "" {
		return nil
	}
	if channel == "" || appid == "" {
		return infraerrors.BadRequest("WECHAT_CALLBACK_INVALID", "wechat callback channel/appid is incomplete")
	}
	if h == nil || h.settingSvc == nil {
		return infraerrors.ServiceUnavailable("CONFIG_NOT_READY", "wechat oauth config not loaded")
	}

	settings, err := h.settingSvc.GetAllSettings(ctx)
	if err != nil {
		return infraerrors.ServiceUnavailable("CONFIG_NOT_READY", "failed to load wechat oauth settings").WithCause(err)
	}

	var enabled bool
	var expectedAppID string
	switch channel {
	case "open":
		enabled = settings.WeChatLoginOpenEnabled
		expectedAppID = strings.TrimSpace(settings.WeChatLoginOpenAppID)
	case "mp":
		enabled = settings.WeChatLoginMPEnabled
		expectedAppID = strings.TrimSpace(settings.WeChatLoginMPAppID)
	default:
		return infraerrors.BadRequest("WECHAT_CALLBACK_INVALID", "wechat callback channel must be open or mp")
	}

	if !enabled {
		return infraerrors.NotFound("OAUTH_DISABLED", "wechat oauth channel is disabled")
	}
	if expectedAppID == "" {
		return infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "wechat oauth channel app id is not configured")
	}
	if appid != expectedAppID {
		return infraerrors.BadRequest("WECHAT_CALLBACK_APPID_MISMATCH", "wechat callback app id does not match configured credentials")
	}
	return nil
}

func fetchWeChatOAuthIdentity(ctx context.Context, cfg wechatOAuthConfig, code string) (*wechatOAuthTokenResponse, *wechatOAuthUserInfoResponse, error) {
	tokenResp, err := exchangeWeChatOAuthCode(ctx, cfg, code)
	if err != nil {
		return nil, nil, err
	}
	userInfo, err := fetchWeChatUserInfo(ctx, tokenResp)
	if err != nil {
		return nil, nil, err
	}
	return tokenResp, userInfo, nil
}

func exchangeWeChatOAuthCode(ctx context.Context, cfg wechatOAuthConfig, code string) (*wechatOAuthTokenResponse, error) {
	endpoint, err := url.Parse(wechatOAuthAccessTokenURL)
	if err != nil {
		return nil, fmt.Errorf("parse wechat access token url: %w", err)
	}
	query := endpoint.Query()
	query.Set("appid", cfg.appID)
	query.Set("secret", cfg.appSecret)
	query.Set("code", strings.TrimSpace(code))
	query.Set("grant_type", "authorization_code")
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build wechat access token request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request wechat access token: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("wechat access token status=%d", resp.StatusCode)
	}

	var payload wechatOAuthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode wechat access token response: %w", err)
	}
	if payload.ErrCode != 0 {
		return nil, fmt.Errorf("wechat access token errcode=%d errmsg=%s", payload.ErrCode, strings.TrimSpace(payload.ErrMsg))
	}
	if strings.TrimSpace(payload.AccessToken) == "" || strings.TrimSpace(payload.OpenID) == "" {
		return nil, fmt.Errorf("wechat access token response missing access_token/openid")
	}
	return &payload, nil
}

func fetchWeChatUserInfo(ctx context.Context, tokenResp *wechatOAuthTokenResponse) (*wechatOAuthUserInfoResponse, error) {
	if tokenResp == nil {
		return nil, fmt.Errorf("wechat token response is required")
	}

	endpoint, err := url.Parse(wechatOAuthUserInfoURL)
	if err != nil {
		return nil, fmt.Errorf("parse wechat userinfo url: %w", err)
	}
	query := endpoint.Query()
	query.Set("access_token", strings.TrimSpace(tokenResp.AccessToken))
	query.Set("openid", strings.TrimSpace(tokenResp.OpenID))
	query.Set("lang", "zh_CN")
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build wechat userinfo request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request wechat userinfo: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("wechat userinfo status=%d", resp.StatusCode)
	}

	var payload wechatOAuthUserInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode wechat userinfo response: %w", err)
	}
	if payload.ErrCode != 0 {
		return nil, fmt.Errorf("wechat userinfo errcode=%d errmsg=%s", payload.ErrCode, strings.TrimSpace(payload.ErrMsg))
	}
	if strings.TrimSpace(payload.OpenID) == "" {
		payload.OpenID = strings.TrimSpace(tokenResp.OpenID)
	}
	if strings.TrimSpace(payload.UnionID) == "" {
		payload.UnionID = strings.TrimSpace(tokenResp.UnionID)
	}
	return &payload, nil
}
