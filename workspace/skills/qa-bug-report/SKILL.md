---
name: qa-bug-report
description: "File bugs as GitHub Issues with structured templates. Use for reporting confirmed bugs with reproduction steps, severity, and evidence. Also creates test session summary issues. Requires `gh` CLI."
metadata: {"nanobot":{"emoji":"🐛","requires":{"bins":["gh"]}}}
---

# QA Bug Report

File confirmed bugs as GitHub Issues using the `gh` CLI.

## Before Filing

1. Verify the bug is reproducible (test at least twice)
2. Check for duplicates:

```bash
gh issue list --repo OWNER/REPO --label bug --search "KEYWORD" --json number,title --jq '.[] | "#\(.number): \(.title)"'
```

3. If a duplicate exists, comment on it instead of filing a new issue

## File a Bug

```bash
gh issue create --repo OWNER/REPO \
  --title "SHORT DESCRIPTION" \
  --label "bug,qa-automated" \
  --body "$(cat <<'EOF'
## Summary
One-line description of the bug.

## Severity
**critical** | **high** | **medium** | **low**

## Steps to Reproduce
1. Step one
2. Step two
3. Step three

## Expected Behavior
What should happen.

## Actual Behavior
What actually happens.

## Evidence
- Status code: XXX
- Response body (relevant excerpt):
```
PASTE HERE
```
- Screenshot: (path or URL if available)

## Environment
- URL: TARGET_URL
- Method: GET/POST/etc.
- Timestamp: YYYY-MM-DD HH:MM UTC

---
*Filed by QA AgentX 🤖*
EOF
)"
```

## Severity Guide

| Severity | Criteria |
|----------|----------|
| critical | Security breach, data loss, app crash, auth bypass |
| high | Major feature broken, no workaround available |
| medium | Feature works incorrectly, workaround exists |
| low | Cosmetic, minor UX, typo |

## Test Session Summary

After completing all tests, create a summary issue:

```bash
gh issue create --repo OWNER/REPO \
  --title "QA Test Session Summary — $(date +%Y-%m-%d)" \
  --label "qa-automated" \
  --body "$(cat <<'EOF'
## QA Test Session Summary

**Target:** TARGET_URL
**Date:** YYYY-MM-DD
**Agent:** QA AgentX 🤖

## Results

| Category | Tests | Pass | Fail | Skip |
|----------|-------|------|------|------|
| API      |       |      |      |      |
| UI       |       |      |      |      |
| Security |       |      |      |      |
| Accessibility |  |      |      |      |
| **Total** |      |      |      |      |

## Issues Filed
- #XX — Description (severity)
- #XX — Description (severity)

## Notes
Additional observations or recommendations.

---
*Filed by QA AgentX 🤖*
EOF
)"
```
