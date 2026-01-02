#!/bin/bash

set -e

CONFIG_FILE="./config.yml"
DEPLOY_FILE="k8s/deployment.yaml"
IMAGE_TAG=$(date +%Y%m%d-%H%M%S)
IMAGE_FULL_NAME="bigtablelite:$IMAGE_TAG"
TEMP_DEPLOY_FILE="/tmp/bigtablelite-deployment-$IMAGE_TAG.yaml"

echo "Deploying BigTableLite to Kubernetes..."

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo "kubectl not found. Please install kubectl."
    exit 1
fi

echo "Building Docker image..."
echo "Using image tag: $IMAGE_FULL_NAME"

# Use Minikube’s Docker daemon
if kubectl config current-context | grep -q "minikube"; then
    echo "Switching Docker environment to Minikube..."
    eval $(minikube -p minikube docker-env)
fi

# Rebuild Docker image
docker build --no-cache -t $IMAGE_FULL_NAME .

# Detect cluster type and load image (for Kind)
if kubectl config current-context | grep -q "kind"; then
    echo "Loading image into Kind..."
    kind load docker-image $IMAGE_FULL_NAME
fi

# EXTRACT SHARD COUNT
SHARD_COUNT=$(grep '^shard_count:' "$CONFIG_FILE" | awk '{print $2}')
if [[ -z "$SHARD_COUNT" ]]; then SHARD_COUNT=1; fi

echo "Syncing Kubernetes replicas to shard_count: $SHARD_COUNT"

# Wipe the old state to avoid port conflicts with old pods
kubectl delete statefulset bigtablelite

# CREATE CONFIGMAP
kubectl create configmap my-go-config \
  --from-file=./config.yml \
  --from-file=./shard-config.yaml \
  --dry-run=client -o yaml | kubectl apply -f -

# PREPARE THE YAML
echo "Updating deployment YAML with replicas ($SHARD_COUNT) and tag ($IMAGE_TAG)"
sed "s/replicas: [0-9]*/replicas: $SHARD_COUNT/g" "$DEPLOY_FILE" | \
sed "s|bigtablelite:LATEST_BUILD|$IMAGE_FULL_NAME|g" > "$TEMP_DEPLOY_FILE"

echo "Deploying Kafka..."
kubectl apply -f k8s/kafka-deployment.yaml
kubectl wait --for=condition=ready pod -l app=kafka --timeout=120s

echo "Deploying Redis..."
kubectl apply -f k8s/redis-deployment.yaml
kubectl wait --for=condition=ready pod -l app=redis --timeout=60s

echo "Deploying BigTableLite..."
kubectl apply -f "$TEMP_DEPLOY_FILE"
kubectl apply -f k8s/service.yaml

echo "Waiting for BigTableLite pods to be ready..."
# Determine if we use rollout for Deployment or StatefulSet
if grep -q "StatefulSet" "$DEPLOY_FILE"; then
    kubectl rollout status statefulset/bigtablelite --timeout=180s
else
    kubectl rollout status deployment/bigtablelite --timeout=180s
fi

echo "Deploying Monitoring (Prometheus/Grafana)..."
kubectl apply -f k8s/prometheus-deployment.yaml
kubectl apply -f k8s/grafana-deployment.yaml

echo ""
echo "Deployment complete!"

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