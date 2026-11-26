package utils

import (
	"bytes"
	"image"
	"image/jpeg"
	"os/exec"
	"testing"
)

func BenchmarkImageConversion(b *testing.B) {
	if _, err := exec.LookPath("cwebp"); err != nil {
		b.Skip("install cwebp (libwebp-tools) to benchmark NAS image conversion")
	}
	img := image.NewRGBA(image.Rect(0, 0, 3840, 2160))
	var noise uint32 = 1
	for i := range img.Pix {
		noise = noise*1664525 + 1013904223
		img.Pix[i] = byte(noise >> 24)
	}
	var input bytes.Buffer
	if err := jpeg.Encode(&input, img, &jpeg.Options{Quality: 85}); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.SetBytes(int64(input.Len()))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _, mime, err := ProcessImage(input.Bytes())
		if err != nil || mime != "image/webp" {
			b.Fatalf("image conversion failed: %s, %v", mime, err)
		}
	}
}
