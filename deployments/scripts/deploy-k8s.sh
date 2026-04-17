#!/bin/bash
# Deploy ReignX to Kubernetes cluster

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"
NAMESPACE="${NAMESPACE:-reignx}"

echo "Deploying ReignX to Kubernetes (namespace: $NAMESPACE)..."

cd "$PROJECT_ROOT/deployments/kubernetes"

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo "Error: kubectl is not installed"
    exit 1
fi

# Check if cluster is accessible
if ! kubectl cluster-info &> /dev/null; then
    echo "Error: Cannot connect to Kubernetes cluster"
    exit 1
fi

# Create namespace
echo "Creating namespace..."
kubectl apply -f namespace.yaml

# Apply RBAC
echo "Applying RBAC..."
kubectl apply -f rbac.yaml

# Apply secrets (WARNING: Update these before production!)
echo "Applying secrets..."
kubectl apply -f secrets.yaml

# Apply ConfigMaps
echo "Applying ConfigMaps..."
kubectl apply -f configmap.yaml

# Deploy PostgreSQL
echo "Deploying PostgreSQL..."
kubectl apply -f postgres-statefulset.yaml

# Deploy etcd
echo "Deploying etcd..."
kubectl apply -f etcd-statefulset.yaml

# Deploy NATS
echo "Deploying NATS..."
kubectl apply -f nats-deployment.yaml

# Wait for dependencies to be ready
echo "Waiting for dependencies to be ready..."
kubectl wait --for=condition=ready pod -l app=postgres -n $NAMESPACE --timeout=300s
kubectl wait --for=condition=ready pod -l app=etcd -n $NAMESPACE --timeout=300s

# Deploy ReignX daemon
echo "Deploying ReignX daemon..."
kubectl apply -f reignxd-deployment.yaml

# Deploy monitoring
echo "Deploying monitoring stack..."
kubectl apply -f monitoring.yaml

# Wait for ReignX to be ready
echo "Waiting for ReignX to be ready..."
kubectl wait --for=condition=ready pod -l app=reignxd -n $NAMESPACE --timeout=300s

# Get service endpoints
echo ""
echo "✓ ReignX deployed successfully!"
echo ""
echo "Services:"
kubectl get svc -n $NAMESPACE

echo ""
echo "Pods:"
kubectl get pods -n $NAMESPACE

echo ""
echo "Access ReignX API:"
echo "  kubectl port-forward -n $NAMESPACE svc/reignxd-service 8080:8080"
echo ""
echo "Access Grafana:"
echo "  kubectl port-forward -n $NAMESPACE svc/grafana-service 3000:3000"
echo ""
echo "View logs:"
echo "  kubectl logs -n $NAMESPACE -l app=reignxd -f"
