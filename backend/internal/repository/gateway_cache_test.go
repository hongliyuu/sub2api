package repository

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestBuildUserAgentKeyPrefix(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.Config
		want string
	}{
		{
			name: "默认前缀",
			cfg:  nil,
			want: "sub2api",
		},
		{
			name: "自定义前缀优先",
			cfg: &config.Config{
				Gateway: config.GatewayConfig{
					UserAgentCacheKeyPrefix: "sub2api:prod",
				},
			},
			want: "sub2api:prod",
		},
		{
			name: "根据运行模式构建",
			cfg: &config.Config{
				RunMode: "standard",
				Server: config.ServerConfig{
					Mode: "release",
				},
			},
			want: "sub2api:standard:release",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, buildUserAgentKeyPrefix(tt.cfg))
		})
	}
}
