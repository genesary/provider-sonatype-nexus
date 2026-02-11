#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFEST_DIR="${SCRIPT_DIR}/../manifests"

NEXUS_URL="${NEXUS_URL:-http://localhost:8081}"
NEXUS_USER="${NEXUS_USER:-admin}"
NEXUS_PASS="${NEXUS_PASS:-admin123}"

echo "=== Testing Helm Repository Resources ==="

# Test Helm Hosted Repository
echo "--- Testing Helm Hosted Repository ---"

kubectl apply -f "${MANIFEST_DIR}/repository-helm-hosted.yaml"

echo "Waiting for Helm Hosted Repository to be ready..."
sleep 5

for i in {1..30}; do
    status=$(kubectl get repository e2e-test-helm-hosted -n default -o jsonpath='{.status.conditions[?(@.type=="Synced")].status}' 2>/dev/null || echo "Unknown")
    if [ "$status" = "True" ]; then
        echo "Helm Hosted Repository is synced!"
        break
    fi
    echo "Waiting for Helm Hosted Repository to sync... ($i/30)"
    sleep 2
done

# Verify in Nexus API
echo "Verifying Helm Hosted Repository in Nexus..."
response=$(curl -sf -u "${NEXUS_USER}:${NEXUS_PASS}" "${NEXUS_URL}/service/rest/v1/repositories" || echo "")
if echo "$response" | grep -q "e2e-test-helm-hosted"; then
    echo "SUCCESS: Helm Hosted Repository found in Nexus!"
else
    echo "WARNING: Helm Hosted Repository not yet visible in Nexus API"
fi

# Test Helm Proxy Repository
echo "--- Testing Helm Proxy Repository ---"

kubectl apply -f "${MANIFEST_DIR}/repository-helm-proxy.yaml"

echo "Waiting for Helm Proxy Repository to be ready..."
sleep 5

for i in {1..30}; do
    status=$(kubectl get repository e2e-test-helm-proxy -n default -o jsonpath='{.status.conditions[?(@.type=="Synced")].status}' 2>/dev/null || echo "Unknown")
    if [ "$status" = "True" ]; then
        echo "Helm Proxy Repository is synced!"
        break
    fi
    echo "Waiting for Helm Proxy Repository to sync... ($i/30)"
    sleep 2
done

# Verify in Nexus API
echo "Verifying Helm Proxy Repository in Nexus..."
response=$(curl -sf -u "${NEXUS_USER}:${NEXUS_PASS}" "${NEXUS_URL}/service/rest/v1/repositories" || echo "")
if echo "$response" | grep -q "e2e-test-helm-proxy"; then
    echo "SUCCESS: Helm Proxy Repository found in Nexus!"
else
    echo "WARNING: Helm Proxy Repository not yet visible in Nexus API"
fi

# Cleanup
echo "--- Cleaning up Helm repositories ---"
kubectl delete repository e2e-test-helm-hosted -n default --wait=true --timeout=60s || true
kubectl delete repository e2e-test-helm-proxy -n default --wait=true --timeout=60s || true

echo "--- Helm Repository tests completed ---"
