# HTTP Status Codes — Quick Reference

## Success (2xx)
| Code | Meaning | When Expected |
|------|---------|---------------|
| 200 | OK | GET, PUT, PATCH success |
| 201 | Created | POST that creates a resource |
| 204 | No Content | DELETE success, or PUT with no body returned |

## Redirection (3xx)
| Code | Meaning | Notes |
|------|---------|-------|
| 301 | Moved Permanently | URL changed, follow Location header |
| 302 | Found (Temporary) | Temporary redirect |
| 304 | Not Modified | Caching — ETag/If-None-Match match |

## Client Errors (4xx)
| Code | Meaning | Common Bug |
|------|---------|------------|
| 400 | Bad Request | Missing validation, should include error message |
| 401 | Unauthorized | Missing or invalid auth — check if 403 is more appropriate |
| 403 | Forbidden | Valid auth but insufficient permissions |
| 404 | Not Found | Resource doesn't exist |
| 405 | Method Not Allowed | Wrong HTTP method on endpoint |
| 409 | Conflict | Duplicate resource or state conflict |
| 422 | Unprocessable Entity | Validation failed on well-formed request |
| 429 | Too Many Requests | Rate limiting — check Retry-After header |

## Server Errors (5xx)
| Code | Meaning | Always a Bug? |
|------|---------|---------------|
| 500 | Internal Server Error | Almost always a bug — server should handle gracefully |
| 502 | Bad Gateway | Upstream service failure |
| 503 | Service Unavailable | Server overloaded or in maintenance |
| 504 | Gateway Timeout | Upstream timeout |

## Common API Bugs by Status Code

- **200 on error**: API returns 200 with error body instead of proper 4xx/5xx
- **500 on bad input**: Should be 400/422, not 500 — indicates unhandled exception
- **404 vs 403**: Returning 404 for forbidden resources (info leak) or 403 for missing resources
- **Missing CORS headers**: Preflight OPTIONS returns wrong status or missing headers
- **No rate limiting**: Endpoint allows unlimited requests (should return 429)
