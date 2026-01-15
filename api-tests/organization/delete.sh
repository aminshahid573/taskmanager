#!/bin/bash

# Delete Organization API Test
source "$(dirname "$0")/../config.sh"

print_header "Testing Delete Organization Endpoint"

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

read -p "Are you sure you want to delete this organization? (yes/no): " CONFIRM

if [ "$CONFIRM" != "yes" ]; then
    print_warning "Delete cancelled"
    exit 0
fi

print_warning "Deleting organization: $ORG_ID"

RESPONSE=$(api_call "DELETE" "/organizations/$ORG_ID" "" "$TOKEN")

echo -e "${YELLOW}Response:${NC}"
echo "$RESPONSE" | jq '.'

# Extract values - Delete returns a message object
MESSAGE=$(echo "$RESPONSE" | jq -r '.message' 2>/dev/null)

if [ "$MESSAGE" != "null" ] && [ "$MESSAGE" != "" ]; then
    print_success "Organization deleted successfully"
    echo "Message: $MESSAGE"
    
    # Clear saved org ID
    rm -f /tmp/org_id.txt
else
    print_error "Failed to delete organization"
    echo "$RESPONSE" | jq '.'
fi
