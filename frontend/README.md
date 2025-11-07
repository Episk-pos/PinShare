# PinShare Frontend

A modern React-based web interface for PinShare - a P2P file sharing system with IPFS integration.

## Features

### File Browser
- **View All Files**: Browse all files shared in the network with comprehensive metadata
- **Search & Filter**: Search files by SHA256 hash, IPFS CID, file type, or tags
- **Metadata Display**:
  - File type and SHA256 hash
  - IPFS Content Identifier (CID)
  - Upload and update timestamps
  - Tags with vote counts
  - Moderation votes
  - Ban status (Policy Violation, Indecent Content, Malware)
- **Real-time Updates**: Automatically refreshes file list every 10 seconds

### Network View
- **Interactive Network Diagram**: Visualize P2P connections using force-directed graph
  - Your node displayed in the center
  - Connected peers shown as nodes
  - Links representing connections
- **Network Statistics**:
  - Node status (online/offline)
  - Connected peer count
  - Node ID and addresses
- **Peer Information**:
  - List of all connected peers
  - Peer IDs and addresses
  - Real-time peer status updates every 5 seconds

## Technology Stack

- **React 18** - UI framework
- **Vite** - Build tool and dev server
- **Axios** - HTTP client for API communication
- **react-force-graph** - Network visualization
- **lucide-react** - Modern icon library
- **date-fns** - Date formatting utilities

## Prerequisites

- Node.js 18+ and npm
- PinShare backend running on port 9090

## Installation

1. Navigate to the frontend directory:
```bash
cd frontend
```

2. Install dependencies:
```bash
npm install
```

## Development

Start the development server:
```bash
npm run dev
```

The application will be available at `http://localhost:3000`

The dev server includes a proxy configuration that forwards API requests from `/api/*` to `http://localhost:9090/*`, so make sure your PinShare backend is running.

## Building for Production

Build the application:
```bash
npm run build
```

The built files will be in the `dist/` directory.

Preview the production build:
```bash
npm run preview
```

## API Endpoints Used

The frontend communicates with the following PinShare backend endpoints:

### Files API
- `GET /files` - Retrieve all file metadata
- `GET /files/{fileSHA256}` - Get specific file metadata
- `PUT /files/{fileSHA256}` - Update file metadata
- `POST /files/{fileSHA256}/tags` - Add tag to file
- `DELETE /files/{fileSHA256}/tags/{tagName}` - Remove tag from file
- `POST /files/{fileSHA256}/votes/removal` - Vote for file removal

### P2P API
- `GET /p2p/status` - Get P2P node status
- `GET /p2p/peers` - List connected peers
- `POST /p2p/peers` - Connect to new peer
- `POST /p2p/peers/{peerID}/message` - Send message to peer

## Project Structure

```
frontend/
├── src/
│   ├── components/
│   │   ├── FileBrowser.jsx      # File browsing and search
│   │   └── NetworkView.jsx      # Network visualization
│   ├── services/
│   │   └── api.js               # API client
│   ├── styles/
│   │   └── App.css              # Global styles
│   ├── App.jsx                  # Main application component
│   └── main.jsx                 # Application entry point
├── public/                      # Static assets
├── index.html                   # HTML template
├── vite.config.js              # Vite configuration
└── package.json                # Dependencies and scripts
```

## Configuration

### Vite Proxy Configuration

The Vite dev server is configured to proxy API requests to the backend. You can modify this in `vite.config.js`:

```javascript
server: {
  port: 3000,
  proxy: {
    '/api': {
      target: 'http://localhost:9090',
      changeOrigin: true,
      rewrite: (path) => path.replace(/^\/api/, '')
    }
  }
}
```

## UI Design

The interface features a modern dark theme with:
- **Color Scheme**: Dark blue/purple gradient with accent colors
- **Responsive Layout**: Works on desktop and mobile devices
- **Card-based Design**: Files displayed in grid cards with hover effects
- **Interactive Elements**: Smooth transitions and animations
- **Accessibility**: Clear contrast ratios and readable fonts

## Troubleshooting

### Backend Connection Issues

If you see errors about connecting to the backend:
1. Ensure PinShare backend is running: `go run main.go start`
2. Verify the backend is listening on port 9090
3. Check that the proxy configuration in `vite.config.js` is correct

### Network Diagram Not Showing

If the network diagram doesn't render:
1. Check browser console for errors
2. Ensure the P2P API endpoints are responding
3. Verify you have at least one peer connection

### Build Errors

If you encounter build errors:
1. Delete `node_modules/` and `package-lock.json`
2. Run `npm install` again
3. Clear the Vite cache: `rm -rf node_modules/.vite`

## Contributing

When contributing to the frontend:
1. Follow React best practices
2. Keep components modular and reusable
3. Add appropriate error handling
4. Test with various data scenarios
5. Ensure responsive design works on mobile

## License

This frontend is part of the PinShare project. See the main LICENSE file in the project root.
