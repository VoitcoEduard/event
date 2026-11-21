package utils

import (
	"bytes"
	"errors"
)

var SupportedAudioTypes = map[string]string{
	"audio/mpeg": ".mp3",
	"audio/wav":  ".wav",
	"audio/ogg":  ".ogg",
}

// ValidateAudio inspects the magic bytes of data and returns a canonical safe
// audio MIME type ("audio/mpeg", "audio/wav", or "audio/ogg") if valid.
func ValidateAudio(data []byte) (string, error) {
	if len(data) == 0 {
		return "", errors.New("empty audio data")
	}

	contentType := audioTypeFromBytes(data)
	if _, ok := SupportedAudioTypes[contentType]; ok {
		return contentType, nil
	}
	return "", errors.New("unsupported audio format")
}

func audioTypeFromBytes(data []byte) string {
	// WAV: RIFF....WAVE
	if len(data) >= 12 &&
		bytes.Equal(data[0:4], []byte("RIFF")) &&
		bytes.Equal(data[8:12], []byte("WAVE")) {
		return "audio/wav"
	}

	// Ogg container (RFC 3533): "OggS" + version 0x00, 27-byte minimum page header.
	if len(data) >= 27 &&
		bytes.Equal(data[0:4], []byte("OggS")) &&
		data[4] == 0x00 {
		return "audio/ogg"
	}

	// MP3 with ID3v2 header: "ID3" + version bytes (< 0xFF) + 4 synchsafe size bytes (MSB 0).
	if len(data) >= 10 &&
		data[0] == 'I' && data[1] == 'D' && data[2] == '3' &&
		data[3] < 0xFF && data[4] < 0xFF &&
		(data[6]&0x80) == 0 && (data[7]&0x80) == 0 &&
		(data[8]&0x80) == 0 && (data[9]&0x80) == 0 {
		return "audio/mpeg"
	}

	// Raw MPEG audio frame sync (MP3 without ID3v2 tag):
	// 11-bit sync (0xFF, top 3 bits of byte 1), valid MPEG version (!= 01),
	// valid layer (!= 00), valid bitrate (!= 0x0F), valid sample rate (!= 0x03).
	if len(data) >= 4 &&
		data[0] == 0xFF &&
		(data[1]&0xE0) == 0xE0 &&
		(data[1]>>3)&0x03 != 0x01 &&
		(data[1]>>1)&0x03 != 0x00 &&
		(data[2]>>4)&0x0F != 0x0F &&
		(data[2]>>2)&0x03 != 0x03 {
		return "audio/mpeg"
	}

	return ""
}
