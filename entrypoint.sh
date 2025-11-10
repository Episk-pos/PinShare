#!/bin/bash
/usr/local/bin/start_ipfs daemon --migrate=true --agent-version-suffix=docker &

# Wait for IPFS to start
sleep 10

# Configure IPFS API CORS for browser access
ipfs config --json API.HTTPHeaders.Access-Control-Allow-Origin '["http://localhost:5174", "http://localhost:5173", "http://localhost:3000"]'
ipfs config --json API.HTTPHeaders.Access-Control-Allow-Methods '["GET", "POST", "PUT"]'
ipfs config --json API.HTTPHeaders.Access-Control-Allow-Headers '["Authorization", "Content-Type"]'
ipfs config --json API.HTTPHeaders.Access-Control-Expose-Headers '["Location"]'
ipfs config --json API.HTTPHeaders.Access-Control-Allow-Credentials '["true"]'

echo "[INFO] IPFS CORS configured for browser access"

# Wait for remaining startup time
sleep 50

/opt/pinshare/bin/pinshare
