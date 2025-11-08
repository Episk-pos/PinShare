import React from 'react'
import { useQuery } from '@tanstack/react-query'
import { pinshareApi } from '../services/api'
import { FileText, Users } from 'lucide-react'

function StatsHeader() {
  const { data: files } = useQuery({
    queryKey: ['metadata'],
    queryFn: () => pinshareApi.get('/files').then(res => res.data),
    staleTime: 5 * 60 * 1000,
  })

  const { data: status } = useQuery({
    queryKey: ['p2p-status'],
    queryFn: () => pinshareApi.get('/p2p/status').then(res => res.data),
    staleTime: 10 * 1000,
  })

  const fileCount = files ? (Array.isArray(files) ? files.length : Object.keys(files).length) : 0
  const peerCount = status?.connectedPeers || 0

  return (
    <div className="flex items-center space-x-6">
      <div className="flex items-center space-x-2 text-gray-600">
        <FileText size={20} />
        <span className="text-sm">
          <span className="font-bold text-blue-600">{fileCount}</span> Files
        </span>
      </div>
      <div className="flex items-center space-x-2 text-gray-600">
        <Users size={20} />
        <span className="text-sm">
          <span className="font-bold text-purple-600">{peerCount}</span> Peers
        </span>
      </div>
    </div>
  )
}

export default StatsHeader
