package logger

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// AccessLogEntry is the structured data emitted per LLM gateway request.
type AccessLogEntry struct {
	RequestID           string `json:"request_id"`
	ClientRequestID     string `json:"client_request_id"`
	SessionHash         string `json:"session_hash,omitempty"`
	Platform            string `json:"platform"`
	Model               string `json:"model"`
	AccountID           int64  `json:"account_id"`
	GroupID             int64  `json:"group_id,omitempty"`
	UserID              int64  `json:"user_id"`
	APIKeyID            int64  `json:"api_key_id"`
	Stream              bool   `json:"stream"`
	RequestType         string `json:"request_type,omitempty"`
	InboundEndpoint     string `json:"inbound_endpoint,omitempty"`
	UpstreamModel       string `json:"upstream_model,omitempty"`
	ClientIP            string `json:"client_ip,omitempty"`
	UserAgent           string `json:"user_agent,omitempty"`
	StatusCode          int    `json:"status_code"`
	Success             bool   `json:"success"`
	DurationMs          int    `json:"duration_ms"`
	FirstTokenMs        int    `json:"first_token_ms,omitempty"`
	AuthLatencyMs       int    `json:"auth_latency_ms,omitempty"`
	RouteLatencyMs      int    `json:"route_latency_ms,omitempty"`
	InputTokens         int    `json:"input_tokens,omitempty"`
	OutputTokens        int    `json:"output_tokens,omitempty"`
	CacheCreationTokens int    `json:"cache_creation_tokens,omitempty"`
	CacheReadTokens     int    `json:"cache_read_tokens,omitempty"`
	RetryCount          int    `json:"retry_count,omitempty"`
	AccountSwitchCount  int    `json:"account_switch_count,omitempty"`
	RequestBody         string `json:"request_body,omitempty"`
	ResponseBody        string `json:"response_body,omitempty"`
}

type AccessLogOptions struct {
	FilePath     string
	IncludeBody  bool
	MaxBodyBytes int
	MaxSizeMB    int
	MaxBackups   int
	MaxAgeDays   int
	Compress     bool
}

type AccessLogger struct {
	logger *zap.Logger
	opts   AccessLogOptions
}

func NewAccessLogger(opts AccessLogOptions) *AccessLogger {
	if opts.FilePath == "" {
		opts.FilePath = resolveDefaultAccessLogPath()
	}
	if opts.MaxBodyBytes <= 0 {
		opts.MaxBodyBytes = 10240
	}
	if opts.MaxSizeMB <= 0 {
		opts.MaxSizeMB = 200
	}
	if opts.MaxBackups <= 0 {
		opts.MaxBackups = 10
	}
	if opts.MaxAgeDays <= 0 {
		opts.MaxAgeDays = 30
	}

	dir := filepath.Dir(opts.FilePath)
	_ = os.MkdirAll(dir, 0o755)

	writer := &lumberjack.Logger{
		Filename:   opts.FilePath,
		MaxSize:    opts.MaxSizeMB,
		MaxBackups: opts.MaxBackups,
		MaxAge:     opts.MaxAgeDays,
		Compress:   opts.Compress,
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:    "ts",
		LevelKey:   "",
		MessageKey: "",
		EncodeTime: zapcore.ISO8601TimeEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(writer),
		zapcore.InfoLevel,
	)

	return &AccessLogger{
		logger: zap.New(core),
		opts:   opts,
	}
}

func (al *AccessLogger) Emit(entry AccessLogEntry) {
	if al == nil || al.logger == nil {
		return
	}

	fields := []zap.Field{
		zap.String("request_id", entry.RequestID),
		zap.String("client_request_id", entry.ClientRequestID),
		zap.String("platform", entry.Platform),
		zap.String("model", entry.Model),
		zap.Int64("account_id", entry.AccountID),
		zap.Int64("user_id", entry.UserID),
		zap.Int64("api_key_id", entry.APIKeyID),
		zap.Bool("stream", entry.Stream),
		zap.Int("status_code", entry.StatusCode),
		zap.Bool("success", entry.Success),
		zap.Int("duration_ms", entry.DurationMs),
	}

	if entry.SessionHash != "" {
		fields = append(fields, zap.String("session_hash", entry.SessionHash))
	}
	if entry.GroupID != 0 {
		fields = append(fields, zap.Int64("group_id", entry.GroupID))
	}
	if entry.RequestType != "" {
		fields = append(fields, zap.String("request_type", entry.RequestType))
	}
	if entry.InboundEndpoint != "" {
		fields = append(fields, zap.String("inbound_endpoint", entry.InboundEndpoint))
	}
	if entry.UpstreamModel != "" {
		fields = append(fields, zap.String("upstream_model", entry.UpstreamModel))
	}
	if entry.ClientIP != "" {
		fields = append(fields, zap.String("client_ip", entry.ClientIP))
	}
	if entry.UserAgent != "" {
		fields = append(fields, zap.String("user_agent", entry.UserAgent))
	}
	if entry.FirstTokenMs > 0 {
		fields = append(fields, zap.Int("first_token_ms", entry.FirstTokenMs))
	}
	if entry.AuthLatencyMs > 0 {
		fields = append(fields, zap.Int("auth_latency_ms", entry.AuthLatencyMs))
	}
	if entry.RouteLatencyMs > 0 {
		fields = append(fields, zap.Int("route_latency_ms", entry.RouteLatencyMs))
	}
	if entry.InputTokens > 0 {
		fields = append(fields, zap.Int("input_tokens", entry.InputTokens))
	}
	if entry.OutputTokens > 0 {
		fields = append(fields, zap.Int("output_tokens", entry.OutputTokens))
	}
	if entry.CacheCreationTokens > 0 {
		fields = append(fields, zap.Int("cache_creation_tokens", entry.CacheCreationTokens))
	}
	if entry.CacheReadTokens > 0 {
		fields = append(fields, zap.Int("cache_read_tokens", entry.CacheReadTokens))
	}
	if entry.RetryCount > 0 {
		fields = append(fields, zap.Int("retry_count", entry.RetryCount))
	}
	if entry.AccountSwitchCount > 0 {
		fields = append(fields, zap.Int("account_switch_count", entry.AccountSwitchCount))
	}
	if entry.RequestBody != "" {
		fields = append(fields, zap.String("request_body", entry.RequestBody))
	}
	if entry.ResponseBody != "" {
		fields = append(fields, zap.String("response_body", entry.ResponseBody))
	}

	al.logger.Info("llm_request", fields...)
}

func (al *AccessLogger) Close() {
	if al == nil || al.logger == nil {
		return
	}
	_ = al.logger.Sync()
}

func resolveDefaultAccessLogPath() string {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "/app/data"
	}
	return filepath.Join(dataDir, "logs", "access.log")
}
