import React, { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import Fuse from 'fuse.js'
import { useMetadata } from '../hooks/useMetadata'
import FileRow from '../components/FileRow.jsx'
import { Search, Filter } from 'lucide-react'
import toast from 'react-hot-toast'

const API_BASE = import.meta.env.VITE_API_BASE || 'http://localhost:9090'
const GATEWAY_BASE = import.meta.env.VITE_GATEWAY_BASE || 'http://localhost:8080/ipfs'

function Browse() {
  const { data: files, isLoading, error } = useMetadata()
  const [searchTerm, setSearchTerm] = useState('')
  const [filterType, setFilterType] = useState('all')
  const [filteredFiles, setFilteredFiles] = useState([])

  React.useEffect(() => {
    let results = files || []
    if (searchTerm) {
      const fuse = new Fuse(results, {
        keys: ['fileType', 'ipfsCID', 'fileSHA256'],
        threshold: 0.3,
      })
      results = fuse.search(searchTerm).map(result => result.item)
    }
    if (filterType !== 'all') {
      results = results.filter(file => file.fileType === filterType)
    }
    setFilteredFiles(results)
  }, [files, searchTerm, filterType])

  if (isLoading) return <div className="text-center py-8">Loading metadata...</div>
  if (error) return <div className="text-center py-8 text-red-500">Error: {error.message}. Check API connection.</div>

  return (
    <div className="px-4 py-6 sm:px-0">
      <div className="mb-6 flex flex-col sm:flex-row gap-4">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-5 h-5" />
          <input
            type="text"
            placeholder="Search by type, CID, or hash..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
        <select
          value={filterType}
          onChange={(e) => setFilterType(e.target.value)}
          className="px-4 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option value="all">All Types</option>
          <option value="pdf">PDF</option>
          <option value="txt">Text</option>
          <option value="html">HTML</option>
          {/* Add more based on allowed types */}
        </select>
      </div>
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200 bg-white">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">CID</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">SHA256 (Short)</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Type / Status</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Added</th>
              <th className="px-6 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider">Mod Votes</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Tags</th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {filteredFiles.length === 0 ? (
              <tr>
                <td colSpan="7" className="px-6 py-4 text-center text-gray-500">
                  No files found. Connect to peers to receive metadata.
                </td>
              </tr>
            ) : (
              filteredFiles.map((file) => (
                <FileRow key={file.fileSHA256} file={file} />
              ))
            )}
          </tbody>
        </table>
      </div>
      {filteredFiles.length > 0 && (
        <div className="mt-4 text-center text-sm text-gray-500">
          Showing {filteredFiles.length} of {files?.length || 0} files
        </div>
      )}
    </div>
  )
}

export default Browse