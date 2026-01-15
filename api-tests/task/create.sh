#!/bin/bash

# Create Task API Test
source "$(dirname "$0")/../config.sh"

print_header "Testing Create Task Endpoint"

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

read -p "Enter task title: " TITLE
read -p "Enter task description: " DESCRIPTION
read -p "Enter due date (YYYY-MM-DD format, optional): " DUE_DATE

TASK_DATA="{
    \"title\": \"$TITLE\",
    \"description\": \"$DESCRIPTION\""

if [ -n "$DUE_DATE" ]; then
    TASK_DATA+=",\"due_date\": \"${DUE_DATE}T00:00:00Z\""
fi

TASK_DATA+="}"

print_warning "Creating task: $TITLE"

RESPONSE=$(api_call "POST" "/organizations/$ORG_ID/tasks" "$TASK_DATA" "$TOKEN")

echo -e "${YELLOW}Response:${NC}"
echo "$RESPONSE" | jq '.'

# Extract values - Create returns task object directly
TASK_ID=$(echo "$RESPONSE" | jq -r '.id' 2>/dev/null)

if [ "$TASK_ID" != "null" ] && [ "$TASK_ID" != "" ]; then
    print_success "Task created successfully"
    echo "Task ID: $TASK_ID"
    
    # Save task ID
    echo "$TASK_ID" > /tmp/task_id.txt
else
    print_error "Failed to create task"
