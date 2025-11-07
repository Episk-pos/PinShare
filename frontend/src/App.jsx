import { useState, useEffect } from 'react';
import FileBrowser from './components/FileBrowser';
import NetworkView from './components/NetworkView';
import { filesAPI, p2pAPI } from './services/api';
import { Share2, FileText, Network as NetworkIcon, Users } from 'lucide-react';
import './styles/App.css';

function App() {
  const [activeTab, setActiveTab] = useState('files');
  const [stats, setStats] = useState({
    fileCount: 0,
    peerCount: 0,
  });

  useEffect(() => {
    fetchStats();
    // Update stats every 10 seconds
    const interval = setInterval(fetchStats, 10000);
    return () => clearInterval(interval);
  }, []);

  const fetchStats = async () => {
    try {
      const [files, peers] = await Promise.all([
        filesAPI.getAllFiles().catch(() => []),
        p2pAPI.getPeers().catch(() => []),
      ]);

      const filesArray = Array.isArray(files) ? files : Object.values(files);
      const peersArray = Array.isArray(peers) ? peers : [];

      setStats({
        fileCount: filesArray.length,
        peerCount: peersArray.length,
      });
    } catch (err) {
      console.error('Error fetching stats:', err);
    }
  };

  return (
    <div className="app">
      <header className="header">
        <div className="header-content">
          <h1>
            <Share2 size={32} />
            PinShare
          </h1>
          <div className="header-stats">
            <div className="stat">
              <FileText size={20} />
              <span>
                <span className="stat-value">{stats.fileCount}</span> Files
              </span>
            </div>
            <div className="stat">
              <Users size={20} />
              <span>
                <span className="stat-value">{stats.peerCount}</span> Peers
              </span>
            </div>
          </div>
        </div>
      </header>

      <main className="main-content">
        <div className="tabs">
          <button
            className={`tab ${activeTab === 'files' ? 'active' : ''}`}
            onClick={() => setActiveTab('files')}
          >
            <FileText size={20} />
            Files
          </button>
          <button
            className={`tab ${activeTab === 'network' ? 'active' : ''}`}
            onClick={() => setActiveTab('network')}
          >
            <NetworkIcon size={20} />
            Network
          </button>
        </div>

        <div className="tab-content">
          {activeTab === 'files' ? <FileBrowser /> : <NetworkView />}
        </div>
      </main>
    </div>
  );
}

export default App;
