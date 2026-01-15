#!/bin/bash

# Remove Member from Organization
source "$(dirname "$0")/../config.sh"

print_header "Testing Remove Member Endpoint"

if [ -f /tmp/access_token.txt ]; then
    TOKEN=$(cat /tmp/access_token.txt)
else
    print_error "No access token found. Run auth/login.sh or auth/verify-otp.sh first."
    exit 1
fi

if [ -f /tmp/org_id.txt ]; then
    ORG_ID=$(cat /tmp/org_id.txt)
else
    read -p "Enter organization ID: " ORG_ID
fi

read -p "Member user ID to remove: " MEMBER_ID

print_warning "Removing member $MEMBER_ID from org $ORG_ID"
RESPONSE=$(api_call "DELETE" "/organizations/${ORG_ID}/members/${MEMBER_ID}" "" "$TOKEN")

echo -e "${YELLOW}Response:${NC}"
echo "$RESPONSE" | jq '.'
