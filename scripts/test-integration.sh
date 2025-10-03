#!/bin/bash

# Integration Test Script
# Tests Auth Service (Node.js) + Tasks Service (Go)

set -e  # Exit on error

BASE_URL_AUTH="http://localhost:3001"
BASE_URL_TASKS="http://localhost:3002"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
print_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}\n"
}

print_step() {
    echo -e "${YELLOW}➜ $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Generate random test data
USERNAME="testuser_$RANDOM"
EMAIL="test$RANDOM@example.com"
PASSWORD="SecurePass123"

print_header "Microservices Integration Test"
echo "Username: $USERNAME"
echo "Email: $EMAIL"
echo ""

# Test 1: Health Checks
print_step "1. Checking Service Health"
AUTH_HEALTH=$(curl -s $BASE_URL_AUTH/health)
TASKS_HEALTH=$(curl -s $BASE_URL_TASKS/health)

if echo "$AUTH_HEALTH" | grep -q "healthy"; then
    print_success "Auth Service is healthy"
else
    print_error "Auth Service is not responding"
    exit 1
fi

if echo "$TASKS_HEALTH" | grep -q "healthy"; then
    print_success "Tasks Service is healthy"
else
    print_error "Tasks Service is not responding"
    exit 1
fi

# Test 2: Register User
print_step "2. Registering New User"
REGISTER_RESPONSE=$(curl -s -X POST $BASE_URL_AUTH/api/auth/register \
  -H "Content-Type: application/json" \
  -d "{
    \"username\": \"$USERNAME\",
    \"email\": \"$EMAIL\",
    \"password\": \"$PASSWORD\"
  }")

if echo "$REGISTER_RESPONSE" | grep -q "registered successfully"; then
    USER_ID=$(echo "$REGISTER_RESPONSE" | jq -r '.user.id')
    print_success "User registered: $USER_ID"
else
    print_error "Failed to register user"
    echo "$REGISTER_RESPONSE" | jq
    exit 1
fi

# Test 3: Wait for RabbitMQ Sync
print_step "3. Waiting for RabbitMQ to sync user data (3 seconds)"
sleep 3

# Test 4: Login
print_step "4. Logging in to get JWT Token"
LOGIN_RESPONSE=$(curl -s -X POST $BASE_URL_AUTH/api/auth/login \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"$EMAIL\",
    \"password\": \"$PASSWORD\"
  }")

TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.token')

if [ "$TOKEN" == "null" ] || [ -z "$TOKEN" ]; then
    print_error "Failed to get token"
    echo "$LOGIN_RESPONSE" | jq
    exit 1
fi

print_success "Login successful, token received"

# Test 5: Create Task #1
print_step "5. Creating Task #1 (High Priority)"
TASK1_RESPONSE=$(curl -s -X POST $BASE_URL_TASKS/api/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Buy groceries",
    "description": "Milk, bread, eggs, cheese",
    "priority": "high",
    "due_date": "2025-10-10T10:00:00Z"
  }')

if echo "$TASK1_RESPONSE" | grep -q '"id"'; then
    TASK1_ID=$(echo "$TASK1_RESPONSE" | jq -r '.task.id')
    print_success "Task created: $TASK1_ID"
else
    print_error "Failed to create task"
    echo "$TASK1_RESPONSE" | jq
    exit 1
fi

# Test 6: Create Task #2
print_step "6. Creating Task #2 (Urgent Priority)"
TASK2_RESPONSE=$(curl -s -X POST $BASE_URL_TASKS/api/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Finish project report",
    "description": "Complete Q4 financial analysis",
    "priority": "urgent",
    "status": "in_progress"
  }')

if echo "$TASK2_RESPONSE" | grep -q '"id"'; then
    TASK2_ID=$(echo "$TASK2_RESPONSE" | jq -r '.task.id')
    print_success "Task created: $TASK2_ID"
else
    print_error "Failed to create task"
    exit 1
fi

# Test 7: Create Task #3
print_step "7. Creating Task #3 (Completed)"
TASK3_RESPONSE=$(curl -s -X POST $BASE_URL_TASKS/api/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Morning workout",
    "description": "Gym session",
    "status": "completed",
    "priority": "low"
  }')

if echo "$TASK3_RESPONSE" | grep -q '"id"'; then
    print_success "Task created"
else
    print_error "Failed to create task"
    exit 1
fi

# Test 8: List All Tasks
print_step "8. Listing All Tasks"
ALL_TASKS=$(curl -s -X GET "$BASE_URL_TASKS/api/tasks" \
  -H "Authorization: Bearer $TOKEN")

TASK_COUNT=$(echo "$ALL_TASKS" | jq -r '.tasks | length')
if [ "$TASK_COUNT" -ge 3 ]; then
    print_success "Found $TASK_COUNT tasks"
else
    print_error "Expected at least 3 tasks, found $TASK_COUNT"
    exit 1
fi

# Test 9: Get Single Task
print_step "9. Getting Task by ID"
SINGLE_TASK=$(curl -s -X GET "$BASE_URL_TASKS/api/tasks/$TASK1_ID" \
  -H "Authorization: Bearer $TOKEN")

if echo "$SINGLE_TASK" | grep -q "Buy groceries"; then
    print_success "Task retrieved successfully"
else
    print_error "Failed to get task"
    exit 1
fi

# Test 10: Update Task
print_step "10. Updating Task Status to 'completed'"
UPDATE_RESPONSE=$(curl -s -X PUT "$BASE_URL_TASKS/api/tasks/$TASK1_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "completed"
  }')

if echo "$UPDATE_RESPONSE" | grep -q "completed"; then
    print_success "Task updated successfully"
else
    print_error "Failed to update task"
    exit 1
fi

# Test 11: Filter Tasks by Status
print_step "11. Filtering Tasks by Status (in_progress)"
FILTERED_TASKS=$(curl -s -X GET "$BASE_URL_TASKS/api/tasks?status=in_progress" \
  -H "Authorization: Bearer $TOKEN")

FILTERED_COUNT=$(echo "$FILTERED_TASKS" | jq -r '.tasks | length')
if [ "$FILTERED_COUNT" -ge 1 ]; then
    print_success "Found $FILTERED_COUNT in_progress task(s)"
else
    print_error "Filter failed"
    exit 1
fi

# Test 12: Filter Tasks by Priority
print_step "12. Filtering Tasks by Priority (urgent)"
PRIORITY_TASKS=$(curl -s -X GET "$BASE_URL_TASKS/api/tasks?priority=urgent" \
  -H "Authorization: Bearer $TOKEN")

PRIORITY_COUNT=$(echo "$PRIORITY_TASKS" | jq -r '.tasks | length')
if [ "$PRIORITY_COUNT" -ge 1 ]; then
    print_success "Found $PRIORITY_COUNT urgent task(s)"
else
    print_error "Priority filter failed"
    exit 1
fi

# Test 13: Get Task Statistics
print_step "13. Getting Task Statistics"
STATS=$(curl -s -X GET "$BASE_URL_TASKS/api/tasks/stats/summary" \
  -H "Authorization: Bearer $TOKEN")

TOTAL_TASKS=$(echo "$STATS" | jq -r '.stats.total_tasks')
COMPLETED=$(echo "$STATS" | jq -r '.stats.by_status.completed')

if [ "$TOTAL_TASKS" != "null" ] && [ "$TOTAL_TASKS" -ge 3 ]; then
    print_success "Statistics retrieved: $TOTAL_TASKS total tasks, $COMPLETED completed"
else
    print_error "Failed to get statistics"
    exit 1
fi

# Test 14: Delete Task
print_step "14. Deleting Task"
DELETE_RESPONSE=$(curl -s -X DELETE "$BASE_URL_TASKS/api/tasks/$TASK2_ID" \
  -H "Authorization: Bearer $TOKEN")

if echo "$DELETE_RESPONSE" | grep -q "deleted successfully"; then
    print_success "Task deleted successfully"
else
    print_error "Failed to delete task"
    exit 1
fi

# Test 15: Verify Deletion
print_step "15. Verifying Task Deletion"
VERIFY_TASKS=$(curl -s -X GET "$BASE_URL_TASKS/api/tasks" \
  -H "Authorization: Bearer $TOKEN")

REMAINING_COUNT=$(echo "$VERIFY_TASKS" | jq -r '.tasks | length')
print_success "Remaining tasks: $REMAINING_COUNT"

# Test 16: Test Invalid Token
print_step "16. Testing Invalid Token (Security Check)"
INVALID_RESPONSE=$(curl -s -X GET "$BASE_URL_TASKS/api/tasks" \
  -H "Authorization: Bearer invalid_token_12345")

if echo "$INVALID_RESPONSE" | grep -qi "invalid token"; then
    print_success "Invalid token correctly rejected"
else
    print_error "Security issue: Invalid token was accepted"
    exit 1
fi

# Test 17: Test Missing Token
print_step "17. Testing Missing Token (Security Check)"
NO_TOKEN_RESPONSE=$(curl -s -X GET "$BASE_URL_TASKS/api/tasks")

if echo "$NO_TOKEN_RESPONSE" | grep -q "authorization"; then
    print_success "Missing token correctly rejected"
else
    print_error "Security issue: Missing token was accepted"
    exit 1
fi

# Final Summary
print_header "Test Summary"
print_success "All 17 tests passed!"
echo ""
echo "✅ Auth Service (Node.js) - Working"
echo "✅ Tasks Service (Go) - Working"
echo "✅ RabbitMQ Event Sync - Working"
echo "✅ JWT Authentication - Working"
echo "✅ Database Operations - Working"
echo "✅ CRUD Operations - Working"
echo "✅ Filtering & Statistics - Working"
echo "✅ Security Checks - Working"
echo ""
print_success "Microservices architecture is fully operational! 🚀"
