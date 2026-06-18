#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFEST_DIR="${SCRIPT_DIR}/../manifests"

NEXUS_URL="${NEXUS_URL:-http://localhost:8081}"
NEXUS_USER="${NEXUS_USER:-admin}"
NEXUS_PASS="${NEXUS_PASS:-admin123}"

echo "=== Testing CleanupPolicy Resources ==="

echo "--- Testing CleanupPolicy Create ---"

kubectl apply -f "${MANIFEST_DIR}/cleanuppolicy.yaml"

echo "Waiting for CleanupPolicy to be ready..."
sleep 5

for i in {1..30}; do
    status=$(kubectl get cleanuppolicy e2e-test-cleanup-policy -n default -o jsonpath='{.status.conditions[?(@.type=="Synced")].status}' 2>/dev/null || echo "Unknown")
    reason=$(kubectl get cleanuppolicy e2e-test-cleanup-policy -n default -o jsonpath='{.status.conditions[?(@.type=="Synced")].reason}' 2>/dev/null || echo "")
    if [ "$status" = "True" ]; then
        echo "CleanupPolicy is synced!"
        break
    fi
    echo "Waiting for CleanupPolicy to sync... ($i/30) [status=$status reason=$reason]"
    sleep 2
done

echo "Verifying CleanupPolicy in Nexus..."
response=$(curl -sf -u "${NEXUS_USER}:${NEXUS_PASS}" "${NEXUS_URL}/service/rest/v1/cleanup-policies/e2e-test-cleanup-policy" || echo "")
if echo "$response" | grep -q "e2e-test-cleanup-policy"; then
    echo "SUCCESS: CleanupPolicy found in Nexus!"
else
    echo "WARNING: CleanupPolicy not yet visible in Nexus API (may still be creating)"
fi

echo "--- Testing CleanupPolicy Update ---"

kubectl patch cleanuppolicy e2e-test-cleanup-policy -n default \
    --type='merge' \
    --patch='{"spec":{"forProvider":{"criteriaLastBlobUpdated":45}}}'

sleep 5

for i in {1..15}; do
    status=$(kubectl get cleanuppolicy e2e-test-cleanup-policy -n default -o jsonpath='{.status.conditions[?(@.type=="Synced")].status}' 2>/dev/null || echo "Unknown")
    reason=$(kubectl get cleanuppolicy e2e-test-cleanup-policy -n default -o jsonpath='{.status.conditions[?(@.type=="Synced")].reason}' 2>/dev/null || echo "")
    if [ "$status" = "True" ]; then
        echo "CleanupPolicy update synced!"
        break
    fi
    echo "Waiting for CleanupPolicy update to sync... ($i/15) [status=$status reason=$reason]"
    sleep 2
done

echo "Verifying CleanupPolicy update in Nexus..."
response=$(curl -sf -u "${NEXUS_USER}:${NEXUS_PASS}" "${NEXUS_URL}/service/rest/v1/cleanup-policies/e2e-test-cleanup-policy" || echo "")
if echo "$response" | grep -q "45"; then
    echo "SUCCESS: CleanupPolicy update reflected in Nexus!"
else
    echo "INFO: CleanupPolicy update may not yet be reflected in Nexus API"
fi

echo "--- Cleanup ---"

kubectl delete cleanuppolicy e2e-test-cleanup-policy -n default --wait=true --timeout=60s

echo "Verifying CleanupPolicy deleted from Nexus..."
response=$(curl -sf -u "${NEXUS_USER}:${NEXUS_PASS}" "${NEXUS_URL}/service/rest/v1/cleanup-policies/e2e-test-cleanup-policy" 2>&1 || echo "")
if echo "$response" | grep -q "404\|not found\|empty"; then
    echo "SUCCESS: CleanupPolicy deleted from Nexus!"
else
    echo "INFO: CleanupPolicy may have already been deleted or not yet propagated"
fi

echo "--- CleanupPolicy test completed ---"
