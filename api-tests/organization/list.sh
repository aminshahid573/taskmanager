#!/bin/bash

# List Organizations API Test
source "$(dirname "$0")/../config.sh"

print_header "Testing List Organizations Endpoint"

# Get token
if [ -f /tmp/access_token.txt ]; then
    TOKEN=$(cat /tmp/access_token.txt)
else
    print_error "No access token found. Please run auth/login.sh or auth/verify-otp.sh first"
    exit 1
fi

print_warning "Fetching organizations"

RESPONSE=$(api_call "GET" "/organizations" "" "$TOKEN")

echo -e "${YELLOW}Response:${NC}"
echo "$RESPONSE" | jq '.'

# Extract values - List returns {organizations: [...]}
ORG_COUNT=$(echo "$RESPONSE" | jq '.organizations | length' 2>/dev/null)

if [ "$ORG_COUNT" != "null" ] && [ "$ORG_COUNT" -ge 0 ]; then
    print_success "Organizations fetched successfully"
    echo "Total organizations: $ORG_COUNT"
    
    # Display organizations in a table format
    echo -e "\n${BLUE}Organizations:${NC}"
    echo "$RESPONSE" | jq -r '.organizations[] | "\(.id) | \(.name) | \(.description)"' | \
