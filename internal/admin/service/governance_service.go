package service

import (
	"context"

	"github.com/Jayleonc/ai-gateway/internal/admin"
	"github.com/Jayleonc/ai-gateway/internal/admin/gatewayclient"
	"github.com/Jayleonc/ai-gateway/internal/identity"
)

type GovernanceService struct {
	identityRepo  identity.APIKeyRepository
	gatewayClient *gatewayclient.GatewayAdminClient
	evaluator     *GovernanceEvaluator
	knownKeyIDs   []string
}

func NewGovernanceService(
	identityRepo identity.APIKeyRepository,
	gatewayClient *gatewayclient.GatewayAdminClient,
	knownKeyIDs []string,
) *GovernanceService {
	return &GovernanceService{
		identityRepo:  identityRepo,
		gatewayClient: gatewayClient,
		evaluator:     NewGovernanceEvaluator(),
		knownKeyIDs:   knownKeyIDs,
	}
}

func (s *GovernanceService) GetSummary(ctx context.Context) (*admin.GovernanceSummary, error) {
	var active, limited, exhausted int

	for _, keyID := range s.knownKeyIDs {
		keyInfo, err := s.identityRepo.GetByID(ctx, keyID)
		if err != nil {
			continue
		}

		quota, err := s.gatewayClient.GetQuota(ctx, keyID)
		if err != nil {
			continue
		}

		status := s.evaluator.EvaluateStatus(quota.Remaining, keyInfo.QuotaLimit)

		switch status {
		case GovernanceStatusActive:
			active++
		case GovernanceStatusLimited:
			limited++
		case GovernanceStatusExhausted:
			exhausted++
		}
	}

	return &admin.GovernanceSummary{
		TotalKeys:     active + limited + exhausted,
		ActiveKeys:    active,
		LimitedKeys:   limited,
		ExhaustedKeys: exhausted,
	}, nil
}
