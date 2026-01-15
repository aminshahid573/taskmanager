#!/bin/bash

# Get Organization API Test
source "$(dirname "$0")/../config.sh"

print_header "Testing Get Organization Endpoint"

# Get token
if [ -f /tmp/access_token.txt ]; then
    TOKEN=$(cat /tmp/access_token.txt)
else
    print_error "No access token found. Please run auth/login.sh or auth/verify-otp.sh first"
    exit 1
fi

# Get org ID
if [ -f /tmp/org_id.txt ]; then
    ORG_ID=$(cat /tmp/org_id.txt)
    echo "Using saved organization ID: $ORG_ID"
else
    read -p "Enter organization ID (UUID): " ORG_ID
fi

print_warning "Fetching organization: $ORG_ID"

RESPONSE=$(api_call "GET" "/organizations/$ORG_ID" "" "$TOKEN")

echo -e "${YELLOW}Response:${NC}"
echo "$RESPONSE" | jq '.'

# Extract values - Get returns organization object directly
ORG_NAME=$(echo "$RESPONSE" | jq -r '.name' 2>/dev/null)
ORG_DESC=$(echo "$RESPONSE" | jq -r '.description' 2>/dev/null)

if [ "$ORG_NAME" != "null" ] && [ "$ORG_NAME" != "" ]; then
    print_success "Organization fetched successfully"
    echo "Name: $ORG_NAME"
    echo "Description: $ORG_DESC"
