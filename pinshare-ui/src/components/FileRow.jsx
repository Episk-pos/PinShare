import React from 'react'
import { ExternalLink, Download, Pin, AlertTriangle, Tag, Copy } from 'lucide-react'
import toast from 'react-hot-toast'
import { Badge } from './Badge.jsx'

const API_BASE = import.meta.env.VITE_API_BASE || ''
const GATEWAY_BASE = import.meta.env.VITE_GATEWAY_BASE || 'http://localhost:8080/ipfs'
const IPFS_API_BASE = import.meta.env.VITE_IPFS_API_BASE || 'http://localhost:5001/api/v0'

// Helper function to highlight matched text
const HighlightedText = ({ text, matches = [] }) => {
  if (!matches || matches.length === 0) {
    return <>{text || 'Unknown'}</>
  }

  // Sort indices by start position
  const sortedIndices = [...matches].sort((a, b) => a[0] - b[0])

  const parts = []
  let lastIndex = 0

  sortedIndices.forEach(([start, end]) => {
    // Add non-highlighted text before match
    if (start > lastIndex) {
      parts.push(<span key={`text-${lastIndex}`}>{text.substring(lastIndex, start)}</span>)
    }
    // Add highlighted match
    parts.push(
      <mark key={`match-${start}`} className="bg-yellow-200 font-semibold">
        {text.substring(start, end + 1)}
      </mark>
    )
    lastIndex = end + 1
  })

  // Add remaining text
  if (lastIndex < text.length) {
    parts.push(<span key={`text-${lastIndex}`}>{text.substring(lastIndex)}</span>)
  }

  return <>{parts}</>
}

function FileRow({ file }) {
  // Extract match indices for fileName from Fuse.js matches
  const getMatchIndices = (field) => {
    if (!file._matches) return []
    const match = file._matches.find(m => m.key === field)
    return match ? match.indices : []
  }

  const getBanLabel = (banSet) => {
    if (banSet === 3) return 'Policy Violation'
    if (banSet === 6) return 'Indecent Content'
    if (banSet === 9) return 'Malware'
    return 'Banned'
  }
  const handlePreview = () => {
    window.open(`${GATEWAY_BASE}/${file.ipfsCID}`, '_blank')
  }

  const handleDownload = () => {
    window.open(`${GATEWAY_BASE}/${file.ipfsCID}`, '_blank')
  }

  const handlePin = async () => {
    try {
      const response = await fetch(`${API_BASE}/api/v1/ipfs/pin/${file.ipfsCID}`, {
        method: 'POST',
      })
      if (response.ok) {
        toast.success(`Pinned ${file.ipfsCID.substring(0, 8)}...`)
      } else {
        const error = await response.json()
        throw new Error(error.message || 'Pin failed')
      }
    } catch (error) {
      toast.error(`Pin error: ${error.message}. Try IPFS CLI: ipfs pin add ${file.ipfsCID}`)
    }
  }

  const handleCopyCID = () => {
    navigator.clipboard.writeText(file.ipfsCID)
    toast.success('CID copied to clipboard!')
  }

  return (
    <tr className="hover:bg-gray-50">
      <td className="px-6 py-4 text-sm font-medium text-gray-900">
        <HighlightedText text={file.fileName || 'Unknown file'} matches={getMatchIndices('fileName')} />
      </td>
      <td className="px-6 py-4 whitespace-nowrap">
        <div className="flex items-center space-x-2">
          <span className="inline-flex px-2 py-1 text-xs font-semibold rounded-full bg-blue-100 text-blue-800">
            {file.fileType || 'unknown'}
          </span>
          {file.banSet > 0 && (
            <span className="inline-flex items-center px-2 py-1 text-xs font-semibold rounded-full bg-red-100 text-red-800">
              <AlertTriangle size={12} className="mr-1" />
              {getBanLabel(file.banSet)}
            </span>
          )}
        </div>
      </td>
      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
        {file.addedAt ? new Date(file.addedAt).toLocaleDateString() : 'N/A'}
      </td>
      <td className="px-6 py-4 whitespace-nowrap text-sm text-center">
        {file.moderationVotes !== undefined && file.moderationVotes !== 0 ? (
          <span className={`font-semibold ${file.moderationVotes > 0 ? 'text-red-600' : 'text-blue-600'}`}>
            {file.moderationVotes}
          </span>
        ) : (
          <span className="text-gray-400">-</span>
        )}
      </td>
      <td className="px-6 py-4">
        {file.tags && Object.keys(file.tags).length > 0 ? (
          <div className="flex flex-wrap gap-1">
            {Object.entries(file.tags).slice(0, 3).map(([tag, count]) => (
              <span key={tag} className="inline-flex items-center px-2 py-1 text-xs rounded-full bg-purple-100 text-purple-800">
                <Tag size={10} className="mr-1" />
                {tag}
                {count > 1 && ` (${count})`}
              </span>
            ))}
            {Object.keys(file.tags).length > 3 && (
              <span className="text-xs text-gray-500">+{Object.keys(file.tags).length - 3}</span>
            )}
          </div>
        ) : (
          <span className="text-xs text-gray-400">No tags</span>
        )}
      </td>
      <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
        <div className="flex items-center justify-end space-x-2">
          <button
            onClick={handlePreview}
            className="text-indigo-600 hover:text-indigo-900 flex items-center space-x-1"
            title="Preview file"
          >
            <ExternalLink className="w-4 h-4" />
          </button>
          <button
            onClick={handleDownload}
            className="text-green-600 hover:text-green-900 flex items-center space-x-1"
            title="Download file"
          >
            <Download className="w-4 h-4" />
          </button>
          <button
            onClick={handlePin}
            className="text-blue-600 hover:text-blue-900 flex items-center space-x-1"
            title="Pin to local IPFS"
          >
            <Pin className="w-4 h-4" />
          </button>
          <button
            onClick={handleCopyCID}
            className="text-gray-600 hover:text-gray-900 flex items-center space-x-1"
            title="Copy CID to clipboard"
          >
            <Copy className="w-4 h-4" />
          </button>
        </div>
      </td>
    </tr>
  )
}

export default FileRow