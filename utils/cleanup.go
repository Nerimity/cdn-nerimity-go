package utils

import (
	"log"
	"os"
	"path/filepath"
	"time"
)

func cleanupOldFiles(dir string, ageThreshold time.Duration) {
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("Error reading directory: %v", err)
		return
	}

	now := time.Now()

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		info, err := file.Info()
		if err != nil {
			log.Printf("Error getting file info: %v", err)
			continue
		}

		if now.Sub(info.ModTime()) > ageThreshold {
			path := filepath.Join(dir, file.Name())
			err := os.Remove(path)
			if err != nil {
				log.Printf("Error removing file %s: %v", path, err)
			} else {
				log.Printf("Removed file %s", path)
			}
		}
	}
}

func StartFileCleanup() {
	targetDir := "temp"
	interval := 1 * time.Minute
	threshold := 1 * time.Minute

	go func() {

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			cleanupOldFiles(targetDir, threshold)
		}
	}()
}

func FlushTempFiles() {
	dir := "temp"

	err := os.RemoveAll(dir)
	if err != nil {
		panic(err)
	}

	err = os.MkdirAll(dir, 0755)
	if err != nil {
		panic(err)
	}
}
