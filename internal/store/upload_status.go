package store

import (
	"sync"
	"time"
)

// UploadStatus represents the current status of a file being processed
type UploadStatus struct {
	FileName      string    `json:"fileName"`
	FileSHA256    string    `json:"fileSHA256,omitempty"`
	Stage         string    `json:"stage"` // "validating", "scanning", "uploading", "storing", "completed", "failed"
	Progress      int       `json:"progress"` // 0-100
	Error         string    `json:"error,omitempty"`
	StartedAt     time.Time `json:"startedAt"`
	LastUpdatedAt time.Time `json:"lastUpdatedAt"`
	CompletedAt   *time.Time `json:"completedAt,omitempty"`
}

// UploadStatusManager manages the status of all files being uploaded
type UploadStatusManager struct {
	mu       sync.RWMutex
	statuses map[string]*UploadStatus // key is fileName
}

var StatusManager *UploadStatusManager

func init() {
	StatusManager = &UploadStatusManager{
		statuses: make(map[string]*UploadStatus),
	}
}

// StartUpload initializes tracking for a new file upload
func (usm *UploadStatusManager) StartUpload(fileName string) {
	usm.mu.Lock()
	defer usm.mu.Unlock()

	usm.statuses[fileName] = &UploadStatus{
		FileName:      fileName,
		Stage:         "validating",
		Progress:      0,
		StartedAt:     time.Now(),
		LastUpdatedAt: time.Now(),
	}
}

// UpdateStatus updates the status of a file upload
func (usm *UploadStatusManager) UpdateStatus(fileName, stage string, progress int, sha256 string) {
	usm.mu.Lock()
	defer usm.mu.Unlock()

	if status, exists := usm.statuses[fileName]; exists {
		status.Stage = stage
		status.Progress = progress
		status.LastUpdatedAt = time.Now()
		if sha256 != "" {
			status.FileSHA256 = sha256
		}
	}
}

// CompleteUpload marks a file upload as completed
func (usm *UploadStatusManager) CompleteUpload(fileName string) {
	usm.mu.Lock()
	defer usm.mu.Unlock()

	if status, exists := usm.statuses[fileName]; exists {
		status.Stage = "completed"
		status.Progress = 100
		now := time.Now()
		status.CompletedAt = &now
		status.LastUpdatedAt = now
	}
}

// FailUpload marks a file upload as failed
func (usm *UploadStatusManager) FailUpload(fileName, errorMsg string) {
	usm.mu.Lock()
	defer usm.mu.Unlock()

	if status, exists := usm.statuses[fileName]; exists {
		status.Stage = "failed"
		status.Error = errorMsg
		now := time.Now()
		status.CompletedAt = &now
		status.LastUpdatedAt = now
	}
}

// GetStatus retrieves the status of a specific file
func (usm *UploadStatusManager) GetStatus(fileName string) (*UploadStatus, bool) {
	usm.mu.RLock()
	defer usm.mu.RUnlock()

	status, exists := usm.statuses[fileName]
	return status, exists
}

// GetAllStatuses returns all upload statuses
func (usm *UploadStatusManager) GetAllStatuses() []*UploadStatus {
	usm.mu.RLock()
	defer usm.mu.RUnlock()

	statuses := make([]*UploadStatus, 0, len(usm.statuses))
	for _, status := range usm.statuses {
		statuses = append(statuses, status)
	}
	return statuses
}

// GetActiveStatuses returns only uploads that are currently in progress
func (usm *UploadStatusManager) GetActiveStatuses() []*UploadStatus {
	usm.mu.RLock()
	defer usm.mu.RUnlock()

	statuses := make([]*UploadStatus, 0)
	for _, status := range usm.statuses {
		if status.Stage != "completed" && status.Stage != "failed" {
			statuses = append(statuses, status)
		}
	}
	return statuses
}

// CleanupOldStatuses removes completed/failed uploads older than the specified duration
func (usm *UploadStatusManager) CleanupOldStatuses(maxAge time.Duration) {
	usm.mu.Lock()
	defer usm.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for fileName, status := range usm.statuses {
		if status.CompletedAt != nil && status.CompletedAt.Before(cutoff) {
			delete(usm.statuses, fileName)
		}
	}
}
