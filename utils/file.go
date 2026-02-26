package utils

import (
	"strings"
)

func SafeFilename(filename string) string {
	trimmed := strings.TrimSpace(filename)

	if trimmed == "" {
		return "unnamed"
	}

	r := strings.NewReplacer("/", "_", "\\", "_")
	return r.Replace(trimmed)
}
