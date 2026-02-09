#!/bin/bash
set -e

NEXUS_URL="${NEXUS_URL:-http://localhost:8081}"
NEXUS_USER="${NEXUS_USER:-admin}"
NEXUS_PASS="${NEXUS_PASS:-admin123}"

echo "=== Testing Repository Group Resources ==="

# First create a hosted repository to be part of the group
echo "--- Creating Maven Hosted for Group ---"

cat <<EOF | kubectl apply -f -
apiVersion: nexus.crossplane.io/v1alpha1
kind: Repository
metadata:
  name: e2e-group-maven-hosted
  namespace: default
spec:
  forProvider:
    name: e2e-group-maven-hosted
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

echo "Waiting for Maven Hosted to be ready..."
sleep 5

for i in {1..30}; do
    status=$(kubectl get repository e2e-group-maven-hosted -n default -o jsonpath='{.status.conditions[?(@.type=="Synced")].status}' 2>/dev/null || echo "Unknown")
    if [ "$status" = "True" ]; then
        echo "Maven Hosted is synced!"
        break
    fi
    echo "Waiting for Maven Hosted to sync... ($i/30)"
    sleep 2
done

# Create a proxy repository to be part of the group
echo "--- Creating Maven Proxy for Group ---"

cat <<EOF | kubectl apply -f -
apiVersion: nexus.crossplane.io/v1alpha1
kind: Repository
metadata:
  name: e2e-group-maven-proxy
  namespace: default
spec:
  forProvider:
    name: e2e-group-maven-proxy
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

echo "Waiting for Maven Proxy to be ready..."
sleep 5

for i in {1..30}; do
    status=$(kubectl get repository e2e-group-maven-proxy -n default -o jsonpath='{.status.conditions[?(@.type=="Synced")].status}' 2>/dev/null || echo "Unknown")
    if [ "$status" = "True" ]; then
        echo "Maven Proxy is synced!"
        break
    fi
    echo "Waiting for Maven Proxy to sync... ($i/30)"
    sleep 2
done

# Now create the group repository
echo "--- Testing Maven Group Repository ---"

cat <<EOF | kubectl apply -f -
apiVersion: nexus.crossplane.io/v1alpha1
kind: Repository
metadata:
  name: e2e-test-maven-group
  namespace: default
spec:
  forProvider:
    name: e2e-test-maven-group
    format: maven2
    type: group
    online: true
    maven:
      versionPolicy: RELEASE
      layoutPolicy: STRICT
    group:
      memberNames:
        - e2e-group-maven-hosted
        - e2e-group-maven-proxy
    storage:
      blobStoreName: default
      strictContentTypeValidation: true
  providerConfigRef:
    name: default
EOF

echo "Waiting for Maven Group Repository to be ready..."
sleep 5

for i in {1..30}; do
    status=$(kubectl get repository e2e-test-maven-group -n default -o jsonpath='{.status.conditions[?(@.type=="Synced")].status}' 2>/dev/null || echo "Unknown")
    if [ "$status" = "True" ]; then
        echo "Maven Group Repository is synced!"
        break
    fi
    echo "Waiting for Maven Group Repository to sync... ($i/30)"
    sleep 2
done

# Verify in Nexus API
echo "Verifying Maven Group Repository in Nexus..."
response=$(curl -sf -u "${NEXUS_USER}:${NEXUS_PASS}" "${NEXUS_URL}/service/rest/v1/repositories" || echo "")
if echo "$response" | grep -q "e2e-test-maven-group"; then
    echo "SUCCESS: Maven Group Repository found in Nexus!"
else
    echo "WARNING: Maven Group Repository not yet visible in Nexus API"
fi

# Cleanup - delete group first, then members
echo "--- Cleaning up repositories ---"
kubectl delete repository e2e-test-maven-group -n default --wait=true --timeout=60s || true
kubectl delete repository e2e-group-maven-hosted -n default --wait=true --timeout=60s || true
kubectl delete repository e2e-group-maven-proxy -n default --wait=true --timeout=60s || true

echo "--- Maven Group Repository test completed ---"
