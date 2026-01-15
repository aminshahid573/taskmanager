#!/bin/bash

# Signup API Test
source "$(dirname "$0")/../config.sh"

print_header "Testing Signup Endpoint"

read -p "Enter email: " EMAIL
read -sp "Enter password: " PASSWORD
echo ""
read -p "Enter name: " NAME

print_warning "Creating new user with email: $EMAIL"

RESPONSE=$(api_call "POST" "/auth/signup" "{
    \"email\": \"$EMAIL\",
    \"password\": \"$PASSWORD\",
    \"name\": \"$NAME\"
}")

echo -e "${YELLOW}Response:${NC}"
echo "$RESPONSE" | jq '.'

# Extract values
OTP_SENT=$(echo "$RESPONSE" | jq -r '.otp_sent' 2>/dev/null)
OTP_EXPIRES=$(echo "$RESPONSE" | jq -r '.otp_expires_in' 2>/dev/null)
MESSAGE=$(echo "$RESPONSE" | jq -r '.message' 2>/dev/null)

if [ "$OTP_SENT" = "true" ]; then

# Save email for later tests
echo "$EMAIL" > /tmp/test_email.txt
