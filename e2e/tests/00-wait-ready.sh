#!/bin/bash
set -e

NEXUS_URL="${NEXUS_URL:-http://localhost:8081}"
TIMEOUT="${TIMEOUT:-300}"

echo "=== Waiting for Nexus to be ready ==="

start_time=$(date +%s)
while true; do
    current_time=$(date +%s)
    elapsed=$((current_time - start_time))

    if [ $elapsed -gt $TIMEOUT ]; then
        echo "ERROR: Timeout waiting for Nexus to be ready"
        exit 1
    fi

    if curl -sf "${NEXUS_URL}/service/rest/v1/status" > /dev/null 2>&1; then
        echo "Nexus is ready!"
        break
    fi

    echo "Waiting for Nexus... (${elapsed}s/${TIMEOUT}s)"
    sleep 5
done

echo "=== Waiting for Provider to be healthy ==="

start_time=$(date +%s)
while true; do
    current_time=$(date +%s)
    elapsed=$((current_time - start_time))

    if [ $elapsed -gt $TIMEOUT ]; then
        echo "ERROR: Timeout waiting for Provider to be healthy"
        echo "=== Provider status ==="
        kubectl get providers.pkg.crossplane.io provider-sonatype-nexus -o yaml 2>/dev/null || true
        echo "=== Pods ==="
        kubectl get pods -n crossplane-system 2>/dev/null || true
        exit 1
    fi

    healthy=$(kubectl get providers.pkg.crossplane.io provider-sonatype-nexus -o jsonpath='{.status.conditions[?(@.type=="Healthy")].status}' 2>/dev/null || echo "Unknown")
    installed=$(kubectl get providers.pkg.crossplane.io provider-sonatype-nexus -o jsonpath='{.status.conditions[?(@.type=="Installed")].status}' 2>/dev/null || echo "Unknown")

    if [ "$healthy" = "True" ] && [ "$installed" = "True" ]; then
        echo "Provider is healthy and installed!"
        break
    fi

    echo "Waiting for Provider... (healthy=$healthy, installed=$installed) (${elapsed}s/${TIMEOUT}s)"
    sleep 5
done

echo "=== All components ready ==="
