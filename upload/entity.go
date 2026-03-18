package upload

import (
	"time"

	"github.com/skiphead/salutespeech/types"
)

// Response UploadResponse represents upload API response
type Response struct {
	Status int `json:"status"`
	Result struct {
		RequestFileID string `json:"request_file_id"`
	} `json:"result"`
}

// Request UploadRequest represents upload request
type Request struct {
	Data        []byte
	ContentType types.ContentType
	RequestID   string
}

// Config represents upload client configuration
type Config struct {
	BaseURL       string
	AllowInsecure bool
	Timeout       time.Duration
	Logger        types.Logger
}
