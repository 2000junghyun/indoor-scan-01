#!/bin/bash

# Build script for Terraform Scanner Service

set -e

echo "==================================="
echo "Building Terraform Scanner Service"
echo "==================================="
echo ""

# 색상 정의
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 1. Clean previous builds
echo "${YELLOW}[1/4] Cleaning previous builds...${NC}"
rm -f terraform-scanner-service
rm -rf vendor/
echo "${GREEN}✓ Cleaned${NC}"
echo ""

# 2. Download dependencies
echo "${YELLOW}[2/4] Downloading dependencies...${NC}"
go mod download
echo "${GREEN}✓ Dependencies downloaded${NC}"
echo ""

# 3. Build the binary
echo "${YELLOW}[3/4] Building binary...${NC}"
go build -o terraform-scanner-service .
echo "${GREEN}✓ Binary built: terraform-scanner-service${NC}"
echo ""

# 4. Verify build
echo "${YELLOW}[4/4] Verifying build...${NC}"
if [ -f "terraform-scanner-service" ]; then
    SIZE=$(du -h terraform-scanner-service | cut -f1)
    echo "${GREEN}✓ Build successful (Size: $SIZE)${NC}"
else
    echo "✗ Build failed"
    exit 1
fi
echo ""

echo "==================================="
echo "Build completed!"
echo "==================================="
echo ""
echo "To run the service:"
echo "  ./terraform-scanner-service"
echo ""
echo "Or with custom policy directory:"
echo "  POLICY_DIR=/path/to/policies ./terraform-scanner-service"
