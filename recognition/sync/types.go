// Package sync provides type definitions for synchronous speech recognition operations.
// It defines models, languages, request/response structures, and related types
// for real-time audio transcription using the SaluteSpeech API.
package sync

import (
	"github.com/skiphead/salutespeech/types"
)

// Model represents the speech recognition model to use for transcription.
// Different models are optimized for various acoustic environments and use cases.
type Model string

const (
	// ModelCallcenter is optimized for telephone conversations and call center audio.
	// Handles typical telephony characteristics (limited bandwidth, background noise).
	ModelCallcenter Model = "callcenter"

	// ModelMedia is optimized for broadcast media, podcasts, and videos.
	// Works well with professional audio and varied speaking styles.
	ModelMedia Model = "media"

	// ModelIVR is optimized for Interactive Voice Response systems.
	// Designed for short utterances and menu navigation scenarios.
	ModelIVR Model = "ivr"

	// ModelGeneral is a general-purpose model suitable for most use cases.
	// Maintained for compatibility with async recognition models.
	ModelGeneral Model = "general"
)

// Language represents the language of the audio being recognized.
// Each language code follows the ISO 639-1 language code with ISO 3166-1 alpha-2 country code.
type Language string

const (
	// LangRuRU represents Russian language (Russia)
	LangRuRU Language = "ru-RU"

	// LangEnUS represents English language (United States)
	LangEnUS Language = "en-US"

	// LangKkKZ represents Kazakh language (Kazakhstan)
	LangKkKZ Language = "kk-KZ"

	// LangKyKG represents Kyrgyz language (Kyrgyzstan)
	LangKyKG Language = "ky-KG"

	// LangUzUZ represents Uzbek language (Uzbekistan)
	LangUzUZ Language = "uz-UZ"
)

// EmotionScores represents detailed emotion analysis scores.
// This structure provides separate scores for different emotion classification
// approaches (channel A, channel T, and combined).
type EmotionScores struct {
	// Channel A emotion scores
	NeutralA  float64 `json:"neutral_a,omitempty"`
	PositiveA float64 `json:"positive_a,omitempty"`
	NegativeA float64 `json:"negative_a,omitempty"`

	// Channel T emotion scores
	NeutralT  float64 `json:"neutral_t,omitempty"`
	PositiveT float64 `json:"positive_t,omitempty"`
	NegativeT float64 `json:"negative_t,omitempty"`

	// Combined emotion scores
	Neutral  float64 `json:"neutral,omitempty"`
	Positive float64 `json:"positive,omitempty"`
	Negative float64 `json:"negative,omitempty"`
}

// SpeakerIdentity represents demographic information about the speaker.
// Includes age group and gender classification with confidence scores.
type SpeakerIdentity struct {
	Age      string  `json:"age,omitempty"`       // Age group classification (e.g., "adult", "child")
	Gender   string  `json:"gender,omitempty"`    // Gender classification ("male", "female")
	AgeScore float64 `json:"age_score,omitempty"` // Confidence score for age classification
}

// Emotion represents emotion probabilities from speech analysis.
// Values are typically between 0 and 1, representing confidence levels.
type Emotion struct {
	Negative float64 `json:"negative"` // Probability of negative emotion
	Neutral  float64 `json:"neutral"`  // Probability of neutral emotion
	Positive float64 `json:"positive"` // Probability of positive emotion
}

// PersonIdentity represents comprehensive person identity information.
// Includes age and gender classification with integer confidence scores.
type PersonIdentity struct {
	Age         string `json:"age"`          // Age group classification
	Gender      string `json:"gender"`       // Gender classification
	AgeScore    int    `json:"age_score"`    // Confidence score for age (0-100)
	GenderScore int    `json:"gender_score"` // Confidence score for gender (0-100)
}

// Response represents a synchronous speech recognition response.
// Contains transcribed text and optional analysis results.
type Response struct {
	Result         []string       `json:"result"`                    // Transcribed text alternatives
	Emotions       []Emotion      `json:"emotions,omitempty"`        // Emotion analysis per utterance
	PersonIdentity PersonIdentity `json:"person_identity,omitempty"` // Speaker demographic info
	Status         int            `json:"status"`                    // HTTP status code
}

// Request represents a synchronous speech recognition request.
// Contains audio data and recognition parameters.
type Request struct {
	Data                  []byte            // Raw audio data
	ContentType           types.ContentType // MIME type of the audio
	Language              Language          // Language of the audio
	EnableProfanityFilter bool              // If true, filters profanity from results
	Model                 Model             // Recognition model to use
	SampleRate            int               // Sample rate of the audio in Hz
	ChannelsCount         int               // Number of audio channels
	RequestID             string            // Optional request ID for tracing
}

// Options provides a convenient way to specify recognition options.
// Used by convenience methods like RecognizeFromFile.
type Options struct {
	Language              Language // Language of the audio
	EnableProfanityFilter bool     // Enable profanity filtering
	Model                 Model    // Recognition model
	SampleRate            int      // Audio sample rate (Hz)
	ChannelsCount         int      // Number of audio channels
	RequestID             string   // Optional request ID
}

// DefaultOptions returns default recognition options suitable for most use cases.
// Defaults:
//   - Language: Russian (ru-RU)
//   - Profanity filtering: Enabled
//   - Model: General purpose
//   - Sample rate: 16000 Hz (optimal for speech recognition)
//   - Channels: 1 (mono)
//
// These defaults provide good performance for typical speech recognition scenarios.
func DefaultOptions() Options {
	return Options{
		Language:              LangRuRU,
		EnableProfanityFilter: true,
		Model:                 ModelGeneral,
		SampleRate:            16000,
		ChannelsCount:         1,
	}
}
