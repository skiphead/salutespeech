package sync

import (
	"github.com/skiphead/salutespeech/types"
)

// Model represents sync recognition model
type Model string

const (
	ModelCallcenter Model = "callcenter"
	ModelMedia      Model = "media"
	ModelIVR        Model = "ivr"
	ModelGeneral    Model = "general" // для совместимости с async
)

// Language represents recognition language
type Language string

const (
	LangRuRU Language = "ru-RU"
	LangEnUS Language = "en-US"
	LangKkKZ Language = "kk-KZ"
	LangKyKG Language = "ky-KG"
	LangUzUZ Language = "uz-UZ"
)

// EmotionScores represents emotion scores
type EmotionScores struct {
	NeutralA  float64 `json:"neutral_a,omitempty"`
	PositiveA float64 `json:"positive_a,omitempty"`
	NegativeA float64 `json:"negative_a,omitempty"`

	NeutralT  float64 `json:"neutral_t,omitempty"`
	PositiveT float64 `json:"positive_t,omitempty"`
	NegativeT float64 `json:"negative_t,omitempty"`

	Neutral  float64 `json:"neutral,omitempty"`
	Positive float64 `json:"positive,omitempty"`
	Negative float64 `json:"negative,omitempty"`
}

// SpeakerIdentity represents speaker identity
type SpeakerIdentity struct {
	Age      string  `json:"age,omitempty"`
	Gender   string  `json:"gender,omitempty"`
	AgeScore float64 `json:"age_score,omitempty"`
}

// Emotion represents emotion data in response
type Emotion struct {
	Negative float64 `json:"negative"`
	Neutral  float64 `json:"neutral"`
	Positive float64 `json:"positive"`
}

// PersonIdentity represents person identity in response
type PersonIdentity struct {
	Age         string `json:"age"`
	Gender      string `json:"gender"`
	AgeScore    int    `json:"age_score"`
	GenderScore int    `json:"gender_score"`
}

// Response represents sync recognition response
type Response struct {
	Result         []string       `json:"result"`
	Emotions       []Emotion      `json:"emotions,omitempty"`
	PersonIdentity PersonIdentity `json:"person_identity,omitempty"`
	Status         int            `json:"status"`
}

// Request represents sync recognition request
type Request struct {
	Data                  []byte
	ContentType           types.ContentType
	Language              Language
	EnableProfanityFilter bool
	Model                 Model
	SampleRate            int
	ChannelsCount         int
	RequestID             string
}

// Options represents sync recognition options for convenience
type Options struct {
	Language              Language
	EnableProfanityFilter bool
	Model                 Model
	SampleRate            int
	ChannelsCount         int
	RequestID             string
}

// DefaultOptions returns default sync recognition options
func DefaultOptions() Options {
	return Options{
		Language:              LangRuRU,
		EnableProfanityFilter: true,
		Model:                 ModelGeneral,
		SampleRate:            16000,
		ChannelsCount:         1,
	}
}
