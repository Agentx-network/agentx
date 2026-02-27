---
name: qa-security
description: "Basic security testing for web applications. Checks security headers, sensitive file exposure, CORS misconfiguration, basic injection vectors, and auth bypass. Use when testing a web app's security posture. Requires `curl`."
metadata: {"nanobot":{"emoji":"🔒","requires":{"bins":["curl"]}}}
---

# QA Security Test

Basic security checks using `curl`. These are non-destructive, read-only tests safe to run against any target with authorization.

**Important:** Only run security tests with explicit user authorization. Ask before testing if scope is unclear.

## 1. Security Headers

Check for required security headers:

```bash
curl -sI "URL" | grep -iE "^(strict-transport|content-security-policy|x-frame-options|x-content-type|referrer-policy|permissions-policy)" || echo "MISSING security headers"
```

Expected headers:
| Header | Expected Value | Risk if Missing |
|--------|---------------|-----------------|
| Strict-Transport-Security | max-age=31536000; includeSubDomains | Downgrade attacks |
| Content-Security-Policy | Restrictive policy | XSS attacks |
| X-Frame-Options | DENY or SAMEORIGIN | Clickjacking |
| X-Content-Type-Options | nosniff | MIME sniffing attacks |
| Referrer-Policy | strict-origin-when-cross-origin | URL leakage |
| Permissions-Policy | Restrictive | Feature abuse |

Full header audit script:

```bash
bash {baseDir}/scripts/check-headers.sh "URL"
```

## 2. Sensitive File Exposure

Check if common sensitive paths are accessible:

```bash
for path in .env .git/config .git/HEAD wp-config.php .DS_Store .htaccess server-status debug phpinfo.php; do
  code=$(curl -s -o /dev/null -w "%{http_code}" "URL/$path")
  if [ "$code" != "404" ] && [ "$code" != "403" ]; then
    echo "EXPOSED: /$path (HTTP $code)"
  fi
done
```

## 3. CORS Misconfiguration

```bash
# Test with arbitrary origin
curl -sI -H "Origin: https://evil.com" "URL" | grep -i "access-control"

# Test with null origin
curl -sI -H "Origin: null" "URL" | grep -i "access-control"
```

**Bugs to report:**
- `Access-Control-Allow-Origin: *` on authenticated endpoints
- Origin reflection (echoes back any Origin header)
- `Access-Control-Allow-Credentials: true` with wildcard origin

## 4. Basic Injection Probes

Test for reflected input in responses (non-destructive):

```bash
# XSS probe — check if input is reflected unescaped
curl -s "URL/search?q=<script>alert(1)</script>" | grep -c "<script>alert(1)</script>" && echo "POTENTIAL XSS" || echo "OK — input escaped or not reflected"

# SQL error probe — check for database error messages
curl -s "URL/search?q='" | grep -ciE "(sql|syntax|mysql|postgresql|sqlite|oracle|unclosed)" && echo "POTENTIAL SQL INJECTION" || echo "OK — no SQL errors"
```

## 5. Auth Bypass Checks

```bash
# Access protected endpoint without auth
curl -s -w "\n%{http_code}" "URL/admin"
curl -s -w "\n%{http_code}" "URL/api/users"

# HTTP method override
curl -s -w "\n%{http_code}" -X POST "URL/admin"
curl -s -w "\n%{http_code}" -H "X-HTTP-Method-Override: GET" -X POST "URL/admin"

# Path traversal on API
curl -s -w "\n%{http_code}" "URL/api/../../etc/passwd"
```

## 6. Cookie Security

```bash
curl -sI "URL" | grep -i "set-cookie"
```

Check each cookie for:
- `Secure` flag (required for HTTPS)
- `HttpOnly` flag (prevents JS access)
- `SameSite` attribute (CSRF protection)

## Recording Results

```bash
mkdir -p test-results/security
bash {baseDir}/scripts/check-headers.sh "URL" > test-results/security/headers.txt 2>&1
```
