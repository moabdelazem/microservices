#!/bin/bash

BASE_URL="http://localhost:3001"

echo "=== Testing Auth Service ==="

# Test 1: Health Check
echo -e "\n1. Health Check:"
curl -s $BASE_URL/health | jq

# Test 2: Register User
echo -e "\n2. Register User:"
curl -s -X POST $BASE_URL/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser_'$RANDOM'",
    "email": "test'$RANDOM'@example.com",
    "password": "SecurePass123"
  }' | jq

# Test 3: Invalid Registration (weak password)
echo -e "\n3. Invalid Registration (weak password):"
curl -s -X POST $BASE_URL/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser2",
    "email": "test2@example.com",
    "password": "weak"
  }' | jq

# Test 4: Login
echo -e "\n4. Login:"
curl -s -X POST $BASE_URL/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "SecurePass123"
  }' | jq

echo -e "\n=== Tests Complete ==="