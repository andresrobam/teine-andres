# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install dbmate for migrations
RUN apk add --no-cache curl && \
    curl -fsSL -o /usr/local/bin/dbmate https://github.com/amacneil/dbmate/releases/latest/download/dbmate-linux-amd64 && \
    chmod +x /usr/local/bin/dbmate

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o teine-andres .

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install ca-certificates for HTTPS connections
RUN apk add --no-cache ca-certificates

# Copy the binary from builder
COPY --from=builder /app/teine-andres .

# Copy dbmate from builder
COPY --from=builder /usr/local/bin/dbmate /usr/local/bin/dbmate

# Copy migrations directory
COPY --from=builder /app/db/migrations ./db/migrations

# Copy entrypoint script
COPY entrypoint.sh .
RUN chmod +x entrypoint.sh

# Set the entrypoint
ENTRYPOINT ["./entrypoint.sh"]
