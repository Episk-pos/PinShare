import React from 'react'
import { Badge } from '../components/Badge.jsx'  // Optional styled badge

function PeerList({ peers, topicPeers }) {
  const allPeers = Array.isArray(topicPeers) ? topicPeers : []
  const peerCount = peers || 0

  return (
    <div>
      <p className="text-sm text-gray-600 mb-4">Total P2P Peers: <span className="font-semibold">{peerCount}</span></p>
      <p className="text-sm text-gray-600 mb-4">Topic Subscribers: <span className="font-semibold">{allPeers.length}</span></p>
      
      {allPeers.length > 0 ? (
        <ul className="space-y-2">
          {allPeers.slice(0, 10).map((peer, index) => (  // Limit to 10 for brevity
            <li key={index} className="flex justify-between items-center p-2 bg-gray-50 rounded">
              <span className="text-sm font-mono text-gray-900">{peer.substring(0, 16)}...</span>
              <Badge variant="green">Subscribed</Badge>
            </li>
          ))}
          {allPeers.length > 10 && (
            <li className="text-sm text-gray-500">... and {allPeers.length - 10} more</li>
          )}
        </ul>
      ) : (
        <p className="text-sm text-gray-500">No peers on topic yet. Connect to more nodes.</p>
      )}
    </div>
  )
}

export default PeerList