# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies: g++, make, protobuf compiler, and other build tools
RUN apk add --no-cache build-base g++ make protobuf protobuf-dev

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Install protoc-gen-go and protoc-gen-go-grpc
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Copy proto files and generate code
COPY proto/ ./proto/
ENV PATH=$PATH:/root/go/bin
RUN protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/bigtablelite.proto

# Copy C++ source files
COPY sstable/ ./sstable/

# Build C++ SSTable library first
RUN make -C sstable

# Copy rest of the project
COPY . .

# Build Go binary (cgo must be enabled!)
RUN CGO_ENABLED=1 GOOS=linux go build -o bigtablelite ./cmd/server

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary
COPY --from=builder /app/bigtablelite .

# Create data directory for SSTable files
RUN mkdir -p /data

EXPOSE 50051 9090

# Run with SSTable backend by default, data directory mounted
CMD ["./bigtablelite", "-data-dir", "/data"]
