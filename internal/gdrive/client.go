package gdrive

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// Client wraps the Google Drive API client
type Client struct {
	service *drive.Service
	ctx     context.Context
}

// DriveFile represents a file or folder in Google Drive
type DriveFile struct {
	ID           string
	Name         string
	MimeType     string
	Size         int64
	ModifiedTime string
	Parents      []string
	IsFolder     bool
}

const (
	// Google Drive folder MIME type
	MimeTypeFolder = "application/vnd.google-apps.folder"
)

// NewClient creates a new Google Drive client with the given OAuth token
func NewClient(ctx context.Context, token *oauth2.Token, config *oauth2.Config) (*Client, error) {
	client := config.Client(ctx, token)

	service, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create drive service: %w", err)
	}

	return &Client{
		service: service,
		ctx:     ctx,
	}, nil
}

// ListFiles lists files in the specified folder (or root if folderID is empty)
func (c *Client) ListFiles(folderID string, pageSize int) ([]*DriveFile, error) {
	if pageSize == 0 {
		pageSize = 100
	}

	query := "trashed = false"
	if folderID != "" {
		query = fmt.Sprintf("'%s' in parents and trashed = false", folderID)
	} else {
		// Root folder
		query = "'root' in parents and trashed = false"
	}

	fileList, err := c.service.Files.List().
		Q(query).
		PageSize(int64(pageSize)).
		Fields("files(id, name, mimeType, size, modifiedTime, parents)").
		Do()

	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	var files []*DriveFile
	for _, f := range fileList.Files {
		size := int64(0)
		if f.Size != nil {
			size = *f.Size
		}

		files = append(files, &DriveFile{
			ID:           f.Id,
			Name:         f.Name,
			MimeType:     f.MimeType,
			Size:         size,
			ModifiedTime: f.ModifiedTime,
			Parents:      f.Parents,
			IsFolder:     f.MimeType == MimeTypeFolder,
		})
	}

	return files, nil
}

// GetFile retrieves metadata for a specific file
func (c *Client) GetFile(fileID string) (*DriveFile, error) {
	f, err := c.service.Files.Get(fileID).
		Fields("id, name, mimeType, size, modifiedTime, parents").
		Do()

	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	size := int64(0)
	if f.Size != nil {
		size = *f.Size
	}

	return &DriveFile{
		ID:           f.Id,
		Name:         f.Name,
		MimeType:     f.MimeType,
		Size:         size,
		ModifiedTime: f.ModifiedTime,
		Parents:      f.Parents,
		IsFolder:     f.MimeType == MimeTypeFolder,
	}, nil
}

// DownloadFile downloads a file from Google Drive to the specified destination path
func (c *Client) DownloadFile(fileID, destPath string) (int64, error) {
	// Get file metadata first
	file, err := c.GetFile(fileID)
	if err != nil {
		return 0, fmt.Errorf("failed to get file metadata: %w", err)
	}

	if file.IsFolder {
		return 0, fmt.Errorf("cannot download folder as file")
	}

	// Create destination directory if it doesn't exist
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Download file
	resp, err := c.service.Files.Get(fileID).Download()
	if err != nil {
		return 0, fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	// Create destination file
	outFile, err := os.Create(destPath)
	if err != nil {
		return 0, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer outFile.Close()

	// Copy data
	bytesWritten, err := io.Copy(outFile, resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to write file: %w", err)
	}

	return bytesWritten, nil
}

// ListFilesRecursive lists all files in a folder and its subfolders
func (c *Client) ListFilesRecursive(folderID string) ([]*DriveFile, error) {
	var allFiles []*DriveFile
	return c.listFilesRecursiveHelper(folderID, allFiles)
}

func (c *Client) listFilesRecursiveHelper(folderID string, accumulated []*DriveFile) ([]*DriveFile, error) {
	files, err := c.ListFiles(folderID, 1000)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		accumulated = append(accumulated, file)

		// If it's a folder, recurse into it
		if file.IsFolder {
			accumulated, err = c.listFilesRecursiveHelper(file.ID, accumulated)
			if err != nil {
				return nil, fmt.Errorf("failed to recurse into folder %s: %w", file.Name, err)
			}
		}
	}

	return accumulated, nil
}

// GetFilesByIDs retrieves metadata for multiple files by their IDs
func (c *Client) GetFilesByIDs(fileIDs []string) ([]*DriveFile, error) {
	var files []*DriveFile

	for _, fileID := range fileIDs {
		file, err := c.GetFile(fileID)
		if err != nil {
			return nil, fmt.Errorf("failed to get file %s: %w", fileID, err)
		}
		files = append(files, file)
	}

	return files, nil
}

// CalculateTotalSize calculates the total size of files (excluding folders)
func CalculateTotalSize(files []*DriveFile) int64 {
	var total int64
	for _, file := range files {
		if !file.IsFolder {
			total += file.Size
		}
	}
	return total
}

// FilterFiles filters files by criteria
func FilterFiles(files []*DriveFile, includeFiles, includeFolders bool) []*DriveFile {
	var filtered []*DriveFile
	for _, file := range files {
		if (file.IsFolder && includeFolders) || (!file.IsFolder && includeFiles) {
			filtered = append(filtered, file)
		}
	}
	return filtered
}
