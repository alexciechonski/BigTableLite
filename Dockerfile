# Build stage
FROM golang:1.21-alpine AS builder

# Install protobuf compiler
# RUN apk add --no-cache protobuf protoc-gen-go protoc-gen-go-grpc

WORKDIR /app

# Copy go mod files
COPY go.mod ./

# Download dependencies
RUN go mod download
RUN go mod tidy

# Copy source code
COPY . .

# Generate protobuf code
RUN protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/bigtablelite.proto

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bigtablelite .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/bigtablelite .

# Expose gRPC and metrics ports
EXPOSE 50051 9090

# Run the application
# Environment variables can be set via Kubernetes deployment
CMD ["./bigtablelite"]

