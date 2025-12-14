package provider

// ChatRequest 统一聊天请求
type ChatRequest struct {
	Model       string
	Messages    []Message
	Temperature *float64
	TopP        *float64
	MaxTokens   *int
	Stream      bool
	Stop        []string
	User        string
}

// Message 消息
type Message struct {
	Role    string
	Content string
}

// CompletionRequest 统一补全请求
type CompletionRequest struct {
	Model       string
	Prompt      string
	MaxTokens   *int
	Temperature *float64
	TopP        *float64
	Stream      bool
	Stop        []string
	User        string
}
