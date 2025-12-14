package openai

// API 端点
const (
	EndpointChatCompletions = "/v1/chat/completions"
	EndpointCompletions     = "/v1/completions"
	EndpointModels          = "/v1/models"
)

// 对象类型
const (
	ObjectChatCompletion      = "chat.completion"
	ObjectChatCompletionChunk = "chat.completion.chunk"
	ObjectCompletion          = "text_completion"
	ObjectModel               = "model"
	ObjectList                = "list"
)

// 角色
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
	RoleFunction  = "function"
)

// 完成原因
const (
	FinishReasonStop          = "stop"
	FinishReasonLength        = "length"
	FinishReasonFunctionCall  = "function_call"
	FinishReasonToolCalls     = "tool_calls"
	FinishReasonContentFilter = "content_filter"
)
