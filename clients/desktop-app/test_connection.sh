#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Backend IP (change this if needed)
BACKEND_IP="192.168.200.252"

echo "=================================="
echo "Backend Connection Test"
echo "=================================="
echo ""

# Test Auth Service
echo -n "Testing Auth Service (Port 8086)... "
if curl -s --connect-timeout 3 "http://${BACKEND_IP}:8086/health" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ OK${NC}"
    echo "  Response: $(curl -s http://${BACKEND_IP}:8086/health)"
else
    echo -e "${RED}✗ FAILED${NC}"
fi
echo ""

# Test Messaging Service
echo -n "Testing Messaging Service (Port 8081)... "
if curl -s --connect-timeout 3 "http://${BACKEND_IP}:8081/health" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ OK${NC}"
else
    echo -e "${RED}✗ FAILED (Service might not be running)${NC}"
fi
echo ""

# Test Presence Service
echo -n "Testing Presence Service (Port 8083)... "
if curl -s --connect-timeout 3 "http://${BACKEND_IP}:8083/health" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ OK${NC}"
else
    echo -e "${RED}✗ FAILED${NC}"
fi
echo ""

# Test File Transfer Service
echo -n "Testing File Transfer Service (Port 8082)... "
if curl -s --connect-timeout 3 "http://${BACKEND_IP}:8082/health" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ OK${NC}"
else
    echo -e "${RED}✗ FAILED${NC}"
fi
echo ""

# Test Admin API
echo -n "Testing Admin API (Port 8090)... "
if curl -s --connect-timeout 3 "http://${BACKEND_IP}:8090/health" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ OK${NC}"
    echo "  Response: $(curl -s http://${BACKEND_IP}:8090/health)"
else
    echo -e "${RED}✗ FAILED${NC}"
fi
echo ""

# Test Login Endpoint
echo "=================================="
echo "Testing Login Endpoint"
echo "=================================="
echo ""
echo "Attempting login with admin/password..."
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "http://${BACKEND_IP}:8086/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"password"}')

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)

if [ "$HTTP_CODE" = "200" ]; then
    echo -e "${GREEN}✓ Login Successful${NC}"
    echo "Response:"
    echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
else
    echo -e "${RED}✗ Login Failed (HTTP $HTTP_CODE)${NC}"
    echo "Response: $BODY"
fi

echo ""
echo "=================================="
echo "Test Complete"
echo "=================================="
