# Configuration variables
ARG GO_VERSION=1.24.3
ARG ALPINE_VERSION=3.22

# Build stage
FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
RUN mkdir -p cmd/termfolio
COPY cmd/termfolio/main.go cmd/termfolio
COPY internal ./internal

# Compile application
RUN CGO_ENABLED=0 GOOS=linux go build -o termfolio ./cmd/termfolio

# Runtime stage
ARG ALPINE_VERSION
FROM alpine:${ALPINE_VERSION}

WORKDIR /work

# Copy binary from builder
COPY --from=builder /app/termfolio /usr/local/bin/termfolio

# Set entrypoint
ENTRYPOINT ["splitans"]

# Default arguments (can be overridden)
CMD []
