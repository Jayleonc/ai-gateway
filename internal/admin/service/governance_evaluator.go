package service

import (
	"github.com/Jayleonc/ai-gateway/internal/admin"
	"github.com/Jayleonc/ai-gateway/internal/admin/gatewayclient"
	"github.com/Jayleonc/ai-gateway/internal/metering"
)

const (
	GovernanceStatusActive    = "active"
	GovernanceStatusLimited   = "limited"
	GovernanceStatusExhausted = "exhausted"
)

type GovernanceEvaluator struct{}

func NewGovernanceEvaluator() *GovernanceEvaluator {
	return &GovernanceEvaluator{}
}

func (e *GovernanceEvaluator) Evaluate(remaining int64, quotaLimit int64, records []*metering.UsageRecord) admin.GovernanceView {
	if remaining <= 0 {
		return admin.GovernanceView{
			Status: GovernanceStatusExhausted,
			Reason: "Quota exhausted, no remaining tokens available",
		}
	}

	if quotaLimit > 0 && remaining < quotaLimit/10 {
		return admin.GovernanceView{
			Status: GovernanceStatusLimited,
			Reason: "Quota running low, less than 10% remaining",
		}
	}

	return admin.GovernanceView{
		Status: GovernanceStatusActive,
		Reason: "Key is active and within quota limits",
	}
}

func (e *GovernanceEvaluator) EvaluateStatus(remaining int64, quotaLimit int64) string {
	if remaining <= 0 {
		return GovernanceStatusExhausted
	}
	if quotaLimit > 0 && remaining < quotaLimit/10 {
		return GovernanceStatusLimited
	}
	return GovernanceStatusActive
}

func (e *GovernanceEvaluator) BuildUsageSummary(records []*metering.UsageRecord, remaining int64) admin.UsageSummaryView {
	var totalTokens int64
	hasPartial := false
	hasQuotaExceeded := remaining <= 0

	for _, r := range records {
		totalTokens += r.ConfirmedTokens
		if r.EndReason == "quota_exceeded" {
			hasQuotaExceeded = true
		}
		if r.EndReason == "partial" || r.Status == "partial" {
			hasPartial = true
		}
	}

	return admin.UsageSummaryView{
		RecentRequests:       len(records),
		TotalConfirmedTokens: totalTokens,
		HasQuotaExceeded:     hasQuotaExceeded,
		HasPartial:           hasPartial,
	}
}

func (e *GovernanceEvaluator) BuildUsageSummaryFromGateway(usages []gatewayclient.UsageFact, remaining int64) admin.UsageSummaryView {
	var totalTokens int64
	hasPartial := false
	hasQuotaExceeded := remaining <= 0

	for _, u := range usages {
		totalTokens += u.ConfirmedTokens
		if u.EndReason == "quota_exceeded" {
			hasQuotaExceeded = true
		}
		if u.EndReason == "partial" || u.Status == "partial" {
			hasPartial = true
		}
	}

	return admin.UsageSummaryView{
		RecentRequests:       len(usages),
		TotalConfirmedTokens: totalTokens,
		HasQuotaExceeded:     hasQuotaExceeded,
		HasPartial:           hasPartial,
	}
}

func (e *GovernanceEvaluator) EvaluateFromGateway(remaining int64, quotaLimit int64, usages []gatewayclient.UsageFact) admin.GovernanceView {
	if remaining <= 0 {
		return admin.GovernanceView{
			Status: GovernanceStatusExhausted,
			Reason: "Quota exhausted, no remaining tokens available",
		}
	}

	if quotaLimit > 0 && remaining < quotaLimit/10 {
		return admin.GovernanceView{
			Status: GovernanceStatusLimited,
			Reason: "Quota running low, less than 10% remaining",
		}
	}

	return admin.GovernanceView{
		Status: GovernanceStatusActive,
		Reason: "Key is active and within quota limits",
	}
}
