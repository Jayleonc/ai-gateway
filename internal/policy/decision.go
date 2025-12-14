package policy

// Decision 策略决策结果
type Decision struct {
	Allowed        bool
	DenyReason     string
	DenyCode       string
	TargetProvider string
	TargetModel    string
	Priority       int
	Metadata       map[string]string
}

// NewAllowDecision 创建允许决策
func NewAllowDecision(provider, model string) *Decision {
	return &Decision{
		Allowed:        true,
		TargetProvider: provider,
		TargetModel:    model,
		Metadata:       make(map[string]string),
	}
}

// NewDenyDecision 创建拒绝决策
func NewDenyDecision(code, reason string) *Decision {
	return &Decision{
		Allowed:    false,
		DenyCode:   code,
		DenyReason: reason,
		Metadata:   make(map[string]string),
	}
}

// EvaluateRequest 策略评估请求
type EvaluateRequest struct {
	Model           string
	EstimatedTokens int
	Metadata        map[string]string
}
