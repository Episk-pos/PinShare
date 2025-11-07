import { useState, useEffect, useRef } from 'react';
import { p2pAPI } from '../services/api';
import ForceGraph2D from 'react-force-graph-2d';
import { Network, Users, Activity, Server } from 'lucide-react';

const NetworkView = () => {
  const [status, setStatus] = useState(null);
  const [peers, setPeers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [graphData, setGraphData] = useState({ nodes: [], links: [] });
  const graphRef = useRef();

  useEffect(() => {
    fetchNetworkData();
    // Poll for updates every 5 seconds
    const interval = setInterval(fetchNetworkData, 5000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    if (status && peers) {
      updateGraphData();
    }
  }, [status, peers]);

  const fetchNetworkData = async () => {
    try {
      const [statusData, peersData] = await Promise.all([
        p2pAPI.getStatus(),
        p2pAPI.getPeers(),
      ]);

      setStatus(statusData);
      setPeers(Array.isArray(peersData) ? peersData : []);
      setError(null);
    } catch (err) {
      console.error('Error fetching network data:', err);
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const updateGraphData = () => {
    const nodes = [];
    const links = [];

    // Add local node
    if (status?.nodeID) {
      nodes.push({
        id: status.nodeID,
        name: 'You',
        type: 'local',
        val: 15,
      });

      // Add peer nodes and links
      peers.forEach((peer, index) => {
        const peerId = peer.peerID || peer.id || `peer-${index}`;
        nodes.push({
          id: peerId,
          name: `Peer ${index + 1}`,
          type: 'peer',
          val: 10,
        });

        links.push({
          source: status.nodeID,
          target: peerId,
        });
      });
    }

    setGraphData({ nodes, links });
  };

  if (loading) {
    return (
      <div className="loading">
        <div className="spinner"></div>
        <p>Loading network information...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="error-state">
        <h3>Error Loading Network Data</h3>
        <p>{error}</p>
        <p style={{ marginTop: '1rem', fontSize: '0.9rem' }}>
          Make sure the PinShare backend is running on port 9090
        </p>
      </div>
    );
  }

  return (
    <div className="network-container">
      <div className="network-graph">
        {graphData.nodes.length > 0 ? (
          <ForceGraph2D
            ref={graphRef}
            graphData={graphData}
            nodeLabel="name"
            nodeColor={(node) => (node.type === 'local' ? '#667eea' : '#764ba2')}
            nodeRelSize={6}
            linkColor={() => '#0f3460'}
            linkWidth={2}
            backgroundColor="#1a1a2e"
            d3VelocityDecay={0.3}
            cooldownTicks={100}
            onEngineStop={() => {
              if (graphRef.current) {
                graphRef.current.zoomToFit(400, 50);
              }
            }}
          />
        ) : (
          <div className="empty-state">
            <Network size={48} style={{ opacity: 0.5, margin: '0 auto 1rem' }} />
            <h3>No Network Data</h3>
            <p>Unable to visualize network</p>
          </div>
        )}
      </div>

      <div className="network-sidebar">
        <div className="network-card">
          <h3>
            <Server size={20} />
            Node Status
          </h3>
          {status ? (
            <div>
              <div className="status-item">
                <span className="status-label">Status</span>
                <span className="status-value" style={{ color: '#2ecc71' }}>
                  <Activity size={14} style={{ marginRight: '4px' }} />
                  Online
                </span>
              </div>
              <div className="status-item">
                <span className="status-label">Connected Peers</span>
                <span className="status-value">{peers.length}</span>
              </div>
              {status.addresses && status.addresses.length > 0 && (
                <div className="status-item">
                  <span className="status-label">Addresses</span>
                  <span className="status-value">{status.addresses.length}</span>
                </div>
              )}
              {status.nodeID && (
                <div>
                  <div style={{ marginTop: '1rem', marginBottom: '0.5rem', color: '#a0a0a0' }}>
                    Node ID:
                  </div>
                  <div className="node-id">{status.nodeID}</div>
                </div>
              )}
            </div>
          ) : (
            <p style={{ color: '#a0a0a0' }}>No status information available</p>
          )}
        </div>

        <div className="network-card">
          <h3>
            <Users size={20} />
            Connected Peers ({peers.length})
          </h3>
          {peers.length > 0 ? (
            <div className="peer-list">
              {peers.map((peer, index) => (
                <div key={peer.peerID || peer.id || index} className="peer-item">
                  <div style={{ marginBottom: '0.5rem', fontWeight: 'bold', color: '#e0e0e0' }}>
                    Peer {index + 1}
                  </div>
                  <div className="peer-id">
                    {peer.peerID || peer.id || 'Unknown ID'}
                  </div>
                  {peer.addresses && peer.addresses.length > 0 && (
                    <div style={{ marginTop: '0.5rem', fontSize: '0.7rem', color: '#a0a0a0' }}>
                      {peer.addresses.length} address{peer.addresses.length !== 1 ? 'es' : ''}
                    </div>
                  )}
                </div>
              ))}
            </div>
          ) : (
            <div style={{ color: '#a0a0a0', textAlign: 'center', padding: '1rem' }}>
              <Users size={32} style={{ opacity: 0.3, margin: '0 auto 0.5rem' }} />
              <p>No peers connected</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default NetworkView;
