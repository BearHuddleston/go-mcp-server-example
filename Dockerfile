# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app

# Copy go mod and sum files
COPY go.mod ./
# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o mcp-server ./cmd/mcpserver

# Final stage
FROM alpine:latest

# Install CA certificates for HTTPS support
RUN apk --no-cache add ca-certificates

# Set the working directory
WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/mcp-server .

# Expose port 8080 for HTTP transport
EXPOSE 8080

# Command to run the executable
CMD ["./mcp-server", "--transport", "http", "--port", "8080"]
