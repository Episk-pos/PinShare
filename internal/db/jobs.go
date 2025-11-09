package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ImportJobStatus represents the status of an import job
type ImportJobStatus string

const (
	JobStatusPending   ImportJobStatus = "pending"
	JobStatusRunning   ImportJobStatus = "running"
	JobStatusCompleted ImportJobStatus = "completed"
	JobStatusFailed    ImportJobStatus = "failed"
	JobStatusCancelled ImportJobStatus = "cancelled"
)

// ImportFileStatus represents the status of a file being imported
type ImportFileStatus string

const (
	FileStatusPending     ImportFileStatus = "pending"
	FileStatusDownloading ImportFileStatus = "downloading"
	FileStatusHashing     ImportFileStatus = "hashing"
	FileStatusScanning    ImportFileStatus = "scanning"
	FileStatusUploading   ImportFileStatus = "uploading"
	FileStatusCompleted   ImportFileStatus = "completed"
	FileStatusFailed      ImportFileStatus = "failed"
)

// ImportJob represents a Google Drive import job
type ImportJob struct {
	ID               string
	UserID           int64
	Status           ImportJobStatus
	TotalFiles       int
	CompletedFiles   int
	FailedFiles      int
	TotalBytes       int64
	TransferredBytes int64
	StartedAt        *time.Time
	CompletedAt      *time.Time
	Options          map[string]interface{}
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// ImportFile represents a file in an import job
type ImportFile struct {
	ID            int64
	JobID         string
	DriveFileID   string
	FileName      string
	FileSize      int64
	Status        ImportFileStatus
	Progress      int
	SHA256Hash    string
	IPFSCID       string
	ErrorMessage  string
	RetryCount    int
	StartedAt     *time.Time
	CompletedAt   *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// CreateImportJob creates a new import job
func (db *DB) CreateImportJob(userID int64, options map[string]interface{}) (*ImportJob, error) {
	jobID := uuid.New().String()
	now := time.Now()

	optionsJSON, err := json.Marshal(options)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal options: %w", err)
	}

	_, err = db.Exec(`
		INSERT INTO import_jobs (id, user_id, status, options, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, jobID, userID, JobStatusPending, string(optionsJSON), now, now)

	if err != nil {
		return nil, fmt.Errorf("failed to create import job: %w", err)
	}

	return db.GetImportJob(jobID)
}

// GetImportJob retrieves an import job by ID
func (db *DB) GetImportJob(jobID string) (*ImportJob, error) {
	job := &ImportJob{}
	var optionsJSON string

	err := db.QueryRow(`
		SELECT id, user_id, status, total_files, completed_files, failed_files,
		       total_bytes, transferred_bytes, started_at, completed_at,
		       options, created_at, updated_at
		FROM import_jobs WHERE id = ?
	`, jobID).Scan(
		&job.ID,
		&job.UserID,
		&job.Status,
		&job.TotalFiles,
		&job.CompletedFiles,
		&job.FailedFiles,
		&job.TotalBytes,
		&job.TransferredBytes,
		&job.StartedAt,
		&job.CompletedAt,
		&optionsJSON,
		&job.CreatedAt,
		&job.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("import job not found")
	} else if err != nil {
		return nil, fmt.Errorf("failed to get import job: %w", err)
	}

	if err := json.Unmarshal([]byte(optionsJSON), &job.Options); err != nil {
		return nil, fmt.Errorf("failed to unmarshal options: %w", err)
	}

	return job, nil
}

// UpdateImportJobStatus updates the status of an import job
func (db *DB) UpdateImportJobStatus(jobID string, status ImportJobStatus) error {
	now := time.Now()
	_, err := db.Exec(`
		UPDATE import_jobs SET status = ?, updated_at = ? WHERE id = ?
	`, status, now, jobID)

	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	return nil
}

// UpdateImportJobProgress updates the progress metrics of an import job
func (db *DB) UpdateImportJobProgress(jobID string, completedFiles, failedFiles int, transferredBytes int64) error {
	now := time.Now()
	_, err := db.Exec(`
		UPDATE import_jobs
		SET completed_files = ?, failed_files = ?, transferred_bytes = ?, updated_at = ?
		WHERE id = ?
	`, completedFiles, failedFiles, transferredBytes, now, jobID)

	if err != nil {
		return fmt.Errorf("failed to update job progress: %w", err)
	}

	return nil
}

// StartImportJob marks a job as started
func (db *DB) StartImportJob(jobID string) error {
	now := time.Now()
	_, err := db.Exec(`
		UPDATE import_jobs SET status = ?, started_at = ?, updated_at = ? WHERE id = ?
	`, JobStatusRunning, now, now, jobID)

	if err != nil {
		return fmt.Errorf("failed to start job: %w", err)
	}

	return nil
}

// CompleteImportJob marks a job as completed
func (db *DB) CompleteImportJob(jobID string) error {
	now := time.Now()
	_, err := db.Exec(`
		UPDATE import_jobs SET status = ?, completed_at = ?, updated_at = ? WHERE id = ?
	`, JobStatusCompleted, now, now, jobID)

	if err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}

	return nil
}

// AddImportFile adds a file to an import job
func (db *DB) AddImportFile(jobID, driveFileID, fileName string, fileSize int64) (*ImportFile, error) {
	now := time.Now()

	result, err := db.Exec(`
		INSERT INTO import_files (job_id, drive_file_id, file_name, file_size, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, jobID, driveFileID, fileName, fileSize, FileStatusPending, now, now)

	if err != nil {
		return nil, fmt.Errorf("failed to add import file: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get file ID: %w", err)
	}

	return db.GetImportFile(id)
}

// GetImportFile retrieves an import file by ID
func (db *DB) GetImportFile(id int64) (*ImportFile, error) {
	file := &ImportFile{}

	err := db.QueryRow(`
		SELECT id, job_id, drive_file_id, file_name, file_size, status, progress,
		       sha256_hash, ipfs_cid, error_message, retry_count,
		       started_at, completed_at, created_at, updated_at
		FROM import_files WHERE id = ?
	`, id).Scan(
		&file.ID,
		&file.JobID,
		&file.DriveFileID,
		&file.FileName,
		&file.FileSize,
		&file.Status,
		&file.Progress,
		&file.SHA256Hash,
		&file.IPFSCID,
		&file.ErrorMessage,
		&file.RetryCount,
		&file.StartedAt,
		&file.CompletedAt,
		&file.CreatedAt,
		&file.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("import file not found")
	} else if err != nil {
		return nil, fmt.Errorf("failed to get import file: %w", err)
	}

	return file, nil
}

// GetImportFilesByJob retrieves all files for a job
func (db *DB) GetImportFilesByJob(jobID string) ([]*ImportFile, error) {
	rows, err := db.Query(`
		SELECT id, job_id, drive_file_id, file_name, file_size, status, progress,
		       sha256_hash, ipfs_cid, error_message, retry_count,
		       started_at, completed_at, created_at, updated_at
		FROM import_files WHERE job_id = ?
		ORDER BY created_at ASC
	`, jobID)

	if err != nil {
		return nil, fmt.Errorf("failed to query import files: %w", err)
	}
	defer rows.Close()

	var files []*ImportFile
	for rows.Next() {
		file := &ImportFile{}
		var sha256Hash, ipfsCID, errorMessage sql.NullString
		err := rows.Scan(
			&file.ID,
			&file.JobID,
			&file.DriveFileID,
			&file.FileName,
			&file.FileSize,
			&file.Status,
			&file.Progress,
			&sha256Hash,
			&ipfsCID,
			&errorMessage,
			&file.RetryCount,
			&file.StartedAt,
			&file.CompletedAt,
			&file.CreatedAt,
			&file.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan import file: %w", err)
		}

		// Convert sql.NullString to regular strings
		if sha256Hash.Valid {
			file.SHA256Hash = sha256Hash.String
		}
		if ipfsCID.Valid {
			file.IPFSCID = ipfsCID.String
		}
		if errorMessage.Valid {
			file.ErrorMessage = errorMessage.String
		}

		files = append(files, file)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating import files: %w", err)
	}

	return files, nil
}

// UpdateImportFileStatus updates the status and progress of an import file
func (db *DB) UpdateImportFileStatus(id int64, status ImportFileStatus, progress int) error {
	now := time.Now()
	_, err := db.Exec(`
		UPDATE import_files SET status = ?, progress = ?, updated_at = ? WHERE id = ?
	`, status, progress, now, id)

	if err != nil {
		return fmt.Errorf("failed to update file status: %w", err)
	}

	return nil
}

// CompleteImportFile marks a file as completed with hash and CID
func (db *DB) CompleteImportFile(id int64, sha256Hash, ipfsCID string) error {
	now := time.Now()
	_, err := db.Exec(`
		UPDATE import_files
		SET status = ?, progress = 100, sha256_hash = ?, ipfs_cid = ?, completed_at = ?, updated_at = ?
		WHERE id = ?
	`, FileStatusCompleted, sha256Hash, ipfsCID, now, now, id)

	if err != nil {
		return fmt.Errorf("failed to complete file: %w", err)
	}

	return nil
}

// FailImportFile marks a file as failed with an error message
func (db *DB) FailImportFile(id int64, errorMessage string) error {
	now := time.Now()
	_, err := db.Exec(`
		UPDATE import_files
		SET status = ?, error_message = ?, updated_at = ?, retry_count = retry_count + 1
		WHERE id = ?
	`, FileStatusFailed, errorMessage, now, id)

	if err != nil {
		return fmt.Errorf("failed to mark file as failed: %w", err)
	}

	return nil
}

// GetImportJobsByUser retrieves all import jobs for a user
func (db *DB) GetImportJobsByUser(userID int64, limit, offset int) ([]*ImportJob, error) {
	rows, err := db.Query(`
		SELECT id, user_id, status, total_files, completed_files, failed_files,
		       total_bytes, transferred_bytes, started_at, completed_at,
		       options, created_at, updated_at
		FROM import_jobs
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, userID, limit, offset)

	if err != nil {
		return nil, fmt.Errorf("failed to query import jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*ImportJob
	for rows.Next() {
		job := &ImportJob{}
		var optionsJSON string
		err := rows.Scan(
			&job.ID,
			&job.UserID,
			&job.Status,
			&job.TotalFiles,
			&job.CompletedFiles,
			&job.FailedFiles,
			&job.TotalBytes,
			&job.TransferredBytes,
			&job.StartedAt,
			&job.CompletedAt,
			&optionsJSON,
			&job.CreatedAt,
			&job.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan import job: %w", err)
		}

		if err := json.Unmarshal([]byte(optionsJSON), &job.Options); err != nil {
			return nil, fmt.Errorf("failed to unmarshal options: %w", err)
		}

		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating import jobs: %w", err)
	}

	return jobs, nil
}
