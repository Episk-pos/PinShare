import axios from 'axios'

const API_BASE = import.meta.env.VITE_API_BASE || '/api'  // Proxy for both dev/prod

export const pinshareApi = axios.create({
  baseURL: API_BASE,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Add auth or other interceptors later

export default pinshareApi