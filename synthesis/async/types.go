// Package async provides type definitions for asynchronous speech synthesis operations.
// It defines request/response structures, audio encodings, and task result types
// for long-running text-to-speech conversion using the SaluteSpeech API.
package async

import (
	"github.com/skiphead/salutespeech/types"
)

// Encoding represents the audio encoding format for synthesized speech output.
// Different encodings offer various trade-offs between audio quality and file size.
type Encoding string

const (
	// EncodingWAV16 represents WAV format with 16-bit depth and 16kHz sampling rate.
	// Provides high quality with wide compatibility.
	EncodingWAV16 Encoding = "wav16"

	// EncodingPCM16 represents raw PCM format with 16-bit depth and 16kHz sampling rate.
	// Minimal processing overhead, suitable for streaming.
	EncodingPCM16 Encoding = "pcm16"

	// EncodingOpus represents Opus audio encoding in Ogg container.
	// Opus provides excellent compression while maintaining high audio quality,
	// making it ideal for network transmission and storage.
	EncodingOpus Encoding = "opus"

	// EncodingALaw represents A-law encoded audio.
	// Standard telephony format, 8-bit depth at 8kHz sampling rate.
	EncodingALaw Encoding = "alaw"

	// EncodingG729 represents G.729 compressed audio format.
	// Very high compression ratio, commonly used in VoIP applications.
	EncodingG729 Encoding = "g729"
)

// Request represents an asynchronous speech synthesis task creation request.
// It contains all parameters needed to initiate a text-to-speech conversion job.
type Request struct {
	// RequestFileID is the ID of the uploaded text file containing the text to synthesize.
	// The file must be uploaded to SaluteSpeech storage before creating the task.
	RequestFileID string `json:"request_file_id"`

	// AudioEncoding specifies the desired output audio format (e.g., "Opus").
	AudioEncoding Encoding `json:"audio_encoding"`

	// Voice determines which voice model to use for synthesis.
	// Different voices have different characteristics (gender, age, style).
	Voice types.Voice `json:"voice"`
}

// Result represents the metadata of a synthesis task returned by the API.
// It contains task identification and status information.
type Result struct {
	// ID is the unique identifier of the synthesis task.
	ID string `json:"id"`

	// CreatedAt is the timestamp when the task was created (RFC3339 format).
	CreatedAt string `json:"created_at"`

	// UpdatedAt is the timestamp when the task was last updated (RFC3339 format).
	UpdatedAt string `json:"updated_at"`

	// Status indicates the current state of the task (NEW, PROCESSING, DONE, ERROR, CANCELED).
	Status types.TaskStatus `json:"status"`
	// ResponseFileId object id
	ResponseFileId string `json:"response_file_id"`
}

// Response represents the API response for a synthesis task creation request.
// It wraps the task result with an HTTP status code.
type Response struct {
	// Status is the HTTP status code of the API response.
	Status int `json:"status"`

	// Result contains the task metadata for the created synthesis job.
	Result Result `json:"result"`
}

// TaskResult represents the complete result of a synthesis task after processing.
// It includes task metadata and optionally the synthesized audio data.
type TaskResult struct {
	// ID is the unique identifier of the synthesis task.
	ID string `json:"id"`

	// Status indicates the final state of the task (DONE, ERROR, etc.).
	Status types.TaskStatus `json:"status"`

	// ResponseFileID is the ID of the generated audio file (available when status is DONE).
	ResponseFileID string `json:"response_file_id,omitempty"`

	// AudioData contains the synthesized audio bytes (populated only when downloaded).
	// This field is omitted from JSON serialization.
	AudioData []byte `json:"-"`

	// ErrorMessage provides details about any error that occurred during processing.
	// Present only when Status is ERROR.
	ErrorMessage *string `json:"error_message,omitempty"`
}
