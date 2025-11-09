package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

	"pinshare/internal/db"
	"pinshare/internal/gdrive"
	"pinshare/internal/jobs"

	"golang.org/x/oauth2"
)

// GDriveServer manages Google Drive import functionality
type GDriveServer struct {
	database     *db.DB
	oauthManager *gdrive.OAuthManager
	jobQueue     *jobs.Queue
	tempDir      string
	maxFileSize  int64

	// Session management for OAuth flow
	mu          sync.RWMutex
	oauthStates map[string]*gdrive.OAuthManager // Map state to OAuth manager instances
}

// NewGDriveServer creates a new Google Drive API server
func NewGDriveServer(
	database *db.DB,
	oauthConfig *gdrive.OAuthConfig,
	jobQueue *jobs.Queue,
	tempDir string,
	maxFileSize int64,
) *GDriveServer {
	return &GDriveServer{
		database:     database,
		oauthManager: gdrive.NewOAuthManager(oauthConfig),
		jobQueue:     jobQueue,
		tempDir:      tempDir,
		maxFileSize:  maxFileSize,
		oauthStates:  make(map[string]*gdrive.OAuthManager),
	}
}

// AuthorizeRequest handles the authorization request
func (s *GDriveServer) AuthorizeRequest(w http.ResponseWriter, r *http.Request) {
	log.Println("[INFO] POST /api/google-drive/authorize")

	// Create new OAuth manager for this session
	manager := s.oauthManager

	authURL, err := manager.GenerateAuthURL()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to generate auth URL: %v", err))
		return
	}

	// Store the OAuth manager by state for later retrieval
	s.mu.Lock()
	s.oauthStates[manager.GetState()] = manager
	s.mu.Unlock()

	response := map[string]string{
		"authorizationUrl": authURL,
		"state":            manager.GetState(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CallbackRequest handles the OAuth callback
func (s *GDriveServer) CallbackRequest(w http.ResponseWriter, r *http.Request) {
	log.Println("[INFO] POST /api/google-drive/callback")

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		writeError(w, http.StatusBadRequest, "missing code or state parameter")
		return
	}

	// Retrieve OAuth manager by state
	s.mu.RLock()
	manager, exists := s.oauthStates[state]
	s.mu.RUnlock()

	if !exists {
		writeError(w, http.StatusBadRequest, "invalid state parameter")
		return
	}

	// Exchange code for token
	ctx := context.Background()
	token, err := manager.ExchangeCode(ctx, code, state)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to exchange code: %v", err))
		return
	}

	// Get user info from Google
	client := manager.GetClient(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get user info: %v", err))
		return
	}
	defer resp.Body.Close()

	var userInfo struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to parse user info: %v", err))
		return
	}

	// Store user and tokens in database
	_, err = s.database.CreateOrUpdateUser(
		userInfo.ID,
		userInfo.Email,
		token.AccessToken,
		token.RefreshToken,
		token.Expiry,
	)

	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to store user: %v", err))
		return
	}

	// Clean up OAuth state
	s.mu.Lock()
	delete(s.oauthStates, state)
	s.mu.Unlock()

	log.Printf("[INFO] User authenticated: %s (%s)", userInfo.Email, userInfo.ID)

	response := map[string]interface{}{
		"success": true,
		"user": map[string]string{
			"googleId": userInfo.ID,
			"email":    userInfo.Email,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SetToken handles token submission from OAuth broker
func (s *GDriveServer) SetToken(w http.ResponseWriter, r *http.Request) {
	log.Println("[INFO] POST /api/google-drive/set-token")

	var token oauth2.Token
	if err := json.NewDecoder(r.Body).Decode(&token); err != nil {
		log.Printf("[ERROR] Failed to decode token: %v", err)
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid token format: %v", err))
		return
	}

	// Validate token by getting user info from Google
	ctx := context.Background()
	client := s.oauthManager.GetClient(ctx, &token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		log.Printf("[ERROR] Failed to validate token with Google: %v", err)
		writeError(w, http.StatusUnauthorized, fmt.Sprintf("invalid token: %v", err))
		return
	}
	defer resp.Body.Close()

	var userInfo struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		log.Printf("[ERROR] Failed to parse user info: %v", err)
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to parse user info: %v", err))
		return
	}

	log.Printf("[DEBUG] Received user info - ID: %s, Email: %s", userInfo.ID, userInfo.Email)

	// Store user and tokens in database
	_, err = s.database.CreateOrUpdateUser(
		userInfo.ID,
		userInfo.Email,
		token.AccessToken,
		token.RefreshToken,
		token.Expiry,
	)

	if err != nil {
		log.Printf("[ERROR] Failed to store user in database: %v", err)
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to store user: %v", err))
		return
	}

	log.Printf("[INFO] Token saved for user: %s (%s)", userInfo.Email, userInfo.ID)

	response := map[string]interface{}{
		"success": true,
		"user": map[string]string{
			"googleId": userInfo.ID,
			"email":    userInfo.Email,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAuthStatus returns the authentication status for a user
func (s *GDriveServer) GetAuthStatus(w http.ResponseWriter, r *http.Request) {
	log.Println("[INFO] GET /api/google-drive/auth-status")

	// Check for google_id query parameter (optional)
	// If not provided, return the first user (single-user mode for OAuth broker)
	googleID := r.URL.Query().Get("google_id")

	var user *db.User
	var err error

	if googleID != "" {
		user, err = s.database.GetUserByGoogleID(googleID)
	} else {
		// Get any authenticated user (OAuth broker flow - single user mode)
		users, err := s.database.GetAllUsers()
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get users: %v", err))
			return
		}
		if len(users) > 0 {
			user = users[0]
		} else {
			// No users authenticated
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]bool{"authenticated": false})
			return
		}
	}

	if err != nil {
		if err == db.ErrUserNotFound {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]bool{"authenticated": false})
			return
		}
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get user: %v", err))
		return
	}

	response := map[string]interface{}{
		"authenticated": true,
		"email":         user.Email,
		"tokenExpired":  user.IsTokenExpired(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RevokeAccess revokes Google Drive access for a user
func (s *GDriveServer) RevokeAccess(w http.ResponseWriter, r *http.Request) {
	log.Println("[INFO] DELETE /api/google-drive/revoke")

	googleID := r.URL.Query().Get("google_id")
	if googleID == "" {
		writeError(w, http.StatusBadRequest, "missing google_id parameter")
		return
	}

	user, err := s.database.GetUserByGoogleID(googleID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	// Get decrypted tokens
	accessToken, _, err := user.GetDecryptedTokens()
	if err != nil {
		log.Printf("[WARNING] Failed to decrypt tokens for revocation: %v", err)
	} else {
		// Attempt to revoke token with Google
		ctx := context.Background()
		if err := s.oauthManager.RevokeToken(ctx, accessToken); err != nil {
			log.Printf("[WARNING] Failed to revoke token with Google: %v", err)
		}
	}

	// Delete user from database
	if err := s.database.DeleteUser(googleID); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete user: %v", err))
		return
	}

	log.Printf("[INFO] Revoked access for user: %s", googleID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// ListFolders lists files and folders in a Google Drive folder
func (s *GDriveServer) ListFolders(w http.ResponseWriter, r *http.Request) {
	log.Println("[INFO] GET /api/google-drive/folders")

	googleID := r.URL.Query().Get("google_id")
	folderID := r.URL.Query().Get("path")

	// Support single-user mode - if google_id not provided, use the only user
	if googleID == "" {
		users, err := s.database.GetAllUsers()
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get users: %v", err))
			return
		}
		if len(users) == 0 {
			writeError(w, http.StatusUnauthorized, "no authenticated users found")
			return
		}
		if len(users) > 1 {
			writeError(w, http.StatusBadRequest, "multiple users found, google_id parameter required")
			return
		}
		googleID = users[0].GoogleID
	}

	// Get user and create Drive client
	user, driveClient, err := s.getUserAndClient(googleID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, fmt.Sprintf("failed to authenticate: %v", err))
		return
	}

	// Check if token needs refresh
	if user.IsTokenExpired() {
		if err := s.refreshUserToken(user); err != nil {
			writeError(w, http.StatusUnauthorized, fmt.Sprintf("failed to refresh token: %v", err))
			return
		}
		// Re-create client with new token
		_, driveClient, err = s.getUserAndClient(googleID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create client: %v", err))
			return
		}
	}

	// List files
	files, err := driveClient.ListFiles(folderID, 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list files: %v", err))
		return
	}

	// Convert to response format
	var response []map[string]interface{}
	for _, file := range files {
		response = append(response, map[string]interface{}{
			"id":           file.ID,
			"name":         file.Name,
			"mimeType":     file.MimeType,
			"size":         file.Size,
			"modifiedTime": file.ModifiedTime,
			"parents":      file.Parents,
			"isFolder":     file.IsFolder,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PreviewImport previews files that would be imported
func (s *GDriveServer) PreviewImport(w http.ResponseWriter, r *http.Request) {
	log.Println("[INFO] POST /api/google-drive/preview-import")

	var req struct {
		GoogleID  string   `json:"googleId"`
		FileIDs   []string `json:"fileIds"`
		FolderIDs []string `json:"folderIds"`
		Recursive bool     `json:"recursive"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	// Support single-user mode
	googleID := req.GoogleID
	if googleID == "" {
		users, err := s.database.GetAllUsers()
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get users: %v", err))
			return
		}
		if len(users) == 0 {
			writeError(w, http.StatusUnauthorized, "no authenticated users found")
			return
		}
		if len(users) > 1 {
			writeError(w, http.StatusBadRequest, "multiple users found, googleId parameter required")
			return
		}
		googleID = users[0].GoogleID
	}

	// Get Drive client
	_, driveClient, err := s.getUserAndClient(googleID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, fmt.Sprintf("failed to authenticate: %v", err))
		return
	}

	// Collect all files
	var allFiles []*gdrive.DriveFile

	// Get individual files
	if len(req.FileIDs) > 0 {
		files, err := driveClient.GetFilesByIDs(req.FileIDs)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get files: %v", err))
			return
		}
		allFiles = append(allFiles, files...)
	}

	// Get files from folders
	for _, folderID := range req.FolderIDs {
		var files []*gdrive.DriveFile
		var err error

		if req.Recursive {
			files, err = driveClient.ListFilesRecursive(folderID)
		} else {
			files, err = driveClient.ListFiles(folderID, 1000)
		}

		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list folder: %v", err))
			return
		}

		allFiles = append(allFiles, files...)
	}

	// Filter out folders and calculate totals
	var fileList []map[string]interface{}
	totalSize := int64(0)
	fileCount := 0

	for _, file := range allFiles {
		if file.IsFolder {
			continue
		}

		fileList = append(fileList, map[string]interface{}{
			"id":           file.ID,
			"name":         file.Name,
			"size":         file.Size,
			"modifiedTime": file.ModifiedTime,
		})

		totalSize += file.Size
		fileCount++
	}

	response := map[string]interface{}{
		"files":      fileList,
		"totalSize":  totalSize,
		"totalCount": fileCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// StartImport starts a new import job
func (s *GDriveServer) StartImport(w http.ResponseWriter, r *http.Request) {
	log.Println("[INFO] POST /api/google-drive/import")

	var req struct {
		GoogleID          string   `json:"googleId"`
		FileIDs           []string `json:"fileIds"`
		FolderIDs         []string `json:"folderIds"`
		Recursive         bool     `json:"recursive"`
		PreserveHierarchy bool     `json:"preserveHierarchy"`
		SkipDuplicates    bool     `json:"skipDuplicates"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	// Support single-user mode
	googleID := req.GoogleID
	if googleID == "" {
		users, err := s.database.GetAllUsers()
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get users: %v", err))
			return
		}
		if len(users) == 0 {
			writeError(w, http.StatusUnauthorized, "no authenticated users found")
			return
		}
		if len(users) > 1 {
			writeError(w, http.StatusBadRequest, "multiple users found, googleId parameter required")
			return
		}
		googleID = users[0].GoogleID
	}

	// Get user and Drive client
	user, driveClient, err := s.getUserAndClient(googleID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, fmt.Sprintf("failed to authenticate: %v", err))
		return
	}

	// Create import job in database
	options := map[string]interface{}{
		"recursive":         req.Recursive,
		"preserveHierarchy": req.PreserveHierarchy,
		"skipDuplicates":    req.SkipDuplicates,
	}

	job, err := s.database.CreateImportJob(user.ID, options)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create job: %v", err))
		return
	}

	// Create import job
	importJob := jobs.NewImportJob(
		job.ID,
		user.ID,
		req.FileIDs,
		req.FolderIDs,
		req.Recursive,
		s.tempDir,
		s.database,
		driveClient,
		jobs.ImportOptions{
			PreserveHierarchy: req.PreserveHierarchy,
			SkipDuplicates:    req.SkipDuplicates,
			MaxFileSize:       s.maxFileSize,
		},
	)

	// Enqueue job
	if err := s.jobQueue.Enqueue(importJob); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to enqueue job: %v", err))
		return
	}

	log.Printf("[INFO] Started import job: %s", job.ID)

	response := map[string]interface{}{
		"jobId":  job.ID,
		"status": job.Status,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// GetImportStatus returns the status of an import job
func (s *GDriveServer) GetImportStatus(w http.ResponseWriter, r *http.Request, jobID string) {
	log.Printf("[INFO] GET /api/google-drive/import/%s/status", jobID)

	job, err := s.database.GetImportJob(jobID)
	if err != nil {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}

	files, err := s.database.GetImportFilesByJob(jobID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get files: %v", err))
		return
	}

	// Calculate progress
	percentComplete := 0
	if job.TotalFiles > 0 {
		percentComplete = (job.CompletedFiles * 100) / job.TotalFiles
	}

	// Convert files to response format
	var fileList []map[string]interface{}
	for _, file := range files {
		fileList = append(fileList, map[string]interface{}{
			"driveId":  file.DriveFileID,
			"fileName": file.FileName,
			"status":   file.Status,
			"progress": file.Progress,
			"error":    file.ErrorMessage,
			"hash":     file.SHA256Hash,
			"cid":      file.IPFSCID,
		})
	}

	response := map[string]interface{}{
		"jobId":  job.ID,
		"status": job.Status,
		"progress": map[string]interface{}{
			"totalFiles":       job.TotalFiles,
			"completedFiles":   job.CompletedFiles,
			"failedFiles":      job.FailedFiles,
			"percentComplete":  percentComplete,
			"bytesTransferred": job.TransferredBytes,
			"totalBytes":       job.TotalBytes,
		},
		"files":       fileList,
		"startedAt":   job.StartedAt,
		"completedAt": job.CompletedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetImportHistory returns import job history for a user
func (s *GDriveServer) GetImportHistory(w http.ResponseWriter, r *http.Request) {
	log.Println("[INFO] GET /api/google-drive/import/history")

	googleID := r.URL.Query().Get("google_id")

	// Support single-user mode - if google_id not provided, use the only user
	if googleID == "" {
		users, err := s.database.GetAllUsers()
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get users: %v", err))
			return
		}
		if len(users) == 0 {
			writeError(w, http.StatusUnauthorized, "no authenticated users found")
			return
		}
		if len(users) > 1 {
			writeError(w, http.StatusBadRequest, "multiple users found, google_id parameter required")
			return
		}
		googleID = users[0].GoogleID
	}

	// Parse limit and offset
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit == 0 {
		limit = 20
	}

	user, err := s.database.GetUserByGoogleID(googleID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	jobs, err := s.database.GetImportJobsByUser(user.ID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get jobs: %v", err))
		return
	}

	var response []map[string]interface{}
	for _, job := range jobs {
		response = append(response, map[string]interface{}{
			"jobId":            job.ID,
			"status":           job.Status,
			"totalFiles":       job.TotalFiles,
			"completedFiles":   job.CompletedFiles,
			"failedFiles":      job.FailedFiles,
			"totalBytes":       job.TotalBytes,
			"transferredBytes": job.TransferredBytes,
			"startedAt":        job.StartedAt,
			"completedAt":      job.CompletedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper function to get user and create Drive client
func (s *GDriveServer) getUserAndClient(googleID string) (*db.User, *gdrive.Client, error) {
	user, err := s.database.GetUserByGoogleID(googleID)
	if err != nil {
		return nil, nil, err
	}

	accessToken, refreshToken, err := user.GetDecryptedTokens()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt tokens: %w", err)
	}

	token := &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Expiry:       user.TokenExpiry,
	}

	ctx := context.Background()
	client, err := gdrive.NewClient(ctx, token, s.oauthManager.GetConfig())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create drive client: %w", err)
	}

	return user, client, nil
}

// Helper function to refresh user token
func (s *GDriveServer) refreshUserToken(user *db.User) error {
	_, refreshToken, err := user.GetDecryptedTokens()
	if err != nil {
		return fmt.Errorf("failed to decrypt tokens: %w", err)
	}

	ctx := context.Background()
	newToken, err := s.oauthManager.RefreshToken(ctx, refreshToken)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	// Update user in database
	_, err = s.database.CreateOrUpdateUser(
		user.GoogleID,
		user.Email,
		newToken.AccessToken,
		newToken.RefreshToken,
		newToken.Expiry,
	)

	return err
}
