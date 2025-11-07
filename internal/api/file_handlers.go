package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"pinshare/internal/psfs"
	"pinshare/internal/store"
)

// DownloadFile handles GET /files/{fileSHA256}/download
// Downloads the file from IPFS on-demand and streams it to the client
func (s *Server) DownloadFile(w http.ResponseWriter, r *http.Request, fileSHA256 string) {
	// Get file metadata
	file, found := store.GlobalStore.GetFile(fileSHA256)
	if !found {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}

	// Check if file is banned
	if file.BanSet > 0 {
		writeError(w, http.StatusForbidden, "file is banned and cannot be downloaded")
		return
	}

	// Construct cache file path
	cacheDir := "./cache"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create cache directory: %v", err))
		return
	}

	filePath := filepath.Join(cacheDir, file.IPFSCID+"."+file.FileType)

	// Check if file already exists in cache
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// File not in cache, fetch from IPFS
		fmt.Printf("[API] Fetching file %s from IPFS\n", file.IPFSCID)
		if err := psfs.GetFileIPFS(file.IPFSCID, filePath); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to fetch file from IPFS: %v", err))
			return
		}
	}

	// Open the file
	f, err := os.Open(filePath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to open file: %v", err))
		return
	}
	defer f.Close()

	// Get file info for size
	fileInfo, err := f.Stat()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get file info: %v", err))
		return
	}

	// Set appropriate headers
	w.Header().Set("Content-Type", getContentType(file.FileType))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.%s\"", file.FileSHA256[:12], file.FileType))

	// Stream the file to the client
	if _, err := io.Copy(w, f); err != nil {
		fmt.Printf("[API ERROR] Failed to stream file to client: %v\n", err)
		return
	}

	fmt.Printf("[API] Successfully served file %s\n", file.IPFSCID)
}

// ServeFile handles GET /files/{fileSHA256}/content
// Serves the file for preview/inline display (e.g., PDF viewer)
func (s *Server) ServeFile(w http.ResponseWriter, r *http.Request, fileSHA256 string) {
	// Get file metadata
	file, found := store.GlobalStore.GetFile(fileSHA256)
	if !found {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}

	// Check if file is banned
	if file.BanSet > 0 {
		writeError(w, http.StatusForbidden, "file is banned and cannot be viewed")
		return
	}

	// Construct cache file path
	cacheDir := "./cache"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create cache directory: %v", err))
		return
	}

	filePath := filepath.Join(cacheDir, file.IPFSCID+"."+file.FileType)

	// Check if file already exists in cache
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// File not in cache, fetch from IPFS
		fmt.Printf("[API] Fetching file %s from IPFS for preview\n", file.IPFSCID)
		if err := psfs.GetFileIPFS(file.IPFSCID, filePath); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to fetch file from IPFS: %v", err))
			return
		}
	}

	// Open the file
	f, err := os.Open(filePath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to open file: %v", err))
		return
	}
	defer f.Close()

	// Get file info for size
	fileInfo, err := f.Stat()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get file info: %v", err))
		return
	}

	// Set appropriate headers for inline display
	w.Header().Set("Content-Type", getContentType(file.FileType))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	w.Header().Set("Content-Disposition", "inline") // Display inline instead of download
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour

	// Stream the file to the client
	if _, err := io.Copy(w, f); err != nil {
		fmt.Printf("[API ERROR] Failed to stream file to client: %v\n", err)
		return
	}

	fmt.Printf("[API] Successfully served file %s for preview\n", file.IPFSCID)
}

// getContentType returns the appropriate MIME type for a file extension
func getContentType(fileType string) string {
	contentTypes := map[string]string{
		"pdf":  "application/pdf",
		"txt":  "text/plain",
		"doc":  "application/msword",
		"docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"jpg":  "image/jpeg",
		"jpeg": "image/jpeg",
		"png":  "image/png",
		"gif":  "image/gif",
		"mp4":  "video/mp4",
		"mp3":  "audio/mpeg",
	}

	if ct, ok := contentTypes[fileType]; ok {
		return ct
	}
	return "application/octet-stream" // Default binary type
}
