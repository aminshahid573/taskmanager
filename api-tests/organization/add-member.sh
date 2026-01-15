#!/bin/bash

# Add Member to Organization
source "$(dirname "$0")/../config.sh"

print_header "Testing Add Member Endpoint"

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

read -p "Member email to add: " MEMBER_EMAIL
read -p "Role (owner|admin|member): " MEMBER_ROLE

DATA="{
  \"user_email\": \"$MEMBER_EMAIL\",
  \"role\": \"$MEMBER_ROLE\"
}"

print_warning "Adding member $MEMBER_EMAIL as $MEMBER_ROLE to org $ORG_ID"
RESPONSE=$(api_call "POST" "/organizations/${ORG_ID}/members" "$DATA" "$TOKEN")

echo -e "${YELLOW}Response:${NC}"
echo "$RESPONSE" | jq '.'
