# Nebula Server Dockerfile
# Multi-stage build for minimal image size

# Build stage
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build server
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o nebula-server ./cmd/nebula-server

# Build CLI
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o nebula ./cmd/nebula

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata git

# Create non-root user
RUN adduser -D -h /app nebula

WORKDIR /app

# Copy binaries
COPY --from=builder /build/nebula-server /app/
COPY --from=builder /build/nebula /usr/local/bin/

# Create data directories
RUN mkdir -p /data/apps /data/builds /data/compose /data/databases && \
    chown -R nebula:nebula /data /app

USER nebula

EXPOSE 8080

ENV NEBULA_DATA_DIR=/data
ENV NEBULA_DB_PATH=/data/nebula.db

ENTRYPOINT ["/app/nebula-server"]
