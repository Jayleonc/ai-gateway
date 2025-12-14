package provider

// ChatResponse 统一聊天响应
type ChatResponse struct {
	ID           string
	Model        string
	Choices      []ChatChoice
	Usage        *Usage
	FinishReason string
}

// ChatChoice 聊天选择
type ChatChoice struct {
	Index        int
	Message      *Message
	FinishReason string
}

// CompletionResponse 统一补全响应
type CompletionResponse struct {
	ID           string
	Model        string
	Choices      []CompletionChoice
	Usage        *Usage
	FinishReason string
}

// CompletionChoice 补全选择
type CompletionChoice struct {
	Index        int
	Text         string
	FinishReason string
}

// Usage Token 使用量
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// StreamEvent 流式事件
type StreamEvent struct {
	ID           string
	Model        string
	Delta        *Message
	FinishReason string
	Usage        *Usage
}
