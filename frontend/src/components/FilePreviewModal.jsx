import { X } from 'lucide-react';
import { filesAPI } from '../services/api';

const FilePreviewModal = ({ file, onClose }) => {
  if (!file) return null;

  const contentURL = filesAPI.getFileContentURL(file.fileSHA256);

  const handleBackdropClick = (e) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  return (
    <div className="modal-backdrop" onClick={handleBackdropClick}>
      <div className="modal-content">
        <div className="modal-header">
          <h2>File Preview</h2>
          <button className="modal-close" onClick={onClose}>
            <X size={24} />
          </button>
        </div>
        <div className="modal-body">
          <div className="file-preview-info">
            <div><strong>Type:</strong> {file.fileType}</div>
            <div><strong>SHA256:</strong> <code>{file.fileSHA256}</code></div>
            {file.ipfsCID && (
              <div><strong>IPFS CID:</strong> <code>{file.ipfsCID}</code></div>
            )}
          </div>
          <div className="file-preview-container">
            {file.fileType === 'pdf' ? (
              <iframe
                src={contentURL}
                className="file-preview-iframe"
                title="File Preview"
              />
            ) : (
              <div className="file-preview-placeholder">
                <p>Preview not available for this file type.</p>
                <p>Use the download button to save the file.</p>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default FilePreviewModal;
