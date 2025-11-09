import { useState, useEffect } from 'react'

export default function GoogleDriveImport() {
  const [authStatus, setAuthStatus] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [showTokenPaste, setShowTokenPaste] = useState(false)
  const [tokenInput, setTokenInput] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    checkAuthStatus()
  }, [])

  const checkAuthStatus = async () => {
    try {
      const response = await fetch('/api/google-drive/auth-status')

      if (!response.ok) {
        if (response.status === 404) {
          setError('Google Drive import is not configured. Please set GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, and GOOGLE_REDIRECT_URL environment variables.')
          setLoading(false)
          return
        }
        throw new Error(`HTTP error! status: ${response.status}`)
      }

      const data = await response.json()
      setAuthStatus(data)
    } catch (err) {
      if (!error) {
        setError('Failed to check authentication status')
      }
    } finally {
      setLoading(false)
    }
  }

  const handleAuthorize = async () => {
    try {
      const response = await fetch('/api/google-drive/authorize', {
        method: 'POST'
      })
      const data = await response.json()
      if (data.authUrl) {
        window.location.href = data.authUrl
      }
    } catch (err) {
      setError('Failed to initiate authorization')
    }
  }

  const handleRevoke = async () => {
    try {
      await fetch('/api/google-drive/revoke', {
        method: 'DELETE'
      })
      setAuthStatus(null)
    } catch (err) {
      setError('Failed to revoke access')
    }
  }

  const handleTokenSubmit = async () => {
    setSubmitting(true)
    setError(null)

    try {
      // Parse the token JSON
      let token
      try {
        token = JSON.parse(tokenInput)
      } catch (e) {
        setError('Invalid token format. Please paste the entire JSON token from the OAuth broker.')
        setSubmitting(false)
        return
      }

      // Submit token to backend
      const response = await fetch('/api/google-drive/set-token', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(token)
      })

      if (!response.ok) {
        throw new Error('Failed to save token')
      }

      // Success - refresh auth status
      await checkAuthStatus()
      setShowTokenPaste(false)
      setTokenInput('')
    } catch (err) {
      setError('Failed to save token: ' + err.message)
    } finally {
      setSubmitting(false)
    }
  }

  const openOAuthBroker = () => {
    window.open('http://localhost:8888', '_blank')
    setShowTokenPaste(true)
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-gray-600">Loading...</div>
      </div>
    )
  }

  return (
    <div className="px-4 sm:px-0">
      <div className="bg-white shadow rounded-lg p-6">
        <h2 className="text-2xl font-bold text-gray-900 mb-4">Google Drive Import</h2>

        {error && (
          <div className="mb-4 bg-yellow-50 border border-yellow-200 rounded-lg p-4">
            <div className="flex">
              <div className="flex-shrink-0">
                <svg className="h-5 w-5 text-yellow-400" viewBox="0 0 20 20" fill="currentColor">
                  <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
                </svg>
              </div>
              <div className="ml-3">
                <h3 className="text-sm font-medium text-yellow-800">Configuration Required</h3>
                <div className="mt-2 text-sm text-yellow-700">
                  <p>{error}</p>
                  {error.includes('GOOGLE_CLIENT_ID') && (
                    <div className="mt-3 space-y-2">
                      <p className="font-semibold">Setup Instructions:</p>
                      <ol className="list-decimal list-inside space-y-1 ml-2">
                        <li>Create a Google Cloud Project at <a href="https://console.cloud.google.com" target="_blank" rel="noopener noreferrer" className="underline">console.cloud.google.com</a></li>
                        <li>Enable the Google Drive API</li>
                        <li>Create OAuth 2.0 credentials (Web application type)</li>
                        <li>Add authorized redirect URI: <code className="bg-yellow-100 px-1 rounded">http://localhost:9090/api/google-drive/callback</code></li>
                        <li>Copy credentials to <code className="bg-yellow-100 px-1 rounded">.env</code> file or docker-compose.yml environment variables</li>
                      </ol>
                      <p className="mt-2 text-xs">See <code className="bg-yellow-100 px-1 rounded">.env.example</code> for complete configuration details.</p>
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>
        )}

        {!authStatus?.authenticated ? (
          <div>
            <p className="text-gray-700 mb-6">
              Import files from your Google Drive into PinShare. Choose one of the authentication methods below.
            </p>

            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-6">
              <h3 className="font-semibold text-blue-900 mb-2">Option 1: OAuth Broker (Recommended)</h3>
              <p className="text-blue-800 text-sm mb-3">
                Use the PinShare OAuth Broker to get a token without setting up OAuth credentials.
              </p>
              <button
                onClick={openOAuthBroker}
                className="bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-4 rounded mb-3"
              >
                Get Token from OAuth Broker
              </button>

              {showTokenPaste && (
                <div className="mt-4 p-4 bg-white rounded border border-blue-300">
                  <h4 className="font-semibold text-blue-900 mb-2">Paste Your Token:</h4>
                  <p className="text-sm text-gray-600 mb-2">
                    Copy the entire JSON token from the OAuth Broker and paste it below.
                  </p>
                  <textarea
                    value={tokenInput}
                    onChange={(e) => setTokenInput(e.target.value)}
                    className="w-full h-32 p-2 border border-gray-300 rounded font-mono text-xs"
                    placeholder='{"access_token":"...","token_type":"Bearer",...}'
                  />
                  <div className="mt-2 flex gap-2">
                    <button
                      onClick={handleTokenSubmit}
                      disabled={submitting || !tokenInput}
                      className="bg-green-600 hover:bg-green-700 text-white font-medium py-2 px-4 rounded disabled:bg-gray-400"
                    >
                      {submitting ? 'Saving...' : 'Save Token'}
                    </button>
                    <button
                      onClick={() => {
                        setShowTokenPaste(false)
                        setTokenInput('')
                      }}
                      className="bg-gray-500 hover:bg-gray-600 text-white font-medium py-2 px-4 rounded"
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              )}
            </div>

            <div className="bg-gray-50 border border-gray-200 rounded-lg p-4 mb-6">
              <h3 className="font-semibold text-gray-900 mb-2">Option 2: Direct OAuth (Advanced)</h3>
              <p className="text-gray-700 text-sm mb-3">
                If you have configured your own OAuth credentials in the backend.
              </p>
              <div className="bg-gray-100 border border-gray-200 rounded-lg p-4 mb-3">
                <h4 className="font-semibold text-gray-800 mb-2">Required Permissions:</h4>
                <ul className="list-disc list-inside text-gray-700 space-y-1 text-sm">
                  <li>Read-only access to your Google Drive files</li>
                  <li>View file metadata and folder structure</li>
                </ul>
              </div>
              <button
                onClick={handleAuthorize}
                className="bg-gray-600 hover:bg-gray-700 text-white font-medium py-2 px-4 rounded"
              >
                Connect with Direct OAuth
              </button>
            </div>
          </div>
        ) : (
          <div>
            <div className="mb-6 bg-green-50 border border-green-200 rounded-lg p-4">
              <p className="text-green-800">
                ✓ Connected as {authStatus.email || 'Google Account'}
              </p>
            </div>

            <div className="space-y-4">
              <div>
                <h3 className="text-lg font-semibold mb-2">Import Options</h3>
                <p className="text-gray-600 mb-4">
                  Select files and folders from your Google Drive to import into PinShare.
                </p>
                <button
                  className="bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-4 rounded mr-2"
                  onClick={() => alert('File browser coming soon!')}
                >
                  Browse Files
                </button>
                <button
                  className="bg-gray-600 hover:bg-gray-700 text-white font-medium py-2 px-4 rounded"
                  onClick={() => alert('Import history coming soon!')}
                >
                  View Import History
                </button>
              </div>

              <div className="pt-4 mt-4 border-t border-gray-200">
                <button
                  onClick={handleRevoke}
                  className="text-red-600 hover:text-red-800 font-medium"
                >
                  Disconnect Google Drive
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
