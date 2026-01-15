#!/bin/bash

# Get User Profile API Test
source "$(dirname "$0")/../config.sh"

print_header "Testing Get User Profile Endpoint"

# Get token
if [ -f /tmp/access_token.txt ]; then
    TOKEN=$(cat /tmp/access_token.txt)
else
    print_error "No access token found. Please run auth/login.sh or auth/verify-otp.sh first"
    exit 1
fi

print_warning "Fetching user profile"

RESPONSE=$(api_call "GET" "/users/me" "" "$TOKEN")

echo -e "${YELLOW}Response:${NC}"
echo "$RESPONSE" | jq '.'

# Extract values
SUCCESS=$(echo "$RESPONSE" | jq -r '.success' 2>/dev/null)
USER_ID=$(echo "$RESPONSE" | jq -r '.data.id' 2>/dev/null)
USER_NAME=$(echo "$RESPONSE" | jq -r '.data.name' 2>/dev/null)
USER_EMAIL=$(echo "$RESPONSE" | jq -r '.data.email' 2>/dev/null)
EMAIL_VERIFIED=$(echo "$RESPONSE" | jq -r '.data.email_verified' 2>/dev/null)

if [ "$SUCCESS" = "true" ] && [ "$USER_ID" != "null" ]; then
    print_success "Profile fetched successfully"
    echo "ID: $USER_ID"
    echo "Name: $USER_NAME"
    echo "Email: $USER_EMAIL"
    echo "Email Verified: $EMAIL_VERIFIED"
else
    print_error "Failed to fetch profile"
fi
