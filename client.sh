#!/bin/bash

BASE_URL="http://localhost:5000"

echo "Starting API calls..."

function call_api() {
    local endpoint=$1
    echo "Calling $BASE_URL$endpoint..."
    curl -s -N "$BASE_URL$endpoint" &
}

# Call each API **only once** and keep the output visible
call_api "/joblistings"
call_api "/loginlinkedin"
call_api "/uploaddb"

echo "✅ All API calls initiated. Waiting for continuous output..."

# Keep the script running to display API output
wait
