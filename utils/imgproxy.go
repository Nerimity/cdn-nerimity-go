package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strings"

	"github.com/cshum/vipsgen/vips"
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
	Crop   *ImageProxyCrop
}

func GenerateImageProxyURL(opts ImageProxyOptions) (string, error) {
	var parts []string

	var path = "local:///" + opts.Path

	var encodedPath = encodeURIComponent(path)

	image, err := vips.NewImageFromFile(opts.Path, nil)
	if err != nil {
		return "", err
	}
	defer image.Close()

	// isAnimated := image.Pages()
	width := image.Width()
	height := image.Height()

	aspectRatio := float64(opts.Size.Width) / float64(opts.Size.Height)

	targetDims := calculateTargetDimensions(
		calculateTargetDimensionsOptions{
			OriginalDimensions: Dimensions{Width: width, Height: height},
			MaxDimensions:      Dimensions{Width: opts.Size.Width, Height: opts.Size.Height},
			AspectRatio:        aspectRatio,
		},
	)

	if opts.Size.ResizeType == ResizeTypeFit {
		parts = append(parts, "rs:fit:"+fmt.Sprintf("%d:%d", targetDims.Width, targetDims.Height))
	}
	if opts.Size.ResizeType == ResizeTypeFill {
		parts = append(parts, "rs:fill:"+fmt.Sprintf("%d:%d", targetDims.Width, targetDims.Height))
	}

	if opts.Crop != nil {
		parts = append(parts, "crop:"+fmt.Sprintf("%d:%d:nowe:%d:%d", opts.Crop.Width, opts.Crop.Height, opts.Crop.X, opts.Crop.Y))

	}

	parts = append(parts, "plain/"+encodedPath)

	return BASE_PROXY + strings.Join(parts, "/") + "@webp", nil

}

func encodeURIComponent(str string) string {
	escaped := url.QueryEscape(str)
	// replace + with %20 to match JavaScript's encodeURIComponent
	return strings.ReplaceAll(escaped, "+", "%20")
}

type Dimensions struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type calculateTargetDimensionsOptions struct {
	OriginalDimensions Dimensions
	MaxDimensions      Dimensions
	AspectRatio        float64
}

func calculateTargetDimensions(opts calculateTargetDimensionsOptions) Dimensions {
	origWidth := float64(opts.OriginalDimensions.Width)
	origHeight := float64(opts.OriginalDimensions.Height)

	maxWidth := float64(opts.MaxDimensions.Width)
	maxHeight := float64(opts.MaxDimensions.Height)

	originalRatio := origWidth / origHeight

	var targetW, targetH float64

	// Fit the requested aspect ratio inside the original image bounds
	if originalRatio > opts.AspectRatio {
		targetH = origHeight
		targetW = targetH * opts.AspectRatio
	} else {
		targetW = origWidth
		targetH = targetW / opts.AspectRatio
	}

	// Calculate scales to fit within MaxDimensions
	widthScale := maxWidth / targetW
	heightScale := maxHeight / targetH

	// Use the smallest scale to ensure it fits both bounds
	// math.Min requires float64; ensures we only downscale
	finalScale := math.Min(1.0, math.Min(widthScale, heightScale))

	return Dimensions{
		Width:  int(math.Round(targetW * finalScale)),
		Height: int(math.Round(targetH * finalScale)),
	}
}

func PointsToDimensions(pointsStr string) (*Dimensions, []float64, error) {
	if pointsStr == "" {
		return nil, nil, nil
	}

	var parsedPoints []float64
	err := json.Unmarshal([]byte(pointsStr), &parsedPoints)
	if err != nil {
		return nil, nil, errors.New("Invalid crop points.")
	}

	if len(parsedPoints) != 4 {
		return nil, nil, errors.New("Invalid crop points.")
	}

	for _, point := range parsedPoints {
		if math.IsNaN(point) || point < 0 || point > 9999 {
			return nil, nil, errors.New("Invalid crop points.")
		}
	}

	dimensions := getDimensions(parsedPoints)
	return &dimensions, parsedPoints, nil
}

func getDimensions(points []float64) Dimensions {
	startX := points[0]
	startY := points[1]
	endX := points[2]
	endY := points[3]

	width := math.Abs(endX - startX)
	height := math.Abs(endY - startY)

	return Dimensions{
		Width:  int(math.Round(width)),
		Height: int(math.Round(height)),
	}
}
