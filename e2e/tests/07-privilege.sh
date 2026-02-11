#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFEST_DIR="${SCRIPT_DIR}/../manifests"

NEXUS_URL="${NEXUS_URL:-http://localhost:8081}"
NEXUS_USER="${NEXUS_USER:-admin}"
NEXUS_PASS="${NEXUS_PASS:-admin123}"

echo "=== Testing Privilege Resources ==="

# Test Application Privilege
echo "--- Testing Application Privilege ---"

kubectl apply -f "${MANIFEST_DIR}/privilege-application.yaml"

echo "Waiting for Application Privilege to be ready..."
sleep 5

for i in {1..30}; do
    status=$(kubectl get privilege e2e-test-app-privilege -n default -o jsonpath='{.status.conditions[?(@.type=="Synced")].status}' 2>/dev/null || echo "Unknown")
    if [ "$status" = "True" ]; then
        echo "Application Privilege is synced!"
        break
    fi
    echo "Waiting for Application Privilege to sync... ($i/30)"
    sleep 2
done

# Verify in Nexus API
echo "Verifying Application Privilege in Nexus..."
response=$(curl -sf -u "${NEXUS_USER}:${NEXUS_PASS}" "${NEXUS_URL}/service/rest/v1/security/privileges/e2e-test-app-privilege" || echo "")
if echo "$response" | grep -q "e2e-test-app-privilege"; then
    echo "SUCCESS: Application Privilege found in Nexus!"
else
    echo "WARNING: Application Privilege not yet visible in Nexus API"
fi

# Test Repository-View Privilege (using default maven-central repository)
echo "--- Testing Repository-View Privilege ---"

kubectl apply -f "${MANIFEST_DIR}/privilege-repository-view.yaml"

echo "Waiting for Repository-View Privilege to be ready..."
sleep 5

for i in {1..30}; do
    status=$(kubectl get privilege e2e-test-repo-privilege -n default -o jsonpath='{.status.conditions[?(@.type=="Synced")].status}' 2>/dev/null || echo "Unknown")
    if [ "$status" = "True" ]; then
        echo "Repository-View Privilege is synced!"
        break
    fi
    echo "Waiting for Repository-View Privilege to sync... ($i/30)"
    sleep 2
done

# Verify in Nexus API
echo "Verifying Repository-View Privilege in Nexus..."
response=$(curl -sf -u "${NEXUS_USER}:${NEXUS_PASS}" "${NEXUS_URL}/service/rest/v1/security/privileges/e2e-test-repo-privilege" || echo "")
if echo "$response" | grep -q "e2e-test-repo-privilege"; then
    echo "SUCCESS: Repository-View Privilege found in Nexus!"
else
    echo "WARNING: Repository-View Privilege not yet visible in Nexus API"
fi

# Test Wildcard Privilege
echo "--- Testing Wildcard Privilege ---"

kubectl apply -f "${MANIFEST_DIR}/privilege-wildcard.yaml"

echo "Waiting for Wildcard Privilege to be ready..."
sleep 5

for i in {1..30}; do
    status=$(kubectl get privilege e2e-test-wildcard-privilege -n default -o jsonpath='{.status.conditions[?(@.type=="Synced")].status}' 2>/dev/null || echo "Unknown")
    if [ "$status" = "True" ]; then
        echo "Wildcard Privilege is synced!"
        break
    fi
    echo "Waiting for Wildcard Privilege to sync... ($i/30)"
    sleep 2
done

# Verify in Nexus API
echo "Verifying Wildcard Privilege in Nexus..."
response=$(curl -sf -u "${NEXUS_USER}:${NEXUS_PASS}" "${NEXUS_URL}/service/rest/v1/security/privileges/e2e-test-wildcard-privilege" || echo "")
if echo "$response" | grep -q "e2e-test-wildcard-privilege"; then
    echo "SUCCESS: Wildcard Privilege found in Nexus!"
else
    echo "WARNING: Wildcard Privilege not yet visible in Nexus API"
fi

# Cleanup
echo "Cleaning up Privileges..."
kubectl delete privilege e2e-test-app-privilege -n default --wait=true --timeout=60s
kubectl delete privilege e2e-test-repo-privilege -n default --wait=true --timeout=60s
kubectl delete privilege e2e-test-wildcard-privilege -n default --wait=true --timeout=60s

echo "--- Privilege test completed ---"
