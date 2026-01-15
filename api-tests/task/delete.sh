#!/bin/bash

# Delete Task API Test
source "$(dirname "$0")/../config.sh"

print_header "Testing Delete Task Endpoint"

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

read -p "Are you sure you want to delete this task? (yes/no): " CONFIRM

if [ "$CONFIRM" != "yes" ]; then
    print_warning "Delete cancelled"
    exit 0
fi

print_warning "Deleting task: $TASK_ID"

RESPONSE=$(api_call "DELETE" "/organizations/$ORG_ID/tasks/$TASK_ID" "" "$TOKEN")

echo -e "${YELLOW}Response:${NC}"
echo "$RESPONSE" | jq '.'

# Extract values - Delete returns message
MESSAGE=$(echo "$RESPONSE" | jq -r '.message' 2>/dev/null)

if [ "$MESSAGE" != "null" ] && [ "$MESSAGE" != "" ]; then
    print_success "Task deleted successfully"
    echo "Message: $MESSAGE"
    
    # Clear saved task ID
    rm -f /tmp/task_id.txt
else
    print_error "Failed to delete task"
