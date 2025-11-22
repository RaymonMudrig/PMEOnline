#!/bin/bash

# Test script for APME API Service
# Make sure the service is running before executing this script

BASE_URL="http://localhost:8080"

echo "=========================================="
echo "APME API Test Script"
echo "=========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Function to print section headers
print_section() {
    echo ""
    echo -e "${BLUE}=========================================="
    echo "$1"
    echo -e "==========================================${NC}"
    echo ""
}

# Function to print test results
print_result() {
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Test passed${NC}"
    else
        echo -e "${RED}✗ Test failed${NC}"
    fi
}

# Test 1: Health Check
print_section "Test 1: Health Check"
curl -s -X GET "$BASE_URL/health" | jq .
print_result

# Test 2: Submit Borrowing Order
print_section "Test 2: Submit Borrowing Order"
BORR_RESPONSE=$(curl -s -X POST "$BASE_URL/api/order/new" \
  -H "Content-Type: application/json" \
  -d '{
    "reff_request_id": "TEST-BORR-001",
    "account_code": "YU-012345",
    "participant_code": "YU",
    "instrument_code": "BBRI",
    "side": "BORR",
    "quantity": 1000,
    "settlement_date": "2025-11-25T00:00:00Z",
    "reimbursement_date": "2025-12-25T00:00:00Z",
    "periode": 30,
    "market_price": 5000,
    "rate": 0.18,
    "instruction": "Test borrowing order",
    "aro": false
  }')
echo "$BORR_RESPONSE" | jq .
BORR_ORDER_NID=$(echo "$BORR_RESPONSE" | jq -r '.data.order_nid')
print_result

# Test 3: Submit Lending Order
print_section "Test 3: Submit Lending Order"
LEND_RESPONSE=$(curl -s -X POST "$BASE_URL/api/order/new" \
  -H "Content-Type: application/json" \
  -d '{
    "reff_request_id": "TEST-LEND-001",
    "account_code": "AA-067890",
    "participant_code": "AA",
    "instrument_code": "BBRI",
    "side": "LEND",
    "quantity": 2000,
    "settlement_date": "2025-11-25T00:00:00Z",
    "reimbursement_date": "2025-12-25T00:00:00Z",
    "periode": 30,
    "market_price": 5000,
    "rate": 0.15,
    "instruction": "Test lending order",
    "aro": false
  }')
echo "$LEND_RESPONSE" | jq .
LEND_ORDER_NID=$(echo "$LEND_RESPONSE" | jq -r '.data.order_nid')
print_result

# Wait for processing
sleep 2

# Test 4: Get Account Info
print_section "Test 4: Get Account Info"
curl -s -X GET "$BASE_URL/api/account/info?sid=SIDA1234567890AB" | jq .
print_result

# Test 5: Get Order List (All)
print_section "Test 5: Get Order List (All)"
curl -s -X GET "$BASE_URL/api/order/list" | jq .
print_result

# Test 6: Get Order List (Filtered by Participant)
print_section "Test 6: Get Order List (Filtered by Participant YU)"
curl -s -X GET "$BASE_URL/api/order/list?participant=YU" | jq .
print_result

# Test 7: Get Order List (Filtered by State)
print_section "Test 7: Get Order List (Open Orders)"
curl -s -X GET "$BASE_URL/api/order/list?state=O" | jq .
print_result

# Test 8: Get Contract List
print_section "Test 8: Get Contract List"
curl -s -X GET "$BASE_URL/api/contract/list" | jq .
print_result

# Test 9: Get Contract List (Filtered by SID)
print_section "Test 9: Get Contract List (Filtered by SID)"
curl -s -X GET "$BASE_URL/api/contract/list?sid=SIDA1234567890AB&state=O" | jq .
print_result

# Test 10: Get SBL Detail (All)
print_section "Test 10: Get SBL Detail (All)"
curl -s -X GET "$BASE_URL/api/sbl/detail" | jq .
print_result

# Test 11: Get SBL Detail (Filtered by Instrument)
print_section "Test 11: Get SBL Detail (Instrument BBRI)"
curl -s -X GET "$BASE_URL/api/sbl/detail?instrument=BBRI" | jq .
print_result

# Test 12: Get SBL Detail (Filtered by Side)
print_section "Test 12: Get SBL Detail (Borrowing Side)"
curl -s -X GET "$BASE_URL/api/sbl/detail?side=BORR" | jq .
print_result

# Test 13: Get SBL Aggregate
print_section "Test 13: Get SBL Aggregate (All)"
curl -s -X GET "$BASE_URL/api/sbl/aggregate" | jq .
print_result

# Test 14: Get SBL Aggregate (Filtered by Instrument)
print_section "Test 14: Get SBL Aggregate (Instrument BBRI)"
curl -s -X GET "$BASE_URL/api/sbl/aggregate?instrument=BBRI" | jq .
print_result

# Test 15: Amend Order
if [ ! -z "$BORR_ORDER_NID" ] && [ "$BORR_ORDER_NID" != "null" ]; then
    print_section "Test 15: Amend Order"
    curl -s -X POST "$BASE_URL/api/order/amend" \
      -H "Content-Type: application/json" \
      -d "{
        \"order_nid\": $BORR_ORDER_NID,
        \"reff_request_id\": \"TEST-BORR-001-AMEND\",
        \"quantity\": 1500,
        \"aro\": true
      }" | jq .
    print_result

    # Wait for processing
    sleep 1
fi

# Test 16: Withdraw Order
if [ ! -z "$LEND_ORDER_NID" ] && [ "$LEND_ORDER_NID" != "null" ]; then
    print_section "Test 16: Withdraw Order"
    curl -s -X POST "$BASE_URL/api/order/withdraw" \
      -H "Content-Type: application/json" \
      -d "{
        \"order_nid\": $LEND_ORDER_NID,
        \"reff_request_id\": \"TEST-LEND-001-WITHDRAW\"
      }" | jq .
    print_result
fi

# Test 17: Error Handling - Invalid Request
print_section "Test 17: Error Handling - Missing Required Field"
curl -s -X POST "$BASE_URL/api/order/new" \
  -H "Content-Type: application/json" \
  -d '{
    "reff_request_id": "TEST-ERROR-001",
    "account_code": "YU-012345",
    "side": "BORR"
  }' | jq .
print_result

# Test 18: Error Handling - Invalid Account
print_section "Test 18: Error Handling - Non-existent SID"
curl -s -X GET "$BASE_URL/api/account/info?sid=INVALID_SID_999" | jq .
print_result

echo ""
echo -e "${GREEN}=========================================="
echo "Test Script Complete"
echo -e "==========================================${NC}"
echo ""
echo "Note: For WebSocket testing, use the test_websocket.html file"
echo "or run: go run test_websocket_client.go"
