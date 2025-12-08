package tokencount

import (
	"bytes"
	"encoding/base64"
	"image"
	_ "image/jpeg"
	_ "image/png"
)

// estimateImageBlock estimates tokens for an image block based on resolution
// Claude's image token calculation: (width * height) / 750
// Reference: https://docs.anthropic.com/en/docs/build-with-claude/vision
func estimateImageBlock(m map[string]any) int {
	source, ok := m["source"].(map[string]any)
	if !ok {
		return 1500 // fallback
	}

	sourceType, _ := source["type"].(string)
	if sourceType != "base64" {
		return 1500 // URL images use fallback
	}

	data, ok := source["data"].(string)
	if !ok || data == "" {
		return 1500
	}

	// Decode base64 and get image dimensions
	width, height := getImageDimensions(data)
	if width == 0 || height == 0 {
		return 1500 // fallback if decode fails
	}

	// Claude's formula: (width * height) / 750
	tokens := (width * height) / 750
	if tokens < 85 {
		tokens = 85 // minimum tokens for any image
	}
	return tokens
}

// getImageDimensions decodes base64 image and returns width, height
func getImageDimensions(base64Data string) (int, int) {
	decoded, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return 0, 0
	}

	config, _, err := image.DecodeConfig(bytes.NewReader(decoded))
	if err != nil {
		return 0, 0
	}

	return config.Width, config.Height
}
