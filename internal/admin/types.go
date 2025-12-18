package admin

type KeyOverview struct {
	APIKeyID     string           `json:"api_key_id"`
	Allowance    AllowanceView    `json:"allowance"`
	Quota        QuotaView        `json:"quota"`
	UsageSummary UsageSummaryView `json:"usage_summary"`
	Governance   GovernanceView   `json:"governance"`
}

type AllowanceView struct {
	AllowedModels []string `json:"allowed_models"`
	QuotaLimit    int64    `json:"quota_limit"`
	RateLimit     int      `json:"rate_limit"`
}

type QuotaView struct {
	Used      int64 `json:"used"`
	Remaining int64 `json:"remaining"`
	Total     int64 `json:"total"`
}

type UsageSummaryView struct {
	RecentRequests       int   `json:"recent_requests"`
	TotalConfirmedTokens int64 `json:"total_confirmed_tokens"`
	HasQuotaExceeded     bool  `json:"has_quota_exceeded"`
	HasPartial           bool  `json:"has_partial"`
}

type GovernanceView struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type UsageView struct {
	RequestID       string `json:"request_id"`
	Model           string `json:"model"`
	Status          string `json:"status"`
	EndReason       string `json:"end_reason"`
	ConfirmedTokens int64  `json:"confirmed_tokens"`
	StartedAt       int64  `json:"started_at"`
	EndedAt         int64  `json:"ended_at"`
}

type UsageListResponse struct {
	APIKeyID string      `json:"api_key_id"`
	Usages   []UsageView `json:"usages"`
	Total    int         `json:"total"`
}

type GovernanceState struct {
	APIKeyID string `json:"api_key_id"`
	Status   string `json:"status"`
	Reason   string `json:"reason"`
}

type GovernanceSummary struct {
	TotalKeys     int `json:"total_keys"`
	ActiveKeys    int `json:"active_keys"`
	LimitedKeys   int `json:"limited_keys"`
	ExhaustedKeys int `json:"exhausted_keys"`
}

type ResetQuotaResponse struct {
	APIKeyID     string `json:"api_key_id"`
	Success      bool   `json:"success"`
	NewRemaining int64  `json:"new_remaining"`
	Message      string `json:"message"`
}
