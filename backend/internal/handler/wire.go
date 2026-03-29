package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/google/wire"
)

// ProvideAccessLogger creates an AccessLogger from config, or nil when disabled.
func ProvideAccessLogger(cfg *config.Config) *logger.AccessLogger {
	if !cfg.AccessLog.Enabled {
		return nil
	}
	opts := logger.AccessLogOptions{
		FilePath:     cfg.AccessLog.FilePath,
		IncludeBody:  cfg.AccessLog.IncludeBody,
		MaxBodyBytes: cfg.AccessLog.MaxBodyBytes,
	}
	if cfg.AccessLog.Rotation.MaxSizeMB > 0 {
		opts.MaxSizeMB = cfg.AccessLog.Rotation.MaxSizeMB
	}
	if cfg.AccessLog.Rotation.MaxBackups > 0 {
		opts.MaxBackups = cfg.AccessLog.Rotation.MaxBackups
	}
	if cfg.AccessLog.Rotation.MaxAgeDays > 0 {
		opts.MaxAgeDays = cfg.AccessLog.Rotation.MaxAgeDays
	}
	opts.Compress = cfg.AccessLog.Rotation.Compress
	return logger.NewAccessLogger(opts)
}

// ProvideAdminHandlers creates the AdminHandlers struct
func ProvideAdminHandlers(
	dashboardHandler *admin.DashboardHandler,
	userHandler *admin.UserHandler,
	groupHandler *admin.GroupHandler,
	accountHandler *admin.AccountHandler,
	announcementHandler *admin.AnnouncementHandler,
	dataManagementHandler *admin.DataManagementHandler,
	backupHandler *admin.BackupHandler,
	oauthHandler *admin.OAuthHandler,
	openaiOAuthHandler *admin.OpenAIOAuthHandler,
	geminiOAuthHandler *admin.GeminiOAuthHandler,
	antigravityOAuthHandler *admin.AntigravityOAuthHandler,
	proxyHandler *admin.ProxyHandler,
	redeemHandler *admin.RedeemHandler,
	promoHandler *admin.PromoHandler,
	settingHandler *admin.SettingHandler,
	opsHandler *admin.OpsHandler,
	systemHandler *admin.SystemHandler,
	subscriptionHandler *admin.SubscriptionHandler,
	usageHandler *admin.UsageHandler,
	userAttributeHandler *admin.UserAttributeHandler,
	errorPassthroughHandler *admin.ErrorPassthroughHandler,
	tlsFingerprintProfileHandler *admin.TLSFingerprintProfileHandler,
	apiKeyHandler *admin.AdminAPIKeyHandler,
	scheduledTestHandler *admin.ScheduledTestHandler,
) *AdminHandlers {
	return &AdminHandlers{
		Dashboard:             dashboardHandler,
		User:                  userHandler,
		Group:                 groupHandler,
		Account:               accountHandler,
		Announcement:          announcementHandler,
		DataManagement:        dataManagementHandler,
		Backup:                backupHandler,
		OAuth:                 oauthHandler,
		OpenAIOAuth:           openaiOAuthHandler,
		GeminiOAuth:           geminiOAuthHandler,
		AntigravityOAuth:      antigravityOAuthHandler,
		Proxy:                 proxyHandler,
		Redeem:                redeemHandler,
		Promo:                 promoHandler,
		Setting:               settingHandler,
		Ops:                   opsHandler,
		System:                systemHandler,
		Subscription:          subscriptionHandler,
		Usage:                 usageHandler,
		UserAttribute:         userAttributeHandler,
		ErrorPassthrough:      errorPassthroughHandler,
		TLSFingerprintProfile: tlsFingerprintProfileHandler,
		APIKey:                apiKeyHandler,
		ScheduledTest:         scheduledTestHandler,
	}
}

// ProvideSystemHandler creates admin.SystemHandler with UpdateService
func ProvideSystemHandler(updateService *service.UpdateService, lockService *service.SystemOperationLockService) *admin.SystemHandler {
	return admin.NewSystemHandler(updateService, lockService)
}

// ProvideSettingHandler creates SettingHandler with version from BuildInfo
func ProvideSettingHandler(settingService *service.SettingService, buildInfo BuildInfo) *SettingHandler {
	return NewSettingHandler(settingService, buildInfo.Version)
}

// ProvideHandlers creates the Handlers struct
func ProvideHandlers(
	authHandler *AuthHandler,
	userHandler *UserHandler,
	apiKeyHandler *APIKeyHandler,
	usageHandler *UsageHandler,
	redeemHandler *RedeemHandler,
	subscriptionHandler *SubscriptionHandler,
	announcementHandler *AnnouncementHandler,
	adminHandlers *AdminHandlers,
	gatewayHandler *GatewayHandler,
	openaiGatewayHandler *OpenAIGatewayHandler,
	soraGatewayHandler *SoraGatewayHandler,
	soraClientHandler *SoraClientHandler,
	settingHandler *SettingHandler,
	totpHandler *TotpHandler,
	_ *service.IdempotencyCoordinator,
	_ *service.IdempotencyCleanupService,
) *Handlers {
	return &Handlers{
		Auth:          authHandler,
		User:          userHandler,
		APIKey:        apiKeyHandler,
		Usage:         usageHandler,
		Redeem:        redeemHandler,
		Subscription:  subscriptionHandler,
		Announcement:  announcementHandler,
		Admin:         adminHandlers,
		Gateway:       gatewayHandler,
		OpenAIGateway: openaiGatewayHandler,
		SoraGateway:   soraGatewayHandler,
		SoraClient:    soraClientHandler,
		Setting:       settingHandler,
		Totp:          totpHandler,
	}
}

// ProviderSet is the Wire provider set for all handlers
var ProviderSet = wire.NewSet(
	// Access logger provider
	ProvideAccessLogger,

	// Top-level handlers
	NewAuthHandler,
	NewUserHandler,
	NewAPIKeyHandler,
	NewUsageHandler,
	NewRedeemHandler,
	NewSubscriptionHandler,
	NewAnnouncementHandler,
	NewGatewayHandler,
	NewOpenAIGatewayHandler,
	NewSoraGatewayHandler,
	NewTotpHandler,
	ProvideSettingHandler,

	// Admin handlers
	admin.NewDashboardHandler,
	admin.NewUserHandler,
	admin.NewGroupHandler,
	admin.NewAccountHandler,
	admin.NewAnnouncementHandler,
	admin.NewDataManagementHandler,
	admin.NewBackupHandler,
	admin.NewOAuthHandler,
	admin.NewOpenAIOAuthHandler,
	admin.NewGeminiOAuthHandler,
	admin.NewAntigravityOAuthHandler,
	admin.NewProxyHandler,
	admin.NewRedeemHandler,
	admin.NewPromoHandler,
	admin.NewSettingHandler,
	admin.NewOpsHandler,
	ProvideSystemHandler,
	admin.NewSubscriptionHandler,
	admin.NewUsageHandler,
	admin.NewUserAttributeHandler,
	admin.NewErrorPassthroughHandler,
	admin.NewTLSFingerprintProfileHandler,
	admin.NewAdminAPIKeyHandler,
	admin.NewScheduledTestHandler,

	// AdminHandlers and Handlers constructors
	ProvideAdminHandlers,
	ProvideHandlers,
)
