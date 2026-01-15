#!/bin/bash

# Update Organization API Test
source "$(dirname "$0")/../config.sh"

print_header "Testing Update Organization Endpoint"

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

read -p "Enter new name (optional): " NEW_NAME
read -p "Enter new description (optional): " NEW_DESC

UPDATE_DATA="{"

if [ -n "$NEW_NAME" ]; then
    UPDATE_DATA+="\"name\": \"$NEW_NAME\","
fi

if [ -n "$NEW_DESC" ]; then
    UPDATE_DATA+="\"description\": \"$NEW_DESC\","
fi

# Remove trailing comma
UPDATE_DATA="${UPDATE_DATA%,}"
UPDATE_DATA+="}"

print_warning "Updating organization: $ORG_ID"

RESPONSE=$(api_call "PUT" "/organizations/$ORG_ID" "$UPDATE_DATA" "$TOKEN")

echo -e "${YELLOW}Response:${NC}"
echo "$RESPONSE" | jq '.'

# Extract values - Update returns organization object directly
ORG_NAME=$(echo "$RESPONSE" | jq -r '.name' 2>/dev/null)

if [ "$ORG_NAME" != "null" ] && [ "$ORG_NAME" != "" ]; then
    print_success "Organization updated successfully"
else
    print_error "Failed to update organization"
