// Package async provides type definitions for asynchronous speech recognition operations.
// It defines models, audio encodings, request/response structures, and result types
// for long-running audio transcription tasks using the SaluteSpeech API.
package async

import (
	"time"

	"github.com/skiphead/salutespeech/types"
)

// Model represents the speech recognition model to use for transcription.
// Different models are specialized for specific domains and vocabulary.
type Model string

const (
	// ModelGeneral is a general-purpose model suitable for most conversational speech.
	// Works well with everyday conversations, meetings, and general content.
	ModelGeneral Model = "general"

	// ModelFinance is optimized for financial domain vocabulary.
	// Better recognition of financial terms, numbers, and industry-specific terminology.
	ModelFinance Model = "finance"

	// ModelMedicine is optimized for medical domain vocabulary.
	// Improved accuracy for medical terminology, drug names, and clinical terms.
	ModelMedicine Model = "medicine"
)

// AudioEncoding represents the encoding format of the input audio file.
// The encoding must match the actual format of the uploaded audio.
type AudioEncoding string

const (
	// EncodingPCM_S16LE represents raw PCM audio with signed 16-bit little-endian samples.
	// This is the recommended format for optimal recognition accuracy.
	EncodingPCM_S16LE AudioEncoding = "PCM_S16LE"

	// EncodingMP3 represents MP3 compressed audio format.
	// Widely supported but may have slightly lower accuracy than PCM.
	EncodingMP3 AudioEncoding = "MP3"

	// EncodingOGG_OPUS represents Ogg container with Opus codec.
	// Excellent compression with good quality, suitable for network transmission.
	EncodingOGG_OPUS AudioEncoding = "opus"

	// EncodingFLAC represents FLAC (Free Lossless Audio Codec) format.
	// Lossless compression that maintains original audio quality.
	EncodingFLAC AudioEncoding = "FLAC"

	// FormatWAV16 represents WAV format with 16-bit depth and 16kHz sampling rate.
	// Provides high quality with wide compatibility.
	FormatWAV16 AudioEncoding = "wav16"

	// FormatPCM16 represents raw PCM format with 16-bit depth and 16kHz sampling rate.
	// Minimal processing overhead, suitable for streaming.
	FormatPCM16 AudioEncoding = "pcm16"

	// FormatALaw represents A-law encoded audio.
	// Standard telephony format, 8-bit depth at 8kHz sampling rate.
	FormatALaw AudioEncoding = "alaw"

	// FormatG729 represents G.729 compressed audio format.
	// Very high compression ratio, commonly used in VoIP applications.
	FormatG729 AudioEncoding = "g729"
)

// Hints provides additional context to improve recognition accuracy.
// These hints help the recognition engine better understand domain-specific terms.
type Hints struct {
	// Words is a list of domain-specific words or phrases to boost recognition.
	Words []string `json:"words,omitempty"`

	// EnableLetters enables recognition of individual letters (for spelling).
	EnableLetters bool `json:"enable_letters,omitempty"`

	// EOUTimeout sets the End-Of-Utterance timeout in milliseconds.
	EOUTimeout int `json:"eou_timeout,omitempty"`
}

// SpeakerSeparationOptions configures diarization (speaker separation) features.
// Allows the recognition system to distinguish between different speakers.
type SpeakerSeparationOptions struct {
	// Enable enables speaker separation/diarization.
	Enable bool `json:"enable,omitempty"`

	// EnableOnlyMainSpeaker filters to only include the main speaker.
	EnableOnlyMainSpeaker bool `json:"enable_only_main_speaker,omitempty"`

	// Count specifies the expected number of speakers.
	Count int `json:"count,omitempty"`
}

// Options represents the complete set of recognition parameters.
// These options control how the audio is processed and what features are enabled.
type Options struct {
	Model                 Model                     `json:"model"`                                // Recognition model to use
	AudioEncoding         AudioEncoding             `json:"audio_encoding"`                       // Format of input audio
	SampleRate            int                       `json:"sample_rate"`                          // Sample rate in Hz
	Language              string                    `json:"language,omitempty"`                   // Language code (e.g., "ru-RU")
	EnableProfanityFilter bool                      `json:"enable_profanity_filter,omitempty"`    // Filter profanity from results
	HypothesesCount       int                       `json:"hypotheses_count,omitempty"`           // Number of alternative transcriptions
	NoSpeechTimeout       int                       `json:"no_speech_timeout,omitempty"`          // Timeout if no speech detected (ms)
	MaxSpeechTimeout      int                       `json:"max_speech_timeout,omitempty"`         // Maximum speech segment duration (ms)
	Hints                 *Hints                    `json:"hints,omitempty"`                      // Domain-specific hints
	ChannelsCount         int                       `json:"channels_count,omitempty"`             // Number of audio channels
	SpeakerSeparation     *SpeakerSeparationOptions `json:"speaker_separation_options,omitempty"` // Diarization settings
	InsightModels         []string                  `json:"insight_models,omitempty"`             // Additional analysis models
}

// Request represents an asynchronous recognition task creation request.
// It combines recognition options with the uploaded audio file identifier.
type Request struct {
	Options       *Options `json:"options"`         // Recognition configuration
	RequestFileID string   `json:"request_file_id"` // ID of uploaded audio file
}

// Result represents the metadata of a recognition task returned by the API.
// Contains task identification and current status information.
type Result struct {
	ID        string           `json:"id"`         // Unique task identifier
	CreatedAt string           `json:"created_at"` // Task creation timestamp (RFC3339)
	UpdatedAt string           `json:"updated_at"` // Last update timestamp (RFC3339)
	Status    types.TaskStatus `json:"status"`     // Current task status
}

// Response represents the API response for a recognition task creation request.
// Wraps the task result with an HTTP status code.
type Response struct {
	Status int    `json:"status"` // HTTP status code
	Result Result `json:"result"` // Task metadata
}

// Alternative represents a single transcription alternative.
// Multiple alternatives may be returned when HypothesesCount > 1.
type Alternative struct {
	Text       string  `json:"text"`            // Transcribed text
	Confidence float64 `json:"confidence"`      // Confidence score (0-1)
	Words      []Word  `json:"words,omitempty"` // Word-level details
}

// Word represents detailed information about a single recognized word.
// Includes timing information for alignment with audio.
type Word struct {
	Text       string  `json:"text"`       // Word text
	StartTime  float64 `json:"start_time"` // Start time in seconds
	EndTime    float64 `json:"end_time"`   // End time in seconds
	Confidence float64 `json:"confidence"` // Word-level confidence score
}

// TaskResult represents the complete result of a recognition task.
// Contains transcribed text, metadata, and any error information.
type TaskResult struct {
	Status int `json:"status"`
	Result struct {
		Id             string    `json:"id"`
		CreatedAt      time.Time `json:"created_at"`
		UpdatedAt      time.Time `json:"updated_at"`
		Status         string    `json:"status"`
		ResponseFileId string    `json:"response_file_id"`
	} `json:"result"`
}

// DownloadFileResponse represents the detailed recognition results from a completed task.
// This structure is used when parsing the downloaded result file.
type DownloadFileResponse struct {
	Results             []DownloadFileResult `json:"results"`
	Eou                 bool                 `json:"eou"`
	EmotionsResult      EmotionsResult       `json:"emotions_result"`
	ProcessedAudioStart string               `json:"processed_audio_start"`
	ProcessedAudioEnd   string               `json:"processed_audio_end"`
	BackendInfo         BackendInfo          `json:"backend_info"`
	Channel             int                  `json:"channel"`
	SpeakerInfo         SpeakerInfo          `json:"speaker_info"`
	EouReason           string               `json:"eou_reason"`
	Insight             string               `json:"insight"`
	PersonIdentity      PersonIdentity       `json:"person_identity"`
}

// DownloadFileResult represents a single recognition segment in the result file.
type DownloadFileResult struct {
	Text           string          `json:"text"`
	NormalizedText string          `json:"normalized_text"`
	Start          string          `json:"start"`
	End            string          `json:"end"`
	WordAlignments []WordAlignment `json:"word_alignments"`
}

// WordAlignment represents timing information for individual words.
type WordAlignment struct {
	Word  string `json:"word"`
	Start string `json:"start"`
	End   string `json:"end"`
}

// EmotionsResult contains emotion analysis results.
type EmotionsResult struct {
	Positive float64 `json:"positive"`
	Neutral  float64 `json:"neutral"`
	Negative float64 `json:"negative"`
}

// BackendInfo contains metadata about the recognition backend.
type BackendInfo struct {
	ModelName     string `json:"model_name"`
	ModelVersion  string `json:"model_version"`
	ServerVersion string `json:"server_version"`
}

// SpeakerInfo contains diarization results.
type SpeakerInfo struct {
	SpeakerID             int `json:"speaker_id"`
	MainSpeakerConfidence int `json:"main_speaker_confidence"`
}

// PersonIdentity contains demographic analysis results.
type PersonIdentity struct {
	Age         string `json:"age"`
	Gender      string `json:"gender"`
	AgeScore    int    `json:"age_score"`
	GenderScore int    `json:"gender_score"`
}
