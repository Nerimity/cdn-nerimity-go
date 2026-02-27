package utils

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	vips "github.com/cshum/vipsgen/vips"
)

const BASE_PROXY = "http://localhost:8888/pr:sharp/"

type BasicImageProxyOptions struct {
	URL        string
	IsLocalURL bool
	Static     bool
	Size       int
}

func GenerateBasicImageProxyURL(opts BasicImageProxyOptions) string {
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

type ImageProxyResizeType string

const (
	ResizeTypeFit  ImageProxyResizeType = "fit"
	ResizeTypeFill ImageProxyResizeType = "fill"
)

type ImageProxySize struct {
	Width      int
	Height     int
	ResizeType ImageProxyResizeType
}
type ImageProxyCrop struct {
	Width  int
	Height int
	X      int
	Y      int
}

type ImageProxyOptions struct {
	Path   string
	Static bool
	Size   ImageProxySize
	Crop   ImageProxyCrop
}

func GenerateImageProxyURL(opts ImageProxyOptions) string {
	var parts []string

	var path = "local:///" + opts.Path

	var encodedPath = encodeURIComponent(path)

	img, err := vips.NewImageFromFile("input.jpg", nil)
	if err != nil {
		log.Fatal(err)
	}
	println(img)

	// if opts.Size != 0 {
	// 	var size = fmt.Sprintf("rs:fit:%d:%d", opts.Size, opts.Size)
	// 	parts = append(parts, size)
	// }

	parts = append(parts, "plain/"+encodedPath)

	return BASE_PROXY + strings.Join(parts, "/") + "@webp"

}

func encodeURIComponent(str string) string {
	escaped := url.QueryEscape(str)
	// replace + with %20 to match JavaScript's encodeURIComponent
	return strings.ReplaceAll(escaped, "+", "%20")
}
