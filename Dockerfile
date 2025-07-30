# Linux Dockerfile for aks-mcp
# Build stage
FROM golang:1.24-alpine AS builder
ARG TARGETOS=linux
ARG TARGETARCH
ARG VERSION
ARG GIT_COMMIT
ARG BUILD_DATE
ARG GIT_TREE_STATE

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application for target platform
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o aks-mcp ./cmd/aks-mcp

# Runtime stage
FROM alpine:3.22

# Install required packages for kubectl and helm, plus build tools for Azure CLI
RUN apk add --no-cache curl bash openssl ca-certificates git python3 py3-pip \
    gcc python3-dev musl-dev linux-headers

# Install Azure CLI
RUN pip3 install --break-system-packages azure-cli

# Create the mcp user and group
RUN addgroup -S mcp && \
    adduser -S -G mcp -h /home/mcp mcp && \
    mkdir -p /home/mcp/.kube && \
    chown -R mcp:mcp /home/mcp

# Copy binary from builder
COPY --from=builder /app/aks-mcp /usr/local/bin/aks-mcp

# Set working directory
WORKDIR /home/mcp

# Expose the default port for sse/streamable-http transports
EXPOSE 8000

# Switch to non-root user
USER mcp

# Set environment variables
ENV HOME=/home/mcp

# Command to run
ENTRYPOINT ["/usr/local/bin/aks-mcp"]
CMD ["--transport", "streamable-http", "--host", "0.0.0.0"]
