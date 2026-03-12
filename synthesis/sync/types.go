package sync

import (
	"github.com/skiphead/salutespeech/types"
)

// Format represents synthesis output format
type Format string

const (
	FormatWAV16 Format = "wav16"
	FormatPCM16 Format = "pcm16"
	FormatOpus  Format = "opus"
	FormatALaw  Format = "alaw"
	FormatG729  Format = "g729"
)

// ContentType represents synthesis content type
type ContentType string

const (
	ContentTypeText ContentType = "application/text"
	ContentTypeSSML ContentType = "application/ssml"
)

// Request represents sync synthesis request
type Request struct {
	Text         string
	ContentType  ContentType
	Format       Format
	Voice        types.Voice
	RebuildCache bool
	BypassCache  bool
	RequestID    string
}

// Response represents sync synthesis response
type Response struct {
	AudioData     []byte
	ContentType   string
	ContentLength int
}

// Options represents sync synthesis options for convenience
type Options struct {
	Format       Format
	Voice        types.Voice
	RebuildCache bool
	BypassCache  bool
	RequestID    string
}

// DefaultOptions returns default sync synthesis options
func DefaultOptions() Options {
	return Options{
		Format:       FormatWAV16,
		Voice:        types.VoiceMay24000,
		RebuildCache: false,
		BypassCache:  false,
	}
}

// GetContentType returns MIME type for format
func (f Format) GetContentType() string {
	switch f {
	case FormatWAV16:
		return "audio/x-wav"
	case FormatPCM16:
		return "audio/x-pcm;bit=16;rate=24000"
	case FormatOpus:
		return "audio/ogg; codecs=opus"
	case FormatALaw:
		return "audio/pcma;rate=8000"
	case FormatG729:
		return "audio/g729"
	default:
		return "audio/x-wav"
	}
}
