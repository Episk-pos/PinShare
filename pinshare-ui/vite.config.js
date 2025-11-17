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
  // Vite automatically loads .env files based on mode (development, production, etc.)
  // No need to manually define env vars here - they're available via import.meta.env.VITE_*
}))