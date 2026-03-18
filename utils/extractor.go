package utils

import (
	"encoding/json"
	"fmt"
	"strings"
)

// WordAlignment Структуры для десериализации (как в предыдущем примере)
type WordAlignment struct {
	Word  string `json:"word"`
	Start string `json:"start"`
	End   string `json:"end"`
}

type ExtractResult struct {
	Text           string          `json:"text"`
	NormalizedText string          `json:"normalized_text"`
	Start          string          `json:"start"`
	End            string          `json:"end"`
	WordAlignments []WordAlignment `json:"word_alignments"`
}

type EmotionsResult struct {
	Positive float64 `json:"positive"`
	Neutral  float64 `json:"neutral"`
	Negative float64 `json:"negative"`
}

type BackendInfo struct {
	ModelName     string `json:"model_name"`
	ModelVersion  string `json:"model_version"`
	ServerVersion string `json:"server_version"`
}

type SpeakerInfo struct {
	SpeakerID             int     `json:"speaker_id"`
	MainSpeakerConfidence float64 `json:"main_speaker_confidence"`
}

type PersonIdentity struct {
	Age         string  `json:"age"`
	Gender      string  `json:"gender"`
	AgeScore    float64 `json:"age_score"`
	GenderScore float64 `json:"gender_score"`
}

type TranscriptionData struct {
	Results             []ExtractResult `json:"results"`
	Eou                 bool            `json:"eou"`
	EmotionsResult      EmotionsResult  `json:"emotions_result"`
	ProcessedAudioStart string          `json:"processed_audio_start"`
	ProcessedAudioEnd   string          `json:"processed_audio_end"`
	BackendInfo         BackendInfo     `json:"backend_info"`
	Channel             int             `json:"channel"`
	SpeakerInfo         SpeakerInfo     `json:"speaker_info"`
	EouReason           string          `json:"eou_reason"`
	Insight             string          `json:"insight"`
	PersonIdentity      PersonIdentity  `json:"person_identity"`
}

// ExtractTextFromResults Функция для десериализации и извлечения всех полей text из results
func ExtractTextFromResults(data []byte) (string, error) {
	var transcriptions []TranscriptionData

	// Десериализуем JSON
	err := json.Unmarshal(data, &transcriptions)
	if err != nil {
		return "", fmt.Errorf("ошибка десериализации JSON: %v", err)
	}

	// Собираем все тексты
	var texts []string
	for _, item := range transcriptions {
		for _, result := range item.Results {
			if result.Text != "" {
				texts = append(texts, result.Text)
			}
		}
	}

	// Объединяем все тексты в одну строку через пробел
	return strings.Join(texts, " "), nil
}

// ExtractTextFromTranscriptions Функция для извлечения текста из уже десериализованных данных
func ExtractTextFromTranscriptions(transcriptions []TranscriptionData) string {
	var texts []string

	for _, item := range transcriptions {
		for _, result := range item.Results {
			if result.Text != "" {
				texts = append(texts, result.Text)
			}
		}
	}

	return strings.Join(texts, " ")
}
