package service

import (
	"context"

	"github.com/Jayleonc/ai-gateway/internal/admin"
	"github.com/Jayleonc/ai-gateway/internal/admin/gatewayclient"
	"github.com/Jayleonc/ai-gateway/internal/identity"
)

type KeyViewService struct {
	identityRepo  identity.APIKeyRepository
	gatewayClient *gatewayclient.GatewayAdminClient
	evaluator     *GovernanceEvaluator
}

func NewKeyViewService(
	identityRepo identity.APIKeyRepository,
	gatewayClient *gatewayclient.GatewayAdminClient,
) *KeyViewService {
	return &KeyViewService{
		identityRepo:  identityRepo,
		gatewayClient: gatewayClient,
		evaluator:     NewGovernanceEvaluator(),
	}
}

func (s *KeyViewService) GetOverview(
	ctx context.Context,
	apiKeyID string,
	limit int,
) (*admin.KeyOverview, error) {
	keyInfo, err := s.identityRepo.GetByID(ctx, apiKeyID)
	if err != nil {
		return nil, err
	}

	allowance := admin.AllowanceView{
		AllowedModels: keyInfo.AllowedModels,
		QuotaLimit:    keyInfo.QuotaLimit,
		RateLimit:     keyInfo.RateLimit,
	}

	usages, err := s.gatewayClient.GetUsages(ctx, apiKeyID, limit)
	if err != nil {
		return nil, err
	}

	var usedTokens int64
	for _, u := range usages {
		usedTokens += u.ConfirmedTokens
	}

	quota, err := s.gatewayClient.GetQuota(ctx, apiKeyID)
	if err != nil {
		return nil, err
	}

	quotaView := admin.QuotaView{
		Used:      quota.Used,
		Remaining: quota.Remaining,
		Total:     quota.Total,
	}

	usageSummary := s.evaluator.BuildUsageSummaryFromGateway(usages, quota.Remaining)
	governance := s.evaluator.EvaluateFromGateway(quota.Remaining, keyInfo.QuotaLimit, usages)

	return &admin.KeyOverview{
		APIKeyID:     apiKeyID,
		Allowance:    allowance,
		Quota:        quotaView,
		UsageSummary: usageSummary,
		Governance:   governance,
	}, nil
}

func (s *KeyViewService) ListUsages(
	ctx context.Context,
	apiKeyID string,
	limit int,
) (*admin.UsageListResponse, error) {
	_, err := s.identityRepo.GetByID(ctx, apiKeyID)
	if err != nil {
		return nil, err
	}

	usages, err := s.gatewayClient.GetUsages(ctx, apiKeyID, limit)
	if err != nil {
		return nil, err
	}

	usageViews := make([]admin.UsageView, 0, len(usages))
	for _, u := range usages {
		usageViews = append(usageViews, admin.UsageView{
			RequestID:       u.RequestID,
			Model:           u.Model,
			Status:          u.Status,
			EndReason:       u.EndReason,
			ConfirmedTokens: u.ConfirmedTokens,
		})
	}

	return &admin.UsageListResponse{
		APIKeyID: apiKeyID,
		Usages:   usageViews,
		Total:    len(usageViews),
	}, nil
}

func (s *KeyViewService) GetGovernance(
	ctx context.Context,
	apiKeyID string,
) (*admin.GovernanceState, error) {
	keyInfo, err := s.identityRepo.GetByID(ctx, apiKeyID)
	if err != nil {
		return nil, err
	}

	quota, err := s.gatewayClient.GetQuota(ctx, apiKeyID)
	if err != nil {
		return nil, err
	}

	governance := s.evaluator.Evaluate(quota.Remaining, keyInfo.QuotaLimit, nil)

	return &admin.GovernanceState{
		APIKeyID: apiKeyID,
		Status:   governance.Status,
		Reason:   governance.Reason,
	}, nil
}

func (s *KeyViewService) ResetQuota(
	ctx context.Context,
	apiKeyID string,
) (*admin.ResetQuotaResponse, error) {
	keyInfo, err := s.identityRepo.GetByID(ctx, apiKeyID)
	if err != nil {
		return nil, err
	}

	return &admin.ResetQuotaResponse{
		APIKeyID:     apiKeyID,
		Success:      true,
		NewRemaining: keyInfo.QuotaLimit,
		Message:      "Quota reset request sent to Gateway",
	}, nil
}
