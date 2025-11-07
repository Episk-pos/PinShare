import { useState, useEffect } from 'react';
import { filesAPI } from '../services/api';
import { Search, FileText, Calendar, Tag, AlertTriangle } from 'lucide-react';
import { formatDistanceToNow } from 'date-fns';

const FileBrowser = () => {
  const [files, setFiles] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [searchTerm, setSearchTerm] = useState('');

  useEffect(() => {
    fetchFiles();
    // Poll for updates every 10 seconds
    const interval = setInterval(fetchFiles, 10000);
    return () => clearInterval(interval);
  }, []);

  const fetchFiles = async () => {
    try {
      const data = await filesAPI.getAllFiles();
      // Convert the response to an array if it's an object
      const filesArray = Array.isArray(data) ? data : Object.values(data);
      setFiles(filesArray);
      setError(null);
    } catch (err) {
      console.error('Error fetching files:', err);
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const filteredFiles = files.filter(file => {
    if (!searchTerm) return true;

    const searchLower = searchTerm.toLowerCase();
    const matchesHash = file.fileSHA256?.toLowerCase().includes(searchLower);
    const matchesCID = file.ipfsCID?.toLowerCase().includes(searchLower);
    const matchesType = file.fileType?.toLowerCase().includes(searchLower);
    const matchesTags = file.tags && Object.keys(file.tags).some(tag =>
      tag.toLowerCase().includes(searchLower)
    );

    return matchesHash || matchesCID || matchesType || matchesTags;
  });

  const getBanLabel = (banSet) => {
    if (banSet === 3) return 'Policy Violation';
    if (banSet === 6) return 'Indecent Content';
    if (banSet === 9) return 'Malware';
    return 'Banned';
  };

  if (loading) {
    return (
      <div className="loading">
        <div className="spinner"></div>
        <p>Loading files...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="error-state">
        <h3>Error Loading Files</h3>
        <p>{error}</p>
        <p style={{ marginTop: '1rem', fontSize: '0.9rem' }}>
          Make sure the PinShare backend is running on port 9090
        </p>
      </div>
    );
  }

  return (
    <div>
      <div className="search-bar">
        <Search className="search-icon" size={20} />
        <input
          type="text"
          className="search-input"
          placeholder="Search by hash, CID, file type, or tags..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
        />
      </div>

      {filteredFiles.length === 0 ? (
        <div className="empty-state">
          <FileText size={48} style={{ opacity: 0.5, margin: '0 auto 1rem' }} />
          <h3>No files found</h3>
          <p>
            {searchTerm ? 'Try adjusting your search terms' : 'Upload some files to get started'}
          </p>
        </div>
      ) : (
        <>
          <div style={{ marginBottom: '1rem', color: '#a0a0a0' }}>
            Showing {filteredFiles.length} of {files.length} files
          </div>
          <div className="file-grid">
            {filteredFiles.map((file) => (
              <div key={file.fileSHA256} className="file-card">
                <div className="file-card-header">
                  <span className="file-type">{file.fileType || 'Unknown'}</span>
                  {file.banSet > 0 && (
                    <span className="ban-badge">
                      <AlertTriangle size={12} style={{ marginRight: '4px' }} />
                      {getBanLabel(file.banSet)}
                    </span>
                  )}
                </div>

                <div className="file-hash">
                  <strong>SHA256:</strong>
                  <br />
                  {file.fileSHA256}
                </div>

                {file.ipfsCID && (
                  <div className="file-hash">
                    <strong>IPFS CID:</strong>
                    <br />
                    {file.ipfsCID}
                  </div>
                )}

                <div className="file-info">
                  <div className="file-info-row">
                    <span style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                      <Calendar size={14} />
                      Added
                    </span>
                    <span>
                      {file.addedAt
                        ? formatDistanceToNow(new Date(file.addedAt), { addSuffix: true })
                        : 'Unknown'}
                    </span>
                  </div>

                  {file.lastUpdated && (
                    <div className="file-info-row">
                      <span>Last Updated</span>
                      <span>
                        {formatDistanceToNow(new Date(file.lastUpdated), { addSuffix: true })}
                      </span>
                    </div>
                  )}

                  {file.moderationVotes !== undefined && file.moderationVotes !== 0 && (
                    <div className="file-info-row">
                      <span>Moderation Votes</span>
                      <span style={{ color: file.moderationVotes > 0 ? '#e74c3c' : '#667eea' }}>
                        {file.moderationVotes}
                      </span>
                    </div>
                  )}
                </div>

                {file.tags && Object.keys(file.tags).length > 0 && (
                  <div className="file-tags">
                    {Object.entries(file.tags).map(([tag, count]) => (
                      <span key={tag} className="tag">
                        <Tag size={12} />
                        {tag}
                        {count > 1 && ` (${count})`}
                      </span>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  );
};

export default FileBrowser;
