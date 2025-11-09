import React, { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'
import Fuse from 'fuse.js'
import { useMetadata } from '../hooks/useMetadata'
import FileRow from '../components/FileRow.jsx'
import { Search, Filter } from 'lucide-react'
import toast from 'react-hot-toast'

const API_BASE = import.meta.env.VITE_API_BASE || 'http://localhost:9090'
const GATEWAY_BASE = import.meta.env.VITE_GATEWAY_BASE || 'http://localhost:8080/ipfs'

function Browse() {
  const { data: files, isLoading, error } = useMetadata()
  const [searchParams, setSearchParams] = useSearchParams()

  // Initialize state from URL params
  const [searchTerm, setSearchTerm] = useState(searchParams.get('search') || '')
  const [filterType, setFilterType] = useState(searchParams.get('type') || 'all')
  const [filteredFiles, setFilteredFiles] = useState([])

  // Update URL when search/filter changes
  const handleSearchChange = (value) => {
    setSearchTerm(value)
    const newParams = new URLSearchParams(searchParams)
    if (value) {
      newParams.set('search', value)
    } else {
      newParams.delete('search')
    }
    setSearchParams(newParams, { replace: true })
  }

  const handleFilterChange = (value) => {
    setFilterType(value)
    const newParams = new URLSearchParams(searchParams)
    if (value && value !== 'all') {
      newParams.set('type', value)
    } else {
      newParams.delete('type')
    }
    setSearchParams(newParams, { replace: true })
  }

  // Get unique file types for dynamic filter
  const fileTypes = React.useMemo(() => {
    if (!files || files.length === 0) return []
    const types = [...new Set(files.map(file => file.fileType).filter(Boolean))]
    return types.sort()
  }, [files])

  // Memoize Fuse instance to avoid recreating on every search
  const fuse = React.useMemo(() => {
    if (!files || files.length === 0) return null
    return new Fuse(files, {
      keys: [
        { name: 'fileName', weight: 0.4 },
        { name: 'ipfsCID', weight: 0.3 },
        { name: 'fileType', weight: 0.2 },
        { name: 'fileSHA256', weight: 0.1 },
      ],
      threshold: 0.3,
      includeScore: true,
      includeMatches: true,
      ignoreLocation: true,
      minMatchCharLength: 2,
    })
  }, [files])

  React.useEffect(() => {
    let results = files || []
    let resultsWithMatches = []

    if (searchTerm && fuse) {
      const fuseResults = fuse.search(searchTerm)
      resultsWithMatches = fuseResults.map(result => ({
        ...result.item,
        _matches: result.matches,
        _score: result.score
      }))
      results = resultsWithMatches
    } else {
      results = files || []
    }

    if (filterType !== 'all') {
      results = results.filter(file => file.fileType === filterType)
    }

    setFilteredFiles(results)
  }, [files, searchTerm, filterType, fuse])

  if (isLoading) return <div className="text-center py-8">Loading metadata...</div>
  if (error) return <div className="text-center py-8 text-red-500">Error: {error.message}. Check API connection.</div>

  return (
    <div className="px-4 py-6 sm:px-0">
      <div className="mb-6 flex flex-col sm:flex-row gap-4">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-5 h-5" />
          <input
            type="text"
            placeholder="Search by file name, type, CID, or hash..."
            value={searchTerm}
            onChange={(e) => handleSearchChange(e.target.value)}
            className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
        <select
          value={filterType}
          onChange={(e) => handleFilterChange(e.target.value)}
          className="px-4 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option value="all">All Types</option>
          {fileTypes.map(type => (
            <option key={type} value={type}>{type.toUpperCase()}</option>
          ))}
        </select>
      </div>
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200 bg-white">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">File Name</th>
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
                <td colSpan="6" className="px-6 py-4 text-center text-gray-500">
                  {(!files || files.length === 0) ? (
                    "No files found. Connect to peers to receive metadata."
                  ) : (
                    <>No results found for "{searchTerm || filterType}". Try a different search term or filter.</>
                  )}
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