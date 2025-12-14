package errors

import "errors"

// 标准错误定义
var (
	ErrNotImplemented   = errors.New("not implemented")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrForbidden        = errors.New("forbidden")
	ErrNotFound         = errors.New("not found")
	ErrBadRequest       = errors.New("bad request")
	ErrInternalServer   = errors.New("internal server error")
	ErrQuotaExceeded    = errors.New("quota exceeded")
	ErrRateLimited      = errors.New("rate limited")
	ErrProviderError    = errors.New("provider error")
	ErrInvalidAPIKey    = errors.New("invalid api key")
	ErrAPIKeyExpired    = errors.New("api key expired")
	ErrAPIKeyDisabled   = errors.New("api key disabled")
	ErrModelNotAllowed  = errors.New("model not allowed")
	ErrProviderNotFound = errors.New("provider not found")
)

// APIError 统一 API 错误结构
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

func (e *APIError) Error() string {
	return e.Message
}

// NewAPIError 创建 API 错误
func NewAPIError(code, message, errType string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
		Type:    errType,
	}
}
