#!/bin/bash

# Get Task API Test
source "$(dirname "$0")/../config.sh"

print_header "Testing Get Task Endpoint"

# Get token
if [ -f /tmp/access_token.txt ]; then
    TOKEN=$(cat /tmp/access_token.txt)
else
    print_error "No access token found. Please run auth/login.sh or auth/verify-otp.sh first"
    exit 1
fi

# Get org and task IDs
if [ -f /tmp/org_id.txt ]; then
    ORG_ID=$(cat /tmp/org_id.txt)
else
    read -p "Enter organization ID (UUID): " ORG_ID
fi

if [ -f /tmp/task_id.txt ]; then
    TASK_ID=$(cat /tmp/task_id.txt)
    echo "Using saved task ID: $TASK_ID"
else
    read -p "Enter task ID (UUID): " TASK_ID
fi

print_warning "Fetching task: $TASK_ID"

RESPONSE=$(api_call "GET" "/organizations/$ORG_ID/tasks/$TASK_ID" "" "$TOKEN")

echo -e "${YELLOW}Response:${NC}"
echo "$RESPONSE" | jq '.'

# Extract values - Get returns task object directly
TASK_TITLE=$(echo "$RESPONSE" | jq -r '.title' 2>/dev/null)
TASK_STATUS=$(echo "$RESPONSE" | jq -r '.status' 2>/dev/null)

if [ "$TASK_TITLE" != "null" ] && [ "$TASK_TITLE" != "" ]; then
    print_success "Task fetched successfully"
    echo "Title: $TASK_TITLE"
    echo "Status: $TASK_STATUS"
