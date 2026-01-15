#!/bin/bash

# Resend OTP API Test
source "$(dirname "$0")/../config.sh"

print_header "Testing Resend OTP Endpoint"

# Get email from saved file or prompt
if [ -f /tmp/test_email.txt ]; then
    EMAIL=$(cat /tmp/test_email.txt)
    echo "Using saved email: $EMAIL"
else
    read -p "Enter email: " EMAIL
fi

print_warning "Requesting OTP for email: $EMAIL"

RESPONSE=$(api_call "POST" "/auth/resend-otp" "{
    \"email\": \"$EMAIL\"
}")

echo -e "${YELLOW}Response:${NC}"
echo "$RESPONSE" | jq '.'

# Extract values
SUCCESS=$(echo "$RESPONSE" | jq -r '.success' 2>/dev/null)
MESSAGE=$(echo "$RESPONSE" | jq -r '.message' 2>/dev/null)
OTP_EXPIRES=$(echo "$RESPONSE" | jq -r '.otp_expires_in' 2>/dev/null)
RETRY_AFTER=$(echo "$RESPONSE" | jq -r '.retry_after' 2>/dev/null)
COOLDOWN=$(echo "$RESPONSE" | jq -r '.cooldown_until' 2>/dev/null)

if [ "$SUCCESS" = "true" ]; then
    print_success "OTP resent successfully"
    echo "OTP Expires In: $OTP_EXPIRES seconds"
    echo "Message: $MESSAGE"
elif [ "$SUCCESS" = "false" ]; then
    print_warning "Cooldown active"
    echo "Message: $MESSAGE"
    echo "OTP Expires In: $OTP_EXPIRES seconds"
    echo "Retry After: $RETRY_AFTER seconds"
    echo "Cooldown Until: $COOLDOWN (unix timestamp)"
else
    print_error "OTP request failed"
fi
