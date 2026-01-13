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

echo "=== Waiting for Provider to be ready ==="

kubectl wait --for=condition=available deployment/provider-sonatype-nexus \
    -n crossplane-system --timeout=${TIMEOUT}s

echo "Provider is ready!"

echo "=== All components ready ==="
