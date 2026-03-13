package utils

// Audio signature constants (magic bytes)
const (
	// MP3: ID3 tag or frame sync
	mp3ID3Tag    = "ID3"
	mp3FrameSync = 0xFFB0 // 11111111 1011XXXX (mask)

	// FLAC: "fLaC"
	flacSignature = "fLaC"

	// OGG: "OggS"
	oggSignature = "OggS"

	// WAV: RIFF header with "WAVE" subtype
	wavRIFFPrefix = "RIFF"
	wavWAVEType   = "WAVE"

	// AAC/ADTS: Sync word 0xFFF
	aacADTSSync = 0xFFF0 // 1111111111110000 (mask)

	// M4A/MP4: FTYP box with brand
	mp4FTYP   = "ftyp"
	m4aBrand  = "M4A "
	mp4Brand  = "mp42"
	isomBrand = "isom"

	// AIFF: "FORM" + "AIFF"
	aiffFORM = "FORM"
	aiffType = "AIFF"
	aifcType = "AIFC"
)

// AudioFormat represents detected audio format metadata
type AudioFormat struct {
	ContentType string
	Extension   string
	Description string
}

// Known audio formats by signature
var audioFormats = map[string]AudioFormat{
	"mp3":  {ContentType: "audio/mpeg", Extension: ".mp3", Description: "MPEG Audio Layer III"},
	"flac": {ContentType: "audio/flac", Extension: ".flac", Description: "Free Lossless Audio Codec"},
	"ogg":  {ContentType: "audio/ogg", Extension: ".ogg", Description: "Ogg Vorbis/Opus"},
	"opus": {ContentType: "audio/opus", Extension: ".opus", Description: "Opus Interactive Audio"},
	"wav":  {ContentType: "audio/wav", Extension: ".wav", Description: "Waveform Audio File Format"},
	"aac":  {ContentType: "audio/aac", Extension: ".aac", Description: "Advanced Audio Coding (ADTS)"},
	"m4a":  {ContentType: "audio/mp4", Extension: ".m4a", Description: "MPEG-4 Audio"},
	"aiff": {ContentType: "audio/aiff", Extension: ".aiff", Description: "Audio Interchange File Format"},
}
