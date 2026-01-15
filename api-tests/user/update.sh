#!/bin/bash

# Update User Profile API Test
source "$(dirname "$0")/../config.sh"

print_header "Testing Update Profile Endpoint"

# Get token
if [ -f /tmp/access_token.txt ]; then
    TOKEN=$(cat /tmp/access_token.txt)
else
    print_error "No access token found. Please run auth/login.sh or auth/verify-otp.sh first"
    exit 1
fi

read -p "Enter new name for profile: " NEW_NAME

print_warning "Updating profile with name: $NEW_NAME"

PAYLOAD=$(cat <<EOF
{
  "name": "$NEW_NAME"
}
EOF
)

RESPONSE=$(api_call "PATCH" "/users/me" "$PAYLOAD" "$TOKEN")

echo -e "${YELLOW}Response:${NC}"
echo "$RESPONSE" | jq '.'

# Extract values
SUCCESS=$(echo "$RESPONSE" | jq -r '.success' 2>/dev/null)
MESSAGE=$(echo "$RESPONSE" | jq -r '.message' 2>/dev/null)

if [ "$SUCCESS" = "true" ]; then
    print_success "Profile updated successfully"
    echo "Message: $MESSAGE"
else
    print_error "Failed to update profile"
    echo "$RESPONSE" | jq -r '.error' 2>/dev/null
fi
