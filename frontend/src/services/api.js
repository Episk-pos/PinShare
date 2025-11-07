import axios from 'axios';

const API_BASE = '/api';

const api = axios.create({
  baseURL: API_BASE,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Files API
export const filesAPI = {
  // Get all files
  getAllFiles: async () => {
    const response = await api.get('/files');
    return response.data;
  },

  // Get specific file by SHA256
  getFile: async (fileSHA256) => {
    const response = await api.get(`/files/${fileSHA256}`);
    return response.data;
  },

  // Add or update file metadata
  updateFile: async (fileSHA256, metadata) => {
    const response = await api.put(`/files/${fileSHA256}`, metadata);
    return response.data;
  },

  // Add tag to file
  addTag: async (fileSHA256, tag) => {
    const response = await api.post(`/files/${fileSHA256}/tags`, { tag });
    return response.data;
  },

  // Remove tag from file
  removeTag: async (fileSHA256, tagName) => {
    const response = await api.delete(`/files/${fileSHA256}/tags/${tagName}`);
    return response.data;
  },

  // Vote for file removal
  voteRemoval: async (fileSHA256) => {
    const response = await api.post(`/files/${fileSHA256}/votes/removal`);
    return response.data;
  },
};

// P2P API
export const p2pAPI = {
  // Get P2P node status
  getStatus: async () => {
    const response = await api.get('/p2p/status');
    return response.data;
  },

  // Get connected peers
  getPeers: async () => {
    const response = await api.get('/p2p/peers');
    return response.data;
  },

  // Connect to a new peer
  connectPeer: async (multiaddress) => {
    const response = await api.post('/p2p/peers', { multiaddress });
    return response.data;
  },

  // Send message to peer
  sendMessage: async (peerID, message) => {
    const response = await api.post(`/p2p/peers/${peerID}/message`, { message });
    return response.data;
  },
};

export default api;
