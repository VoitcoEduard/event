package utils

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"slices"
	"strings"
	"testing"
)

var jpegMagic = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00}

var pngMagic = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

var gifMagic = []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00}

var webpMagic = []byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50}

func TestImageTypeFromBytes(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{"valid JPEG", jpegMagic, "image/jpeg"},
		{"valid PNG", pngMagic, "image/png"},
		{"valid GIF", gifMagic, "image/gif"},
		{"valid WebP", webpMagic, "image/webp"},
		{"too short - 2 bytes", []byte{0xFF, 0xD8}, ""},
		{"too short - 7 bytes", []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A}, ""},
		{"corrupted random bytes", []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77}, ""},
		{"JPEG wrong 3rd byte", []byte{0xFF, 0xD8, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, ""},
		{"PNG wrong 4th byte", []byte{0x89, 0x50, 0x4E, 0x00, 0x0D, 0x0A, 0x1A, 0x0A}, ""},
		{"GIF wrong 4th byte", []byte{0x47, 0x49, 0x46, 0x00, 0x39, 0x61, 0x01, 0x00}, ""},
		{"WebP wrong RIFF bytes", []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50}, ""},
		{"WebP wrong WEBP bytes", []byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, ""},
		{"empty data", []byte{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := imageTypeFromBytes(tt.data)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestValidateImage(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		expectedErr error
		expectedMT  string
	}{
		{
			name:        "empty data",
			data:        []byte{},
			expectedErr: errors.New("empty image data"),
			expectedMT:  "",
		},
		{
			name:        "valid short JPEG",
			data:        jpegMagic,
			expectedErr: nil,
			expectedMT:  "image/jpeg",
		},
		{
			name:        "valid short PNG",
			data:        pngMagic,
			expectedErr: nil,
			expectedMT:  "image/png",
		},
		{
			name:        "valid short GIF",
			data:        gifMagic,
			expectedErr: nil,
			expectedMT:  "image/gif",
		},
		{
			name:        "valid short WebP",
			data:        webpMagic,
			expectedErr: nil,
			expectedMT:  "image/webp",
		},
		{
			name:        "corrupted short data",
			data:        []byte{0x00, 0x11, 0x22},
			expectedErr: errors.New("unsupported image format"),
			expectedMT:  "",
		},
		{
			name:        "corrupted full data",
			data:        []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99},
			expectedErr: errors.New("unsupported image format"),
			expectedMT:  "",
		},
		{
			name:        "valid JPEG full data",
			data:        append(jpegMagic, make([]byte, 520)...),
			expectedErr: nil,
			expectedMT:  "image/jpeg",
		},
		{
			name:        "valid PNG full data",
			data:        append(pngMagic, make([]byte, 520)...),
			expectedErr: nil,
			expectedMT:  "image/png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateImage(tt.data)

			if tt.expectedErr != nil {
				if err == nil {
					t.Errorf("expected error %q, got nil", tt.expectedErr)
				} else if err.Error() != tt.expectedErr.Error() {
					t.Errorf("got error %q, want %q", err.Error(), tt.expectedErr.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expectedMT {
					t.Errorf("got mime type %q, want %q", result, tt.expectedMT)
				}
			}
		})
	}
}

func TestGetImageDimensions_EmptyData(t *testing.T) {
	_, _, err := GetImageDimensions([]byte{})
	if err == nil {
		t.Error("expected error for empty data, got nil")
	}
	if err.Error() != "empty image data" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGetImageDimensions_UnknownType(t *testing.T) {
	_, _, err := GetImageDimensions([]byte{0x00, 0x11, 0x22, 0x33})
	if err == nil {
		t.Error("expected error for unknown image type, got nil")
	}
}

func TestProcessImage_EmptyData(t *testing.T) {
	data, width, height, mimeType, err := ProcessImage([]byte{})

	if err == nil {
		t.Error("expected error for empty data, got nil")
	}
	if err.Error() != "empty image data" {
		t.Errorf("unexpected error: %v", err)
	}
	if data != nil {
		t.Errorf("expected nil data, got %v", data)
	}
	if width != 0 || height != 0 {
		t.Errorf("expected zero dimensions, got %dx%d", width, height)
	}
	if mimeType != "" {
		t.Errorf("expected empty mime type, got %q", mimeType)
	}
}

func TestProcessImage_UnsupportedFormat(t *testing.T) {
	data := []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77}

	_, _, _, _, err := ProcessImage(data)
	if err == nil {
		t.Error("expected error for unsupported format, got nil")
	}
	if err.Error() != "unsupported image format" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestProcessImage_InvalidImageData(t *testing.T) {
	data := jpegMagic

	_, _, _, _, err := ProcessImage(data)
	if err == nil {
		t.Error("expected error for invalid image data, got nil")
	}
}

func TestSupportedImageTypes_HasExpectedFormats(t *testing.T) {
	expected := map[string]string{
		"image/jpeg": "jpeg",
		"image/png":  "png",
		"image/gif":  "gif",
		"image/webp": "webp",
	}

	for mimeType, extension := range expected {
		if ext, ok := SupportedImageTypes[mimeType]; !ok {
			t.Errorf("expected %q to be in SupportedImageTypes", mimeType)
		} else if ext != extension {
			t.Errorf("got extension %q for %q, want %q", ext, mimeType, extension)
		}
	}

	if len(SupportedImageTypes) != len(expected) {
		t.Errorf("got %d supported types, want %d", len(SupportedImageTypes), len(expected))
	}
}

func TestProcessImage_ValidPNG(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode test png: %v", err)
	}

	processedData, width, height, mimeType, err := ProcessImage(buf.Bytes())
	if err != nil {
		t.Fatalf("ProcessImage returned error: %v", err)
	}

	if len(processedData) == 0 {
		t.Error("expected non-empty processed data")
	}

	if width != 200 || height != 100 {
		t.Errorf("expected 200x100, got %dx%d", width, height)
	}

	if mimeType != "image/webp" && mimeType != "image/png" {
		t.Errorf("expected image/webp or image/png, got %s", mimeType)
	}
}

func TestConvertToWebP_Resizing(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2400, 1200))
	for y := 0; y < 1200; y++ {
		for x := 0; x < 2400; x++ {
			img.Set(x, y, color.RGBA{R: 0, G: 255, B: 0, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode test png: %v", err)
	}

	webpData, err := ConvertToWebP(buf.Bytes(), 1200)
	if err != nil {
		t.Skipf("skipping ConvertToWebP test since no external tool available: %v", err)
	}

	w, h, err := GetImageDimensions(webpData)
	if err != nil {
		t.Fatalf("failed to get converted image dimensions: %v", err)
	}

	if w > 1200 {
		t.Errorf("expected width <= 1200, got %d", w)
	}
	if h > 600 {
		t.Errorf("expected height <= 600, got %d", h)
	}
}

func TestBuildWebPArgsPortraitAndNoUpscale(t *testing.T) {
	cwebpPortrait := buildWebPArgs("cwebp", 800, 2400, 1920, "in.png", "out.webp")
	if !slices.Contains(cwebpPortrait, "-resize") || !strings.Contains(strings.Join(cwebpPortrait, " "), "-resize 0 1920") {
		t.Fatalf("cwebp portrait args = %v, want -resize 0 1920", cwebpPortrait)
	}
	for _, tool := range []string{"cwebp", "sips", "ffmpeg"} {
		args := buildWebPArgs(tool, 400, 600, 1920, "in.png", "out.webp")
		joined := strings.Join(args, " ")
		if strings.Contains(joined, "-resize") || strings.Contains(joined, "-Z") || strings.Contains(joined, "-vf") {
			t.Fatalf("%s unexpectedly resized small image: %v", tool, args)
		}
	}
	ffmpegPortrait := buildWebPArgs("ffmpeg", 800, 2400, 1920, "in.png", "out.webp")
	if !strings.Contains(strings.Join(ffmpegPortrait, " "), "force_original_aspect_ratio=decrease") {
		t.Fatalf("ffmpeg portrait args = %v, want force_original_aspect_ratio=decrease", ffmpegPortrait)
	}
}

func TestConvertToWebP_TallPortraitShrinksProportionally(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 800, 2400))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode portrait png: %v", err)
	}
	webpData, err := ConvertToWebP(buf.Bytes(), MaxImageDimension)
	if err != nil {
		t.Skipf("skipping ConvertToWebP portrait test: %v", err)
	}
	w, h, err := GetImageDimensions(webpData)
	if err != nil {
		t.Fatalf("GetImageDimensions failed: %v", err)
	}
	if w != 640 || h != 1920 {
		t.Fatalf("tall portrait dimensions = %dx%d, want 640x1920", w, h)
	}
}
