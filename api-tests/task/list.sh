#!/bin/bash

# List Tasks API Test
source "$(dirname "$0")/../config.sh"

print_header "Testing List Tasks Endpoint"

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

print_warning "Fetching tasks for organization: $ORG_ID"

RESPONSE=$(api_call "GET" "/organizations/$ORG_ID/tasks" "" "$TOKEN")

echo -e "${YELLOW}Response:${NC}"
echo "$RESPONSE" | jq '.'

# Extract values - List returns paginated result with tasks
TASK_COUNT=$(echo "$RESPONSE" | jq '.tasks | length' 2>/dev/null)

if [ "$TASK_COUNT" != "null" ] && [ "$TASK_COUNT" -ge 0 ]; then
    print_success "Tasks fetched successfully"
    echo "Total tasks: $TASK_COUNT"
    
    # Display tasks in a table format
    echo -e "\n${BLUE}Tasks:${NC}"
    echo "$RESPONSE" | jq -r '.tasks[] | "\(.id) | \(.title) | \(.status)"' | \
