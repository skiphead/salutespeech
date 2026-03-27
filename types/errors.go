package types

import "errors"

// Common errors
var (
	// Authentication errors
	ErrAuthKeyRequired      = errors.New("auth key is required")
	ErrScopeRequired        = errors.New("scope is required")
	ErrTokenManagerRequired = errors.New("token manager is required")
	ErrInvalidAuthHeader    = errors.New("invalid authorization header")

	// Request errors
	ErrEmptyFileData         = errors.New("empty file data")
	ErrInvalidContentType    = errors.New("invalid content type")
	ErrFileTooSmall          = errors.New("file too small")
	ErrEmptyTaskID           = errors.New("empty task id")
	ErrRequestNil            = errors.New("request is nil")
	ErrRequestFileIDRequired = errors.New("request_file_id is required")
	ErrOptionsRequired       = errors.New("options is required")
	ErrEmptyText             = errors.New("text is empty")
	ErrFileTooLarge          = errors.New("file too large")

	// API errors
	ErrUnauthorized    = errors.New("unauthorized")
	ErrTooManyRequests = errors.New("too many requests")
	ErrServerError     = errors.New("server error")
	ErrBadRequest      = errors.New("bad request")
)
