#!/bin/bash

# Create Organization API Test
source "$(dirname "$0")/../config.sh"

print_header "Testing Create Organization Endpoint"

# Get token
if [ -f /tmp/access_token.txt ]; then
    TOKEN=$(cat /tmp/access_token.txt)
else
    print_error "No access token found. Please run auth/login.sh or auth/verify-otp.sh first"
    exit 1
fi

read -p "Enter organization name: " ORG_NAME
read -p "Enter organization description: " ORG_DESC

ORG_DATA="{
    \"name\": \"$ORG_NAME\",
    \"description\": \"$ORG_DESC\"
}"

print_warning "Creating organization: $ORG_NAME"

RESPONSE=$(api_call "POST" "/organizations" "$ORG_DATA" "$TOKEN")

echo -e "${YELLOW}Response:${NC}"
echo "$RESPONSE" | jq '.'

# Extract values - Create returns organization object directly
ORG_ID=$(echo "$RESPONSE" | jq -r '.id' 2>/dev/null)

if [ "$ORG_ID" != "null" ] && [ "$ORG_ID" != "" ]; then
    print_success "Organization created successfully"
    echo "Organization ID: $ORG_ID"
    
    # Save org ID
    echo "$ORG_ID" > /tmp/org_id.txt
else
    print_error "Failed to create organization"
