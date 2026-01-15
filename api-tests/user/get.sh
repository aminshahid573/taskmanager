#!/bin/bash

# Get User By ID API Test
source "$(dirname "$0")/../config.sh"

print_header "Testing Get User By ID Endpoint"

# Get token
if [ -f /tmp/access_token.txt ]; then
    TOKEN=$(cat /tmp/access_token.txt)
else
    print_error "No access token found. Please run auth/login.sh or auth/verify-otp.sh first"
    exit 1
fi

read -p "Enter user ID (UUID): " USER_ID

print_warning "Fetching user: $USER_ID"

RESPONSE=$(api_call "GET" "/users/$USER_ID" "" "$TOKEN")

echo -e "${YELLOW}Response:${NC}"
echo "$RESPONSE" | jq '.'

# Extract values
SUCCESS=$(echo "$RESPONSE" | jq -r '.success' 2>/dev/null)
PROFILE_ID=$(echo "$RESPONSE" | jq -r '.data.id' 2>/dev/null)
PROFILE_NAME=$(echo "$RESPONSE" | jq -r '.data.name' 2>/dev/null)

if [ "$SUCCESS" = "true" ] && [ "$PROFILE_ID" != "null" ]; then
    print_success "User fetched successfully"
    echo "ID: $PROFILE_ID"
    echo "Name: $PROFILE_NAME"
else
    print_error "Failed to fetch user"
fi
