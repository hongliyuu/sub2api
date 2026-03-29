package handler

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

// TestNewCopilotGatewayHandler_DefaultMaxBodyBytes verifies that the handler
// correctly resolves the system-level body size limit from configuration.
func TestNewCopilotGatewayHandler_DefaultMaxBodyBytes(t *testing.T) {
	tests := []struct {
		name                string
		copilotDefaultMaxKB int64
		wantBytes           int
	}{
		{
			name:                "nil config falls back to built-in default",
			copilotDefaultMaxKB: 0,
			wantBytes:           fallbackCopilotMaxBodyBytes,
		},
		{
			name:                "zero config value falls back to built-in default",
			copilotDefaultMaxKB: 0,
			wantBytes:           fallbackCopilotMaxBodyBytes,
		},
		{
			name:                "configured value is converted from KB to bytes",
			copilotDefaultMaxKB: 1000,
			wantBytes:           1000 * 1024,
		},
		{
			name:                "large configured value is accepted",
			copilotDefaultMaxKB: 8000,
			wantBytes:           8000 * 1024,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Gateway.CopilotDefaultMaxBodyKB = tc.copilotDefaultMaxKB

			h := NewCopilotGatewayHandler(nil, nil, nil, nil, nil, cfg)
			require.Equal(t, tc.wantBytes, h.defaultMaxBodyBytes)
		})
	}
}

func TestNewCopilotGatewayHandler_NilConfig(t *testing.T) {
	h := NewCopilotGatewayHandler(nil, nil, nil, nil, nil, nil)
	require.Equal(t, fallbackCopilotMaxBodyBytes, h.defaultMaxBodyBytes,
		"nil config should fall back to built-in default")
}
