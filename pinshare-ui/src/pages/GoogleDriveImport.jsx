import { useState, useEffect } from 'react'

const OAUTH_BASE = import.meta.env.VITE_OAUTH_BASE || 'http://localhost:8888'
const API_BASE = import.meta.env.VITE_API_BASE || 'http://localhost:9090'

export default function GoogleDriveImport() {
  const [authStatus, setAuthStatus] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [showTokenPaste, setShowTokenPaste] = useState(false)
  const [tokenInput, setTokenInput] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [oauthStatus, setOauthStatus] = useState(null) // null, 'waiting', 'popup-blocked', 'timeout'
  const [showManualHint, setShowManualHint] = useState(false)

  // File browser state
  const [showBrowser, setShowBrowser] = useState(false)
  const [currentPath, setCurrentPath] = useState('')
  const [files, setFiles] = useState([])
  const [selectedFiles, setSelectedFiles] = useState(new Set())
  const [loadingFiles, setLoadingFiles] = useState(false)
  const [breadcrumbs, setBreadcrumbs] = useState([{ id: '', name: 'My Drive' }])

  // Import state
  const [showPreview, setShowPreview] = useState(false)
  const [preview, setPreview] = useState(null)
  const [importing, setImporting] = useState(false)

  useEffect(() => {
    checkAuthStatus()
  }, [])

  const checkAuthStatus = async () => {
    try {
      const response = await fetch(`${API_BASE}/api/google-drive/auth-status`)

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
      const response = await fetch(`${API_BASE}/api/google-drive/authorize`, {
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
      await fetch(`${API_BASE}/api/google-drive/revoke`, {
        method: 'DELETE'
      })
      setAuthStatus(null)
      setShowBrowser(false)
    } catch (err) {
      setError('Failed to revoke access')
    }
  }

  const handleTokenSubmit = async () => {
    setSubmitting(true)
    setError(null)

    try {
      let token
      try {
        token = JSON.parse(tokenInput)
      } catch (e) {
        setError('Invalid token format. Please paste the entire JSON token from the OAuth broker.')
        setSubmitting(false)
        return
      }

      const response = await fetch(`${API_BASE}/api/google-drive/set-token`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(token)
      })

      if (!response.ok) {
        throw new Error('Failed to save token')
      }

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
    const brokerUrl = `${OAUTH_BASE}/authorize`

    // Reset state
    setError(null)
    setShowTokenPaste(false)
    setShowManualHint(false)

    // Open in popup
    const popup = window.open(
      brokerUrl,
      'pinshare-oauth',
      'width=600,height=700,scrollbars=yes'
    )

    if (!popup || popup.closed) {
      // Popup blocked - show options
      setOauthStatus('popup-blocked')
      return
    }

    // Popup opened successfully - show waiting status
    setOauthStatus('waiting')

    // Set up postMessage listener
    let messageHandlerActive = true
    const handleMessage = async (event) => {
      if (!messageHandlerActive) return

      // Validate origin
      const allowedOrigin = new URL(OAUTH_BASE).origin
      if (event.origin !== allowedOrigin) {
        console.warn('Received message from unexpected origin:', event.origin)
        return
      }

      // Validate message structure
      if (event.data?.type !== 'PINSHARE_OAUTH_TOKEN' ||
          event.data?.source !== 'pinshare-oauth-broker') {
        return
      }

      // Send acknowledgment
      try {
        popup.postMessage({ type: 'PINSHARE_TOKEN_RECEIVED' }, allowedOrigin)
      } catch (err) {
        console.error('Failed to send acknowledgment:', err)
      }

      // Clean up listener
      messageHandlerActive = false
      window.removeEventListener('message', handleMessage)
      clearTimeout(timeoutId)
      clearTimeout(hintTimeoutId)

      // Auto-submit token
      try {
        const response = await fetch(`${API_BASE}/api/google-drive/set-token`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(event.data.token)
        })

        if (response.ok) {
          setError(null)
          setOauthStatus(null)
          setShowManualHint(false)
          await checkAuthStatus()
          try {
            popup.close()
          } catch (err) {
            // Ignore if popup already closed
          }
        } else {
          throw new Error('Token validation failed')
        }
      } catch (err) {
        setError('Error storing token: ' + err.message)
        setOauthStatus(null)
      }
    }

    window.addEventListener('message', handleMessage)

    // After 10 seconds, show hint for manual copy/paste
    const hintTimeoutId = setTimeout(() => {
      if (messageHandlerActive) {
        setShowManualHint(true)
      }
    }, 10000)

    // Cleanup after 30 seconds
    const timeoutId = setTimeout(() => {
      if (messageHandlerActive) {
        messageHandlerActive = false
        window.removeEventListener('message', handleMessage)
        setOauthStatus('timeout')
      }
    }, 30000)
  }

  const openOAuthBrokerInNewTab = () => {
    const brokerUrl = `${OAUTH_BASE}/authorize`
    window.open(brokerUrl, '_blank')
    setShowTokenPaste(true)
    setOauthStatus(null)
  }

  const loadFiles = async (folderId = '') => {
    setLoadingFiles(true)
    try {
      const params = new URLSearchParams()
      if (folderId) params.append('path', folderId)

      const response = await fetch(`${API_BASE}/api/google-drive/folders?${params}`)
      if (!response.ok) throw new Error('Failed to load files')

      const data = await response.json()

      // Sort: directories first (alphabetically), then files (alphabetically)
      const sortedData = data.sort((a, b) => {
        // Folders come before files
        if (a.isFolder && !b.isFolder) return -1
        if (!a.isFolder && b.isFolder) return 1
        // Within the same type, sort alphabetically by name (case-insensitive)
        return a.name.localeCompare(b.name, undefined, { sensitivity: 'base' })
      })

      setFiles(sortedData)
      setCurrentPath(folderId)
    } catch (err) {
      setError('Failed to load files: ' + err.message)
    } finally {
      setLoadingFiles(false)
    }
  }

  const openBrowser = () => {
    setShowBrowser(true)
    setBreadcrumbs([{ id: '', name: 'My Drive' }])
    loadFiles('')
  }

  const navigateToFolder = (file) => {
    const newBreadcrumbs = [...breadcrumbs, { id: file.id, name: file.name }]
    setBreadcrumbs(newBreadcrumbs)
    loadFiles(file.id)
  }

  const navigateToBreadcrumb = (index) => {
    const newBreadcrumbs = breadcrumbs.slice(0, index + 1)
    setBreadcrumbs(newBreadcrumbs)
    const folderId = newBreadcrumbs[newBreadcrumbs.length - 1].id
    loadFiles(folderId)
  }

  const toggleFileSelection = (fileId) => {
    const newSelection = new Set(selectedFiles)
    if (newSelection.has(fileId)) {
      newSelection.delete(fileId)
    } else {
      newSelection.add(fileId)
    }
    setSelectedFiles(newSelection)
  }

  const selectAll = () => {
    const allFileIds = files.map(f => f.id)
    setSelectedFiles(new Set(allFileIds))
  }

  const clearSelection = () => {
    setSelectedFiles(new Set())
  }

  const previewImport = async () => {
    try {
      const selectedFilesList = Array.from(selectedFiles)
      const fileIds = files.filter(f => selectedFilesList.includes(f.id) && !f.isFolder).map(f => f.id)
      const folderIds = files.filter(f => selectedFilesList.includes(f.id) && f.isFolder).map(f => f.id)

      const response = await fetch(`${API_BASE}/api/google-drive/preview-import`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          fileIds,
          folderIds,
          recursive: true
        })
      })

      if (!response.ok) throw new Error('Failed to preview import')

      const data = await response.json()
      setPreview(data)
      setShowPreview(true)
    } catch (err) {
      setError('Failed to preview import: ' + err.message)
    }
  }

  const startImport = async () => {
    setImporting(true)
    try {
      const selectedFilesList = Array.from(selectedFiles)
      const fileIds = files.filter(f => selectedFilesList.includes(f.id) && !f.isFolder).map(f => f.id)
      const folderIds = files.filter(f => selectedFilesList.includes(f.id) && f.isFolder).map(f => f.id)

      const response = await fetch(`${API_BASE}/api/google-drive/import`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          fileIds,
          folderIds,
          recursive: true,
          preserveHierarchy: false,
          skipDuplicates: true
        })
      })

      if (!response.ok) throw new Error('Failed to start import')

      const data = await response.json()
      alert(`Import started! Job ID: ${data.jobId}`)
      setShowPreview(false)
      setShowBrowser(false)
      clearSelection()
    } catch (err) {
      setError('Failed to start import: ' + err.message)
    } finally {
      setImporting(false)
    }
  }

  const formatBytes = (bytes) => {
    if (bytes === 0) return '0 Bytes'
    const k = 1024
    const sizes = ['Bytes', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i]
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
                <p className="text-sm text-yellow-700">{error}</p>
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
                disabled={oauthStatus === 'waiting'}
                className="bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-4 rounded disabled:bg-blue-400"
              >
                {oauthStatus === 'waiting' ? 'Authorizing with Google...' : 'Get Token from OAuth Broker'}
              </button>

              {/* Popup blocked - show options */}
              {oauthStatus === 'popup-blocked' && (
                <div className="mt-4 p-4 bg-yellow-50 border border-yellow-300 rounded">
                  <p className="text-yellow-800 font-semibold mb-2">Popup was blocked</p>
                  <p className="text-sm text-yellow-700 mb-3">
                    Please allow popups for this site and try again, or use the manual method:
                  </p>
                  <div className="flex gap-2">
                    <button
                      onClick={openOAuthBroker}
                      className="bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-4 rounded text-sm"
                    >
                      Try Again
                    </button>
                    <button
                      onClick={openOAuthBrokerInNewTab}
                      className="bg-gray-600 hover:bg-gray-700 text-white font-medium py-2 px-4 rounded text-sm"
                    >
                      Open in New Tab (Manual Copy)
                    </button>
                  </div>
                </div>
              )}

              {/* Waiting for postMessage - show hint after delay */}
              {oauthStatus === 'waiting' && showManualHint && (
                <div className="mt-3 p-3 bg-blue-100 border border-blue-300 rounded">
                  <p className="text-sm text-blue-800">
                    Taking longer than expected? {' '}
                    <button
                      onClick={() => setShowTokenPaste(true)}
                      className="text-blue-600 hover:text-blue-800 underline font-medium"
                    >
                      Click here if you need to copy/paste manually
                    </button>
                  </p>
                </div>
              )}

              {/* Timeout */}
              {oauthStatus === 'timeout' && (
                <div className="mt-3 p-3 bg-orange-100 border border-orange-300 rounded">
                  <p className="text-sm text-orange-800">
                    Connection timed out. Please try again or use manual copy/paste.
                  </p>
                </div>
              )}

              {/* Manual token paste area */}
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

            {!showBrowser ? (
              <div className="space-y-4">
                <div>
                  <h3 className="text-lg font-semibold mb-2">Import Options</h3>
                  <p className="text-gray-600 mb-4">
                    Select files and folders from your Google Drive to import into PinShare.
                  </p>
                  <button
                    className="bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-4 rounded mr-2"
                    onClick={openBrowser}
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
            ) : (
              <div>
                {/* Breadcrumb navigation */}
                <div className="mb-4 flex items-center text-sm">
                  {breadcrumbs.map((crumb, index) => (
                    <div key={crumb.id} className="flex items-center">
                      {index > 0 && <span className="mx-2 text-gray-400">/</span>}
                      <button
                        onClick={() => navigateToBreadcrumb(index)}
                        className="text-blue-600 hover:text-blue-800"
                      >
                        {crumb.name}
                      </button>
                    </div>
                  ))}
                </div>

                {/* Selection toolbar */}
                {selectedFiles.size > 0 && (
                  <div className="mb-4 p-3 bg-blue-50 border border-blue-200 rounded flex items-center justify-between">
                    <span className="text-sm text-blue-900">
                      {selectedFiles.size} item{selectedFiles.size !== 1 ? 's' : ''} selected
                    </span>
                    <div className="space-x-2">
                      <button
                        onClick={previewImport}
                        className="bg-blue-600 hover:bg-blue-700 text-white text-sm py-1 px-3 rounded"
                      >
                        Preview Import
                      </button>
                      <button
                        onClick={clearSelection}
                        className="bg-gray-500 hover:bg-gray-600 text-white text-sm py-1 px-3 rounded"
                      >
                        Clear
                      </button>
                    </div>
                  </div>
                )}

                {/* File list */}
                <div className="border border-gray-200 rounded-lg overflow-hidden">
                  {loadingFiles ? (
                    <div className="p-8 text-center text-gray-500">Loading...</div>
                  ) : files.length === 0 ? (
                    <div className="p-8 text-center text-gray-500">This folder is empty</div>
                  ) : (
                    <table className="min-w-full divide-y divide-gray-200">
                      <thead className="bg-gray-50">
                        <tr>
                          <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase w-12">
                            <input
                              type="checkbox"
                              checked={selectedFiles.size === files.length && files.length > 0}
                              onChange={(e) => e.target.checked ? selectAll() : clearSelection()}
                              className="rounded border-gray-300"
                            />
                          </th>
                          <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Name</th>
                          <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Size</th>
                          <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Modified</th>
                        </tr>
                      </thead>
                      <tbody className="bg-white divide-y divide-gray-200">
                        {files.map((file) => (
                          <tr
                            key={file.id}
                            className="hover:bg-gray-50 cursor-pointer"
                            onClick={() => file.isFolder ? navigateToFolder(file) : toggleFileSelection(file.id)}
                          >
                            <td className="px-4 py-3" onClick={(e) => e.stopPropagation()}>
                              <input
                                type="checkbox"
                                checked={selectedFiles.has(file.id)}
                                onChange={() => toggleFileSelection(file.id)}
                                className="rounded border-gray-300"
                              />
                            </td>
                            <td className="px-4 py-3">
                              <div className="flex items-center">
                                {file.isFolder ? (
                                  <svg className="w-5 h-5 text-blue-500 mr-2" fill="currentColor" viewBox="0 0 20 20">
                                    <path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
                                  </svg>
                                ) : (
                                  <svg className="w-5 h-5 text-gray-400 mr-2" fill="currentColor" viewBox="0 0 20 20">
                                    <path fillRule="evenodd" d="M4 4a2 2 0 012-2h4.586A2 2 0 0112 2.586L15.414 6A2 2 0 0116 7.414V16a2 2 0 01-2 2H6a2 2 0 01-2-2V4z" clipRule="evenodd" />
                                  </svg>
                                )}
                                <span className="text-sm text-gray-900">{file.name}</span>
                              </div>
                            </td>
                            <td className="px-4 py-3 text-sm text-gray-500">
                              {file.isFolder ? '-' : formatBytes(file.size)}
                            </td>
                            <td className="px-4 py-3 text-sm text-gray-500">
                              {new Date(file.modifiedTime).toLocaleDateString()}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  )}
                </div>

                <div className="mt-4 flex gap-2">
                  <button
                    onClick={() => {
                      setShowBrowser(false)
                      clearSelection()
                    }}
                    className="bg-gray-500 hover:bg-gray-600 text-white py-2 px-4 rounded"
                  >
                    Close Browser
                  </button>
                </div>
              </div>
            )}
          </div>
        )}

        {/* Import Preview Modal */}
        {showPreview && preview && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className="bg-white rounded-lg p-6 max-w-2xl w-full mx-4 max-h-96 overflow-y-auto">
              <h3 className="text-xl font-bold mb-4">Import Preview</h3>

              <div className="mb-4 p-4 bg-blue-50 rounded">
                <p className="text-sm text-gray-700">
                  <strong>Total Files:</strong> {preview.totalCount}
                </p>
                <p className="text-sm text-gray-700">
                  <strong>Total Size:</strong> {formatBytes(preview.totalSize)}
                </p>
              </div>

              <div className="mb-4 max-h-48 overflow-y-auto border rounded">
                <table className="min-w-full text-sm">
                  <thead className="bg-gray-50 sticky top-0">
                    <tr>
                      <th className="px-3 py-2 text-left">Name</th>
                      <th className="px-3 py-2 text-left">Size</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y">
                    {preview.files.map((file, index) => (
                      <tr key={index}>
                        <td className="px-3 py-2">{file.name}</td>
                        <td className="px-3 py-2">{formatBytes(file.size)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              <div className="flex gap-2 justify-end">
                <button
                  onClick={() => setShowPreview(false)}
                  className="bg-gray-500 hover:bg-gray-600 text-white py-2 px-4 rounded"
                >
                  Cancel
                </button>
                <button
                  onClick={startImport}
                  disabled={importing}
                  className="bg-blue-600 hover:bg-blue-700 text-white py-2 px-4 rounded disabled:bg-gray-400"
                >
                  {importing ? 'Starting...' : 'Start Import'}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
