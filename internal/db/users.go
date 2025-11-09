package db

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"
)

// User represents a PinShare user with Google Drive credentials
type User struct {
	ID                    int64
	GoogleID              string
	Email                 string
	EncryptedAccessToken  string
	EncryptedRefreshToken string
	TokenExpiry           time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

var (
	ErrUserNotFound = errors.New("user not found")
	// encryptionKey should be loaded from environment variable in production
	// For now, using a placeholder - TODO: Move to config
	encryptionKey = []byte("32-byte-long-encryption-key!") // Must be 32 bytes for AES-256
)

// SetEncryptionKey sets the encryption key for token storage
func SetEncryptionKey(key []byte) error {
	if len(key) != 32 {
		return fmt.Errorf("encryption key must be exactly 32 bytes, got %d", len(key))
	}
	encryptionKey = key
	return nil
}

// encrypt encrypts plaintext using AES-GCM
func encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts ciphertext using AES-GCM
func decrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, cipherData := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// CreateOrUpdateUser creates a new user or updates an existing one
func (db *DB) CreateOrUpdateUser(googleID, email, accessToken, refreshToken string, tokenExpiry time.Time) (*User, error) {
	encryptedAccess, err := encrypt(accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt access token: %w", err)
	}

	encryptedRefresh, err := encrypt(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt refresh token: %w", err)
	}

	now := time.Now()

	// Check if user exists
	var existingID int64
	err = db.QueryRow("SELECT id FROM users WHERE google_id = ?", googleID).Scan(&existingID)

	if err == sql.ErrNoRows {
		// Create new user
		result, err := db.Exec(`
			INSERT INTO users (google_id, email, encrypted_access_token, encrypted_refresh_token, token_expiry, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, googleID, email, encryptedAccess, encryptedRefresh, tokenExpiry, now, now)

		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("failed to get user ID: %w", err)
		}

		return &User{
			ID:                    id,
			GoogleID:              googleID,
			Email:                 email,
			EncryptedAccessToken:  encryptedAccess,
			EncryptedRefreshToken: encryptedRefresh,
			TokenExpiry:           tokenExpiry,
			CreatedAt:             now,
			UpdatedAt:             now,
		}, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	// Update existing user
	_, err = db.Exec(`
		UPDATE users
		SET email = ?, encrypted_access_token = ?, encrypted_refresh_token = ?, token_expiry = ?, updated_at = ?
		WHERE google_id = ?
	`, email, encryptedAccess, encryptedRefresh, tokenExpiry, now, googleID)

	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return db.GetUserByGoogleID(googleID)
}

// GetUserByGoogleID retrieves a user by their Google ID
func (db *DB) GetUserByGoogleID(googleID string) (*User, error) {
	user := &User{}
	err := db.QueryRow(`
		SELECT id, google_id, email, encrypted_access_token, encrypted_refresh_token, token_expiry, created_at, updated_at
		FROM users WHERE google_id = ?
	`, googleID).Scan(
		&user.ID,
		&user.GoogleID,
		&user.Email,
		&user.EncryptedAccessToken,
		&user.EncryptedRefreshToken,
		&user.TokenExpiry,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	} else if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetUserByID retrieves a user by their ID
func (db *DB) GetUserByID(id int64) (*User, error) {
	user := &User{}
	err := db.QueryRow(`
		SELECT id, google_id, email, encrypted_access_token, encrypted_refresh_token, token_expiry, created_at, updated_at
		FROM users WHERE id = ?
	`, id).Scan(
		&user.ID,
		&user.GoogleID,
		&user.Email,
		&user.EncryptedAccessToken,
		&user.EncryptedRefreshToken,
		&user.TokenExpiry,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	} else if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetDecryptedTokens returns decrypted access and refresh tokens
func (user *User) GetDecryptedTokens() (accessToken, refreshToken string, err error) {
	accessToken, err = decrypt(user.EncryptedAccessToken)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt access token: %w", err)
	}

	refreshToken, err = decrypt(user.EncryptedRefreshToken)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// DeleteUser removes a user and all associated data
func (db *DB) DeleteUser(googleID string) error {
	result, err := db.Exec("DELETE FROM users WHERE google_id = ?", googleID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}

// IsTokenExpired checks if the user's access token is expired
func (user *User) IsTokenExpired() bool {
	return time.Now().After(user.TokenExpiry)
}
