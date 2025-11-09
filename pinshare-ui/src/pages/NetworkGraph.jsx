import React, { useState, useEffect, useRef } from 'react'
import { useQuery } from '@tanstack/react-query'
import ForceGraph2D from 'react-force-graph-2d'
import { pinshareApi } from '../services/api'
import { Network, Users, Activity, Server } from 'lucide-react'

function NetworkGraph() {
  const [graphData, setGraphData] = useState({ nodes: [], links: [] })
  const graphRef = useRef()

  const { data: status, isLoading: statusLoading, error: statusError } = useQuery({
    queryKey: ['p2p-status'],
    queryFn: () => pinshareApi.get('/p2p/status').then(res => res.data),
    refetchInterval: 5000, // Poll every 5 seconds
  })

  const { data: peers, isLoading: peersLoading, error: peersError } = useQuery({
    queryKey: ['p2p-peers'],
    queryFn: () => pinshareApi.get('/p2p/peers').then(res => res.data),
    refetchInterval: 5000, // Poll every 5 seconds
  })

  useEffect(() => {
    if (status && peers) {
      updateGraphData()
    }
  }, [status, peers])

  const updateGraphData = () => {
    const nodes = []
    const links = []

    // Add local node
    if (status?.id || status?.nodeID) {
      const nodeId = status.id || status.nodeID
      nodes.push({
        id: nodeId,
        name: 'You',
        type: 'local',
        val: 15,
        fx: 0, // Fix position at center
        fy: 0,
      })

      // Add peer nodes and links in a radial layout
      const peerList = Array.isArray(peers) ? peers : []
      const radius = 300 // Distance from center
      const angleStep = (2 * Math.PI) / peerList.length

      peerList.forEach((peer, index) => {
        // Peers are returned as strings (peer IDs), not objects
        const peerId = typeof peer === 'string' ? peer : (peer.peerID || peer.id || `peer-${index}`)
        const angle = index * angleStep

        nodes.push({
          id: peerId,
          name: `Peer ${index + 1}`,
          type: 'peer',
          val: 10,
          fx: radius * Math.cos(angle), // Fix position in circle
          fy: radius * Math.sin(angle),
        })

        links.push({
          source: nodeId,
          target: peerId,
        })
      })
    }

    setGraphData({ nodes, links })
  }

  if (statusLoading || peersLoading) {
    return (
      <div className="text-center py-8">
        <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        <p className="mt-4">Loading network visualization...</p>
      </div>
    )
  }

  if (statusError || peersError) {
    return (
      <div className="text-center py-8 text-red-500">
        <h3 className="text-lg font-semibold">Error Loading Network Data</h3>
        <p>{statusError?.message || peersError?.message}</p>
        <p className="mt-4 text-sm text-gray-500">
          Make sure the PinShare backend is running on port 9090
        </p>
      </div>
    )
  }

  const peerCount = Array.isArray(peers) ? peers.length : 0

  return (
    <div className="px-4 py-6 sm:px-0">
      <div className="mb-6">
        <h2 className="text-2xl font-bold text-gray-900 mb-2">Network Visualization</h2>
        <p className="text-gray-600">Interactive force-directed graph showing P2P connections.</p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
        {/* Network Graph - Takes 3/4 of the space */}
        <div className="lg:col-span-3">
          <div className="bg-gray-900 rounded-lg shadow overflow-hidden" style={{ height: '600px' }}>
            {graphData.nodes.length > 0 ? (
              <ForceGraph2D
                ref={graphRef}
                graphData={graphData}
                nodeLabel="name"
                nodeColor={(node) => (node.type === 'local' ? '#3b82f6' : '#8b5cf6')}
                nodeRelSize={4}
                nodeCanvasObject={(node, ctx, globalScale) => {
                  const label = node.name
                  const fontSize = 12/globalScale
                  ctx.font = `${fontSize}px Sans-Serif`
                  const textWidth = ctx.measureText(label).width
                  const bckgDimensions = [textWidth, fontSize].map(n => n + fontSize * 0.2)

                  // Draw circle
                  ctx.fillStyle = node.type === 'local' ? '#3b82f6' : '#8b5cf6'
                  ctx.beginPath()
                  ctx.arc(node.x, node.y, node.val || 4, 0, 2 * Math.PI, false)
                  ctx.fill()
                }}
                linkColor={() => '#1e3a8a'}
                linkWidth={1}
                linkDirectionalParticles={0}
                backgroundColor="#111827"
                enableNodeDrag={true}
                enableZoomInteraction={true}
                enablePanInteraction={true}
                onEngineStop={() => {
                  if (graphRef.current) {
                    graphRef.current.zoomToFit(400, 100)
                  }
                }}
              />
            ) : (
              <div className="flex flex-col items-center justify-center h-full text-gray-400">
                <Network size={64} className="mb-4 opacity-50" />
                <h3 className="text-lg font-semibold">No Network Data</h3>
                <p className="text-sm">Connect to peers to visualize the network</p>
              </div>
            )}
          </div>
        </div>

        {/* Sidebar - Takes 1/4 of the space */}
        <div className="lg:col-span-1 space-y-6">
          {/* Node Status Card */}
          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center mb-4">
              <Server className="w-6 h-6 text-blue-500 mr-3" />
              <h3 className="text-lg font-semibold text-gray-900">Node Status</h3>
            </div>
            {status ? (
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm text-gray-600">Status</span>
                  <span className="flex items-center text-sm font-semibold text-green-600">
                    <Activity size={14} className="mr-1" />
                    Online
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-gray-600">Connected Peers</span>
                  <span className="text-sm font-semibold text-gray-900">{peerCount}</span>
                </div>
                {status.addresses && status.addresses.length > 0 && (
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-gray-600">Addresses</span>
                    <span className="text-sm font-semibold text-gray-900">{status.addresses.length}</span>
                  </div>
                )}
                {(status.nodeID || status.id) && (
                  <div className="mt-4">
                    <div className="text-xs text-gray-500 mb-1">Node ID:</div>
                    <div className="text-xs font-mono bg-gray-50 p-2 rounded break-all">
                      {status.nodeID || status.id}
                    </div>
                  </div>
                )}
              </div>
            ) : (
              <p className="text-sm text-gray-500">No status information available</p>
            )}
          </div>

          {/* Connected Peers Card */}
          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center mb-4">
              <Users className="w-6 h-6 text-purple-500 mr-3" />
              <h3 className="text-lg font-semibold text-gray-900">
                Connected Peers ({peerCount})
              </h3>
            </div>
            {peerCount > 0 ? (
              <div className="space-y-3 max-h-96 overflow-y-auto">
                {peers.map((peer, index) => {
                  // Peers are returned as strings (peer IDs), not objects
                  const peerId = typeof peer === 'string' ? peer : (peer.peerID || peer.id || 'Unknown ID')
                  const peerAddresses = typeof peer === 'object' ? peer.addresses : null

                  return (
                    <div key={peerId || index} className="border-b border-gray-100 pb-3 last:border-0">
                      <div className="text-sm font-semibold text-gray-900 mb-1">
                        Peer {index + 1}
                      </div>
                      <div className="text-xs font-mono bg-gray-50 p-2 rounded break-all">
                        {peerId}
                      </div>
                      {peerAddresses && peerAddresses.length > 0 && (
                        <div className="text-xs text-gray-500 mt-1">
                          {peerAddresses.length} address{peerAddresses.length !== 1 ? 'es' : ''}
                        </div>
                      )}
                    </div>
                  )
                })}
              </div>
            ) : (
              <div className="text-center py-8">
                <Users size={32} className="mx-auto mb-2 text-gray-300" />
                <p className="text-sm text-gray-500">No peers connected</p>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

export default NetworkGraph
