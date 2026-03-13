// Package types provides core type definitions, constants, and interfaces
// used throughout the SaluteSpeech API client library. It defines content types,
// error types, logger interfaces, and other shared types.
package types

// ContentType represents audio and content MIME types supported by the SaluteSpeech API.
// These types are used to specify the format of audio files for recognition,
// text content for synthesis, and expected response formats.
type ContentType string

const (
	// ContentAudioMPEG represents MP3 audio format (audio/mpeg)
	ContentAudioMPEG ContentType = "audio/mpeg"

	// ContentAudioFLAC represents FLAC (Free Lossless Audio Codec) format (audio/flac)
	ContentAudioFLAC ContentType = "audio/flac"

	// ContentAudioOGGOpus represents Ogg container with Opus codec (audio/ogg;codecs=opus)
	ContentAudioOGGOpus ContentType = "audio/ogg;codecs=opus"

	// ContentAudioPCM8k16bit represents raw PCM audio at 8kHz sampling rate, 16-bit depth
	ContentAudioPCM8k16bit ContentType = "audio/x-pcm;bit=16;rate=8000"

	// ContentAudioPCM16k16bit represents raw PCM audio at 16kHz sampling rate, 16-bit depth
	// This is the recommended format for speech recognition
	ContentAudioPCM16k16bit ContentType = "audio/x-pcm;bit=16;rate=16000"

	// ContentAudioPCMA8k represents A-law encoded audio at 8kHz sampling rate
	ContentAudioPCMA8k ContentType = "audio/pcma;rate=8000"

	// ContentAudioPCMA16k represents A-law encoded audio at 16kHz sampling rate
	ContentAudioPCMA16k ContentType = "audio/pcma;rate=16000"

	// ContentAudioPCMU8k represents μ-law encoded audio at 8kHz sampling rate
	ContentAudioPCMU8k ContentType = "audio/pcmu;rate=8000"

	// ContentAudioPCMU16k represents μ-law encoded audio at 16kHz sampling rate
	ContentAudioPCMU16k ContentType = "audio/pcmu;rate=16000"

	// ContentTextPlain represents plain text content (text/plain)
	// Used for text uploads in synthesis tasks
	ContentTextPlain ContentType = "text/plain"

	// ContentApplicationSSML represents SSML (Speech Synthesis Markup Language) content
	// Used for advanced speech synthesis with pronunciation control
	ContentApplicationSSML ContentType = "application/ssml"
)

// IsValid checks if the content type is supported by the SaluteSpeech API.
// It returns true for all predefined content type constants, false for any other value.
// This is useful for validating user input and ensuring compatibility with API endpoints.
func (ct ContentType) IsValid() bool {
	switch ct {
	case ContentAudioMPEG, ContentAudioFLAC, ContentAudioOGGOpus,
		ContentAudioPCM8k16bit, ContentAudioPCM16k16bit,
		ContentAudioPCMA8k, ContentAudioPCMA16k,
		ContentAudioPCMU8k, ContentAudioPCMU16k,
		ContentTextPlain, ContentApplicationSSML:
		return true
	default:
		return false
	}
}
