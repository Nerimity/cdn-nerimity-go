package utils

import (
	"fmt"
	"os/exec"
)

func GenerateThumbnail(videoPath string, outputPath string) (string, error) {
	cmd := exec.Command("ffmpeg",
		"-i", videoPath,
		"-vf", "thumbnail,scale='min(1080,iw)':'min(1080,ih)':force_original_aspect_ratio=decrease",
		"-frames:v", "1",
		"-c:v", "libwebp",
		"-quality", "80",
		"-y",
		outputPath,
	)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffmpeg failed: %w", err)
	}

	return outputPath, nil
}
