import React from 'react'
import { ExternalLink, Download, Pin } from 'lucide-react'
import toast from 'react-hot-toast'

const GATEWAY_BASE = import.meta.env.VITE_GATEWAY_BASE || 'http://localhost:8080/ipfs'
const IPFS_API_BASE = import.meta.env.VITE_IPFS_API_BASE || 'http://localhost:5001/api/v0'

function FileRow({ file }) {
  const handlePreview = () => {
    window.open(`${GATEWAY_BASE}${file.ipfsCID}`, '_blank')
  }

  const handleDownload = () => {
    window.open(`${GATEWAY_BASE}${file.ipfsCID}`, '_blank')
  }

  const handlePin = async () => {
    try {
      const response = await fetch(`${IPFS_API_BASE}/pin/add?arg=${file.ipfsCID}`, {
        method: 'POST',
      })
      if (response.ok) {
        toast.success(`Pinned ${file.ipfsCID.substring(0, 8)}...`)
      } else {
        throw new Error('Pin failed')
      }
    } catch (error) {
      toast.error(`Pin error: ${error.message}. Try IPFS CLI: ipfs pin add ${file.ipfsCID}`)
    }
  }

  return (
    <tr className="hover:bg-gray-50">
      <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
        {file.ipfsCID ? file.ipfsCID.substring(0, 32) + '...' : 'N/A'}
      </td>
      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
        {file.fileSHA256 ? file.fileSHA256.substring(0, 16) + '...' : 'N/A'}
      </td>
      <td className="px-6 py-4 whitespace-nowrap">
        <span className="inline-flex px-2 py-1 text-xs font-semibold rounded-full bg-blue-100 text-blue-800">
          {file.fileType || 'unknown'}
        </span>
      </td>
      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
        {file.addedAt ? new Date(file.addedAt).toLocaleDateString() : 'N/A'}
      </td>
      <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium space-x-2">
        <button
          onClick={handlePreview}
          className="text-indigo-600 hover:text-indigo-900 flex items-center space-x-1"
        >
          <ExternalLink className="w-4 h-4" />
          <span>Preview</span>
        </button>
        <button
          onClick={handleDownload}
          className="text-green-600 hover:text-green-900 flex items-center space-x-1"
        >
          <Download className="w-4 h-4" />
          <span>Download</span>
        </button>
        <button
          onClick={handlePin}
          className="text-blue-600 hover:text-blue-900 flex items-center space-x-1"
        >
          <Pin className="w-4 h-4" />
          <span>Pin</span>
        </button>
      </td>
    </tr>
  )
}

export default FileRow