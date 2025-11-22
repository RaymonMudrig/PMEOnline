#!/bin/bash

# Test script for eClear API Service
# This script populates the PME system with master data

set -e

API_URL="${API_URL:-http://localhost:8081}"
TESTDATA_DIR="$(dirname "$0")/testdata"

echo "🧪 Testing eClear API Service at $API_URL"
echo ""

# Function to check if service is healthy
check_health() {
    echo "🔍 Checking service health..."
    response=$(curl -s -w "\n%{http_code}" "$API_URL/health")
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" -eq 200 ]; then
        echo "✅ Service is healthy"
        echo "$body" | jq .
        return 0
    else
        echo "❌ Service is not healthy (HTTP $http_code)"
        return 1
    fi
    echo ""
}

# Function to insert master data
insert_data() {
    local endpoint=$1
    local file=$2
    local description=$3

    echo "📤 $description..."

    if [ ! -f "$file" ]; then
        echo "❌ File not found: $file"
        return 1
    fi

    response=$(curl -s -w "\n%{http_code}" -X POST "$API_URL$endpoint" \
        -H "Content-Type: application/json" \
        -d @"$file")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" -eq 200 ]; then
        echo "✅ $description successful"
        echo "$body" | jq .
    else
        echo "❌ $description failed (HTTP $http_code)"
        echo "$body"
        return 1
    fi
    echo ""
}

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo "⚠️  Warning: jq is not installed. JSON output will not be formatted."
    echo "   Install jq for better output: brew install jq (macOS) or apt-get install jq (Linux)"
    echo ""
fi

# Wait for service to be ready
echo "⏳ Waiting for service to be ready..."
max_attempts=30
attempt=0
while ! check_health > /dev/null 2>&1; do
    attempt=$((attempt + 1))
    if [ $attempt -ge $max_attempts ]; then
        echo "❌ Service did not become healthy after $max_attempts attempts"
        exit 1
    fi
    echo "   Attempt $attempt/$max_attempts..."
    sleep 1
done
echo ""

# Health check
check_health

# Step 1: Insert Participants (must be first)
insert_data "/participant/insert" "$TESTDATA_DIR/participants.json" "Inserting participants"

# Step 2: Insert Instruments
insert_data "/instrument/insert" "$TESTDATA_DIR/instruments.json" "Inserting instruments"

# Step 3: Insert Accounts (requires participants to exist)
insert_data "/account/insert" "$TESTDATA_DIR/accounts.json" "Inserting accounts"

# Step 4: Update Account Limits (requires accounts to exist)
insert_data "/account/limit" "$TESTDATA_DIR/account_limits.json" "Updating account limits"

echo "🎉 All master data inserted successfully!"
echo ""
echo "📊 Summary:"
echo "  - Participants: 5"
echo "  - Instruments: 10"
echo "  - Accounts: 8"
echo "  - Account Limits: 8"
echo ""
echo "🚀 System is ready for order processing!"
