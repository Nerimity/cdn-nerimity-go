package utils

import (
	"net/url"
	"strings"
)

func EncodeURIComponent(str string) string {
	escaped := url.QueryEscape(str)

	res := strings.ReplaceAll(escaped, "+", "%20")

	res = strings.ReplaceAll(res, "%21", "!")
	res = strings.ReplaceAll(res, "%27", "'")
	res = strings.ReplaceAll(res, "%28", "(")
	res = strings.ReplaceAll(res, "%29", ")")
	res = strings.ReplaceAll(res, "%2A", "*")

	return res
}

func DecodeURIComponent(str string) (string, error) {

	return url.QueryUnescape(str)
}
