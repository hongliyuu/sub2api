package admin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMaskString(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		keepPrefix int
		keepSuffix int
		expected   string
	}{
		{
			name:       "empty string",
			input:      "",
			keepPrefix: 6,
			keepSuffix: 4,
			expected:   "",
		},
		{
			name:       "short string not masked",
			input:      "abc123",
			keepPrefix: 6,
			keepSuffix: 4,
			expected:   "abc123",
		},
		{
			name:       "exact length not masked",
			input:      "1234567890",
			keepPrefix: 6,
			keepSuffix: 4,
			expected:   "1234567890",
		},
		{
			name:       "typical appid mask",
			input:      "wx0b35f0a1b2fb07e",
			keepPrefix: 6,
			keepSuffix: 4,
			expected:   "wx0b35*******b07e",
		},
		{
			name:       "long string mask",
			input:      "abcdefghijklmnopqrstuvwxyz",
			keepPrefix: 4,
			keepSuffix: 4,
			expected:   "abcd******************wxyz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskString(tt.input, tt.keepPrefix, tt.keepSuffix)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestMaskAppID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty appid",
			input:    "",
			expected: "",
		},
		{
			name:     "short appid",
			input:    "wx12345",
			expected: "wx12345",
		},
		{
			name:     "typical appid",
			input:    "wx0b35f0a1b2fb07e",
			expected: "wx0b35*******b07e",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskAppID(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestMaskMchID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty mchid",
			input:    "",
			expected: "",
		},
		{
			name:     "short mchid",
			input:    "12345",
			expected: "12345",
		},
		{
			name:     "typical mchid",
			input:    "1234567890",
			expected: "1234**7890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskMchID(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}
