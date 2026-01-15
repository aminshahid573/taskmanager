#!/bin/bash

# Login API Test
source "$(dirname "$0")/../config.sh"

print_header "Testing Login Endpoint"

read -p "Enter email: " EMAIL
read -sp "Enter password: " PASSWORD
echo ""

print_warning "Attempting login for: $EMAIL"

RESPONSE=$(api_call "POST" "/auth/login" "{
    \"email\": \"$EMAIL\",
    \"password\": \"$PASSWORD\"
}")

echo -e "${YELLOW}Response:${NC}"
echo "$RESPONSE" | jq '.'

# Extract values
ACCESS_TOKEN=$(echo "$RESPONSE" | jq -r '.access_token' 2>/dev/null)
REFRESH_TOKEN=$(echo "$RESPONSE" | jq -r '.refresh_token' 2>/dev/null)
EXPIRES_IN=$(echo "$RESPONSE" | jq -r '.expires_in' 2>/dev/null)
ERROR_CODE=$(echo "$RESPONSE" | jq -r '.code' 2>/dev/null)
ERROR_MSG=$(echo "$RESPONSE" | jq -r '.message' 2>/dev/null)

if [ "$ACCESS_TOKEN" != "null" ] && [ "$ACCESS_TOKEN" != "" ]; then
    print_success "Login successful"
    echo "Token Expires In: $EXPIRES_IN seconds"
    
    # Save tokens
    echo "$ACCESS_TOKEN" > /tmp/access_token.txt
    echo "$REFRESH_TOKEN" > /tmp/refresh_token.txt
    echo "$EMAIL" > /tmp/test_email.txt
    
    print_success "Tokens saved for future requests"
elif [ "$ERROR_CODE" = "EMAIL_NOT_VERIFIED" ]; then
    print_success "Login endpoint working correctly - Email verification required"
    echo "Message: $ERROR_MSG"
    echo "Email: $EMAIL"
    echo "Next step: Run verify-otp.sh to verify your email"
    
    # Save email for OTP verification
    echo "$EMAIL" > /tmp/test_email.txt
else
    print_error "Login failed"
    echo "$RESPONSE" | jq '.'
fi
