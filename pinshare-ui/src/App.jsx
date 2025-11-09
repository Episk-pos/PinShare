import { createRoot } from 'react-dom/client'
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom'
import { Link } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Toaster } from 'react-hot-toast'
import Browse from './pages/Browse.jsx'
import NetworkDashboard from './pages/Network.jsx'
import NetworkGraph from './pages/NetworkGraph.jsx'
import UploadStatus from './pages/UploadStatus.jsx'
import StatsHeader from './components/StatsHeader.jsx'
import './index.css'

const queryClient = new QueryClient()

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <Router>
        <div className="min-h-screen bg-gray-100">
          <header className="bg-white shadow">
            <div className="max-w-7xl mx-auto px-4 py-6">
              <div className="flex justify-between items-center">
                <div>
                  <h1 className="text-3xl font-bold text-gray-900">PinShare Dashboard</h1>
                  <p className="mt-1 text-sm text-gray-500">Browse, search, and manage shared files from the P2P network.</p>
                </div>
                <div className="flex items-center space-x-8">
                  <StatsHeader />
                  <nav className="flex items-center space-x-4">
                    <Link to="/" className="text-blue-600 hover:text-blue-800 font-medium">Browse</Link>
                    <Link to="/upload-status" className="text-blue-600 hover:text-blue-800 font-medium">Upload Status</Link>
                    <Link to="/network" className="text-blue-600 hover:text-blue-800 font-medium">Network Info</Link>
                    <Link to="/network-graph" className="text-blue-600 hover:text-blue-800 font-medium">Network Graph</Link>
                  </nav>
                </div>
              </div>
            </div>
          </header>
          <main className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
            <Routes>
              <Route path="/" element={<Browse />} />
              <Route path="/upload-status" element={<UploadStatus />} />
              <Route path="/network" element={<NetworkDashboard />} />
              <Route path="/network-graph" element={<NetworkGraph />} />
            </Routes>
          </main>
          <Toaster position="top-right" />
        </div>
      </Router>
    </QueryClientProvider>
  )
}

export default App