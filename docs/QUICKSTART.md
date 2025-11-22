# Quick Start Guide

## Prerequisites Check

```bash
# Check Go version (need 1.21+)
go version

# Check Docker
docker --version

# Check Kubernetes (Minikube or Kind)
kubectl version --client
minikube version  # or: kind version
```

## Local Development

### 1. Install Dependencies

```bash
make deps
```

### 2. Generate Protobuf Code

```bash
make proto
```

### 3. Start Redis

```bash
docker run -d -p 6379:6379 --name redis redis:7-alpine
```

### 4. Run the Service

```bash
make run
```

### 5. Test the API

In another terminal:

```bash
# Using the example client
go run examples/client.go -op set -key hello -value world
go run examples/client.go -op get -key hello

# Or using grpcurl
grpcurl -plaintext -d '{"key": "test", "value": "hello"}' \
  localhost:50051 bigtablelite.BigTableLite/Set

grpcurl -plaintext -d '{"key": "test"}' \
  localhost:50051 bigtablelite.BigTableLite/Get
```

### 6. Check Metrics

```bash
curl http://localhost:9090/metrics
```

## Kubernetes Deployment

### Option 1: Using the Deployment Script

```bash
# Start your cluster first
minikube start
# or
kind create cluster

# Deploy everything
./deploy.sh
```

### Option 2: Manual Deployment

```bash
# 1. Build and load image
docker build -t bigtablelite:latest .
minikube image load bigtablelite:latest  # or: kind load docker-image bigtablelite:latest

# 2. Deploy Redis
kubectl apply -f k8s/redis-deployment.yaml

# 3. Deploy BigTableLite
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml

# 4. Deploy monitoring (optional)
kubectl apply -f k8s/prometheus-deployment.yaml
kubectl apply -f k8s/grafana-deployment.yaml

# 5. Check status
kubectl get pods
kubectl get svc
```

### Access Services

```bash
# Port forward to access locally
kubectl port-forward svc/bigtablelite-service 50051:50051
kubectl port-forward svc/prometheus-service 9090:9090
kubectl port-forward svc/grafana-service 3000:3000

# Or with Minikube
minikube service bigtablelite-service
```
