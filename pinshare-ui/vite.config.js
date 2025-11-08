import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig(({ mode }) => ({
  plugins: [react()],
  ...(mode === 'development' && {
    server: {
      port: 5174,
      proxy: {
        '/api': {
          target: process.env.VITE_PIN_SHARE_API || 'http://localhost:9090',
          changeOrigin: true,
          rewrite: (path) => path.replace(/^\/api/, '')
        },
        '/ipfs-api': {
          target: process.env.VITE_IPFS_API || 'http://localhost:5001',
          changeOrigin: true,
          rewrite: (path) => path.replace(/^\/ipfs-api/, '/api/v0')
        },
      },
    },
  }),
  define: {
    'import.meta.env.VITE_API_BASE': JSON.stringify(process.env.VITE_API_BASE || ''),
    'import.meta.env.VITE_IPFS_API_BASE': JSON.stringify(process.env.VITE_IPFS_API_BASE || '/ipfs-api'),
    'import.meta.env.VITE_GATEWAY_BASE': JSON.stringify(process.env.VITE_GATEWAY_BASE || 'http://localhost:8080/ipfs'),
  },
}))