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

# Define a unique, dynamic tag for this build
IMAGE_TAG=$(date +%Y%m%d-%H%M%S)
IMAGE_FULL_NAME="bigtablelite:$IMAGE_TAG"
echo "Using image tag: $IMAGE_FULL_NAME"

# Use Minikube’s Docker daemon so images are visible inside the cluster
if kubectl config current-context | grep -q "minikube"; then
    echo "Switching Docker environment to Minikube..."
    eval $(minikube -p minikube docker-env)
    if [ $? -ne 0 ]; then
        echo "Error: minikube docker-env setup failed. Aborting."
        exit 1
    fi
fi

# Rebuild Docker image without cache to ensure new code is used
docker build --no-cache -t $IMAGE_FULL_NAME .

# Detect cluster type and load image
if kubectl config current-context | grep -q "minikube"; then
    echo "Image built directly into Minikube's Docker daemon."
elif kubectl config current-context | grep -q "kind"; then
    echo "Loading image into Kind..."
    kind load docker-image $IMAGE_FULL_NAME
else
    echo "Unknown cluster type. Making sure the image is available in your cluster."
fi

# create ConfigMap from config.yml (ConfigMap is created first)
kubectl create configmap my-go-config --from-file=./config.yml --dry-run=client -o yaml | kubectl apply -f -

echo "Deploying Redis..."
kubectl apply -f k8s/redis-deployment.yaml

echo "Waiting for Redis to be ready..."
kubectl wait --for=condition=ready pod -l app=redis --timeout=60s

# Inject the new image tag into the deployment YAML
echo "Updating deployment YAML with new tag: $IMAGE_TAG"
DEPLOY_FILE="k8s/deployment.yaml"
TEMP_DEPLOY_FILE="/tmp/bigtablelite-deployment-$IMAGE_TAG.yaml"

# Replace the LATEST_BUILD placeholder with the dynamic tag
sed "s|bigtablelite:LATEST_BUILD|$IMAGE_FULL_NAME|g" $DEPLOY_FILE > $TEMP_DEPLOY_FILE

echo "Deploying BigTableLite..."
kubectl apply -f $TEMP_DEPLOY_FILE
kubectl apply -f k8s/service.yaml

echo "Waiting for BigTableLite pods to be ready..."
kubectl rollout status deployment/bigtablelite --timeout=180s

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