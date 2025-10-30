#!/bin/bash

# Terraform Scanner Service - Test Script

set -e

echo "==================================="
echo "Terraform Scanner Service Test"
echo "==================================="
echo ""

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 서비스 URL
SERVICE_URL="http://localhost:8080"

# 1. Health Check
echo "${YELLOW}[1/5] Health Check${NC}"
curl -s "${SERVICE_URL}/health" | jq '.'
echo ""

# 2. List Policies
echo "${YELLOW}[2/5] Listing loaded policies${NC}"
POLICY_COUNT=$(curl -s "${SERVICE_URL}/policies" | jq '.count')
echo "Loaded policies: ${POLICY_COUNT}"
echo ""

# 3. Scan single file (insecure-s3.tf)
echo "${YELLOW}[3/5] Scanning insecure S3 bucket configuration${NC}"
RESULT=$(curl -s -X POST "${SERVICE_URL}/scan" \
  -H "Content-Type: application/json" \
  -d '{"target": "./examples/insecure-s3.tf"}')

echo "$RESULT" | jq '.'
STATUS=$(echo "$RESULT" | jq -r '.status')

if [ "$STATUS" = "success" ]; then
    echo "${GREEN}✓ Scan completed successfully${NC}"
    RESULT_FILE=$(echo "$RESULT" | jq -r '.results[0].file')
    echo "Result saved to: $RESULT_FILE"
    
    # 발견된 misconfiguration 수 출력
    if [ -f "$RESULT_FILE" ]; then
        MISCONFIG_COUNT=$(jq '.Results[0].Misconfigurations | length' "$RESULT_FILE")
        echo "${RED}Found $MISCONFIG_COUNT misconfiguration(s)${NC}"
    fi
else
    echo "${RED}✗ Scan failed${NC}"
fi
echo ""

# 4. Scan single file (secure-s3.tf)
echo "${YELLOW}[4/5] Scanning secure S3 bucket configuration${NC}"
RESULT=$(curl -s -X POST "${SERVICE_URL}/scan" \
  -H "Content-Type: application/json" \
  -d '{"target": "./examples/secure-s3.tf"}')

echo "$RESULT" | jq '.'
STATUS=$(echo "$RESULT" | jq -r '.status')

if [ "$STATUS" = "success" ]; then
    echo "${GREEN}✓ Scan completed successfully${NC}"
    RESULT_FILE=$(echo "$RESULT" | jq -r '.results[0].file')
    echo "Result saved to: $RESULT_FILE"
    
    # 발견된 misconfiguration 수 출력
    if [ -f "$RESULT_FILE" ]; then
        MISCONFIG_COUNT=$(jq '.Results[0].Misconfigurations | length' "$RESULT_FILE")
        if [ "$MISCONFIG_COUNT" -eq 0 ]; then
            echo "${GREEN}No misconfigurations found${NC}"
        else
            echo "${YELLOW}Found $MISCONFIG_COUNT misconfiguration(s)${NC}"
        fi
    fi
else
    echo "${RED}✗ Scan failed${NC}"
fi
echo ""

# 5. Scan directory
echo "${YELLOW}[5/5] Scanning examples directory${NC}"
RESULT=$(curl -s -X POST "${SERVICE_URL}/scan" \
  -H "Content-Type: application/json" \
  -d '{"target": "./examples/"}')

echo "$RESULT" | jq '.'
STATUS=$(echo "$RESULT" | jq -r '.status')

if [ "$STATUS" = "success" ]; then
    echo "${GREEN}✓ Directory scan completed successfully${NC}"
    FILE_COUNT=$(echo "$RESULT" | jq '.results | length')
    echo "Scanned $FILE_COUNT file(s)"
else
    echo "${RED}✗ Directory scan failed${NC}"
fi
echo ""

echo "==================================="
echo "Test completed!"
echo "==================================="
echo ""
echo "Check scan results in: ./scan-results/$(date +%Y-%m-%d)/"
