package utils

import (
	"bytes"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"golang.org/x/image/webp"

	"fmt"
	"log"
	"os"
	"os/exec"
)

var SupportedImageTypes = map[string]string{
	"image/jpeg": "jpeg",
	"image/png":  "png",
	"image/gif":  "gif",
	"image/webp": "webp",
}

func ValidateImage(data []byte) (string, error) {
	if len(data) < 512 {
		if len(data) == 0 {
			return "", errors.New("empty image data")
		}
		contentType := imageTypeFromBytes(data)
		if _, ok := SupportedImageTypes[contentType]; ok {
			return contentType, nil
		}
		return "", errors.New("unsupported image format")
	}

	contentType := imageTypeFromBytes(data[:512])
	if _, ok := SupportedImageTypes[contentType]; ok {
		return contentType, nil
	}
	return "", errors.New("unsupported image format")
}

func imageTypeFromBytes(data []byte) string {
	if len(data) < 8 {
		return ""
	}

	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "image/jpeg"
	}
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "image/png"
	}
	if data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38 {
		return "image/gif"
	}
	if data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
		data[8] == 0x57 && data[9] == 0x45 && data[10] == 0x42 && data[11] == 0x50 {
		return "image/webp"
	}
	return ""
}

func GetImageDimensions(data []byte) (int, int, error) {
	if len(data) == 0 {
		return 0, 0, errors.New("empty image data")
	}

	contentType := imageTypeFromBytes(data)

	reader := bytes.NewReader(data)

	var config image.Config
	var err error

	switch contentType {
	case "image/webp":
		config, err = webp.DecodeConfig(reader)
	default:
		config, _, err = image.DecodeConfig(reader)
	}

	if err != nil {
		return 0, 0, err
	}
	return config.Width, config.Height, nil
}

// ConvertToWebP converts image data to WebP format using available system tools
func ConvertToWebP(data []byte) ([]byte, error) {
	// Create temporary input file
	tmpIn, err := os.CreateTemp("", "img-*.in")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpIn.Name())
	if _, err := tmpIn.Write(data); err != nil {
		return nil, err
	}
	tmpIn.Close()

	// Create temporary output file path
	tmpOut := tmpIn.Name() + ".webp"
	defer os.Remove(tmpOut)

	// Try standard tools based on OS
	var cmd *exec.Cmd
	if _, err := exec.LookPath("cwebp"); err == nil {
		cmd = exec.Command("cwebp", "-q", "80", tmpIn.Name(), "-o", tmpOut)
	} else if _, err := exec.LookPath("sips"); err == nil {
		cmd = exec.Command("sips", "-s", "format", "webp", tmpIn.Name(), "--out", tmpOut)
	} else if _, err := exec.LookPath("ffmpeg"); err == nil {
		cmd = exec.Command("ffmpeg", "-y", "-i", tmpIn.Name(), "-c:v", "libwebp", "-lossless", "0", "-qscale", "80", tmpOut)
	} else {
		return nil, errors.New("no standard tools available for webp conversion (cwebp, sips, ffmpeg)")
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("conversion failed: %v: %s", err, string(output))
	}

	// Read converted data
	return os.ReadFile(tmpOut)
}

func ProcessImage(data []byte) ([]byte, int, int, string, error) {
	if len(data) == 0 {
		return nil, 0, 0, "", errors.New("empty image data")
	}

	contentType, err := ValidateImage(data)
	if err != nil {
		return nil, 0, 0, "", err
	}

	width, height, err := GetImageDimensions(data)
	if err != nil {
		return nil, 0, 0, "", err
	}

	// Convert to WebP if not already WebP
	processedData := data
	finalMimeType := contentType
	if contentType != "image/webp" {
		if webpData, err := ConvertToWebP(data); err == nil {
			processedData = webpData
			finalMimeType = "image/webp"
		} else {
			// Log conversion error and fallback to original
			log.Printf("WebP conversion failed, using original: %v", err)
		}
	}

	return processedData, width, height, finalMimeType, nil
}
