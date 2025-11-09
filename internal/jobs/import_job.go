package jobs

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pinshare/internal/db"
	"pinshare/internal/gdrive"
	"pinshare/internal/p2p"
	"pinshare/internal/psfs"
	"pinshare/internal/store"
)

// ImportJob represents a Google Drive import job
type ImportJob struct {
	jobID        string
	userID       int64
	fileIDs      []string
	folderIDs    []string
	recursive    bool
	tempDir      string
	database     *db.DB
	driveClient  *gdrive.Client
	options      ImportOptions
}

// ImportOptions contains options for import jobs
type ImportOptions struct {
	PreserveHierarchy bool
	SkipDuplicates    bool
	IncludeShared     bool
	MaxFileSize       int64 // in bytes, 0 = unlimited
}

// NewImportJob creates a new import job
func NewImportJob(
	jobID string,
	userID int64,
	fileIDs []string,
	folderIDs []string,
	recursive bool,
	tempDir string,
	database *db.DB,
	driveClient *gdrive.Client,
	options ImportOptions,
) *ImportJob {
	return &ImportJob{
		jobID:       jobID,
		userID:      userID,
		fileIDs:     fileIDs,
		folderIDs:   folderIDs,
		recursive:   recursive,
		tempDir:     tempDir,
		database:    database,
		driveClient: driveClient,
		options:     options,
	}
}

// ID returns the job ID
func (j *ImportJob) ID() string {
	return j.jobID
}

// Execute executes the import job
func (j *ImportJob) Execute(ctx context.Context) error {
	log.Printf("[INFO] Starting import job %s", j.jobID)

	// Mark job as started
	if err := j.database.StartImportJob(j.jobID); err != nil {
		return fmt.Errorf("failed to start job: %w", err)
	}

	// Collect all files to import
	var allFiles []*gdrive.DriveFile

	// Add individual files
	if len(j.fileIDs) > 0 {
		files, err := j.driveClient.GetFilesByIDs(j.fileIDs)
		if err != nil {
			return fmt.Errorf("failed to get files: %w", err)
		}
		allFiles = append(allFiles, files...)
	}

	// Add files from folders
	for _, folderID := range j.folderIDs {
		var files []*gdrive.DriveFile
		var err error

		if j.recursive {
			files, err = j.driveClient.ListFilesRecursive(folderID)
		} else {
			files, err = j.driveClient.ListFiles(folderID, 1000)
		}

		if err != nil {
			return fmt.Errorf("failed to list files in folder %s: %w", folderID, err)
		}

		allFiles = append(allFiles, files...)
	}

	// Filter out folders (we only import files)
	var filesToImport []*gdrive.DriveFile
	totalBytes := int64(0)

	for _, file := range allFiles {
		if file.IsFolder {
			continue
		}

		// Check file size limit
		if j.options.MaxFileSize > 0 && file.Size > j.options.MaxFileSize {
			log.Printf("[WARNING] Skipping file %s (size %d exceeds limit %d)", file.Name, file.Size, j.options.MaxFileSize)
			continue
		}

		filesToImport = append(filesToImport, file)
		totalBytes += file.Size
	}

	log.Printf("[INFO] Job %s: found %d files to import (total size: %d bytes)", j.jobID, len(filesToImport), totalBytes)

	// Update job with total counts
	job, err := j.database.GetImportJob(j.jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}
	job.TotalFiles = len(filesToImport)
	job.TotalBytes = totalBytes

	// Add files to database
	for _, file := range filesToImport {
		_, err := j.database.AddImportFile(j.jobID, file.ID, file.Name, file.Size)
		if err != nil {
			log.Printf("[ERROR] Failed to add file %s to database: %v", file.Name, err)
			continue
		}
	}

	// Process each file
	completedCount := 0
	failedCount := 0
	transferredBytes := int64(0)

	for _, file := range filesToImport {
		// Check if context was cancelled
		select {
		case <-ctx.Done():
			log.Printf("[INFO] Job %s cancelled", j.jobID)
			if err := j.database.UpdateImportJobStatus(j.jobID, db.JobStatusCancelled); err != nil {
				log.Printf("[ERROR] Failed to update job status: %v", err)
			}
			return ctx.Err()
		default:
		}

		// Process file
		bytesProcessed, err := j.processFile(ctx, file)
		if err != nil {
			log.Printf("[ERROR] Failed to process file %s: %v", file.Name, err)
			failedCount++
		} else {
			completedCount++
			transferredBytes += bytesProcessed
		}

		// Update job progress
		if err := j.database.UpdateImportJobProgress(j.jobID, completedCount, failedCount, transferredBytes); err != nil {
			log.Printf("[ERROR] Failed to update job progress: %v", err)
		}
	}

	// Mark job as completed
	if err := j.database.CompleteImportJob(j.jobID); err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}

	log.Printf("[INFO] Job %s completed: %d succeeded, %d failed", j.jobID, completedCount, failedCount)
	return nil
}

// processFile processes a single file through the import pipeline
func (j *ImportJob) processFile(ctx context.Context, driveFile *gdrive.DriveFile) (int64, error) {
	log.Printf("[INFO] Processing file: %s (ID: %s, Size: %d)", driveFile.Name, driveFile.ID, driveFile.Size)

	// Find the import file record
	files, err := j.database.GetImportFilesByJob(j.jobID)
	if err != nil {
		return 0, fmt.Errorf("failed to get import files: %w", err)
	}

	var importFile *db.ImportFile
	for _, f := range files {
		if f.DriveFileID == driveFile.ID {
			importFile = f
			break
		}
	}

	if importFile == nil {
		return 0, fmt.Errorf("import file record not found for Drive file %s", driveFile.ID)
	}

	// 1. Download from Google Drive
	if err := j.database.UpdateImportFileStatus(importFile.ID, db.FileStatusDownloading, 10); err != nil {
		return 0, fmt.Errorf("failed to update status: %w", err)
	}

	// Create temp file path
	tempFilePath := filepath.Join(j.tempDir, fmt.Sprintf("%s_%s", j.jobID, driveFile.Name))
	defer os.Remove(tempFilePath) // Clean up temp file

	bytesDownloaded, err := j.driveClient.DownloadFile(driveFile.ID, tempFilePath)
	if err != nil {
		j.database.FailImportFile(importFile.ID, fmt.Sprintf("Download failed: %v", err))
		return 0, fmt.Errorf("failed to download file: %w", err)
	}

	log.Printf("[INFO] Downloaded %s (%d bytes)", driveFile.Name, bytesDownloaded)

	// 2. Compute SHA256 hash
	if err := j.database.UpdateImportFileStatus(importFile.ID, db.FileStatusHashing, 30); err != nil {
		return 0, fmt.Errorf("failed to update status: %w", err)
	}

	sha256Hash, err := psfs.GetSHA256(tempFilePath)
	if err != nil {
		j.database.FailImportFile(importFile.ID, fmt.Sprintf("Hashing failed: %v", err))
		return 0, fmt.Errorf("failed to compute hash: %w", err)
	}

	log.Printf("[INFO] Computed SHA256: %s", sha256Hash)

	// Check for duplicates if option is enabled
	if j.options.SkipDuplicates {
		if _, exists := store.GlobalStore.GetFile(strings.ToLower(sha256Hash)); exists {
			log.Printf("[INFO] Skipping duplicate file: %s (hash: %s)", driveFile.Name, sha256Hash)
			j.database.CompleteImportFile(importFile.ID, sha256Hash, "")
			return bytesDownloaded, nil
		}
	}

	// 3. Security scanning
	if err := j.database.UpdateImportFileStatus(importFile.ID, db.FileStatusScanning, 50); err != nil {
		return 0, fmt.Errorf("failed to update status: %w", err)
	}

	// Validate file type
	validType, err := psfs.ValidateFileType(tempFilePath)
	if err != nil || !validType {
		j.database.FailImportFile(importFile.ID, "Invalid file type")
		return 0, fmt.Errorf("invalid file type: %w", err)
	}

	// Run security scan
	secResult, err := psfs.SecCheck(tempFilePath, sha256Hash)
	if err != nil {
		j.database.FailImportFile(importFile.ID, fmt.Sprintf("Security scan failed: %v", err))
		return 0, fmt.Errorf("security scan failed: %w", err)
	}

	if !secResult {
		j.database.FailImportFile(importFile.ID, "Security scan detected malware")
		return 0, fmt.Errorf("security scan failed for file %s", driveFile.Name)
	}

	log.Printf("[INFO] Security scan passed for %s", driveFile.Name)

	// 4. Upload to IPFS
	if err := j.database.UpdateImportFileStatus(importFile.ID, db.FileStatusUploading, 70); err != nil {
		return 0, fmt.Errorf("failed to update status: %w", err)
	}

	ipfsCID := psfs.AddFileIPFS(tempFilePath)
	if ipfsCID == "" {
		j.database.FailImportFile(importFile.ID, "IPFS upload failed")
		return 0, fmt.Errorf("failed to add file to IPFS")
	}

	log.Printf("[INFO] Added to IPFS with CID: %s", ipfsCID)

	// 5. Store metadata
	if err := j.database.UpdateImportFileStatus(importFile.ID, db.FileStatusUploading, 90); err != nil {
		return 0, fmt.Errorf("failed to update status: %w", err)
	}

	// Extract file extension
	fileExtension := filepath.Ext(driveFile.Name)
	if len(fileExtension) > 0 {
		fileExtension = fileExtension[1:] // Remove leading dot
	}

	metadata := store.BaseMetadata{
		FileSHA256:  strings.ToLower(sha256Hash),
		IPFSCID:     strings.ToLower(ipfsCID),
		FileType:    strings.ToLower(fileExtension),
		AddedAt:     time.Now(),
		LastUpdated: time.Now(),
	}

	if err := store.GlobalStore.AddFile(metadata); err != nil {
		j.database.FailImportFile(importFile.ID, fmt.Sprintf("Metadata storage failed: %v", err))
		return 0, fmt.Errorf("failed to store metadata: %w", err)
	}

	// Trigger P2P metadata sync
	go p2p.PublishMetadataUpdate(metadata)

	// Mark file as completed
	if err := j.database.CompleteImportFile(importFile.ID, sha256Hash, ipfsCID); err != nil {
		return 0, fmt.Errorf("failed to mark file as completed: %w", err)
	}

	log.Printf("[INFO] Successfully imported %s", driveFile.Name)
	return bytesDownloaded, nil
}
