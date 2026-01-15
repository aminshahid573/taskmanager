# API Test Scripts

This directory contains shell scripts for testing all the Task Manager API endpoints. Each script uses `curl` for HTTP requests and `jq` for JSON parsing.

## Structure

```
api-tests/
├── config.sh                 # Configuration and helper functions
├── auth/                     # Authentication endpoints
│   ├── signup.sh            # Create new user account
│   ├── login.sh             # Login with email and password
│   ├── resend-otp.sh        # Resend OTP to email
│   └── verify-otp.sh        # Verify OTP and get tokens
├── user/                     # User profile endpoints
│   ├── profile.sh           # Get current user's profile
│   ├── get.sh               # Get another user's public profile
│   └── update.sh            # Update current user's profile
├── task/                     # Task management endpoints
│   ├── create.sh            # Create a new task
│   ├── list.sh              # List all tasks
│   ├── get.sh               # Get specific task details
│   ├── update.sh            # Update task information
│   └── delete.sh            # Delete a task
└── organization/             # Organization management endpoints
    ├── create.sh            # Create a new organization
    ├── list.sh              # List all organizations
    ├── get.sh               # Get organization details
    ├── update.sh            # Update organization information
    └── delete.sh            # Delete an organization
```

## Prerequisites

- `bash` - Shell interpreter
- `curl` - Command-line HTTP client
- `jq` - JSON query processor

### Installation

**On macOS (using Homebrew):**

```bash
brew install curl jq
```

**On Ubuntu/Debian:**

```bash
sudo apt-get install curl jq
```

**On Windows (using Git Bash):**

- Git Bash includes curl
- Install jq from: https://stedolan.github.io/jq/download/

## Usage

### 1. Authentication Flow

**First-time signup:**

```bash
bash api-tests/auth/signup.sh
```

This creates a new account and sends an OTP to the email. Save the email displayed.

**Verify OTP:**

```bash
bash api-tests/auth/verify-otp.sh
```

Enter the OTP you received via email. This will save your access token.

**Or Login (if already verified):**

```bash
bash api-tests/auth/login.sh
```

Logs in with email and password, saves the access token.

**Resend OTP:**

```bash
bash api-tests/auth/resend-otp.sh
```

Request a new OTP (useful during development/testing).

### 2. User Profile Management

**Get Current User Profile:**

```bash
bash api-tests/user/profile.sh
```

Shows your own profile information including ID, email, name, and verification status.

**Get Another User's Profile:**

```bash
bash api-tests/user/get.sh
```

Shows another user's public profile (ID, name, creation date). Enter a user ID when prompted.

**Update Your Profile:**

```bash
bash api-tests/user/update.sh
```

Updates your profile information (currently supports name updates).

### 3. Organization Management

**Create Organization:**

```bash
bash api-tests/organization/create.sh
```

Creates a new organization. The ID is automatically saved.

**List Organizations:**

```bash
bash api-tests/organization/list.sh
```

Shows all your organizations in a table format.

**Get Organization Details:**

```bash
bash api-tests/organization/get.sh
```

Shows detailed information about a specific organization.

**Update Organization:**

```bash
bash api-tests/organization/update.sh
```

Updates organization name and/or description.

**Delete Organization:**

```bash
bash api-tests/organization/delete.sh
```

Deletes an organization (requires confirmation).

### 4. Task Management

**Create Task:**

```bash
bash api-tests/task/create.sh
```

Creates a new task. You'll need an organization ID. The task ID is automatically saved.

**List Tasks:**

```bash
bash api-tests/task/list.sh
```

Shows all tasks for an organization in table format.

**Get Task Details:**

```bash
bash api-tests/task/get.sh
```

Shows detailed information about a specific task.

**Update Task:**

```bash
bash api-tests/task/update.sh
```

Updates task title, description, status, or due date.

**Delete Task:**

```bash
bash api-tests/task/delete.sh
```

Deletes a task (requires confirmation).

## Features

### Automatic Token Management

- Access tokens are saved to `/tmp/access_token.txt`
- Refresh tokens are saved to `/tmp/refresh_token.txt`
- Scripts automatically use saved tokens for authenticated requests

### Automatic ID Management

- Organization IDs are saved to `/tmp/org_id.txt`
- Task IDs are saved to `/tmp/task_id.txt`
- You can reuse these without re-entering them

### Color-Coded Output

- 🟢 Green: Success messages
- 🔴 Red: Error messages
- 🟡 Yellow: Warning/info messages
- 🔵 Blue: Headers

### JSON Parsing with jq

- All responses are pretty-printed with jq
- Tables format data for easy reading
- Errors are extracted and displayed clearly

## Configuration

Edit `config.sh` to change:

- `API_BASE_URL` - Default: `http://localhost:8080/api/v1`
- `CONTENT_TYPE` - Default: `application/json`

```bash
API_BASE_URL="http://localhost:8080/api/v1"
```

## Example Workflow

```bash
# 1. Sign up
bash api-tests/auth/signup.sh

# 2. Verify OTP
bash api-tests/auth/verify-otp.sh

# 3. Get your profile
bash api-tests/user/profile.sh

# 4. Update your profile
bash api-tests/user/update.sh

# 5. Create organization
bash api-tests/organization/create.sh

# 6. Create task
bash api-tests/task/create.sh

# 7. List tasks
bash api-tests/task/list.sh

# 8. Update task
bash api-tests/task/update.sh

# 9. Get task details
bash api-tests/task/get.sh
```

## Troubleshooting

### Scripts don't execute

```bash
chmod +x api-tests/**/*.sh
chmod +x api-tests/config.sh
```

### "No access token found" error

Run authentication script first:

```bash
bash api-tests/auth/login.sh
```

### "Command not found: jq"

Install jq for your operating system.

### API connection refused

Make sure your API server is running:

```bash
docker-compose up
```

## API Response Examples

### Success Response (200 OK)

```json
{
  "success": true,
  "message": "Operation successful",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com"
  }
}
```

### Error Response (400/401/500)

```json
{
  "success": false,
  "message": "Error description",
  "error_code": "ERROR_CODE",
  "details": {
    "field": "error details"
  }
}
```

## Notes

- All tokens and IDs are saved to `/tmp/` on Linux/macOS or system temp directory on Windows
- Scripts are interactive - they prompt for required input
- Optional fields can be skipped by pressing Enter without typing
- Tokens expire after a certain time - you may need to login again

For API documentation, see the project README.
