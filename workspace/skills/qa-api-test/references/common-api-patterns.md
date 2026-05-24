# Common API Patterns — Testing Checklist

## REST Conventions to Verify

| Method | Endpoint | Expected Status | Check |
|--------|----------|----------------|-------|
| GET | /resources | 200 + array | Pagination params work? |
| GET | /resources/:id | 200 + object | 404 for invalid ID? |
| POST | /resources | 201 + created object | 400 for missing fields? |
| PUT | /resources/:id | 200 + updated object | 404 for invalid ID? |
| PATCH | /resources/:id | 200 + partial update | Ignores unknown fields? |
| DELETE | /resources/:id | 204 or 200 | Idempotent (2nd delete)? |

## Pagination Testing

```bash
# First page
curl -s "URL/resources?page=1&limit=10" | jq 'length'

# Beyond last page — should return empty array, not error
curl -s "URL/resources?page=99999" | jq 'length'

# Zero / negative limit
curl -s "URL/resources?limit=0"
curl -s "URL/resources?limit=-1"

# Very large limit — resource exhaustion?
curl -s "URL/resources?limit=100000"
```

## Common Bugs to Check

1. **No pagination** — GET /resources returns everything (performance issue)
2. **ID enumeration** — Sequential IDs allow scraping all records
3. **Mass assignment** — POST/PUT accepts fields it shouldn't (role, isAdmin)
4. **Missing Content-Type validation** — Accepts any content type
5. **CORS too permissive** — `Access-Control-Allow-Origin: *` on authenticated endpoints
6. **Error message leaks** — Stack traces, SQL queries, or internal paths in error responses
7. **Inconsistent error format** — Different error shapes across endpoints
