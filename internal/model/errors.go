package model

import "fmt"

// APIError 上游/业务错误，归一化后由 handler 转换为 OpenAI 风格错误响应。
type APIError struct {
	StatusCode int
	Code       string
	Message    string
	Type       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("api error %d: %s", e.StatusCode, e.Message)
}

// NewAPIError 构造一个 APIError，Type 按 HTTP 状态码推断。
func NewAPIError(status int, code, message string) *APIError {
	return &APIError{
		StatusCode: status,
		Code:       code,
		Message:    message,
		Type:       errorType(status),
	}
}

func errorType(status int) string {
	switch {
	case status >= 500:
		return "upstream_error"
	case status == 401, status == 403:
		return "authentication_error"
	case status == 429:
		return "rate_limit_exceeded"
	case status == 404:
		return "not_found"
	default:
		return "invalid_request_error"
	}
}
