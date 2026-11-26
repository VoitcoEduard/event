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
	if len(data) >= 12 &&
		bytes.Equal(data[0:4], []byte("RIFF")) &&
		bytes.Equal(data[8:12], []byte("WAVE")) {
		return "audio/wav"
	}

	if len(data) >= 27 &&
		bytes.Equal(data[0:4], []byte("OggS")) &&
		data[4] == 0x00 {
		return "audio/ogg"
	}

	if len(data) >= 10 &&
		data[0] == 'I' && data[1] == 'D' && data[2] == '3' &&
		data[3] < 0xFF && data[4] < 0xFF &&
		(data[6]&0x80) == 0 && (data[7]&0x80) == 0 &&
		(data[8]&0x80) == 0 && (data[9]&0x80) == 0 {
		return "audio/mpeg"
	}

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
