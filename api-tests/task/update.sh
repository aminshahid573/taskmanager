#!/bin/bash

# Update Task API Test
source "$(dirname "$0")/../config.sh"

print_header "Testing Update Task Endpoint"

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

read -p "Enter new title (optional): " NEW_TITLE
read -p "Enter new description (optional): " NEW_DESCRIPTION
read -p "Enter new status (todo/in_progress/done, optional): " NEW_STATUS
read -p "Enter new due date (YYYY-MM-DD format, optional): " NEW_DUE_DATE

UPDATE_DATA="{"

if [ -n "$NEW_TITLE" ]; then
    UPDATE_DATA+="\"title\": \"$NEW_TITLE\","
fi

if [ -n "$NEW_DESCRIPTION" ]; then
    UPDATE_DATA+="\"description\": \"$NEW_DESCRIPTION\","
fi

if [ -n "$NEW_STATUS" ]; then
    UPDATE_DATA+="\"status\": \"$NEW_STATUS\","
fi

if [ -n "$NEW_DUE_DATE" ]; then
    UPDATE_DATA+="\"due_date\": \"${NEW_DUE_DATE}T00:00:00Z\","
fi

# Remove trailing comma
UPDATE_DATA="${UPDATE_DATA%,}"
UPDATE_DATA+="}"

print_warning "Updating task: $TASK_ID"

RESPONSE=$(api_call "PUT" "/organizations/$ORG_ID/tasks/$TASK_ID" "$UPDATE_DATA" "$TOKEN")

echo -e "${YELLOW}Response:${NC}"
echo "$RESPONSE" | jq '.'

# Extract values - Update returns task object directly
TASK_TITLE=$(echo "$RESPONSE" | jq -r '.title' 2>/dev/null)

if [ "$TASK_TITLE" != "null" ] && [ "$TASK_TITLE" != "" ]; then
    print_success "Task updated successfully"
else
    print_error "Failed to update task"
