package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type vertexAccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

func (p *GeminiTokenProvider) getVertexAccessToken(ctx context.Context, account *Account) (string, error) {
	if account == nil {
		return "", fmt.Errorf("vertex account is nil")
	}
	if account.Platform != PlatformGemini || account.Type != AccountTypeVertex {
		return "", fmt.Errorf("not a vertex gemini account")
	}

	cacheKey := GeminiTokenCacheKey(account)
	if p.tokenCache != nil {
		if token, err := p.tokenCache.GetAccessToken(ctx, cacheKey); err == nil && strings.TrimSpace(token) != "" {
			return token, nil
		}
	}

	if p.tokenCache != nil {
		acquired, err := p.tokenCache.AcquireRefreshLock(ctx, cacheKey, 30*time.Second)
		if err == nil && acquired {
			defer func() { _ = p.tokenCache.ReleaseRefreshLock(ctx, cacheKey) }()
		}
	}

	token, expiresAt, err := p.exchangeVertexServiceAccountToken(ctx, account)
	if err != nil {
		return "", err
	}

	if p.tokenCache != nil {
		ttl := time.Until(expiresAt)
		switch {
		case ttl > geminiTokenCacheSkew:
			ttl -= geminiTokenCacheSkew
		case ttl <= 0:
			ttl = time.Minute
		}
		_ = p.tokenCache.SetAccessToken(ctx, cacheKey, token, ttl)
	}

	return token, nil
}

func (p *GeminiTokenProvider) exchangeVertexServiceAccountToken(ctx context.Context, account *Account) (string, time.Time, error) {
	if p.httpUpstream == nil {
		return "", time.Time{}, fmt.Errorf("http upstream not configured")
	}

	creds, err := parseVertexServiceAccountCredentials(account.GetCredential("service_account_json"))
	if err != nil {
		return "", time.Time{}, err
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(creds.PrivateKey))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("parse vertex private_key: %w", err)
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   strings.TrimSpace(creds.ClientEmail),
		"scope": vertexGeminiCloudScope,
		"aud":   strings.TrimSpace(creds.TokenURI),
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}

	signedJWT := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	if strings.TrimSpace(creds.PrivateKeyID) != "" {
		signedJWT.Header["kid"] = strings.TrimSpace(creds.PrivateKeyID)
	}
	assertion, err := signedJWT.SignedString(privateKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign vertex jwt assertion: %w", err)
	}

	form := url.Values{}
	form.Set("grant_type", vertexJWTGrantType)
	form.Set("assertion", assertion)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSpace(creds.TokenURI), bytes.NewBufferString(form.Encode()))
	if err != nil {
		return "", time.Time{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	proxyURL := p.vertexProxyURL(ctx, account)
	resp, err := p.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return "", time.Time{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", time.Time{}, fmt.Errorf("vertex token exchange failed: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var tokenResp vertexAccessTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", time.Time{}, fmt.Errorf("parse vertex token response: %w", err)
	}
	if strings.TrimSpace(tokenResp.AccessToken) == "" {
		return "", time.Time{}, fmt.Errorf("vertex token exchange returned empty access_token")
	}

	expiresAt := now.Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	if tokenResp.ExpiresIn <= 0 {
		expiresAt = now.Add(time.Hour)
	}
	return tokenResp.AccessToken, expiresAt, nil
}

func (p *GeminiTokenProvider) vertexProxyURL(ctx context.Context, account *Account) string {
	if account == nil || account.ProxyID == nil {
		return ""
	}
	if account.Proxy != nil {
		return account.Proxy.URL()
	}
	if p.proxyRepo == nil {
		return ""
	}
	proxy, err := p.proxyRepo.GetByID(ctx, *account.ProxyID)
	if err != nil || proxy == nil {
		return ""
	}
	return proxy.URL()
}
