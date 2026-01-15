#!/bin/bash

# Update Member Role in Organization
source "$(dirname "$0")/../config.sh"

print_header "Testing Update Member Role Endpoint"

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

read -p "Member user ID to update: " MEMBER_ID
read -p "New role (owner|admin|member): " MEMBER_ROLE

DATA="{
  \"role\": \"$MEMBER_ROLE\"
}"

print_warning "Updating role of member $MEMBER_ID to $MEMBER_ROLE in org $ORG_ID"
RESPONSE=$(api_call "PUT" "/organizations/${ORG_ID}/members/${MEMBER_ID}/role" "$DATA" "$TOKEN")

echo -e "${YELLOW}Response:${NC}"
echo "$RESPONSE" | jq '.'
