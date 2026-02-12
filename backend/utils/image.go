package utils

import (
	"bytes"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"golang.org/x/image/webp"
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

	return data, width, height, contentType, nil
}
