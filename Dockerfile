# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build facilitator
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o facilitator ./cmd/facilitator

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/facilitator .

# Expose port
EXPOSE 8080

# Run facilitator
CMD ["./facilitator"]
