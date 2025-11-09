# Build stage
FROM golang:latest AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY main.go ./

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o oauth-broker .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/oauth-broker .

# Expose port
EXPOSE 8888

# Run the binary
CMD ["./oauth-broker"]
