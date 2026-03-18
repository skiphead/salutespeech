// Package utils provides utility functions for audio file format detection
// and content type determination used throughout the application.
package utils

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/skiphead/salutespeech/synthesis/async"
	"github.com/skiphead/salutespeech/types"
)

// formatMapping maps internal format names to types.ContentType values
var formatMapping = map[string]types.ContentType{
	"mp3":  types.ContentAudioMPEG,
	"flac": types.ContentAudioFLAC,
	"ogg":  types.ContentAudioOGGOpus,
	"opus": types.ContentAudioOGGOpus,
	"pcm":  types.ContentAudioPCM16k16bit,
	"raw":  types.ContentAudioPCM16k16bit,
}

// DetectAudioContentType determines the audio type by file extension OR by header.
// Returns types.ContentType compatible with your type system.
// It first attempts to identify the format using the file extension,
// falling back to header-based detection if the extension is unknown.
func DetectAudioContentType(path string) (types.ContentType, error) {
	if path == "" {
		return "", fmt.Errorf("empty path")
	}

	// 1. Quick detection by file extension
	ext := strings.ToLower(filepath.Ext(path))
	// Remove the dot from extension
	if len(ext) > 0 && ext[0] == '.' {
		ext = ext[1:]
	}

	if ct, ok := formatMapping[ext]; ok {
		return ct, nil
	}

	// 2. If extension is unknown — detect by file header
	return DetectAudioContentTypeByHeader(path)
}

// DetectAudioContentTypeByHeader identifies the audio format by reading the file's magic bytes.
// Opens the specified file and delegates to DetectAudioContentTypeFromReader
// for actual content type determination.
func DetectAudioContentTypeByHeader(path string) (types.ContentType, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			return
		}
	}(file)

	return DetectAudioContentTypeFromReader(file)
}

// DetectAudioContentTypeFromReader identifies the audio format by reading from an io.Reader.
// Useful for streaming scenarios and testing. Reads the first 36 bytes to detect
// known magic signatures for FLAC, OGG/Opus, MP3, and falls back to raw PCM format
// if no known signature is found but data exists.
func DetectAudioContentTypeFromReader(r io.Reader) (types.ContentType, error) {
	const bufSize = 36
	buf := make([]byte, bufSize)

	n, err := io.ReadFull(r, buf)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", fmt.Errorf("read header: %w", err)
	}
	if n == 0 {
		return "", fmt.Errorf("empty file")
	}
	buf = buf[:n]

	// FLAC: "fLaC" signature
	if len(buf) >= 4 && string(buf[0:4]) == "fLaC" {
		return formatMapping["flac"], nil
	}

	// OGG/Opus: "OggS" signature
	if len(buf) >= 4 && string(buf[0:4]) == "OggS" {
		// Opus has "OpusHead" signature at offset 28
		if len(buf) >= 32 && string(buf[28:36]) == "OpusHead" {
			return formatMapping["opus"], nil
		}
		return formatMapping["ogg"], nil
	}

	// MP3: ID3 tag or frame sync
	if len(buf) >= 3 && string(buf[0:3]) == "ID3" {
		return formatMapping["mp3"], nil
	}
	if len(buf) >= 2 {
		sync := binary.BigEndian.Uint16(buf[0:2]) & 0xFFE0
		if sync == 0xFFE0 && isValidMP3Frame(buf[1]) {
			return formatMapping["mp3"], nil
		}
	}

	// Check for PCM/RAW (no header, but can be identified by absence of other signatures)
	// Return raw only if file is not empty and wasn't identified as other format
	if n > 0 {
		// If file is not empty and wasn't recognized as another format,
		// assume it's raw/pcm
		return formatMapping["raw"], nil
	}

	return "", fmt.Errorf("unsupported audio format: no matching signature")
}

// ContentTypeToEncoding converts a types.ContentType to the corresponding async.Encoding.
// Returns an error if the content type has no direct encoding equivalent.
//
// Supported mappings:
//   - ContentAudioPCM16k16bit -> EncodingPCM16
//   - ContentAudioOGGOpus -> EncodingOpus
//   - ContentAudioMPEG, ContentAudioFLAC -> error (no direct equivalent)
func ContentTypeToEncoding(ct types.ContentType) (async.Encoding, error) {
	switch ct {
	case types.ContentAudioPCM16k16bit:
		return async.EncodingPCM16, nil
	case types.ContentAudioOGGOpus:
		return async.EncodingOpus, nil
	case types.ContentAudioMPEG:
		return "", fmt.Errorf("no direct encoding equivalent for content type: %s (mp3 not supported in async.Encoding)", ct)
	case types.ContentAudioFLAC:
		return "", fmt.Errorf("no direct encoding equivalent for content type: %s (flac not supported in async.Encoding)", ct)
	default:
		return "", fmt.Errorf("unsupported content type: %s", ct)
	}
}

// isValidMP3Frame validates MP3 frame header bytes according to the MPEG audio specification.
// Checks version and layer bits to ensure they represent valid combinations.
func isValidMP3Frame(b byte) bool {
	version := (b >> 3) & 0x03
	layer := (b >> 1) & 0x03
	return version != 0x01 && layer != 0x00
}
