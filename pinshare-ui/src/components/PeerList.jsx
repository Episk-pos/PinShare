import React from 'react'

function PeerList({ peers }) {
  const peerList = Array.isArray(peers) ? peers : []

  return (
    <div>
      {peerList.length > 0 ? (
        <div className="space-y-2">
          {peerList.map((peer, index) => {
            // Peers are returned as strings (peer IDs)
            const peerId = typeof peer === 'string' ? peer : (peer.peerID || peer.id || `peer-${index}`)

            return (
              <div key={peerId || index} className="p-3 bg-gray-50 rounded-lg border border-gray-200">
                <div className="flex items-center justify-between">
                  <div className="flex-1">
                    <div className="text-xs text-gray-500 mb-1">Peer {index + 1}</div>
                    <div className="text-sm font-mono text-gray-900 break-all">{peerId}</div>
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      ) : (
        <div className="text-center py-8">
          <p className="text-sm text-gray-500">No connected peers yet.</p>
          <p className="text-xs text-gray-400 mt-2">Start the P2P service to connect to peers.</p>
        </div>
      )}
    </div>
  )
}

export default PeerList