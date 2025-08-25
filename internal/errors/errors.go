package errors

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type ErrorCode string

const (
	CodeInternal       ErrorCode = "INTERNAL_ERROR"
	CodeValidation     ErrorCode = "VALIDATION_ERROR"
	CodeNotFound       ErrorCode = "NOT_FOUND"
	CodeBadRequest     ErrorCode = "BAD_REQUEST"
	CodeUnauthorized   ErrorCode = "UNAUTHORIZED"
	CodeForbidden      ErrorCode = "FORBIDDEN"
	CodeRateLimit      ErrorCode = "RATE_LIMIT_EXCEEDED"
	CodeServiceUnavail ErrorCode = "SERVICE_UNAVAILABLE"
)

type AppError struct {
	Code       ErrorCode `json:"code"`
	Message    string    `json:"message"`
	Details    string    `json:"details,omitempty"`
	StatusCode int       `json:"-"`
	Cause      error     `json:"-"`
	Timestamp  time.Time `json:"timestamp"`
	RequestID  string    `json:"request_id,omitempty"`
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: getStatusCode(code),
		Timestamp:  time.Now().UTC(),
	}
}

func Wrap(err error, code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: getStatusCode(code),
		Cause:      err,
		Timestamp:  time.Now().UTC(),
	}
}

func Internal(message string) *AppError {
	return New(CodeInternal, message)
}

func InternalWrap(err error, message string) *AppError {
	return Wrap(err, CodeInternal, message)
}

func Validation(message string) *AppError {
	return New(CodeValidation, message)
}

func ValidationWrap(err error, message string) *AppError {
	return Wrap(err, CodeValidation, message)
}

func NotFound(message string) *AppError {
	return New(CodeNotFound, message)
}

func BadRequest(message string) *AppError {
	return New(CodeBadRequest, message)
}

func BadRequestWrap(err error, message string) *AppError {
	return Wrap(err, CodeBadRequest, message)
}

func Unauthorized(message string) *AppError {
	return New(CodeUnauthorized, message)
}

func Forbidden(message string) *AppError {
	return New(CodeForbidden, message)
}

func RateLimit(message string) *AppError {
	return New(CodeRateLimit, message)
}

func ServiceUnavailable(message string) *AppError {
	return New(CodeServiceUnavail, message)
}

func getStatusCode(code ErrorCode) int {
	switch code {
	case CodeValidation, CodeBadRequest:
		return http.StatusBadRequest
	case CodeNotFound:
		return http.StatusNotFound
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeRateLimit:
		return http.StatusTooManyRequests
	case CodeServiceUnavail:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

type ErrorResponse struct {
	Error   *AppError `json:"error"`
	Success bool      `json:"success"`
}

func WriteError(w http.ResponseWriter, logger *slog.Logger, err error, requestID string) {
	var appErr *AppError

	switch e := err.(type) {
	case *AppError:
		appErr = e
	default:
		appErr = Internal("An unexpected error occurred")
		appErr.Cause = err
	}

	appErr.RequestID = requestID

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.StatusCode)

	response := ErrorResponse{
		Error:   appErr,
		Success: false,
	}

	if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
		logger.Error("failed to encode error response",
			"encode_error", encodeErr,
			"original_error", err,
			"request_id", requestID,
		)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	logLevel := slog.LevelError
	if appErr.StatusCode < 500 {
		logLevel = slog.LevelWarn
	}

	logger.Log(context.TODO(), logLevel, "request failed",
		"error_code", appErr.Code,
		"error_message", appErr.Message,
		"status_code", appErr.StatusCode,
		"request_id", requestID,
		"cause", appErr.Cause,
	)
}

type SuccessResponse struct {
	Data    any  `json:"data"`
	Success bool `json:"success"`
}

func WriteSuccess(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := SuccessResponse{
		Data:    data,
		Success: true,
	}

	json.NewEncoder(w).Encode(response)
}

func WriteSuccessWithHeaders(w http.ResponseWriter, data any, headers map[string]string) {
	for key, value := range headers {
		w.Header().Set(key, value)
	}
	WriteSuccess(w, data)
}
