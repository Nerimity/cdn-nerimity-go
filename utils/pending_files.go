package utils

import (
	"errors"
	"os"
	"sync"
	"time"
)

type FileCategory string

const (
	AttachmentsCategory    FileCategory = "attachments"
	EmojisCategory         FileCategory = "emojis"
	AvatarsCategory        FileCategory = "avatars"
	ProfileBannersCategory FileCategory = "profile_banners"
)

type PendingFile struct {
	OriginalFilename string
	FileId           int64
	GroupId          int64
	UserId           int64
	Path             string
	Type             FileCategory
	ImageCompressed  bool
	MimeType         string
	Duration         int
	Height           int
	Width            int
	Animated         bool
	FileSize         int
	ExpiresAt        time.Time
}

type PendingFilesManager struct {
	mu    sync.RWMutex
	store map[int64]*PendingFile
}

func NewPendingFilesManager() *PendingFilesManager {
	return &PendingFilesManager{
		store: make(map[int64]*PendingFile),
	}
}

func (m *PendingFilesManager) Add(file PendingFile) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.store[file.FileId] = &file
}

func (m *PendingFilesManager) Verify(fileId int64) (*PendingFile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	file, ok := m.store[fileId]
	if !ok {
		return nil, errors.New("File not found")
	}
	delete(m.store, fileId)

	if time.Now().After(file.ExpiresAt) {
		return nil, errors.New("File expired")
	}

	return file, nil
}

func (m *PendingFilesManager) StartCleanup() {
	interval := 1 * time.Minute
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			m.mu.Lock()
			now := time.Now()
			for id, file := range m.store {
				if now.After(file.ExpiresAt) {
					os.Remove(file.Path)
					delete(m.store, id)
				}
			}
			m.mu.Unlock()
		}
	}()
}
