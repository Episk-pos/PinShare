import axios from 'axios'

const API_BASE = import.meta.env.VITE_API_BASE || ''  // Empty defaults to relative path

export const pinshareApi = axios.create({
  baseURL: `${API_BASE}/api/v1`,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Add auth or other interceptors later

export default pinshareApi