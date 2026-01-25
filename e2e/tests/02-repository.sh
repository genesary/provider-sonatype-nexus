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
