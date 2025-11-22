# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o dbexporter \
    ./cmd/dbexporter/main.go

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata postgresql-client

# Create non-root user
RUN addgroup -g 1000 pme && \
    adduser -D -u 1000 -G pme pme

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/dbexporter .

# Copy migrations
COPY --from=builder /build/cmd/dbexporter/migrations ./migrations

# Change ownership
RUN chown -R pme:pme /app

# Switch to non-root user
USER pme

# Set environment variables with defaults
ENV KAFKA_URL=localhost:9092 \
    KAFKA_TOPIC=pme-ledger \
    DB_HOST=localhost \
    DB_PORT=5432 \
    DB_USER=pmeuser \
    DB_PASSWORD=pmepass \
    DB_NAME=pmedb \
    DB_SSLMODE=disable

# Run the application
CMD ["./dbexporter"]
