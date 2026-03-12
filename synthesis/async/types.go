package async

import (
	"github.com/skiphead/salutespeech/types"
)

// Encoding represents audio encoding for synthesis
type Encoding string

const (
	EncodingOpus Encoding = "Opus"
)

// Request represents async synthesis request
type Request struct {
	RequestFileID string      `json:"request_file_id"`
	AudioEncoding Encoding    `json:"audio_encoding"`
	Voice         types.Voice `json:"voice"`
}

// Result represents synthesis task result
type Result struct {
	ID        string           `json:"id"`
	CreatedAt string           `json:"created_at"`
	UpdatedAt string           `json:"updated_at"`
	Status    types.TaskStatus `json:"status"`
}

// Response represents synthesis API response
type Response struct {
	Status int    `json:"status"`
	Result Result `json:"result"`
}

// TaskResult represents complete synthesis task result
type TaskResult struct {
	ID             string           `json:"id"`
	Status         types.TaskStatus `json:"status"`
	ResponseFileID string           `json:"response_file_id,omitempty"`
	AudioData      []byte           `json:"-"`
	ErrorMessage   *string          `json:"error_message,omitempty"`
}
