package types

import "time"

// Default URLs
const (
	DefaultBaseURL            = "https://smartspeech.sber.ru/rest/v1/"
	DefaultOAuthURL           = "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"
	DefaultUploadURL          = "https://smartspeech.sber.ru/rest/v1/data:upload"
	DefaultSynthesizeURL      = "https://smartspeech.sber.ru/rest/v1/text:async_synthesize"
	DefaultTaskURL            = "https://smartspeech.sber.ru/rest/v1/task:get"
	DefaultDownloadURL        = "https://smartspeech.sber.ru/rest/v1/data:download"
	DefaultResultURL          = "https://smartspeech.sber.ru/rest/v1/speech:recognition_result"
	DefaultSyncRecognitionURL = "https://smartspeech.sber.ru/rest/v1/speech:recognize"
	DefaultSyncSynthesisURL   = "https://smartspeech.sber.ru/rest/v1/text:synthesize"
)

// Limits
const (
	MinFileSize        = 400
	MaxSyncFileSize    = 2 * 1024 * 1024 // 2 MB
	MaxTextLength      = 4000
	MaxSampleRate      = 96000
	MinSampleRate      = 8000
	MaxChannelsCount   = 2
	MinChannelsCount   = 1
	MaxHypothesesCount = 5
	MaxUploadFileSize  = 20 * 1024 * 1024 // 20 MB
)

// Timeouts
const (
	DefaultTimeout       = 30 * time.Second
	DefaultUploadTimeout = 5 * time.Minute
	DefaultAPITimeout    = 60 * time.Second
	DefaultPollInterval  = 2 * time.Second
	DefaultWaitTimeout   = 10 * time.Minute
	DefaultRefreshMargin = 1 * time.Minute
	DefaultMinRefreshInt = 30 * time.Second
)
