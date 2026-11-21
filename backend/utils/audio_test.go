package utils

import "testing"

func TestValidateAudio(t *testing.T) {
	validOgg := make([]byte, 32)
	copy(validOgg, []byte("OggS\x00\x02"))

	validID3MP3 := []byte("ID3\x03\x00\x00\x00\x00\x00\x0f\xff\xfb\x90\x00")
	validRawMP3 := []byte{0xFF, 0xFB, 0x90, 0x00, 0x00, 0x00}
	validWAV := []byte("RIFF\x24\x00\x00\x00WAVEfmt \x10\x00\x00\x00")

	validCases := []struct {
		name     string
		data     []byte
		wantType string
	}{
		{name: "wav", data: validWAV, wantType: "audio/wav"},
		{name: "ogg", data: validOgg, wantType: "audio/ogg"},
		{name: "mp3 with ID3v2", data: validID3MP3, wantType: "audio/mpeg"},
		{name: "mp3 raw frame sync", data: validRawMP3, wantType: "audio/mpeg"},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ValidateAudio(tc.data)
			if err != nil {
				t.Fatalf("expected valid audio, got error: %v", err)
			}
			if got != tc.wantType {
				t.Fatalf("got %q, want %q", got, tc.wantType)
			}
		})
	}

	invalidCases := []struct {
		name string
		data []byte
	}{
		{name: "empty", data: nil},
		{name: "html", data: []byte("<!DOCTYPE html><html><body><script>alert(1)</script></body></html>")},
		{name: "svg", data: []byte(`<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"></svg>`)},
		{name: "javascript", data: []byte("fetch('/auth/me').then(r => r.text())")},
		{name: "plain text", data: []byte("hello world")},
		{name: "webp riff container", data: []byte("RIFF\x24\x00\x00\x00WEBPVP8 ")},
		{name: "jpeg image", data: []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F'}},
		{name: "png image", data: []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}},
		{name: "malformed ID3v2 non-synchsafe", data: []byte{'I', 'D', '3', 0x03, 0x00, 0x00, 0x80, 0x00, 0x00, 0x00}},
		{name: "truncated ogg", data: []byte("OggS\x00")},
		{name: "invalid mpeg reserved bitrate", data: []byte{0xFF, 0xFB, 0xF0, 0x00}},
	}

	for _, tc := range invalidCases {
		t.Run("reject_"+tc.name, func(t *testing.T) {
			if got, err := ValidateAudio(tc.data); err == nil {
				t.Fatalf("expected error for %s, got type %q", tc.name, got)
			}
		})
	}
}
