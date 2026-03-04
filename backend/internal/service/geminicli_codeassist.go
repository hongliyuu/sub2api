package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
)

// GeminiCliCodeAssistClient calls GeminiCli internal Code Assist endpoints.
type GeminiCliCodeAssistClient interface {
	LoadCodeAssist(ctx context.Context, accessToken, proxyURL string, req *geminicli.LoadCodeAssistRequest, userAgent string) (*geminicli.LoadCodeAssistResponse, error)
	OnboardUser(ctx context.Context, accessToken, proxyURL string, req *geminicli.OnboardUserRequest, userAgent string) (*geminicli.OnboardUserResponse, error)
}
