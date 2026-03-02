package utils

import (
	"net/url"
	"strings"
)

func EncodeURIComponent(str string) string {
	escaped := url.QueryEscape(str)
	// replace + with %20 to match JavaScript's encodeURIComponent
	return strings.ReplaceAll(escaped, "+", "%20")
}
