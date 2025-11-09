package config

import (
	"os"
	"strconv"
	"time"
)

// Default values for configuration
const (
	defaultUploadFolder      = "./upload"
	defaultCacheFolder       = "./cache"
	defaultRejectFolder      = "./rejected"
	defaultMetaDataFile      = "metadata.json"
	defaultIdentityKeyFile   = "identity.key"
	defaultLibp2pPort        = 50001
	defaultWatchInterval     = 2 * time.Minute
	defaultOrgName           = "Cypherpunk"
	defaultGroupName         = "TestLab"
	defaultMetadataTopicID   = "/metadata-sync/1.0.0"
	defaultFilteringTopicID  = "/filtering-sync/1.0.0"
	defaultDatabaseFile      = "pinshare.db"
	defaultTempDownloadDir   = "./tmp/gdrive"
	defaultMaxConcurrentJobs = 3
	defaultMaxFileSize       = 1024 * 1024 * 1024 // 1GB in bytes
)

// Default values for Feature Flags
const (
	defaultFF                        = false // ENVVAR NAME
	defaultFFArchiveNode             = false // PS_FF_ARCHIVE_NODE
	defaultFFCache                   = false // PS_FF_CACHE
	defaultFFMoveUpload              = false // PS_FF_MOVE_UPLOAD
	defaultFFSendFileVT              = false // PS_FF_SENDFILE_VT
	defaultFFSkipVT                  = false // PS_FF_SKIP_VT
	defaultFFIgnoreUploadsInMetadata = true  // PS_FF_IGNORE_UPLOADS_IN_METADATA
)

// AppConfig holds all configuration for the application.
type AppConfig struct {
	SecurityCapability        int
	UploadFolder              string
	CacheFolder               string
	RejectFolder              string
	MetaDataFile              string
	IdentityKeyFile           string
	Libp2pPort                int
	WatchInterval             time.Duration
	OrgName                   string
	GroupName                 string
	MetadataTopicID           string
	FilteringTopicID          string
	FFArchiveNode             bool
	FFCache                   bool
	FFMoveUpload              bool
	FFSendFileVT              bool
	FFSkipVT                  bool
	FFIgnoreUploadsInMetadata bool

	// Google Drive import configuration
	DatabaseFile          string
	TempDownloadDir       string
	GoogleClientID        string
	GoogleClientSecret    string
	GoogleRedirectURL     string
	MaxConcurrentJobs     int
	MaxFileSize           int64
	EncryptionKey         string // 32-byte encryption key for token storage
}

// LoadConfig loads configuration from environment variables, falling back to defaults.
// It returns a populated AppConfig struct. Errors are returned if environment
// variables are set but have invalid formats.
func LoadConfig() (*AppConfig, error) {
	conf := &AppConfig{
		SecurityCapability:        0,
		UploadFolder:              defaultUploadFolder,
		CacheFolder:               defaultCacheFolder,
		RejectFolder:              defaultRejectFolder,
		MetaDataFile:              defaultMetaDataFile,
		IdentityKeyFile:           defaultIdentityKeyFile,
		Libp2pPort:                defaultLibp2pPort,
		WatchInterval:             defaultWatchInterval,
		OrgName:                   defaultOrgName,
		GroupName:                 defaultGroupName,
		MetadataTopicID:           defaultMetadataTopicID,
		FilteringTopicID:          defaultFilteringTopicID,
		FFArchiveNode:             defaultFFArchiveNode,
		FFCache:                   defaultFFCache,
		FFMoveUpload:              defaultFFMoveUpload,
		FFSendFileVT:              defaultFFSendFileVT,
		FFSkipVT:                  defaultFFSkipVT,
		FFIgnoreUploadsInMetadata: defaultFFIgnoreUploadsInMetadata,
		DatabaseFile:              defaultDatabaseFile,
		TempDownloadDir:           defaultTempDownloadDir,
		MaxConcurrentJobs:         defaultMaxConcurrentJobs,
		MaxFileSize:               defaultMaxFileSize,
	}

	// Helper function to parse boolean environment variables
	parseBoolEnv := func(key string, target *bool) error {
		if val, ok := os.LookupEnv(key); ok {
			b, err := strconv.ParseBool(val)
			if err != nil {
				return err
			}
			*target = b
		}
		return nil
	}

	// Helper function to parse string environment variables
	parseStringEnv := func(key string, target *string) error {
		if val, ok := os.LookupEnv(key); ok {
			*target = val
		}
		return nil
	}

	// Helper function to parse int environment variables
	parseIntEnv := func(key string, target *int) error {
		if val, ok := os.LookupEnv(key); ok {
			b, err := strconv.ParseInt(val, 0, 64)
			if err != nil {
				return err
			}
			*target = int(b)
		}
		return nil
	}

	//TODO: Loadin the Org/Group names
	if err := parseStringEnv("PS_ORGNAME", &conf.OrgName); err != nil {
		return nil, err
	}
	if err := parseStringEnv("PS_GROUPNAME", &conf.GroupName); err != nil {
		return nil, err
	}
	conf.MetadataTopicID = "/" + conf.OrgName + "/" + conf.GroupName + conf.MetadataTopicID
	conf.FilteringTopicID = "/" + conf.OrgName + "/" + conf.GroupName + conf.FilteringTopicID

	if err := parseIntEnv("PS_LIBP2P_PORT", &conf.Libp2pPort); err != nil {
		return nil, err
	}
	if err := parseBoolEnv("PS_FF_MOVE_UPLOAD", &conf.FFMoveUpload); err != nil {
		return nil, err
	}
	if err := parseBoolEnv("PS_FF_SENDFILE_VT", &conf.FFSendFileVT); err != nil {
		return nil, err
	}
	if err := parseBoolEnv("PS_FF_SKIP_VT", &conf.FFSkipVT); err != nil {
		return nil, err
	}
	if err := parseBoolEnv("PS_FF_IGNORE_UPLOADS_IN_METADATA", &conf.FFIgnoreUploadsInMetadata); err != nil {
		return nil, err
	}

	// Google Drive configuration
	if err := parseStringEnv("PS_DATABASE_FILE", &conf.DatabaseFile); err != nil {
		return nil, err
	}
	if err := parseStringEnv("PS_TEMP_DOWNLOAD_DIR", &conf.TempDownloadDir); err != nil {
		return nil, err
	}
	if err := parseStringEnv("GOOGLE_CLIENT_ID", &conf.GoogleClientID); err != nil {
		return nil, err
	}
	if err := parseStringEnv("GOOGLE_CLIENT_SECRET", &conf.GoogleClientSecret); err != nil {
		return nil, err
	}
	if err := parseStringEnv("GOOGLE_REDIRECT_URL", &conf.GoogleRedirectURL); err != nil {
		return nil, err
	}
	if err := parseIntEnv("PS_MAX_CONCURRENT_JOBS", &conf.MaxConcurrentJobs); err != nil {
		return nil, err
	}

	// Parse max file size
	var maxFileSizeInt int
	if err := parseIntEnv("PS_MAX_FILE_SIZE_MB", &maxFileSizeInt); err != nil {
		return nil, err
	}
	if maxFileSizeInt > 0 {
		conf.MaxFileSize = int64(maxFileSizeInt) * 1024 * 1024 // Convert MB to bytes
	}

	// Encryption key (must be 32 bytes for AES-256)
	if err := parseStringEnv("PS_ENCRYPTION_KEY", &conf.EncryptionKey); err != nil {
		return nil, err
	}

	return conf, nil
}
