#!/usr/bin/env bash
set -euo pipefail

URL="${1:?Usage: check-headers.sh URL}"

echo "Security Header Audit: $URL"
echo "Date: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "---"

headers=$(curl -sI "$URL")

check_header() {
  local name="$1"
  local value
  value=$(echo "$headers" | grep -i "^$name:" | head -1 | sed "s/^[^:]*: //" | tr -d '\r')
  if [ -n "$value" ]; then
    echo "PASS  $name: $value"
  else
    echo "FAIL  $name: MISSING"
  fi
}

check_header "Strict-Transport-Security"
check_header "Content-Security-Policy"
check_header "X-Frame-Options"
check_header "X-Content-Type-Options"
check_header "Referrer-Policy"
check_header "Permissions-Policy"

echo "---"

# Check for info-leaking headers
for h in Server X-Powered-By X-AspNet-Version X-AspNetMvc-Version; do
  value=$(echo "$headers" | grep -i "^$h:" | head -1 | sed "s/^[^:]*: //" | tr -d '\r')
  if [ -n "$value" ]; then
    echo "WARN  $h: $value (information disclosure)"
  fi
done

echo "---"

# Cookie security
cookies=$(echo "$headers" | grep -i "^set-cookie:" || true)
if [ -n "$cookies" ]; then
  echo "Cookie Analysis:"
  echo "$cookies" | while IFS= read -r line; do
    cookie=$(echo "$line" | sed "s/^[^:]*: //" | tr -d '\r')
    name=$(echo "$cookie" | cut -d= -f1)
    issues=""
    echo "$cookie" | grep -qi "secure" || issues="${issues} missing-Secure"
    echo "$cookie" | grep -qi "httponly" || issues="${issues} missing-HttpOnly"
    echo "$cookie" | grep -qi "samesite" || issues="${issues} missing-SameSite"
    if [ -n "$issues" ]; then
      echo "  WARN  $name:$issues"
    else
      echo "  PASS  $name: all flags present"
    fi
  done
else
  echo "No cookies set."
fi
