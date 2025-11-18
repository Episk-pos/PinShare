import React from 'react'
import { useQuery } from '@tanstack/react-query'
import { RefreshCw, CheckCircle, XCircle, Loader, AlertCircle } from 'lucide-react'

const API_BASE = import.meta.env.VITE_API_BASE || ''

function UploadStatus() {
  const { data: statuses, isLoading, error, refetch } = useQuery({
    queryKey: ['uploadStatus'],
    queryFn: async () => {
      const response = await fetch(`${API_BASE}/api/upload-status`)
      if (!response.ok) throw new Error('Failed to fetch upload status')
      return response.json()
    },
    refetchInterval: 2000, // Refresh every 2 seconds for real-time updates
    staleTime: 0,
  })

  const getStageDisplay = (stage) => {
    const stages = {
      validating: { label: 'Validating', color: 'text-blue-600', icon: Loader },
      hashing: { label: 'Calculating Hash', color: 'text-blue-600', icon: Loader },
      scanning: { label: 'Security Scan', color: 'text-yellow-600', icon: Loader },
      uploading: { label: 'Uploading to IPFS', color: 'text-purple-600', icon: Loader },
      storing: { label: 'Storing Metadata', color: 'text-indigo-600', icon: Loader },
      completed: { label: 'Completed', color: 'text-green-600', icon: CheckCircle },
      failed: { label: 'Failed', color: 'text-red-600', icon: XCircle },
    }
    return stages[stage] || { label: stage, color: 'text-gray-600', icon: AlertCircle }
  }

  const formatDuration = (start, end) => {
    const duration = (end ? new Date(end) : new Date()) - new Date(start)
    if (duration < 1000) return `${duration}ms`
    if (duration < 60000) return `${(duration / 1000).toFixed(1)}s`
    return `${(duration / 60000).toFixed(1)}m`
  }

  // Separate active and completed uploads
  const activeUploads = statuses?.filter(s => s.stage !== 'completed' && s.stage !== 'failed') || []
  const completedUploads = statuses?.filter(s => s.stage === 'completed' || s.stage === 'failed') || []

  if (isLoading) return <div className="text-center py-8">Loading upload status...</div>
  if (error) return <div className="text-center py-8 text-red-500">Error: {error.message}</div>

  return (
    <div className="px-4 py-6 sm:px-0">
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold text-gray-900">Upload Status</h2>
        <button
          onClick={() => refetch()}
          className="flex items-center space-x-2 px-4 py-2 bg-blue-500 text-white rounded-md hover:bg-blue-600"
        >
          <RefreshCw className="w-4 h-4" />
          <span>Refresh</span>
        </button>
      </div>

      {/* Active Uploads */}
      {activeUploads.length > 0 && (
        <div className="mb-8">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">In Progress ({activeUploads.length})</h3>
          <div className="space-y-4">
            {activeUploads.map((status) => {
              const stageInfo = getStageDisplay(status.stage)
              const StageIcon = stageInfo.icon
              return (
                <div key={status.fileName} className="bg-white border border-gray-200 rounded-lg p-4 shadow-sm">
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center space-x-2">
                      <StageIcon className={`w-5 h-5 ${stageInfo.color} ${status.stage !== 'completed' && status.stage !== 'failed' ? 'animate-spin' : ''}`} />
                      <span className="font-medium text-gray-900">{status.fileName}</span>
                    </div>
                    <span className={`text-sm font-semibold ${stageInfo.color}`}>{stageInfo.label}</span>
                  </div>

                  {/* Progress Bar */}
                  <div className="w-full bg-gray-200 rounded-full h-2 mb-2">
                    <div
                      className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                      style={{ width: `${status.progress}%` }}
                    ></div>
                  </div>

                  <div className="flex justify-between text-xs text-gray-500">
                    <span>{status.progress}% Complete</span>
                    <span>Duration: {formatDuration(status.startedAt, status.lastUpdatedAt)}</span>
                  </div>

                  {status.fileSHA256 && (
                    <div className="mt-2 text-xs text-gray-400 font-mono truncate">
                      SHA256: {status.fileSHA256}
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        </div>
      )}

      {/* Completed/Failed Uploads */}
      <div>
        <h3 className="text-lg font-semibold text-gray-900 mb-4">
          Recent ({completedUploads.length})
        </h3>

        {completedUploads.length === 0 ? (
          <div className="text-center py-8 text-gray-500">
            No recent uploads
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200 bg-white">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">File Name</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Duration</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Completed At</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Details</th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {completedUploads.map((status) => {
                  const stageInfo = getStageDisplay(status.stage)
                  const StageIcon = stageInfo.icon
                  return (
                    <tr key={status.fileName} className="hover:bg-gray-50">
                      <td className="px-6 py-4 text-sm font-medium text-gray-900 truncate max-w-xs">
                        {status.fileName}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="flex items-center space-x-2">
                          <StageIcon className={`w-4 h-4 ${stageInfo.color}`} />
                          <span className={`text-sm font-semibold ${stageInfo.color}`}>{stageInfo.label}</span>
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        {formatDuration(status.startedAt, status.completedAt)}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        {status.completedAt ? new Date(status.completedAt).toLocaleString() : 'N/A'}
                      </td>
                      <td className="px-6 py-4 text-sm text-gray-500">
                        {status.error ? (
                          <span className="text-red-600">{status.error}</span>
                        ) : status.fileSHA256 ? (
                          <span className="font-mono text-xs truncate block max-w-xs">
                            {status.fileSHA256}
                          </span>
                        ) : (
                          '-'
                        )}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}

export default UploadStatus
