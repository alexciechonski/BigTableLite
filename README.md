# BigTableLite MVP

A fault-tolerant, observable, and concurrent key-value store service built with Go, gRPC, Redis, Kubernetes, and Prometheus.

## Features

- **gRPC API**: High-performance SET/GET operations
- **Concurrent Handling**: Leverages Go goroutines for handling many simultaneous requests
- **Persistent Storage**: Uses Redis for distributed storage
- **Containerized**: Single Docker image deployment
- **Kubernetes Orchestration**: 3 replicas with load balancing and health probes
- **Observability**: Prometheus metrics for request count and latency

## Tech Stack

- **Go (Golang)**: Core application language
- **gRPC**: API protocol
- **Redis**: Backend storage
- **Docker**: Containerization
- **Kubernetes**: Orchestration
- **Prometheus**: Metrics collection
- **Grafana**: Metrics visualization

## Prerequisites

- Go 1.21 or later
- Docker
- Kubernetes cluster (Minikube or Kind)
- `protoc` (Protocol Buffers compiler)
- `protoc-gen-go` and `protoc-gen-go-grpc` plugins

## Quick Start

### 1. Install Dependencies

```bash
# Install Go dependencies
make deps

# Or manually:
go mod download
go mod tidy
```

### 2. Generate Protobuf Code

```bash
make proto
```

### 3. Build the Application

```bash
make build
```

### 4. Run Locally (with Redis)

First, start Redis:

```bash
docker run -d -p 6379:6379 redis:7-alpine
```

Then run the service:

```bash
make run
# Or:
./bigtablelite -redis-addr localhost:6379
```

The service will start:
- gRPC server on port `50051`
- Prometheus metrics on port `9090` at `/metrics`

## Docker Build

Build the Docker image:

```bash
make docker-build
# Or:
docker build -t bigtablelite:latest .
```

## Kubernetes Deployment

### Prerequisites

Ensure you have a Kubernetes cluster running (Minikube or Kind):

```bash
# For Minikube
minikube start

# For Kind
kind create cluster
```

### Deploy to Kubernetes

1. **Build and load the Docker image** (for local clusters):

```bash
# Build the image
make docker-build

# Load into Minikube
minikube image load bigtablelite:latest

# Or for Kind
kind load docker-image bigtablelite:latest
```

2. **Deploy Redis**:

```bash
kubectl apply -f k8s/redis-deployment.yaml
```

3. **Deploy BigTableLite**:

```bash
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
```

4. **Deploy Prometheus** (optional, for monitoring):

```bash
kubectl apply -f k8s/prometheus-deployment.yaml
```

5. **Deploy Grafana** (optional, for visualization):

```bash
kubectl apply -f k8s/grafana-deployment.yaml
```

### Verify Deployment

```bash
# Check pods
kubectl get pods

# Check services
kubectl get services

# Check logs
kubectl logs -l app=bigtablelite --tail=50
```

### Access Services

```bash
# Get service URLs
minikube service bigtablelite-service --url
minikube service prometheus-service --url
minikube service grafana-service --url
```

## Testing the API

### Using grpcurl (recommended)

Install `grpcurl`:

```bash
# macOS
brew install grpcurl

# Or download from: https://github.com/fullstorydev/grpcurl
```

Test the API:

```bash
# Set a value
grpcurl -plaintext -d '{"key": "test", "value": "hello world"}' \
  localhost:50051 bigtablelite.BigTableLite/Set

# Get a value
grpcurl -plaintext -d '{"key": "test"}' \
  localhost:50051 bigtablelite.BigTableLite/Get
```

### Using a Go client

Create a simple test client:

```go
package main

import (
    "context"
    "log"
    "bigtablelite/proto"
    "google.golang.org/grpc"
)

func main() {
    conn, _ := grpc.Dial("localhost:50051", grpc.WithInsecure())
    client := proto.NewBigTableLiteClient(conn)
    
    // Set
    client.Set(context.Background(), &proto.SetRequest{
        Key: "test", Value: "hello",
    })
    
    // Get
    resp, _ := client.Get(context.Background(), &proto.GetRequest{
        Key: "test",
    })
    log.Println(resp)
}
```

## Monitoring

### Prometheus Metrics

The service exposes the following metrics at `http://localhost:9090/metrics`:

- `bigtablelite_requests_total`: Total number of requests (labeled by method and status)
- `bigtablelite_request_duration_seconds`: Request latency histogram (labeled by method)

### View Metrics

```bash
# Local
curl http://localhost:9090/metrics

# In Kubernetes
kubectl port-forward svc/bigtablelite-service 9090:9090
curl http://localhost:9090/metrics
```

### Grafana Setup

1. Access Grafana at the service URL (default: http://localhost:3000)
2. Login with `admin` / `admin`
3. Add Prometheus as a data source:
   - URL: `http://prometheus-service:9090`
4. Create dashboards to visualize:
   - Request rate
   - Request latency (p50, p95, p99)
   - Error rate

## Configuration

The service accepts the following command-line flags:

- `-grpc-port`: gRPC server port (default: `50051`)
- `-metrics-port`: Prometheus metrics port (default: `9090`)
- `-redis-addr`: Redis address (default: `localhost:6379`)

## Development

### Run Tests

```bash
make test
```

### Clean Build Artifacts

```bash
make clean
```

## API Reference

### Set

Stores a key-value pair.

**Request:**
```protobuf
message SetRequest {
  string key = 1;
  string value = 2;
}
```

**Response:**
```protobuf
message SetResponse {
  bool success = 1;
  string message = 2;
}
```

### Get

Retrieves a value by key.

**Request:**
```protobuf
message GetRequest {
  string key = 1;
}
```

**Response:**
```protobuf
message GetResponse {
  bool found = 1;
  string value = 2;
  string message = 3;
}
```

## Troubleshooting

### Redis Connection Issues

Ensure Redis is running and accessible:

```bash
# Check Redis connection
redis-cli ping

# In Kubernetes, check Redis service
kubectl get svc redis-service
kubectl logs -l app=redis
```

### gRPC Connection Issues

Verify the service is running:

```bash
# Check if port is listening
lsof -i :50051

# In Kubernetes
kubectl get pods -l app=bigtablelite
kubectl logs -l app=bigtablelite
```

### Metrics Not Available

Check the metrics endpoint:

```bash
curl http://localhost:9090/metrics
```

