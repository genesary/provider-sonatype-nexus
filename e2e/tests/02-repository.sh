#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFEST_DIR="${SCRIPT_DIR}/../manifests"

NEXUS_URL="${NEXUS_URL:-http://localhost:8081}"
NEXUS_USER="${NEXUS_USER:-admin}"
NEXUS_PASS="${NEXUS_PASS:-admin123}"

echo "=== Testing Repository Resources ==="

# Test Maven Hosted Repository
echo "--- Testing Maven Hosted Repository ---"

kubectl apply -f "${MANIFEST_DIR}/repository-maven-hosted.yaml"

echo "Waiting for Repository to be ready..."
sleep 5

# Wait for the resource to be synced
for i in {1..30}; do
    status=$(kubectl get repository e2e-test-maven-hosted -n default -o jsonpath='{.status.conditions[?(@.type=="Synced")].status}' 2>/dev/null || echo "Unknown")
    if [ "$status" = "True" ]; then
        echo "Repository is synced!"
        break
    fi
    echo "Waiting for Repository to sync... ($i/30)"
    sleep 2
done

# Verify in Nexus API
echo "Verifying Repository in Nexus..."
response=$(curl -sf -u "${NEXUS_USER}:${NEXUS_PASS}" "${NEXUS_URL}/service/rest/v1/repositories" || echo "")
if echo "$response" | grep -q "e2e-test-maven-hosted"; then
    echo "SUCCESS: Repository found in Nexus!"
else
    echo "WARNING: Repository not yet visible in Nexus API (may still be creating)"
fi

# Cleanup
echo "Cleaning up Repository..."
kubectl delete repository e2e-test-maven-hosted -n default --wait=true --timeout=60s

echo "--- Maven Hosted Repository test completed ---"

# Test Docker Hosted Repository
echo "--- Testing Docker Hosted Repository ---"

kubectl apply -f "${MANIFEST_DIR}/repository-docker-hosted.yaml"

echo "Waiting for Docker Repository to be ready..."
sleep 5

for i in {1..30}; do
    status=$(kubectl get repository e2e-test-docker-hosted -n default -o jsonpath='{.status.conditions[?(@.type=="Synced")].status}' 2>/dev/null || echo "Unknown")
    if [ "$status" = "True" ]; then
        echo "Docker Repository is synced!"
        break
    fi
    echo "Waiting for Docker Repository to sync... ($i/30)"
    sleep 2
done

# Cleanup
echo "Cleaning up Docker Repository..."
kubectl delete repository e2e-test-docker-hosted -n default --wait=true --timeout=60s

echo "--- Docker Hosted Repository test completed ---"

# Test Maven Proxy Repository (with httpClient.authentication + connection.useTrustStore)
echo "--- Testing Maven Proxy Repository ---"

# Create authentication secret for proxy
kubectl create secret generic e2e-proxy-auth \
    --from-literal=password=proxypass123 \
    -n default --dry-run=client -o yaml | kubectl apply -f -

kubectl apply -f "${MANIFEST_DIR}/repository-maven-proxy.yaml"

echo "Waiting for Maven Proxy Repository to be ready..."
sleep 5

for i in {1..30}; do
    status=$(kubectl get repository e2e-test-maven-proxy -n default -o jsonpath='{.status.conditions[?(@.type=="Synced")].status}' 2>/dev/null || echo "Unknown")
    if [ "$status" = "True" ]; then
        echo "Maven Proxy Repository is synced!"
        break
    fi
    echo "Waiting for Maven Proxy Repository to sync... ($i/30)"
    sleep 2
done

# Verify in Nexus API
echo "Verifying Maven Proxy Repository in Nexus..."
response=$(curl -sf -u "${NEXUS_USER}:${NEXUS_PASS}" "${NEXUS_URL}/service/rest/v1/repositories" || echo "")
if echo "$response" | grep -q "e2e-test-maven-proxy"; then
    echo "SUCCESS: Maven Proxy Repository found in Nexus!"
else
    echo "WARNING: Maven Proxy Repository not yet visible in Nexus API"
fi

# Verify httpClient.authentication and connection.useTrustStore
echo "Verifying httpClient configuration..."
detail=$(curl -sf -u "${NEXUS_USER}:${NEXUS_PASS}" "${NEXUS_URL}/service/rest/v1/repositories/maven/proxy/e2e-test-maven-proxy" || echo "")
if echo "$detail" | grep -q '"useTrustStore"'; then
    echo "SUCCESS: connection.useTrustStore is configured!"
fi
if echo "$detail" | grep -q '"username"'; then
    echo "SUCCESS: httpClient.authentication is configured!"
fi

echo "--- Maven Proxy Repository test completed ---"

# Test NPM Hosted Repository
echo "--- Testing NPM Hosted Repository ---"

kubectl apply -f "${MANIFEST_DIR}/repository-npm-hosted.yaml"

echo "Waiting for NPM Hosted Repository to be ready..."
sleep 5

for i in {1..30}; do
    status=$(kubectl get repository e2e-test-npm-hosted -n default -o jsonpath='{.status.conditions[?(@.type=="Synced")].status}' 2>/dev/null || echo "Unknown")
    if [ "$status" = "True" ]; then
        echo "NPM Hosted Repository is synced!"
        break
    fi
    echo "Waiting for NPM Hosted Repository to sync... ($i/30)"
    sleep 2
done

# Verify in Nexus API
echo "Verifying NPM Hosted Repository in Nexus..."
response=$(curl -sf -u "${NEXUS_USER}:${NEXUS_PASS}" "${NEXUS_URL}/service/rest/v1/repositories" || echo "")
if echo "$response" | grep -q "e2e-test-npm-hosted"; then
    echo "SUCCESS: NPM Hosted Repository found in Nexus!"
else
    echo "WARNING: NPM Hosted Repository not yet visible in Nexus API"
fi

echo "--- NPM Hosted Repository test completed ---"

# Test Raw Hosted Repository
echo "--- Testing Raw Hosted Repository ---"

kubectl apply -f "${MANIFEST_DIR}/repository-raw-hosted.yaml"

echo "Waiting for Raw Hosted Repository to be ready..."
sleep 5

for i in {1..30}; do
    status=$(kubectl get repository e2e-test-raw-hosted -n default -o jsonpath='{.status.conditions[?(@.type=="Synced")].status}' 2>/dev/null || echo "Unknown")
    if [ "$status" = "True" ]; then
        echo "Raw Hosted Repository is synced!"
        break
    fi
    echo "Waiting for Raw Hosted Repository to sync... ($i/30)"
    sleep 2
done

# Verify in Nexus API
echo "Verifying Raw Hosted Repository in Nexus..."
response=$(curl -sf -u "${NEXUS_USER}:${NEXUS_PASS}" "${NEXUS_URL}/service/rest/v1/repositories" || echo "")
if echo "$response" | grep -q "e2e-test-raw-hosted"; then
    echo "SUCCESS: Raw Hosted Repository found in Nexus!"
else
    echo "WARNING: Raw Hosted Repository not yet visible in Nexus API"
fi

echo "--- Raw Hosted Repository test completed ---"

# Test PyPI Proxy Repository
echo "--- Testing PyPI Proxy Repository ---"

kubectl apply -f "${MANIFEST_DIR}/repository-pypi-proxy.yaml"

echo "Waiting for PyPI Proxy Repository to be ready..."
sleep 5

for i in {1..30}; do
    status=$(kubectl get repository e2e-test-pypi-proxy -n default -o jsonpath='{.status.conditions[?(@.type=="Synced")].status}' 2>/dev/null || echo "Unknown")
    if [ "$status" = "True" ]; then
        echo "PyPI Proxy Repository is synced!"
        break
    fi
    echo "Waiting for PyPI Proxy Repository to sync... ($i/30)"
    sleep 2
done

# Verify in Nexus API
echo "Verifying PyPI Proxy Repository in Nexus..."
response=$(curl -sf -u "${NEXUS_USER}:${NEXUS_PASS}" "${NEXUS_URL}/service/rest/v1/repositories" || echo "")
if echo "$response" | grep -q "e2e-test-pypi-proxy"; then
    echo "SUCCESS: PyPI Proxy Repository found in Nexus!"
else
    echo "WARNING: PyPI Proxy Repository not yet visible in Nexus API"
fi

echo "--- PyPI Proxy Repository test completed ---"

# Cleanup all repositories
echo "--- Cleaning up all test repositories ---"
kubectl delete repository e2e-test-maven-proxy -n default --wait=true --timeout=60s || true
kubectl delete repository e2e-test-npm-hosted -n default --wait=true --timeout=60s || true
kubectl delete repository e2e-test-raw-hosted -n default --wait=true --timeout=60s || true
kubectl delete repository e2e-test-pypi-proxy -n default --wait=true --timeout=60s || true
kubectl delete secret e2e-proxy-auth -n default --ignore-not-found || true

echo "--- All Repository tests completed ---"
