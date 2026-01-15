#!/bin/bash

# Assign Task to a User
source "$(dirname "$0")/../config.sh"

print_header "Testing Assign Task Endpoint"

if [ -f /tmp/access_token.txt ]; then
    TOKEN=$(cat /tmp/access_token.txt)
else
    print_error "No access token found. Run auth/login.sh or auth/verify-otp.sh first."
    exit 1
fi

if [ -f /tmp/org_id.txt ]; then
    ORG_ID=$(cat /tmp/org_id.txt)
else
    read -p "Enter organization ID: " ORG_ID
fi

if [ -f /tmp/task_id.txt ]; then
    TASK_ID=$(cat /tmp/task_id.txt)
else
    read -p "Enter task ID: " TASK_ID
fi

read -p "Assignee user ID: " ASSIGNEE_ID

DATA="{
  \"user_id\": \"$ASSIGNEE_ID\"
}"

print_warning "Assigning task $TASK_ID in org $ORG_ID to $ASSIGNEE_ID"
RESPONSE=$(api_call "PUT" "/organizations/${ORG_ID}/tasks/${TASK_ID}/assign" "$DATA" "$TOKEN")

echo -e "${YELLOW}Response:${NC}"
echo "$RESPONSE" | jq '.'
