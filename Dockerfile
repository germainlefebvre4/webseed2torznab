# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY main.go ./

# Build the application
RUN go build -o webseed2torznab main.go

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/webseed2torznab .

# Create torrents directory
RUN mkdir -p /torrents

# Expose port
EXPOSE 8080

# Set default environment variables
ENV PORT=8080
ENV BASE_URL=http://localhost:8080

# Run the application
CMD ["./webseed2torznab", "/torrents"]
