package errors

import (
	"errors"
	"net/http"
)

const (
	OpenAIErrorTypeRateLimit = "rate_limit_error"
	OpenAIErrorTypeServer    = "server_error"
	OpenAIErrorTypeInvalid   = "invalid_request_error"
	OpenAIErrorTypeAuth      = "authentication_error"
)

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
	ErrClientDisconnect = errors.New("client disconnected")
	ErrInvalidAPIKey    = errors.New("invalid api key")
	ErrAPIKeyExpired    = errors.New("api key expired")
	ErrAPIKeyDisabled   = errors.New("api key disabled")
	ErrModelNotAllowed  = errors.New("model not allowed")
	ErrProviderNotFound = errors.New("provider not found")
)

type APIError interface {
	error
	Code() string    // 内部错误码
	Message() string // 人类可读信息
	Type() string    // openai error type
	HTTPStatus() int // 推荐 HTTP 状态码
}

// apiError 统一 API 错误结构
type apiError struct {
	code    string
	message string
	errType string
	status  int
}

func (e *apiError) Error() string {
	return e.message
}

func (e *apiError) Code() string {
	return e.code
}

func (e *apiError) Message() string {
	return e.message
}

func (e *apiError) Type() string {
	return e.errType
}

func (e *apiError) HTTPStatus() int {
	if e.status != 0 {
		return e.status
	}
	return http.StatusInternalServerError
}

// NewAPIError 创建 API 错误
func NewAPIError(code, message, errType string) APIError {
	return &apiError{
		code:    code,
		message: message,
		errType: errType,
		status:  http.StatusInternalServerError,
	}
}

func NewAPIErrorWithStatus(code, message, errType string, status int) APIError {
	return &apiError{
		code:    code,
		message: message,
		errType: errType,
		status:  status,
	}
}

func AsAPIError(err error) APIError {
	if err == nil {
		return nil
	}
	var apiErr APIError
	if errors.As(err, &apiErr) {
		return apiErr
	}

	switch {
	case errors.Is(err, ErrInvalidAPIKey):
		return NewAPIErrorWithStatus("invalid_api_key", "invalid api key", OpenAIErrorTypeAuth, http.StatusUnauthorized)
	case errors.Is(err, ErrAPIKeyExpired):
		return NewAPIErrorWithStatus("api_key_expired", "api key expired", OpenAIErrorTypeAuth, http.StatusUnauthorized)
	case errors.Is(err, ErrAPIKeyDisabled):
		return NewAPIErrorWithStatus("api_key_disabled", "api key disabled", OpenAIErrorTypeAuth, http.StatusUnauthorized)
	case errors.Is(err, ErrQuotaExceeded):
		return NewAPIErrorWithStatus("quota_exceeded", "quota exceeded", OpenAIErrorTypeRateLimit, http.StatusTooManyRequests)
	case errors.Is(err, ErrRateLimited):
		return NewAPIErrorWithStatus("rate_limited", "rate limited", OpenAIErrorTypeRateLimit, http.StatusTooManyRequests)
	case errors.Is(err, ErrProviderError):
		return NewAPIErrorWithStatus("provider_error", "provider error", OpenAIErrorTypeServer, http.StatusBadGateway)
	case errors.Is(err, ErrProviderNotFound):
		return NewAPIErrorWithStatus("provider_error", "provider error", OpenAIErrorTypeServer, http.StatusBadGateway)
	case errors.Is(err, ErrClientDisconnect):
		return NewAPIErrorWithStatus("client_disconnect", "client disconnected", OpenAIErrorTypeServer, http.StatusInternalServerError)
	case errors.Is(err, ErrBadRequest):
		return NewAPIErrorWithStatus("invalid_request", "bad request", OpenAIErrorTypeInvalid, http.StatusBadRequest)
	default:
		return NewAPIErrorWithStatus("internal_error", "internal server error", OpenAIErrorTypeServer, http.StatusInternalServerError)
	}
}
