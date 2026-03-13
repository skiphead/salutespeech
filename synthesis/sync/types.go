// Package sync provides type definitions for synchronous speech synthesis operations.
// It defines formats, content types, request/response structures, and utility functions
// for text-to-speech conversion using the SaluteSpeech API.
package sync

import (
	"github.com/skiphead/salutespeech/types"
)

// Format represents the output audio format for speech synthesis.
// Different formats offer trade-offs between audio quality, file size,
// and compatibility with different playback systems.
type Format string

const (
	// FormatWAV16 represents WAV format with 16-bit depth and 16kHz sampling rate.
	// Provides high quality with wide compatibility.
	FormatWAV16 Format = "wav16"

	// FormatPCM16 represents raw PCM format with 16-bit depth and 16kHz sampling rate.
	// Minimal processing overhead, suitable for streaming.
	FormatPCM16 Format = "pcm16"

	// FormatOpus represents Opus audio format in Ogg container.
	// Excellent compression while maintaining good quality, ideal for web applications.
	FormatOpus Format = "opus"

	// FormatALaw represents A-law encoded audio.
	// Standard telephony format, 8-bit depth at 8kHz sampling rate.
	FormatALaw Format = "alaw"

	// FormatG729 represents G.729 compressed audio format.
	// Very high compression ratio, commonly used in VoIP applications.
	FormatG729 Format = "g729"
)

// ContentType represents the type of input text content for synthesis.
// Determines how the input text should be interpreted by the synthesis engine.
type ContentType string

const (
	// ContentTypeText represents plain text input.
	// Simple text without any markup or formatting instructions.
	ContentTypeText ContentType = "application/text"

	// ContentTypeSSML represents SSML (Speech Synthesis Markup Language) input.
	// Allows fine-grained control over pronunciation, pitch, rate, and other speech attributes.
	ContentTypeSSML ContentType = "application/ssml"
)

// Request represents a synchronous speech synthesis request.
// It contains all parameters needed to convert text to speech.
type Request struct {
	Text         string      // Input text to synthesize (plain text or SSML)
	ContentType  ContentType // Type of input content (text or SSML)
	Format       Format      // Desired output audio format
	Voice        types.Voice // Voice to use for synthesis
	RebuildCache bool        // If true, forces cache rebuild for this request
	BypassCache  bool        // If true, bypasses cache entirely
	RequestID    string      // Optional request ID for tracing (auto-generated if empty)
}

// Response represents a synchronous speech synthesis response.
// Contains the generated audio data and metadata.
type Response struct {
	AudioData     []byte // Synthesized audio data in the requested format
	ContentType   string // MIME type of the audio data
	ContentLength int    // Length of audio data in bytes
}

// Options provides a convenient way to specify synthesis options.
// Used by convenience methods like SynthesizeText and SynthesizeSSML.
type Options struct {
	Format       Format      // Output audio format
	Voice        types.Voice // Voice to use for synthesis
	RebuildCache bool        // Force cache rebuild
	BypassCache  bool        // Bypass cache
	RequestID    string      // Optional request ID for tracing
}

// DefaultOptions returns default synthesis options.
// Defaults:
//   - Format: WAV16 (high quality, widely compatible)
//   - Voice: May at 24000 Hz (natural sounding female voice)
//   - Cache: Enabled (both RebuildCache and BypassCache are false)
//
// This provides sensible defaults for most use cases while allowing
// customization when needed.
func DefaultOptions() Options {
	return Options{
		Format:       FormatWAV16,
		Voice:        types.VoiceMay24000,
		RebuildCache: false,
		BypassCache:  false,
	}
}

// GetContentType returns the MIME type string corresponding to the audio format.
// This is useful for setting HTTP Content-Type headers and for file handling.
// Returns appropriate MIME type for each format:
//   - FormatWAV16: "audio/x-wav"
//   - FormatPCM16: "audio/x-pcm;bit=16;rate=24000"
//   - FormatOpus: "audio/ogg; codecs=opus"
//   - FormatALaw: "audio/pcma;rate=8000"
//   - FormatG729: "audio/g729"
//   - Default: "audio/x-wav" (fallback for unknown formats)
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
