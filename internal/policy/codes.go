package policy

const (
	DenyCodeQuotaExceeded = "quota_exceeded"
	DenyCodeRateLimited   = "rate_limited"

	ErrQuotaExceeded = DenyCodeQuotaExceeded
	ErrRateLimited   = DenyCodeRateLimited
)
