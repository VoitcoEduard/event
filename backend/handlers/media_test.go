package handlers

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/user/go-expo-sample/backend/postgres"
)

func newMultipartTrackRequest(t *testing.T, filename, contentType string, payload []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, filename))
	if contentType != "" {
		partHeader.Set("Content-Type", contentType)
	}
	part, err := writer.CreatePart(partHeader)
	if err != nil {
		t.Fatalf("create multipart part: %v", err)
	}
	if _, err := part.Write(payload); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/tracks", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestCreateTrackRejectsNonAudioUploads(t *testing.T) {
	h := &Handler{}

	cases := []struct {
		name        string
		filename    string
		contentType string
		payload     []byte
	}{
		{
			name:        "html disguised as mp3",
			filename:    "track.mp3",
			contentType: "audio/mpeg",
			payload:     []byte("<!DOCTYPE html><html><script>alert(document.cookie)</script></html>"),
		},
		{
			name:        "html file",
			filename:    "evil.html",
			contentType: "text/html",
			payload:     []byte("<html><body><script>alert(1)</script></body></html>"),
		},
		{
			name:        "svg file",
			filename:    "vector.svg",
			contentType: "image/svg+xml",
			payload:     []byte(`<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"/>`),
		},
		{
			name:        "plain text",
			filename:    "notes.txt",
			contentType: "text/plain",
			payload:     []byte("not an audio file"),
		},
		{
			name:        "empty file",
			filename:    "empty.mp3",
			contentType: "audio/mpeg",
			payload:     nil,
		},
		{
			name:        "png image uploaded to tracks",
			filename:    "image.png",
			contentType: "image/png",
			payload:     []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := newMultipartTrackRequest(t, tc.filename, tc.contentType, tc.payload)
			w := httptest.NewRecorder()
			h.CreateTrack(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d (body: %s)", w.Code, http.StatusBadRequest, w.Body.String())
			}
		})
	}
}

func TestServeMediaSecurityHeadersAndUnsafeMimeFallback(t *testing.T) {
	h := &Handler{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /media/{mediaID}/file", h.ServeMedia)

	cases := []struct {
		name            string
		storedMime      string
		filename        string
		data            []byte
		wantContentType string
		wantAttachment  bool
	}{
		{
			name:            "legacy html record forced to download",
			storedMime:      "text/html; charset=utf-8",
			filename:        "legacy.mp3",
			data:            []byte("<html><script>alert(1)</script></html>"),
			wantContentType: "application/octet-stream",
			wantAttachment:  true,
		},
		{
			name:            "legacy svg record forced to download",
			storedMime:      "image/svg+xml",
			filename:        "legacy.svg",
			data:            []byte(`<svg onload="alert(1)"/>`),
			wantContentType: "application/octet-stream",
			wantAttachment:  true,
		},
		{
			name:            "empty mime with html body never sniffed as html",
			storedMime:      "",
			filename:        "unknown.html",
			data:            []byte("<html><script>alert(1)</script></html>"),
			wantContentType: "application/octet-stream",
			wantAttachment:  true,
		},
		{
			name:            "mp3 served inline safely",
			storedMime:      "audio/mpeg",
			filename:        "song.mp3",
			data:            []byte("ID3\x03\x00\x00\x00\x00\x00\x0f"),
			wantContentType: "audio/mpeg",
			wantAttachment:  false,
		},
		{
			name:            "legacy audio/wave normalized to audio/wav",
			storedMime:      "audio/wave",
			filename:        "sound.wav",
			data:            []byte("RIFF\x24\x00\x00\x00WAVEfmt "),
			wantContentType: "audio/wav",
			wantAttachment:  false,
		},
		{
			name:            "legacy application/ogg normalized to audio/ogg",
			storedMime:      "application/ogg",
			filename:        "sound.ogg",
			data:            []byte("OggS\x00\x0201234567890123456789012345"),
			wantContentType: "audio/ogg",
			wantAttachment:  false,
		},
		{
			name:            "webp image served inline safely",
			storedMime:      "image/webp",
			filename:        "photo.webp",
			data:            []byte("RIFF\x24\x00\x00\x00WEBPVP8 "),
			wantContentType: "image/webp",
			wantAttachment:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id := uuid.NewString()
			cacheMedia(postgres.MediaRecord{
				ID:        id,
				Filename:  tc.filename,
				MimeType:  tc.storedMime,
				Size:      int64(len(tc.data)),
				CreatedAt: time.Now().UTC(),
			}, tc.data)
			defer InvalidateMediaCache(id)

			req := httptest.NewRequest(http.MethodGet, "/media/"+id+"/file", nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", w.Code)
			}
			if got := w.Header().Get("Content-Type"); got != tc.wantContentType {
				t.Fatalf("Content-Type = %q, want %q", got, tc.wantContentType)
			}
			if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
				t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
			}
			csp := w.Header().Get("Content-Security-Policy")
			if !strings.Contains(csp, "default-src 'none'") || !strings.Contains(csp, "sandbox") {
				t.Fatalf("Content-Security-Policy = %q, want default-src 'none' and sandbox", csp)
			}
			disp := w.Header().Get("Content-Disposition")
			if tc.wantAttachment && disp != "attachment" {
				t.Fatalf("Content-Disposition = %q, want attachment", disp)
			}
			if !tc.wantAttachment && disp != "" {
				t.Fatalf("Content-Disposition = %q, want empty", disp)
			}
		})
	}
}
