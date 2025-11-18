package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

type OAuthBroker struct {
	config   *oauth2.Config
	sessions map[string]*Session
	mu       sync.RWMutex
}

type Session struct {
	State     string
	Token     *oauth2.Token
	CreatedAt time.Time
	ExpiresAt time.Time
}

var broker *OAuthBroker

func main() {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURL := os.Getenv("REDIRECT_URL")

	if clientID == "" || clientSecret == "" {
		log.Fatal("GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET must be set")
	}

	if redirectURL == "" {
		redirectURL = "http://localhost:8888/callback"
	}

	broker = &OAuthBroker{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes: []string{
				drive.DriveReadonlyScope,
				"openid",
				"profile",
				"email",
			},
			Endpoint: google.Endpoint,
		},
		sessions: make(map[string]*Session),
	}

	// Clean up expired sessions every hour
	go broker.cleanupExpiredSessions()

	http.HandleFunc("/", handleHome)
	http.HandleFunc("/authorize", broker.handleAuthorize)
	http.HandleFunc("/callback", broker.handleCallback)
	http.HandleFunc("/health", handleHealth)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8888"
	}

	log.Printf("OAuth Broker starting on :%s", port)
	log.Printf("Redirect URL: %s", redirectURL)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>PinShare OAuth Broker</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            max-width: 800px;
            margin: 50px auto;
            padding: 20px;
            background: #f5f5f5;
        }
        .container {
            background: white;
            padding: 40px;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        h1 { color: #333; }
        .info {
            background: #e3f2fd;
            padding: 15px;
            border-radius: 5px;
            margin: 20px 0;
        }
        .btn {
            background: #4285f4;
            color: white;
            padding: 12px 24px;
            border: none;
            border-radius: 4px;
            font-size: 16px;
            cursor: pointer;
            text-decoration: none;
            display: inline-block;
        }
        .btn:hover { background: #3367d6; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔐 PinShare OAuth Broker</h1>
        <p>This service helps PinShare instances connect to Google Drive without requiring each user to create their own OAuth credentials.</p>

        <div class="info">
            <strong>How it works:</strong>
            <ol>
                <li>Click "Authorize with Google" below</li>
                <li>Sign in with your Google account</li>
                <li>Copy the token provided</li>
                <li>Paste it into your PinShare instance</li>
            </ol>
        </div>

        <a href="/authorize" class="btn">Authorize with Google Drive</a>
    </div>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, tmpl)
}

func (b *OAuthBroker) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	// Generate random state
	state := generateState()

	// Create session
	session := &Session{
		State:     state,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(10 * time.Minute), // Session expires in 10 minutes
	}

	b.mu.Lock()
	b.sessions[state] = session
	b.mu.Unlock()

	// Redirect to Google OAuth
	url := b.config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (b *OAuthBroker) handleCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	errorParam := r.URL.Query().Get("error")

	if errorParam != "" {
		renderError(w, "Authorization failed: "+errorParam)
		return
	}

	// Verify state
	b.mu.RLock()
	session, exists := b.sessions[state]
	b.mu.RUnlock()

	if !exists {
		renderError(w, "Invalid or expired session")
		return
	}

	if time.Now().After(session.ExpiresAt) {
		renderError(w, "Session expired. Please try again.")
		return
	}

	// Exchange code for token
	ctx := context.Background()
	token, err := b.config.Exchange(ctx, code)
	if err != nil {
		log.Printf("Failed to exchange code: %v", err)
		renderError(w, "Failed to exchange authorization code")
		return
	}

	// Store token in session
	session.Token = token

	// Render success page with token
	renderSuccess(w, token)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (b *OAuthBroker) cleanupExpiredSessions() {
	ticker := time.NewTicker(1 * time.Hour)
	for range ticker.C {
		b.mu.Lock()
		for state, session := range b.sessions {
			if time.Now().After(session.ExpiresAt) {
				delete(b.sessions, state)
			}
		}
		b.mu.Unlock()
	}
}

func generateState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func renderError(w http.ResponseWriter, message string) {
	tmpl := template.Must(template.New("error").Parse(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Error - PinShare OAuth</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            max-width: 600px;
            margin: 50px auto;
            padding: 20px;
        }
        .error {
            background: #ffebee;
            color: #c62828;
            padding: 20px;
            border-radius: 8px;
            border-left: 4px solid #c62828;
        }
        .btn {
            background: #4285f4;
            color: white;
            padding: 10px 20px;
            border: none;
            border-radius: 4px;
            text-decoration: none;
            display: inline-block;
            margin-top: 20px;
        }
    </style>
</head>
<body>
    <div class="error">
        <h2>❌ Error</h2>
        <p>{{.}}</p>
    </div>
    <a href="/" class="btn">Try Again</a>
</body>
</html>`))
	w.WriteHeader(http.StatusBadRequest)
	tmpl.Execute(w, message)
}

func renderSuccess(w http.ResponseWriter, token *oauth2.Token) {
	tokenJSON, _ := json.MarshalIndent(token, "", "  ")

	tmpl := template.Must(template.New("success").Parse(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Success - PinShare OAuth</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            max-width: 800px;
            margin: 50px auto;
            padding: 20px;
            background: #f5f5f5;
        }
        .container {
            background: white;
            padding: 40px;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        .success {
            background: #e8f5e9;
            color: #2e7d32;
            padding: 20px;
            border-radius: 8px;
            border-left: 4px solid #4caf50;
            margin-bottom: 30px;
        }
        .token-box {
            background: #f5f5f5;
            padding: 20px;
            border-radius: 8px;
            border: 1px solid #ddd;
            font-family: monospace;
            font-size: 12px;
            white-space: pre-wrap;
            word-break: break-all;
            margin: 20px 0;
            max-height: 300px;
            overflow-y: auto;
        }
        .btn {
            background: #4285f4;
            color: white;
            padding: 12px 24px;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 16px;
        }
        .btn:hover { background: #3367d6; }
        .instructions {
            background: #fff3e0;
            padding: 15px;
            border-radius: 5px;
            margin: 20px 0;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="success">
            <h2>✅ Authorization Successful!</h2>
            <p>Your Google Drive access has been authorized.</p>
        </div>

        <div class="instructions">
            <strong>Next Steps:</strong>
            <ol>
                <li>Click the "Copy Token" button below</li>
                <li>Go to your PinShare instance</li>
                <li>Navigate to Settings → Google Drive Import</li>
                <li>Paste the token and save</li>
            </ol>
        </div>

        <h3>Your Access Token:</h3>
        <div class="token-box" id="token">{{.}}</div>

        <button class="btn" onclick="copyToken()">📋 Copy Token</button>
        <span id="copied" style="color: green; margin-left: 10px; display: none;">✓ Copied!</span>

        <script>
            const tokenData = {{.}};

            // Try postMessage if opened as popup
            if (window.opener && !window.opener.closed) {
                tryPostMessage();
            } else {
                showCopyPasteUI();
            }

            function tryPostMessage() {
                const message = {
                    type: 'PINSHARE_OAUTH_TOKEN',
                    token: tokenData,
                    source: 'pinshare-oauth-broker'
                };

                // Send to opener (frontend) - receiver will validate origin
                window.opener.postMessage(message, '*');

                // Wait for acknowledgment
                let acknowledged = false;
                const handleAck = (event) => {
                    if (event.data && event.data.type === 'PINSHARE_TOKEN_RECEIVED') {
                        acknowledged = true;
                        window.removeEventListener('message', handleAck);

                        // Show success and close
                        document.querySelector('.container').innerHTML = `
                            <div class="success">
                                <h2>✓ Token Sent Successfully!</h2>
                                <p>Closing window...</p>
                            </div>
                        `;
                        setTimeout(() => window.close(), 1000);
                    }
                };

                window.addEventListener('message', handleAck);

                // Timeout after 5 seconds - fallback to copy/paste
                setTimeout(() => {
                    if (!acknowledged) {
                        window.removeEventListener('message', handleAck);
                        showCopyPasteUI();
                    }
                }, 5000);
            }

            function showCopyPasteUI() {
                // UI is already visible, just ensure it's shown
                document.querySelector('.instructions').style.display = 'block';
                document.querySelector('.token-box').style.display = 'block';
                document.querySelector('.btn').style.display = 'inline-block';
            }

            function copyToken() {
                const tokenText = document.getElementById('token').textContent;
                navigator.clipboard.writeText(tokenText).then(() => {
                    document.getElementById('copied').style.display = 'inline';
                    setTimeout(() => {
                        document.getElementById('copied').style.display = 'none';
                    }, 3000);
                });
            }
        </script>

        <p style="margin-top: 30px; color: #666; font-size: 14px;">
            <strong>Note:</strong> This token expires on {{.Expiry.Format "2006-01-02 15:04:05 MST"}}.
            Your PinShare instance will automatically refresh it as needed.
        </p>
    </div>
</body>
</html>`))
	tmpl.Execute(w, string(tokenJSON))
}
