# Build stage - simplified without IPFS
FROM golang:latest as builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY main.go .
COPY internal ./internal

# Build binary
RUN go build -o pinshare .

# Runtime stage
FROM debian:stable-slim

# Install runtime dependencies
RUN set -eux; \
	apt-get update; \
	apt-get install -y \
		ca-certificates \
		chromium \
		clamav \
		clamav-daemon \
		clamav-freshclam \
		clamdscan \
		wget \
	; \
	rm -rf /var/lib/apt/lists/*

# Install IPFS CLI (kubo) for connecting to external IPFS API
# We only need the ipfs command, not the daemon
RUN set -eux; \
	KUBO_VERSION=v0.31.0; \
	ARCH=$(uname -m); \
	case $ARCH in \
		x86_64) ARCH=amd64 ;; \
		aarch64) ARCH=arm64 ;; \
	esac; \
	wget -q "https://dist.ipfs.tech/kubo/${KUBO_VERSION}/kubo_${KUBO_VERSION}_linux-${ARCH}.tar.gz"; \
	tar -xzf "kubo_${KUBO_VERSION}_linux-${ARCH}.tar.gz"; \
	cd kubo; \
	./install.sh; \
	cd ..; \
	rm -rf kubo "kubo_${KUBO_VERSION}_linux-${ARCH}.tar.gz"

# Create application directory
RUN mkdir -p /opt/pinshare/bin /opt/pinshare/data

# Copy binary from builder
COPY --from=builder /build/pinshare /opt/pinshare/bin/pinshare

# Set working directory
WORKDIR /opt/pinshare/data

# Add to PATH
ENV PATH=/opt/pinshare/bin:$PATH

# Expose ports
EXPOSE 9090 50001

# Run the application directly
CMD ["pinshare"]
