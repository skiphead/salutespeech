package async

import (
	"github.com/skiphead/salutespeech/types"
)

// RecognitionModel represents recognition model
type Model string

const (
	ModelGeneral  Model = "general"
	ModelFinance  Model = "finance"
	ModelMedicine Model = "medicine"
)

// AudioEncoding represents audio encoding format
type AudioEncoding string

const (
	EncodingPCM_S16LE AudioEncoding = "PCM_S16LE"
	EncodingMP3       AudioEncoding = "MP3"
	EncodingOGG_OPUS  AudioEncoding = "OGG_OPUS"
	EncodingFLAC      AudioEncoding = "FLAC"
)

// Hints represents recognition hints
type Hints struct {
	Words         []string `json:"words,omitempty"`
	EnableLetters bool     `json:"enable_letters,omitempty"`
	EOUTimeout    int      `json:"eou_timeout,omitempty"`
}

// SpeakerSeparationOptions represents speaker separation options
type SpeakerSeparationOptions struct {
	Enable                bool `json:"enable,omitempty"`
	EnableOnlyMainSpeaker bool `json:"enable_only_main_speaker,omitempty"`
	Count                 int  `json:"count,omitempty"`
}

// Options represents recognition options
type Options struct {
	Model                 Model                     `json:"model"`
	AudioEncoding         AudioEncoding             `json:"audio_encoding"`
	SampleRate            int                       `json:"sample_rate"`
	Language              string                    `json:"language,omitempty"`
	EnableProfanityFilter bool                      `json:"enable_profanity_filter,omitempty"`
	HypothesesCount       int                       `json:"hypotheses_count,omitempty"`
	NoSpeechTimeout       int                       `json:"no_speech_timeout,omitempty"`
	MaxSpeechTimeout      int                       `json:"max_speech_timeout,omitempty"`
	Hints                 *Hints                    `json:"hints,omitempty"`
	ChannelsCount         int                       `json:"channels_count,omitempty"`
	SpeakerSeparation     *SpeakerSeparationOptions `json:"speaker_separation_options,omitempty"`
	InsightModels         []string                  `json:"insight_models,omitempty"`
}

// Request represents recognition request
type Request struct {
	Options       *Options `json:"options"`
	RequestFileID string   `json:"request_file_id"`
}

// Result represents recognition task result
type Result struct {
	ID        string           `json:"id"`
	CreatedAt string           `json:"created_at"`
	UpdatedAt string           `json:"updated_at"`
	Status    types.TaskStatus `json:"status"`
}

// Response represents recognition API response
type Response struct {
	Status int    `json:"status"`
	Result Result `json:"result"`
}

// Alternative represents recognition alternative
type Alternative struct {
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
	Words      []Word  `json:"words,omitempty"`
}

// Word represents recognized word
type Word struct {
	Text       string  `json:"text"`
	StartTime  float64 `json:"start_time"`
	EndTime    float64 `json:"end_time"`
	Confidence float64 `json:"confidence"`
}

// TaskResult represents complete recognition task result
type TaskResult struct {
	ID             string           `json:"id"`
	Status         string           `json:"status"`
	UnifiedStatus  types.TaskStatus `json:"-"`
	Alternatives   []Alternative    `json:"alternatives,omitempty"`
	ErrorCode      *string          `json:"error_code,omitempty"`
	ErrorMessage   *string          `json:"error_message,omitempty"`
	ResponseFileID string           `json:"response_file_id,omitempty"`
}
