#!/bin/bash

# Health endpoint test
source "$(dirname "$0")/../config.sh"

print_header "Testing GET /health"

RESPONSE=$(curl -s "${API_BASE_URL%/api/v1}/health")

echo -e "${YELLOW}Response:${NC}"
echo "$RESPONSE" | jq '.'
