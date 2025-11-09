import React from 'react'
import { useQuery } from '@tanstack/react-query'
import { pinshareApi } from '../services/api'
import PeerList from '../components/PeerList.jsx'
import { Users } from 'lucide-react'

function NetworkDashboard() {
  const { data: status, isLoading: statusLoading, error: statusError } = useQuery({
    queryKey: ['p2p-status'],
    queryFn: () => pinshareApi.get('/p2p/status').then(res => res.data),
  })

  const { data: peers, isLoading: peersLoading, error: peersError } = useQuery({
    queryKey: ['p2p-peers'],
    queryFn: () => pinshareApi.get('/p2p/peers').then(res => res.data),
  })

  if (statusLoading || peersLoading) return <div className="text-center py-8">Loading network...</div>
  if (statusError || peersError) return <div className="text-center py-8 text-red-500">Error loading network data.</div>

  return (
    <div className="px-4 py-6 sm:px-0">
      <div className="mb-6">
        <h2 className="text-2xl font-bold text-gray-900 mb-2">Network Dashboard</h2>
        <p className="text-gray-600">View your P2P node status and connected peers.</p>
      </div>

      <div className="bg-white p-6 rounded-lg shadow mb-8">
        <div className="flex items-center mb-4">
          <Users className="w-8 h-8 text-blue-500 mr-3" />
          <h3 className="text-lg font-semibold text-gray-900">P2P Status</h3>
        </div>
        <div className="space-y-3">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <p className="text-sm text-gray-600">Connected Peers</p>
              <p className="text-2xl font-bold text-gray-900">{status?.connectedPeers || 0}</p>
            </div>
            <div className="md:col-span-2">
              <p className="text-sm text-gray-600 mb-1">Node ID</p>
              <p className="text-sm font-mono bg-gray-50 p-2 rounded break-all">{status?.id || 'N/A'}</p>
            </div>
          </div>
          {status?.addresses && status.addresses.length > 0 && (
            <div>
              <p className="text-sm text-gray-600 mb-2">Listening Addresses</p>
              <ul className="space-y-1">
                {status.addresses.map((addr, idx) => (
                  <li key={idx} className="text-xs font-mono bg-gray-50 p-2 rounded break-all">{addr}</li>
                ))}
              </ul>
            </div>
          )}
        </div>
      </div>

      <div className="bg-white rounded-lg shadow">
        <div className="px-6 py-4 border-b border-gray-200">
          <h3 className="text-lg font-semibold text-gray-900">Connected Peers ({Array.isArray(peers) ? peers.length : 0})</h3>
        </div>
        <div className="p-6">
          <PeerList peers={peers || []} />
        </div>
      </div>
    </div>
  )
}

export default NetworkDashboard