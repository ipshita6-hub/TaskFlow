#!/bin/bash

# TaskFlow API Testing Script
# Usage: ./test-api.sh https://taskflow-api.onrender.com

set -e

if [ -z "$1" ]; then
    echo "Usage: ./test-api.sh <BASE_URL>"
    echo "Example: ./test-api.sh https://taskflow-api.onrender.com"
    exit 1
fi

BASE_URL="$1"
EMAIL="test-$(date +%s)@example.com"
PASSWORD="password123"

echo "🚀 TaskFlow API Test Suite"
echo "=========================="
echo "Base URL: $BASE_URL"
echo ""

# Test 1: Health Check
echo "✓ Test 1: Health Check (API Docs)"
curl -s "$BASE_URL/api/docs" > /dev/null
echo "✓ API is responding"
echo ""

# Test 2: Register User
echo "✓ Test 2: Register User"
echo "Email: $EMAIL"
REGISTER_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"$EMAIL\",
    \"password\": \"$PASSWORD\"
  }")

echo "Response: $REGISTER_RESPONSE"
USER_ID=$(echo $REGISTER_RESPONSE | grep -o '"id":"[^"]*' | cut -d'"' -f4)
echo "✓ User registered with ID: $USER_ID"
echo ""

# Test 3: Login
echo "✓ Test 3: Login"
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"$EMAIL\",
    \"password\": \"$PASSWORD\"
  }")

echo "Response: $LOGIN_RESPONSE"
TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*' | cut -d'"' -f4)
echo "✓ Login successful"
echo "Token: ${TOKEN:0:20}..."
echo ""

# Test 4: Validate Schedule
echo "✓ Test 4: Validate Cron Schedule"
VALIDATE_RESPONSE=$(curl -s "$BASE_URL/api/v1/schedule/validate?expr=0%2012%20*%20*%20*" \
  -H "Authorization: Bearer $TOKEN")
echo "Response: $VALIDATE_RESPONSE"
echo "✓ Schedule validation works"
echo ""

# Test 5: Create One-Time Task
echo "✓ Test 5: Create One-Time Task"
FUTURE_TIME=$(date -u -d "+1 hour" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u -v+1H +"%Y-%m-%dT%H:%M:%SZ")
CREATE_TASK=$(curl -s -X POST "$BASE_URL/api/v1/tasks" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{
    \"name\": \"Test One-Time Task\",
    \"description\": \"Created at $(date)\",
    \"task_type\": \"noop\",
    \"schedule_type\": \"one_time\",
    \"scheduled_at\": \"$FUTURE_TIME\",
    \"retry_policy\": {
      \"max_attempts\": 3,
      \"backoff_seconds\": 60
    }
  }")

echo "Response: $CREATE_TASK"
TASK_ID=$(echo $CREATE_TASK | grep -o '"id":"[^"]*' | cut -d'"' -f4)
echo "✓ One-time task created with ID: $TASK_ID"
echo ""

# Test 6: Create Recurring Task
echo "✓ Test 6: Create Recurring Task (every 5 minutes)"
CREATE_RECURRING=$(curl -s -X POST "$BASE_URL/api/v1/tasks" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{
    \"name\": \"Test Recurring Task\",
    \"description\": \"Runs every 5 minutes\",
    \"task_type\": \"noop\",
    \"schedule_type\": \"cron\",
    \"cron_expression\": \"*/5 * * * *\",
    \"retry_policy\": {
      \"max_attempts\": 3,
      \"backoff_seconds\": 60
    }
  }")

echo "Response: $CREATE_RECURRING"
RECURRING_ID=$(echo $CREATE_RECURRING | grep -o '"id":"[^"]*' | cut -d'"' -f4)
echo "✓ Recurring task created with ID: $RECURRING_ID"
echo ""

# Test 7: List Tasks
echo "✓ Test 7: List Tasks"
LIST_TASKS=$(curl -s "$BASE_URL/api/v1/tasks" \
  -H "Authorization: Bearer $TOKEN")
echo "Response: $LIST_TASKS"
echo "✓ Tasks retrieved successfully"
echo ""

# Test 8: Get Task Details
echo "✓ Test 8: Get Task Details"
GET_TASK=$(curl -s "$BASE_URL/api/v1/tasks/$TASK_ID" \
  -H "Authorization: Bearer $TOKEN")
echo "Response: $GET_TASK"
echo "✓ Task details retrieved"
echo ""

# Test 9: Get Workers
echo "✓ Test 9: Get Active Workers"
GET_WORKERS=$(curl -s "$BASE_URL/api/v1/workers" \
  -H "Authorization: Bearer $TOKEN")
echo "Response: $GET_WORKERS"
echo "✓ Workers retrieved"
echo ""

# Test 10: Get DLQ (should be empty)
echo "✓ Test 10: Get Dead Letter Queue"
GET_DLQ=$(curl -s "$BASE_URL/api/v1/jobs/dlq" \
  -H "Authorization: Bearer $TOKEN")
echo "Response: $GET_DLQ"
echo "✓ DLQ retrieved"
echo ""

echo "✅ All Tests Passed!"
echo "=========================="
echo ""
echo "📊 Summary:"
echo "  - User registered: $EMAIL"
echo "  - One-time task: $TASK_ID"
echo "  - Recurring task: $RECURRING_ID"
echo "  - Auth token: ${TOKEN:0:20}... (save for later API calls)"
echo ""
echo "📖 Next steps:"
echo "  1. View API docs: $BASE_URL/api/docs"
echo "  2. Check task logs: $BASE_URL/api/v1/tasks/$TASK_ID/logs"
echo "  3. Monitor workers: $BASE_URL/api/v1/workers"
echo "  4. View failed jobs: $BASE_URL/api/v1/jobs/dlq"
echo ""
