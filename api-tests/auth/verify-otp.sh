#!/bin/bash

# Verify OTP API Test
source "$(dirname "$0")/../config.sh"

print_header "Testing Verify OTP Endpoint"

# Get email from signup test
if [ -f /tmp/test_email.txt ]; then
    EMAIL=$(cat /tmp/test_email.txt)
else
    EMAIL="testuser@example.com"
fi

read -p "Enter OTP code: " OTP_CODE

print_warning "Verifying OTP for email: $EMAIL"

RESPONSE=$(api_call "POST" "/auth/verify-otp" "{
    \"email\": \"$EMAIL\",
    \"otp\": \"$OTP_CODE\"
}")

echo -e "${YELLOW}Response:${NC}"
echo "$RESPONSE" | jq '.'

# Extract values
SUCCESS=$(echo "$RESPONSE" | jq -r '.success' 2>/dev/null)
MESSAGE=$(echo "$RESPONSE" | jq -r '.message' 2>/dev/null)
ACCESS_TOKEN=$(echo "$RESPONSE" | jq -r '.access_token' 2>/dev/null)
REFRESH_TOKEN=$(echo "$RESPONSE" | jq -r '.refresh_token' 2>/dev/null)

if [ "$SUCCESS" = "true" ] && [ "$ACCESS_TOKEN" != "null" ] && [ "$ACCESS_TOKEN" != "" ]; then
    print_success "OTP verification successful"
    echo "Message: $MESSAGE"
    
    # Save tokens
    echo "$ACCESS_TOKEN" > /tmp/access_token.txt
    echo "$REFRESH_TOKEN" > /tmp/refresh_token.txt
    echo "$EMAIL" > /tmp/test_email.txt
    
    print_success "Tokens saved for future requests"
else
    print_error "OTP verification failed"
    echo "Message: $MESSAGE"
fi
