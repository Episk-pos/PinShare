#!/bin/sh

# Chown data dir to host user (1000:1000 default)
chown -R 1000:1000 /opt/pinshare/data || true

# Run original entrypoint (IPFS + pinshare)
exec /opt/pinshare/bin/entrypoint.sh