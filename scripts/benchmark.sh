#!/bin/bash
# Performance benchmarking script for PME Online

echo "======================================"
echo "PME Online Performance Benchmark"
echo "======================================"
echo ""

# Check if hey is installed
if ! command -v hey &> /dev/null; then
    echo "Installing hey load testing tool..."
    go install github.com/rakyll/hey@latest
fi

API_URL="http://localhost:8080"

# Test 1: LEND Orders (Simple validation)
echo "Test 1: LEND Orders (10,000 requests, 50 concurrent)"
echo "--------------------------------------"
hey -n 10000 -c 50 -m POST \
  -H "Content-Type: application/json" \
  -d '{"reff_request_id":"LEND'$(date +%s)'","participant_code":"AA","account_code":"AA-067890","instrument_code":"BBRI","side":"LEND","quantity":1000,"settlement_date":"1970-01-01T00:00:00Z","reimbursement_date":"1970-01-01T00:00:00Z","periode":0,"market_price":0,"rate":0.15,"aro":false}' \
  $API_URL/api/order/new

echo ""
echo ""

# Test 2: BORR Orders (Full validation)
echo "Test 2: BORR Orders (5,000 requests, 50 concurrent)"
echo "--------------------------------------"
hey -n 5000 -c 50 -m POST \
  -H "Content-Type: application/json" \
  -d '{"reff_request_id":"BORR'$(date +%s)'","participant_code":"YU","account_code":"YU-012345","instrument_code":"BBRI","side":"BORR","quantity":1000,"settlement_date":"2025-11-26T00:00:00Z","reimbursement_date":"2025-12-26T00:00:00Z","periode":30,"market_price":0,"rate":0.18,"aro":false}' \
  $API_URL/api/order/new

echo ""
echo ""

# Test 3: Order List Query
echo "Test 3: Order List Query (1,000 requests, 20 concurrent)"
echo "--------------------------------------"
hey -n 1000 -c 20 \
  $API_URL/api/order/list

echo ""
echo "======================================"
echo "Benchmark Complete"
echo "======================================"
