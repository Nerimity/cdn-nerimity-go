package utils

import (
	"cdn_nerimity_go/database"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strconv"
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
			}
		}
	}
}

// func StartFileCleanup() {
// 	targetDir := "temp"
// 	interval := 5 * time.Minute
// 	threshold := 5 * time.Minute

// 	go func() {

// 		ticker := time.NewTicker(interval)
// 		defer ticker.Stop()

// 		for range ticker.C {
// 			cleanupOldFiles(targetDir, threshold)
// 		}
// 	}()
// }

func FlushTempFiles() {
	FlushTempFilesWithRoot(".")
}

func FlushTempFilesWithRoot(root string) {
	flushDir(filepath.Join(root, "temp"))
	flushDir(filepath.Join(root, "video-thumb-cache"))
}

func flushDir(dir string) {
	err := os.RemoveAll(dir)
	if err != nil {
		panic(err)
	}

	err = os.MkdirAll(dir, 0755)
	if err != nil {
		panic(err)
	}
}

func deleteExpiredFiles(databaseService *database.DatabaseService) {
	expiredFiles, err := databaseService.GetExpiredFiles()
	if err != nil {
		log.Printf("Error getting expired files: %v", err)
		return
	}

	fileIds := make([]int64, len(expiredFiles))

	for i, file := range expiredFiles {
		fileIds[i] = file.FileID

		path := "public/attachments/" + strconv.FormatInt(file.GroupID, 10) + "/" + strconv.FormatInt(file.FileID, 10)
		if err != nil {
			_, err := os.Stat(path)
			if os.IsNotExist(err) {
				continue
			}
			log.Printf("Error removing expired file %s: %v", path, err)
			return
		}
		err := os.RemoveAll(path)
		if err != nil {
			log.Printf("Error removing expired file %s: %v", path, err)
			return
		}
	}

	err = databaseService.DeleteByFileIds(fileIds)
	if err != nil {
		log.Printf("Error deleting expired files: %v", err)
		return
	}
}

func StartDeleteExpiredFiles(databaseService *database.DatabaseService) {
	interval := 1 * time.Minute

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			deleteExpiredFiles(databaseService)
		}
	}()
}

func StartVideoThumbnailCleanup(root string) {
	cacheDir := filepath.Join(root, "video-thumb-cache")
	interval := 5 * time.Minute
	maxAge := 12 * time.Hour

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			cleanupOldFiles(cacheDir, maxAge)
		}
	}()
}

func DeleteRecursiveEmpty(filePath string) error {
	const stopAt = "public"

	filePath = filepath.Clean(filePath)
	absStopAt, _ := filepath.Abs(stopAt)
	absFilePath, _ := filepath.Abs(filePath)

	err := DeleteWithRetry(absFilePath, 5)
	if err != nil {
		return err
	}

	currentDir := filepath.Dir(absFilePath)

	for {
		if currentDir == absStopAt || currentDir == "." || currentDir == "/" {
			break
		}

		err = DeleteWithRetry(currentDir, 3)
		if err != nil {
			return nil
		}

		nextDir := filepath.Dir(currentDir)
		if nextDir == currentDir {
			break
		}
		currentDir = nextDir
	}

	return nil
}

func DeleteWithRetry(path string, attempts int) error {
	for i := 0; i < attempts; i++ {
		err := os.Remove(path)
		if err == nil || errors.Is(err, os.ErrNotExist) {
			return nil
		}

		time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
	}

	return errors.New("path is locked or busy after multiple attempts")
}
