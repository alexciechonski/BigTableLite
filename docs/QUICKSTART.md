BigTableLite — Full Quick Start & Kubernetes Guide

Prerequisites

    go version          # Go 1.21+
    docker --version
    kubectl version --client
    minikube version    # or kind version

------------------------------------------------------------------------

Local Development

1. Install Dependencies

    make deps

2. Generate Protobuf Code

    make proto

3. Start Redis (optional unless using Redis backend)

    docker run -d -p 6379:6379 --name redis redis:7-alpine

4. Run the Service Locally

    make run

5. Test Using Client

    go run pkg/client/client.go -op set -key hello -value world
    go run pkg/client/client.go -op get -key hello

6. Check Metrics

    curl http://localhost:9090/metrics

------------------------------------------------------------------------

Kubernetes Deployment Guide

IMPORTANT: Build Image Inside Minikube

    eval $(minikube -p minikube docker-env)
    docker build -t bigtablelite:latest .

1. Apply Redis

    kubectl apply -f k8s/redis-deployment.yaml

2. Deploy BigTableLite

    kubectl apply -f k8s/bigtable-deployment.yaml
    kubectl apply -f k8s/bigtable-service.yaml

3. Deploy Prometheus & Grafana

    kubectl apply -f k8s/prometheus-deployment.yaml
    kubectl apply -f k8s/grafana-deployment.yaml

------------------------------------------------------------------------

Verifying Deployment

    kubectl get pods
    kubectl get svc

------------------------------------------------------------------------

Accessing Services

gRPC API

    kubectl port-forward svc/bigtablelite-service 50051:50051

Prometheus

    kubectl port-forward svc/prometheus-service 9090:9090
    open http://localhost:9090/targets

Grafana

    kubectl port-forward svc/grafana-service 3000:3000
    open http://localhost:3000
    # Default password: admin / admin

------------------------------------------------------------------------

Troubleshooting Guide

1. Verify Service Exposes Both Ports

    kubectl get svc bigtablelite-service -o yaml

Expected:

    ports:
      - name: grpc
        port: 50051
      - name: metrics
        port: 9090

2. Verify Deployment Exposes containerPort 9090

    ports:
      - containerPort: 50051
      - containerPort: 9090

3. Check Endpoints

    kubectl get endpoints bigtablelite-service -o yaml

Must contain both ports.

4. If port-forward stops working, restart:

    kubectl port-forward svc/bigtablelite-service 50051:50051
