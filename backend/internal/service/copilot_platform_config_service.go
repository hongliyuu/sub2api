package service

import "context"

// CopilotPlatformConfigService 管理 Copilot 平台级参数配置。
type CopilotPlatformConfigService struct {
	repo CopilotPlatformConfigRepository
}

func NewCopilotPlatformConfigService(repo CopilotPlatformConfigRepository) *CopilotPlatformConfigService {
	return &CopilotPlatformConfigService{repo: repo}
}

// GetAll 返回全部 5 个 plan_type 的配置。
func (s *CopilotPlatformConfigService) GetAll(ctx context.Context) ([]CopilotPlatformConfigEntry, error) {
	return s.repo.GetAll(ctx)
}

// GetByPlanType 返回指定 plan_type 的配置。
func (s *CopilotPlatformConfigService) GetByPlanType(ctx context.Context, planType string) (*CopilotPlatformConfigEntry, error) {
	return s.repo.GetByPlanType(ctx, planType)
}

// UpdateByPlanType 更新指定 plan_type 的配置。
func (s *CopilotPlatformConfigService) UpdateByPlanType(ctx context.Context, planType string, patch CopilotPlatformConfigPatch) (*CopilotPlatformConfigEntry, error) {
	return s.repo.Upsert(ctx, planType, patch)
}
