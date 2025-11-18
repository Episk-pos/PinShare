package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"pinshare/internal/p2p"
	"pinshare/internal/store"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server implements the ServerInterface.
type Server struct{}

// NewServer creates a new server.
func NewServer() *Server {
	return &Server{}
}

// writeError is a helper for sending a JSON error response.
func writeError(w http.ResponseWriter, code int, message string) {
	errResp := Error{
		Code:    func() *int32 { c := int32(code); return &c }(),
		Message: message,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(errResp)
}

// ListAllFiles handles GET /files
func (s *Server) ListAllFiles(w http.ResponseWriter, r *http.Request) {
	allStoreFiles := store.GlobalStore.GetAllFiles()
	apiFiles := make([]BaseMetadata, len(allStoreFiles))
	// TODO: should we filter out the bansets if so we need to adjust our length. or use add instead of make

	for i, f := range allStoreFiles {
		// Create copies of the time values to take their addresses.
		lastUpdated := f.LastUpdated
		addedAt := f.AddedAt
		// Create copy of fileName to take its address
		var fileName *string
		if f.FileName != "" {
			fileName = &f.FileName
		}
		apiFiles[i] = BaseMetadata{
			FileSHA256:  f.FileSHA256,
			IpfsCID:     f.IPFSCID,
			FileName:    fileName,
			FileType:    f.FileType,
			LastUpdated: &lastUpdated,
			AddedAt:     &addedAt,
			// TODO: add remaining fields (tags, moderationVotes, banSet)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(apiFiles)
}

func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// GetUploadStatus returns the status of all file uploads
func (s *Server) GetUploadStatus(w http.ResponseWriter, r *http.Request) {
	statuses := store.StatusManager.GetAllStatuses()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(statuses)
}

// GetActiveUploadStatus returns only active uploads (in progress)
func (s *Server) GetActiveUploadStatus(w http.ResponseWriter, r *http.Request) {
	statuses := store.StatusManager.GetActiveStatuses()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(statuses)
}

// CancelUpload cancels an in-progress upload by filename
func (s *Server) CancelUpload(w http.ResponseWriter, r *http.Request) {
	// Only handle paths that end with /cancel
	path := r.URL.Path
	if !strings.HasSuffix(path, "/cancel") {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract filename from URL path: /upload-status/{fileName}/cancel
	// Remove the /upload-status/ prefix and /cancel suffix
	fileName := path[len("/upload-status/") : len(path)-len("/cancel")]

	if fileName == "" {
		writeError(w, http.StatusBadRequest, "fileName is required")
		return
	}

	err := store.StatusManager.CancelUpload(fileName)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "Upload cancelled successfully",
	})
}

// PinContent pins content to IPFS by CID
func (s *Server) PinContent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract CID from URL path: /ipfs/pin/{cid}
	path := r.URL.Path
	cid := path[len("/ipfs/pin/"):]
	if cid == "" {
		writeError(w, http.StatusBadRequest, "CID is required")
		return
	}

	// Call IPFS pin add command
	cmd := fmt.Sprintf("ipfs pin add %s", cid)
	output, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("pin failed: %v - %s", err, string(output)))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]string{
		"status": "pinned",
		"cid":    cid,
	}
	_ = json.NewEncoder(w).Encode(response)
}

// GetFileBySHA256 handles GET /files/{fileSHA256}
func (s *Server) GetFileBySHA256(w http.ResponseWriter, r *http.Request, fileSHA256 string) {
	file, found := store.GlobalStore.GetFile(fileSHA256)
	if !found {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}

	lastUpdated := file.LastUpdated
	addedAt := file.AddedAt
	// Create copy of fileName to take its address
	var fileName *string
	if file.FileName != "" {
		fileName = &file.FileName
	}
	apiFile := BaseMetadata{
		FileSHA256:  file.FileSHA256,
		IpfsCID:     file.IPFSCID,
		FileName:    fileName,
		FileType:    file.FileType,
		LastUpdated: &lastUpdated,
		AddedAt:     &addedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(apiFile)
}

// AddOrUpdateFile handles PUT /files/{fileSHA256}
func (s *Server) AddOrUpdateFile(w http.ResponseWriter, r *http.Request, fileSHA256 string) {
	var body WritableMetadata
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Check if the file already exists to determine status code (200 vs 201).
	_, exists := store.GlobalStore.GetFile(fileSHA256)

	meta := store.BaseMetadata{
		FileSHA256: fileSHA256,
		IPFSCID:    body.IpfsCID,
		FileType:   body.FileType,
	}

	if err := store.GlobalStore.AddFile(meta); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add or update file")
		return
	}

	// Retrieve the updated metadata to return it in the response.
	updatedFile, _ := store.GlobalStore.GetFile(fileSHA256)
	lastUpdated := updatedFile.LastUpdated
	addedAt := updatedFile.AddedAt
	apiFile := BaseMetadata{
		FileSHA256:  updatedFile.FileSHA256,
		IpfsCID:     updatedFile.IPFSCID,
		FileType:    updatedFile.FileType,
		LastUpdated: &lastUpdated,
		AddedAt:     &addedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	_ = json.NewEncoder(w).Encode(apiFile)
}

func (s *Server) AddTag(w http.ResponseWriter, r *http.Request, fileSHA256 string) {
	// TODO:
}

func (s *Server) RemoveTag(w http.ResponseWriter, r *http.Request, fileSHA256 string, tagName string) {
	// TODO:
}

func (s *Server) VoteForRemoval(w http.ResponseWriter, r *http.Request, fileSHA256 string) {
	// TODO:
}

func (s *Server) ListP2PPeers(w http.ResponseWriter, r *http.Request) {
	node := GetNode()
	if node == nil {
		writeError(w, http.StatusInternalServerError, "P2P node not initialized")
		return
	}

	peers := (*node).Network().Peers()
	peerIDs := make([]string, len(peers))
	for i, p := range peers {
		peerIDs[i] = p.String()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(peerIDs)
}

func (s *Server) ConnectToPeer(w http.ResponseWriter, r *http.Request) {
	var body ConnectToPeerJSONBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	node := GetNode()
	if node == nil {
		writeError(w, http.StatusInternalServerError, "P2P node not initialized")
		return
	}

	// Parse the multiaddress from the request body.
	peerMA, err := multiaddr.NewMultiaddr(body.Multiaddr)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid multiaddress: %v", err))
		return
	}

	// Extract peer info from the multiaddress.
	peerInfo, err := peer.AddrInfoFromP2pAddr(peerMA)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("could not get peer info from multiaddress: %v", err))
		return
	}

	// Run connection attempt in the background to not block the API response.
	go func() {
		log.Printf("[API-INFO] Attempting to connect to peer %s", peerInfo.ID.String())
		if err := (*node).Connect(context.Background(), *peerInfo); err != nil {
			log.Printf("[API-WARN] Failed to connect to peer %s: %v", peerInfo.ID.String(), err)
		} else {
			log.Printf("[API-INFO] Successfully connected to peer %s", peerInfo.ID.String())
		}
	}()

	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte("Connection attempt initiated"))
}

func (s *Server) SendDirectMessage(w http.ResponseWriter, r *http.Request, peerID string) {
	var body SendDirectMessageJSONBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	peerIDParsed, err := p2p.ParsePeerID(peerID)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid peer ID: %v", err))
		return
	}

	err = p2p.DirectMessagePeer(context.Background(), *GetNode(), peerIDParsed, []byte(body.Message))
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to send direct message: %v", err))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Message sent successfully"))
}

func (s *Server) GetP2PStatus(w http.ResponseWriter, r *http.Request) {
	node := GetNode()
	if node == nil {
		writeError(w, http.StatusInternalServerError, "P2P node not initialized")
		return
	}

	addrs := make([]string, 0, len((*node).Addrs()))
	for _, addr := range (*node).Addrs() {
		addrs = append(addrs, addr.String())
	}

	status := P2PStatus{
		Id:             (*node).ID().String(),
		Addresses:      addrs,
		ConnectedPeers: len((*node).Network().Peers()),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(status)
}

// TODO: ListTopicPeers is currently disabled because p2p.GetPubSubManager() doesn't exist
// func (s *Server) ListTopicPeers(w http.ResponseWriter, r *http.Request) {
// 	manager := p2p.GetPubSubManager()
// 	if manager == nil {
// 		writeError(w, http.StatusInternalServerError, "PubSub manager not initialized")
// 		return
// 	}
//
// 	topicPeers := manager.ListPeers()
// 	peerIDs := make([]string, len(topicPeers))
// 	for i, p := range topicPeers {
// 		peerIDs[i] = p.String()
// 	}
//
// 	config, _ := config.LoadConfig()
// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusOK)
// 	_ = json.NewEncoder(w).Encode(map[string]interface{}{
// 		"topic":     config.MetadataTopicID,
// 		"peerCount": len(topicPeers),
// 		"peers":     peerIDs,
// 	})
// }

var p2pNodeInstance *host.Host

// SetP2PManager allows main to set the global PubSubManager instance
func SetNode(node *host.Host) {
	p2pNodeInstance = node
}

func GetNode() *host.Host {
	return p2pNodeInstance
}

// corsMiddleware adds CORS headers to allow cross-origin requests from the UI
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow requests from the UI dev server and production
		origin := r.Header.Get("Origin")

		// Read allowed origins from environment variable (comma-separated)
		// Defaults to localhost dev servers if not set
		allowedOriginsEnv := os.Getenv("PS_ALLOWED_ORIGINS")
		var allowedOrigins []string
		if allowedOriginsEnv != "" {
			// Split by comma and trim whitespace
			for _, o := range strings.Split(allowedOriginsEnv, ",") {
				allowedOrigins = append(allowedOrigins, strings.TrimSpace(o))
			}
		} else {
			// Default to localhost for development
			allowedOrigins = []string{
				"http://localhost:5174",
				"http://localhost:5173",
			}
		}

		log.Printf("[CORS DEBUG] Origin: %q, AllowedOrigins: %v", origin, allowedOrigins)

		// Check if origin is in allowed list
		for _, allowed := range allowedOrigins {
			log.Printf("[CORS DEBUG] Comparing origin %q with allowed %q", origin, allowed)
			if origin == allowed {
				log.Printf("[CORS DEBUG] MATCH! Setting Access-Control-Allow-Origin to %q", origin)
				w.Header().Set("Access-Control-Allow-Origin", origin)
				break
			}
		}

		// If no origin header (same-origin request), allow it
		if origin == "" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "3600")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func Start(ctx context.Context, node host.Host, gdriveServer *GDriveServer) {
	SetNode(&node)
	server := NewServer()

	// get an `http.Handler` that we can use
	apiHandler := Handler(server)

	// Create a new ServeMux to combine the API handler and metrics handler
	mux := http.NewServeMux()
	mux.Handle("/api/", apiHandler)
	mux.Handle("/metrics", promhttp.Handler())
	// TODO: Disabled until p2p.GetPubSubManager() is implemented
	// mux.HandleFunc("/api/p2p/topic-peers", server.ListTopicPeers)
	mux.HandleFunc("/health", server.Health)
	mux.HandleFunc("/api/upload-status", server.GetUploadStatus)
	mux.HandleFunc("/api/upload-status/active", server.GetActiveUploadStatus)
	mux.HandleFunc("/api/upload-status/", server.CancelUpload) // Handles /api/upload-status/{fileName}/cancel
	mux.HandleFunc("/api/ipfs/pin/", server.PinContent)
	mux.Handle("/ui/", http.StripPrefix("/ui", http.FileServer(http.Dir("../../pinshare-ui/dist"))))

	// Add Google Drive import routes if server is configured
	if gdriveServer != nil {
		log.Println("[INFO] Registering Google Drive import API routes")

		// OAuth routes
		mux.HandleFunc("/api/google-drive/authorize", gdriveServer.AuthorizeRequest)
		mux.HandleFunc("/api/google-drive/callback", gdriveServer.CallbackRequest)
		mux.HandleFunc("/api/google-drive/set-token", gdriveServer.SetToken)
		mux.HandleFunc("/api/google-drive/auth-status", gdriveServer.GetAuthStatus)
		mux.HandleFunc("/api/google-drive/revoke", gdriveServer.RevokeAccess)

		// Drive operations routes
		mux.HandleFunc("/api/google-drive/folders", gdriveServer.ListFolders)
		mux.HandleFunc("/api/google-drive/preview-import", gdriveServer.PreviewImport)
		mux.HandleFunc("/api/google-drive/import", gdriveServer.StartImport)
		mux.HandleFunc("/api/google-drive/import/history", gdriveServer.GetImportHistory)

		// Job status route (pattern matching for job ID)
		mux.HandleFunc("/api/google-drive/import/", func(w http.ResponseWriter, r *http.Request) {
			// Extract job ID from path
			path := r.URL.Path
			if len(path) > len("/api/google-drive/import/") {
				jobID := path[len("/api/google-drive/import/"):]
				// Check if it ends with /status
				if len(jobID) > 7 && jobID[len(jobID)-7:] == "/status" {
					jobID = jobID[:len(jobID)-7]
					gdriveServer.GetImportStatus(w, r, jobID)
					return
				}
			}
			writeError(w, http.StatusNotFound, "endpoint not found")
		})
	}

	portStr := os.Getenv("API_PORT")
	var port int
	if portStr != "" {
		var err error
		port, err = strconv.Atoi(portStr)
		if err != nil {
			log.Fatalf("[ERROR] Invalid API_PORT env: %v", err)
		}
	} else {
		port = 9090
	}

	addr := fmt.Sprintf("0.0.0.0:%d", port)
	s := &http.Server{
		Handler: corsMiddleware(mux), // Wrap with CORS middleware
		Addr:    addr,
	}

	log.Printf("[INFO] Starting API server on %s", addr)
	// And we serve HTTP until the world ends.
	log.Fatal(s.ListenAndServe())
}
