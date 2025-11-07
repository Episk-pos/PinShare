import React from 'react'
import { useQuery } from '@tanstack/react-query'
import { pinshareApi } from '../services/api'
import PeerList from '../components/PeerList.jsx'
import { Users, Share2 } from 'lucide-react'  // Renamed Network to Share2 to avoid conflict

function NetworkDashboard() {
  const { data: status, isLoading: statusLoading, error: statusError } = useQuery({
    queryKey: ['p2p-status'],
    queryFn: () => pinshareApi.get('/p2p/status').then(res => res.data),
  })

  const { data: topicData, isLoading: topicLoading, error: topicError } = useQuery({
    queryKey: ['topic-peers'],
    queryFn: () => pinshareApi.get('/p2p/topic-peers').then(res => res.data),
  })

  if (statusLoading || topicLoading) return <div className="text-center py-8">Loading network...</div>
  if (statusError || topicError) return <div className="text-center py-8 text-red-500">Error loading network data.</div>

  return (
    <div className="px-4 py-6 sm:px-0">
      <div className="mb-6">
        <h2 className="text-2xl font-bold text-gray-900 mb-2">Network Dashboard</h2>
        <p className="text-gray-600">View connected peers and PubSub topic subscribers.</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
        <div className="bg-white p-6 rounded-lg shadow">
          <div className="flex items-center mb-4">
            <Users className="w-8 h-8 text-blue-500 mr-3" />
            <h3 className="text-lg font-semibold text-gray-900">P2P Status</h3>
          </div>
          <div className="space-y-2">
            <p><span className="font-medium">Node ID:</span> {status?.id || 'N/A'}</p>
            <p><span className="font-medium">Connected Peers:</span> {status?.connectedPeers || 0}</p>
            <p><span className="font-medium">Addresses:</span> {status?.addresses?.join(', ') || 'None'}</p>
          </div>
        </div>

        <div className="bg-white p-6 rounded-lg shadow">
          <div className="flex items-center mb-4">
            <Share2 className="w-8 h-8 text-green-500 mr-3" />
            <h3 className="text-lg font-semibold text-gray-900">Gossip Topic</h3>
          </div>
          <div className="space-y-2">
            <p><span className="font-medium">Topic:</span> {topicData?.topic || 'N/A'}</p>
            <p><span className="font-medium">Subscribers:</span> {topicData?.peerCount || 0}</p>
          </div>
        </div>
      </div>

      <div className="bg-white rounded-lg shadow">
        <div className="px-6 py-4 border-b border-gray-200">
          <h3 className="text-lg font-semibold text-gray-900">Connected Peers</h3>
        </div>
        <div className="p-6">
          <PeerList peers={status?.connectedPeers || 0} topicPeers={topicData?.peers || []} />
        </div>
      </div>
    </div>
  )
}

export default NetworkDashboard