package metering

// Calculator 成本计算器接口
type Calculator interface {
	Calculate(provider, model string, promptTokens, completionTokens int) (*Cost, error)
}

// calculator 计算器实现
type calculator struct {
	pricing map[string]*ModelPricing
}

// ModelPricing 模型定价
type ModelPricing struct {
	Provider        string
	Model           string
	PromptPrice     float64 // 每 1K tokens
	CompletionPrice float64 // 每 1K tokens
	Currency        string
}

// NewCalculator 创建计算器
func NewCalculator() Calculator {
	return &calculator{
		pricing: defaultPricing(),
	}
}

// Calculate 计算成本
func (c *calculator) Calculate(provider, model string, promptTokens, completionTokens int) (*Cost, error) {
	key := provider + ":" + model
	pricing, ok := c.pricing[key]
	if !ok {
		// 使用默认定价
		pricing = &ModelPricing{
			PromptPrice:     0.001,
			CompletionPrice: 0.002,
			Currency:        "USD",
		}
	}

	promptCost := float64(promptTokens) / 1000 * pricing.PromptPrice
	completionCost := float64(completionTokens) / 1000 * pricing.CompletionPrice

	return &Cost{
		PromptCost:     promptCost,
		CompletionCost: completionCost,
		TotalCost:      promptCost + completionCost,
		Currency:       pricing.Currency,
	}, nil
}

// defaultPricing 默认定价表
func defaultPricing() map[string]*ModelPricing {
	return map[string]*ModelPricing{
		"openai:gpt-4": {
			Provider:        "openai",
			Model:           "gpt-4",
			PromptPrice:     0.03,
			CompletionPrice: 0.06,
			Currency:        "USD",
		},
		"openai:gpt-4-turbo": {
			Provider:        "openai",
			Model:           "gpt-4-turbo",
			PromptPrice:     0.01,
			CompletionPrice: 0.03,
			Currency:        "USD",
		},
		"openai:gpt-3.5-turbo": {
			Provider:        "openai",
			Model:           "gpt-3.5-turbo",
			PromptPrice:     0.0005,
			CompletionPrice: 0.0015,
			Currency:        "USD",
		},
	}
}
