package store

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// UploadStatus represents the current status of a file being processed
type UploadStatus struct {
	FileName      string     `json:"fileName"`
	FileSHA256    string     `json:"fileSHA256,omitempty"`
	Stage         string     `json:"stage"` // "validating", "hashing", "scanning", "uploading", "storing", "completed", "failed", "cancelled"
	Progress      int        `json:"progress"` // 0-100
	Error         string     `json:"error,omitempty"`
	StartedAt     time.Time  `json:"startedAt"`
	LastUpdatedAt time.Time  `json:"lastUpdatedAt"`
	CompletedAt   *time.Time `json:"completedAt,omitempty"`
}

// UploadStatusManager manages the status of all files being uploaded
type UploadStatusManager struct {
	mu          sync.RWMutex
	statuses    map[string]*UploadStatus             // key is fileName
	cancelFuncs map[string]context.CancelFunc        // cancel functions for each upload
}

var StatusManager *UploadStatusManager

func init() {
	StatusManager = &UploadStatusManager{
		statuses:    make(map[string]*UploadStatus),
		cancelFuncs: make(map[string]context.CancelFunc),
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
	// Clean up cancel function since upload is complete
	delete(usm.cancelFuncs, fileName)
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
	// Clean up cancel function since upload has failed
	delete(usm.cancelFuncs, fileName)
}

// GetStatus retrieves the status of a specific file
func (usm *UploadStatusManager) GetStatus(fileName string) (*UploadStatus, bool) {
	usm.mu.RLock()
	defer usm.mu.RUnlock()

	status, exists := usm.statuses[fileName]
	return status, exists
}

// GetAllStatuses returns all upload statuses sorted by start time (most recent first)
func (usm *UploadStatusManager) GetAllStatuses() []*UploadStatus {
	usm.mu.RLock()
	defer usm.mu.RUnlock()

	statuses := make([]*UploadStatus, 0, len(usm.statuses))
	for _, status := range usm.statuses {
		statuses = append(statuses, status)
	}

	// Sort by StartedAt in descending order (most recent first)
	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].StartedAt.After(statuses[j].StartedAt)
	})

	return statuses
}

// GetActiveStatuses returns only uploads that are currently in progress, sorted by start time (most recent first)
func (usm *UploadStatusManager) GetActiveStatuses() []*UploadStatus {
	usm.mu.RLock()
	defer usm.mu.RUnlock()

	statuses := make([]*UploadStatus, 0)
	for _, status := range usm.statuses {
		if status.Stage != "completed" && status.Stage != "failed" && status.Stage != "cancelled" {
			statuses = append(statuses, status)
		}
	}

	// Sort by StartedAt in descending order (most recent first)
	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].StartedAt.After(statuses[j].StartedAt)
	})

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
			delete(usm.cancelFuncs, fileName) // Also remove cancel function
		}
	}
}

// StartUploadWithContext initializes tracking for a new file upload with cancellation support
func (usm *UploadStatusManager) StartUploadWithContext(fileName string, cancelFunc context.CancelFunc) {
	usm.mu.Lock()
	defer usm.mu.Unlock()

	usm.statuses[fileName] = &UploadStatus{
		FileName:      fileName,
		Stage:         "validating",
		Progress:      0,
		StartedAt:     time.Now(),
		LastUpdatedAt: time.Now(),
	}
	usm.cancelFuncs[fileName] = cancelFunc
}

// CancelUpload cancels an in-progress upload
func (usm *UploadStatusManager) CancelUpload(fileName string) error {
	usm.mu.Lock()
	defer usm.mu.Unlock()

	cancelFunc, exists := usm.cancelFuncs[fileName]
	if !exists {
		return fmt.Errorf("upload not found or already completed")
	}

	status, statusExists := usm.statuses[fileName]
	if !statusExists {
		return fmt.Errorf("upload status not found")
	}

	// Check if upload is in a cancellable state
	if status.Stage == "completed" || status.Stage == "failed" || status.Stage == "cancelled" {
		return fmt.Errorf("upload cannot be cancelled in current state: %s", status.Stage)
	}

	// Cancel the context
	cancelFunc()

	// Update status to cancelled
	status.Stage = "cancelled"
	status.Error = "Upload cancelled by user"
	now := time.Now()
	status.CompletedAt = &now
	status.LastUpdatedAt = now

	// Remove cancel function
	delete(usm.cancelFuncs, fileName)

	return nil
}

// RemoveCancelFunc removes the cancel function for a completed upload (cleanup)
func (usm *UploadStatusManager) RemoveCancelFunc(fileName string) {
	usm.mu.Lock()
	defer usm.mu.Unlock()

	delete(usm.cancelFuncs, fileName)
}
