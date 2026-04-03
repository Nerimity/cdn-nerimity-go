package utils

import (
	"fmt"
	"os"
	"os/exec"
)

func GenerateThumbnail(videoPath string, outputPath string) ([]byte, error) {
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
		return nil, fmt.Errorf("ffmpeg failed: %w", err)
	}

	return os.ReadFile(outputPath)
}
