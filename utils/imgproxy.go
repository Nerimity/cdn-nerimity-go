package utils

import (
	"fmt"
	"net/url"
	"strings"
)

type ImageProxyOptions struct {
	URL        string
	IsLocalURL bool
	Static     bool
	Size       int
}

const BASE_PROXY = "http://localhost:8888/pr:sharp/"

func GenerateImageProxyURL(opts ImageProxyOptions) string {
	var parts []string

	var path = opts.URL
	if opts.IsLocalURL {
		path = "local:///" + path
	}
	var encodedPath = encodeURIComponent(path)

	if opts.Static {
		var static = "page:0"
		parts = append(parts, static)
	}

	if opts.Size != 0 {
		var size = fmt.Sprintf("rs:fit:%d:%d", opts.Size, opts.Size)
		parts = append(parts, size)
	}

	parts = append(parts, "plain/"+encodedPath)

	return BASE_PROXY + strings.Join(parts, "/") + "@webp"

}

func encodeURIComponent(str string) string {
	escaped := url.QueryEscape(str)
	// replace + with %20 to match JavaScript's encodeURIComponent
	return strings.ReplaceAll(escaped, "+", "%20")
}
