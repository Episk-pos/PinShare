package gdrive

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

// OAuthConfig holds OAuth 2.0 configuration
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// OAuthManager manages OAuth 2.0 flows
type OAuthManager struct {
	config      *oauth2.Config
	state       string
	codeVerifier string
}

// NewOAuthManager creates a new OAuth manager with PKCE support
func NewOAuthManager(cfg *OAuthConfig) *OAuthManager {
	config := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  cfg.RedirectURL,
		Scopes: []string{
			drive.DriveReadonlyScope, // Read-only access to Google Drive
		},
	}

	return &OAuthManager{
		config: config,
	}
}

// GenerateAuthURL generates the authorization URL with PKCE
func (m *OAuthManager) GenerateAuthURL() (string, error) {
	// Generate state for CSRF protection
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}
	m.state = base64.URLEncoding.EncodeToString(stateBytes)

	// Generate PKCE code verifier (43-128 characters)
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return "", fmt.Errorf("failed to generate code verifier: %w", err)
	}
	m.codeVerifier = base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(verifierBytes)

	// Generate code challenge from verifier
	codeChallenge := generateCodeChallenge(m.codeVerifier)

	// Build authorization URL with PKCE parameters
	authURL := m.config.AuthCodeURL(
		m.state,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.AccessTypeOffline, // Request refresh token
		oauth2.ApprovalForce,      // Force approval to get refresh token
	)

	return authURL, nil
}

// ExchangeCode exchanges the authorization code for tokens
func (m *OAuthManager) ExchangeCode(ctx context.Context, code, state string) (*oauth2.Token, error) {
	// Verify state matches
	if state != m.state {
		return nil, fmt.Errorf("invalid state parameter")
	}

	// Exchange code for token with PKCE verifier
	token, err := m.config.Exchange(
		ctx,
		code,
		oauth2.SetAuthURLParam("code_verifier", m.codeVerifier),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	return token, nil
}

// RefreshToken refreshes an expired access token
func (m *OAuthManager) RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	tokenSource := m.config.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	return newToken, nil
}

// GetTokenFromCode is a convenience method that combines state validation and code exchange
func (m *OAuthManager) GetTokenFromCode(ctx context.Context, code, state string) (*oauth2.Token, error) {
	return m.ExchangeCode(ctx, code, state)
}

// GetClient creates an HTTP client with the given token
func (m *OAuthManager) GetClient(ctx context.Context, token *oauth2.Token) *http.Client {
	return m.config.Client(ctx, token)
}

// GetConfig returns the OAuth config
func (m *OAuthManager) GetConfig() *oauth2.Config {
	return m.config
}

// RevokeToken revokes an access token
func (m *OAuthManager) RevokeToken(ctx context.Context, token string) error {
	revokeURL := fmt.Sprintf("https://oauth2.googleapis.com/revoke?token=%s", token)
	req, err := http.NewRequestWithContext(ctx, "POST", revokeURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create revoke request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token revocation failed with status: %d", resp.StatusCode)
	}

	return nil
}

// generateCodeChallenge generates a PKCE code challenge from the verifier
func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(hash[:])
}

// GetState returns the current state value (for verification)
func (m *OAuthManager) GetState() string {
	return m.state
}

// GetCodeVerifier returns the current code verifier (for debugging/testing)
func (m *OAuthManager) GetCodeVerifier() string {
	return m.codeVerifier
}
