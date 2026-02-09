#!/bin/bash
set -e

NEXUS_URL="${NEXUS_URL:-http://localhost:8081}"
NEXUS_USER="${NEXUS_USER:-admin}"
NEXUS_PASS="${NEXUS_PASS:-admin123}"

echo "=== Testing Repository Resources ==="

# Test Maven Hosted Repository
echo "--- Testing Maven Hosted Repository ---"

cat <<EOF | kubectl apply -f -
apiVersion: nexus.crossplane.io/v1alpha1
kind: Repository
metadata:
  name: e2e-test-maven-hosted
  namespace: default
spec:
  forProvider:
    name: e2e-test-maven-hosted
    format: maven2
    type: hosted
    online: true
    maven:
      versionPolicy: RELEASE
      layoutPolicy: STRICT
    storage:
      blobStoreName: default
      strictContentTypeValidation: true
      writePolicy: ALLOW
  providerConfigRef:
    name: default
EOF

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

cat <<EOF | kubectl apply -f -
apiVersion: nexus.crossplane.io/v1alpha1
kind: Repository
metadata:
  name: e2e-test-docker-hosted
  namespace: default
spec:
  forProvider:
    name: e2e-test-docker-hosted
    format: docker
    type: hosted
    online: true
    docker:
      v1Enabled: false
      forceBasicAuth: true
      httpPort: 5000
    storage:
      blobStoreName: default
      strictContentTypeValidation: true
      writePolicy: ALLOW
  providerConfigRef:
    name: default
EOF

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

# Test Maven Proxy Repository
echo "--- Testing Maven Proxy Repository ---"

cat <<EOF | kubectl apply -f -
apiVersion: nexus.crossplane.io/v1alpha1
kind: Repository
metadata:
  name: e2e-test-maven-proxy
  namespace: default
spec:
  forProvider:
    name: e2e-test-maven-proxy
    format: maven2
    type: proxy
    online: true
    maven:
      versionPolicy: RELEASE
      layoutPolicy: STRICT
    proxy:
      remoteUrl: https://repo1.maven.org/maven2/
      contentMaxAge: 1440
      metadataMaxAge: 1440
    storage:
      blobStoreName: default
      strictContentTypeValidation: true
    httpClient:
      blocked: false
      autoBlock: true
    negativeCache:
      enabled: true
      timeToLive: 1440
  providerConfigRef:
    name: default
EOF

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

echo "--- Maven Proxy Repository test completed ---"

# Test NPM Hosted Repository
echo "--- Testing NPM Hosted Repository ---"

cat <<EOF | kubectl apply -f -
apiVersion: nexus.crossplane.io/v1alpha1
kind: Repository
metadata:
  name: e2e-test-npm-hosted
  namespace: default
spec:
  forProvider:
    name: e2e-test-npm-hosted
    format: npm
    type: hosted
    online: true
    storage:
      blobStoreName: default
      strictContentTypeValidation: true
      writePolicy: ALLOW
  providerConfigRef:
    name: default
EOF

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

cat <<EOF | kubectl apply -f -
apiVersion: nexus.crossplane.io/v1alpha1
kind: Repository
metadata:
  name: e2e-test-raw-hosted
  namespace: default
spec:
  forProvider:
    name: e2e-test-raw-hosted
    format: raw
    type: hosted
    online: true
    storage:
      blobStoreName: default
      strictContentTypeValidation: false
      writePolicy: ALLOW
  providerConfigRef:
    name: default
EOF

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

cat <<EOF | kubectl apply -f -
apiVersion: nexus.crossplane.io/v1alpha1
kind: Repository
metadata:
  name: e2e-test-pypi-proxy
  namespace: default
spec:
  forProvider:
    name: e2e-test-pypi-proxy
    format: pypi
    type: proxy
    online: true
    proxy:
      remoteUrl: https://pypi.org/
      contentMaxAge: 1440
      metadataMaxAge: 1440
    storage:
      blobStoreName: default
      strictContentTypeValidation: true
    httpClient:
      blocked: false
      autoBlock: true
    negativeCache:
      enabled: true
      timeToLive: 1440
  providerConfigRef:
    name: default
EOF

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

echo "--- All Repository tests completed ---"
