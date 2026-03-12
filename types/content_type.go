package types

// ContentType represents audio/content MIME types
type ContentType string

const (
	ContentAudioMPEG        ContentType = "audio/mpeg"
	ContentAudioFLAC        ContentType = "audio/flac"
	ContentAudioOGGOpus     ContentType = "audio/ogg;codecs=opus"
	ContentAudioPCM8k16bit  ContentType = "audio/x-pcm;bit=16;rate=8000"
	ContentAudioPCM16k16bit ContentType = "audio/x-pcm;bit=16;rate=16000"
	ContentAudioPCMA8k      ContentType = "audio/pcma;rate=8000"
	ContentAudioPCMA16k     ContentType = "audio/pcma;rate=16000"
	ContentAudioPCMU8k      ContentType = "audio/pcmu;rate=8000"
	ContentAudioPCMU16k     ContentType = "audio/pcmu;rate=16000"
	ContentTextPlain        ContentType = "text/plain"
	ContentApplicationSSML  ContentType = "application/ssml"
)

// IsValid checks if content type is supported
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
