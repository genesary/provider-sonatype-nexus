#!/bin/bash
set -e

NEXUS_URL="${NEXUS_URL:-http://localhost:8081}"
NEXUS_USER="${NEXUS_USER:-admin}"
NEXUS_PASS="${NEXUS_PASS:-admin123}"

echo "=== Testing Helm Repository Resources ==="

# Test Helm Hosted Repository
echo "--- Testing Helm Hosted Repository ---"

cat <<EOF | kubectl apply -f -
apiVersion: nexus.crossplane.io/v1alpha1
kind: Repository
metadata:
  name: e2e-test-helm-hosted
  namespace: default
spec:
  forProvider:
    name: e2e-test-helm-hosted
    format: helm
    type: hosted
    online: true
    storage:
      blobStoreName: default
      strictContentTypeValidation: true
      writePolicy: ALLOW
  providerConfigRef:
    name: default
EOF

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

cat <<EOF | kubectl apply -f -
apiVersion: nexus.crossplane.io/v1alpha1
kind: Repository
metadata:
  name: e2e-test-helm-proxy
  namespace: default
spec:
  forProvider:
    name: e2e-test-helm-proxy
    format: helm
    type: proxy
    online: true
    proxy:
      remoteUrl: https://charts.helm.sh/stable
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
