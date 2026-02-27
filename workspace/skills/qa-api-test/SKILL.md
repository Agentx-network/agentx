---
name: qa-api-test
description: "Test REST and GraphQL APIs systematically with curl + jq. Covers happy path, input validation, auth, error handling, and performance. Use when testing API endpoints, verifying status codes, or checking response schemas. Requires `curl` and `jq`."
metadata: {"nanobot":{"emoji":"🔌","requires":{"bins":["curl","jq"]}}}
---

# QA API Test

Systematic API testing using `curl` and `jq`.

## Test Categories

Run in this order for each endpoint:

### 1. Happy Path
Verify the endpoint works with valid input.

```bash
curl -s -w "\n%{http_code}" "URL/endpoint" | {
  body=$(head -n -1)
  code=$(tail -1)
  echo "Status: $code"
  echo "$body" | jq .
}
```

### 2. Input Validation
Test with invalid, missing, boundary, and malformed inputs.

```bash
# Empty body
curl -s -w "\n%{http_code}" -X POST "URL/endpoint" -H "Content-Type: application/json" -d '{}'

# Missing required fields
curl -s -w "\n%{http_code}" -X POST "URL/endpoint" -H "Content-Type: application/json" -d '{"title":""}'

# Boundary values
curl -s -w "\n%{http_code}" "URL/endpoint?page=0"
curl -s -w "\n%{http_code}" "URL/endpoint?page=-1"
curl -s -w "\n%{http_code}" "URL/endpoint?page=999999999"

# Type confusion
curl -s -w "\n%{http_code}" "URL/endpoint?id=abc"
```

### 3. Auth & Authorization
Test without credentials, with invalid credentials, and across user boundaries.

```bash
# No auth header
curl -s -w "\n%{http_code}" "URL/protected-endpoint"

# Invalid token
curl -s -w "\n%{http_code}" -H "Authorization: Bearer invalid_token" "URL/protected-endpoint"
```

### 4. Error Handling
Verify proper error responses.

```bash
# Non-existent resource
curl -s -w "\n%{http_code}" "URL/endpoint/99999999"

# Wrong HTTP method
curl -s -w "\n%{http_code}" -X DELETE "URL/endpoint"

# Malformed JSON
curl -s -w "\n%{http_code}" -X POST "URL/endpoint" -H "Content-Type: application/json" -d '{invalid}'
```

### 5. Performance
Check response times.

```bash
curl -s -o /dev/null -w "time_total: %{time_total}s\ntime_connect: %{time_connect}s\nsize_download: %{size_download} bytes\n" "URL/endpoint"
```

## Response Validation

For each response, verify:
- **Status code** matches expectation (see `references/http-status-codes.md`)
- **Content-Type** header is correct
- **JSON structure** contains expected fields
- **Data types** are correct (string vs number vs null)
- **Error responses** include a message field

```bash
# Validate JSON structure has expected fields
curl -s "URL/endpoint" | jq 'keys'

# Check specific field types
curl -s "URL/endpoint" | jq 'type'

# Count array items
curl -s "URL/endpoint" | jq 'length'
```

## Recording Results

Save test output to `test-results/api/`:

```bash
mkdir -p test-results/api
curl -s -w "\nHTTP_CODE:%{http_code}" "URL/endpoint" > test-results/api/endpoint-get.txt 2>&1
```
