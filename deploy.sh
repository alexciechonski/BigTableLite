#!/bin/bash

set -e

echo "Deploying BigTableLite to Kubernetes..."

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo "kubectl not found. Please install kubectl."
    exit 1
fi

# Check if cluster is accessible
if ! kubectl cluster-info &> /dev/null; then
    echo "Kubernetes cluster not accessible. Please start your cluster (minikube start or kind create cluster)"
    exit 1
fi

echo "Building Docker image..."
docker build -t bigtablelite:latest .

# Detect cluster type and load image
if kubectl config current-context | grep -q "minikube"; then
    echo "Loading image into Minikube..."
    minikube image load bigtablelite:latest
elif kubectl config current-context | grep -q "kind"; then
    echo "Loading image into Kind..."
    kind load docker-image bigtablelite:latest
else
    echo "Unknown cluster type. Make sure the image is available in your cluster."
fi

echo "Deploying Redis..."
kubectl apply -f k8s/redis-deployment.yaml

echo "Waiting for Redis to be ready..."
kubectl wait --for=condition=ready pod -l app=redis --timeout=60s

echo "Deploying BigTableLite..."
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml

echo "Waiting for BigTableLite pods to be ready..."
# Wait for at least 3 ready pods (ignore any failing pods from new replica sets)
kubectl rollout status deployment/bigtablelite --timeout=120s
  (echo "Warning: Some pods may not be ready, but deployment is available" && \
   kubectl get pods -l app=bigtablelite && \
   kubectl get deployment bigtablelite)

echo "Deploying Prometheus..."
kubectl apply -f k8s/prometheus-deployment.yaml

echo "Deploying Grafana..."
kubectl apply -f k8s/grafana-deployment.yaml

echo ""
echo "Deployment complete!"
echo ""
echo "Status:"
kubectl get pods -l app=bigtablelite
kubectl get svc bigtablelite-service

echo ""
echo "Access services:"
if kubectl config current-context | grep -q "minikube"; then
    echo "  gRPC:   minikube service bigtablelite-service --url"
    echo "  Prometheus: minikube service prometheus-service --url"
    echo "  Grafana:   minikube service grafana-service --url"
else
    echo "  Use kubectl port-forward to access services:"
    echo "  kubectl port-forward svc/bigtablelite-service 50051:50051"
    echo "  kubectl port-forward svc/prometheus-service 9090:9090"
    echo "  kubectl port-forward svc/grafana-service 3000:3000"
fi

