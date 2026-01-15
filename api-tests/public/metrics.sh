#!/bin/bash

# Metrics endpoint test
source "$(dirname "$0")/../config.sh"

print_header "Testing GET /metrics"

RESPONSE=$(curl -s "${API_BASE_URL%/api/v1}/metrics")

echo -e "${YELLOW}Response:${NC}"
echo "$RESPONSE" | head -n 20
