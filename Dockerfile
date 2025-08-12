FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go module files
COPY go.mod ./
# Copy go.sum if it exists (projects without dependencies won't have this file)
COPY go.su[m] ./

# Download dependencies (no-op if no dependencies exist)
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o cachemir-server cmd/server/main.go

FROM alpine:latest

# Update package index and install ca-certificates
RUN apk update && apk --no-cache add ca-certificates && rm -rf /var/cache/apk/*

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/cachemir-server .

# Make sure the binary is executable
RUN chmod +x ./cachemir-server

EXPOSE 8080

CMD ["./cachemir-server"]
