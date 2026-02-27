package utils

import "strings"

func IsOtherMedia(ext string) bool {
	switch ext {
	case ".mp4", ".webm", ".ogg", ".mp3", ".wav":
		return true
	default:
		return false
	}
}

func IsImage(ext string) bool {
	switch strings.ToLower(ext) {
	case ".webp", ".png", ".jpg", ".jpeg", ".gif":
		return true
	default:
		return false
	}
}
