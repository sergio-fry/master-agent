---
id: TASK-25
title: 'Web UI: login/password and session auth'
status: Done
assignee: []
created_date: '2026-08-31 21:02'
updated_date: '2026-09-01 08:55'
labels: []
milestone: m-1
dependencies:
  - TASK-13
  - TASK-19
references:
  - specs/07-web-ui.md
  - deploy/.env.example
  - docker-compose.yml
priority: high
type: feature
ordinal: 320
modified_files:
  - internal/api/session.go
  - internal/api/session_test.go
  - internal/api/session_manager_test.go
  - internal/api/server.go
  - internal/api/middleware.go
  - internal/cli/http.go
  - internal/webui/static/auth.js
  - internal/webui/static/login.html
  - internal/webui/static/login.js
  - internal/webui/static/app.js
  - internal/webui/static/tasks.js
  - internal/webui/static/runs.js
  - internal/webui/static/index.html
  - internal/webui/static/tasks.html
  - internal/webui/static/runs.html
  - internal/webui/static/style.css
  - internal/webui/webui_test.go
  - deploy/install.sh
  - deploy/upgrade.sh
  - deploy/.env.example
  - deploy/docker-compose.prod.yml
  - docker-compose.yml
  - README.md
  - specs/07-web-ui.md
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Replace the Bearer-token prompt in the Web UI with a standard login form. A single admin account is configured via environment variables (login + password). After successful login, the server issues a session (cookie) that persists across browser restarts until logout or expiry. Unauthenticated browser/API requests are rejected when admin credentials are configured.

Deprecate or remove the manual `MASTER_AGENT_TOKEN` paste flow from the UI; document new env vars in README and deploy templates.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 When `ADMIN_USERNAME` and `ADMIN_PASSWORD` are set, unauthenticated requests to `/api/v1/*` return 401; static login page is served without prior auth
- [x] #2 POST login with valid credentials sets an HttpOnly session cookie; subsequent API/UI requests succeed without Bearer header
- [x] #3 Invalid credentials return a clear error; logout clears the session
- [x] #4 Session survives browser refresh/reopen (within configured TTL); password is never returned in API responses
- [x] #5 When admin env vars are unset, behavior matches today (open API / optional Bearer) or is documented explicitly
- [x] #6 deploy/.env.example, docker-compose, and README updated for the new variables
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add env vars `ADMIN_USERNAME`, `ADMIN_PASSWORD`, optional `SESSION_SECRET` and session TTL.
2. Implement session store (signed cookie or server-side in SQLite) and login/logout handlers.
3. Extend auth middleware: accept valid session cookie in addition to (or instead of) Bearer for browser use.
4. Add login page and remove token panel from static UI; redirect unauthenticated users to login.
5. Unit tests for login, session, logout, and middleware; update acceptance tests if needed.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Added ADMIN_USERNAME/ADMIN_PASSWORD session auth with signed HttpOnly cookies and server-side session IDs (revoked on logout). Web UI uses login.html and shared auth.js; MASTER_AGENT_TOKEN remains optional for API clients.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Web UI login/password auth: POST /api/v1/auth/login and /logout, session cookies with TTL (default 7d), login page replaces Bearer token panel. Deploy/install generates admin credentials. Verified with go test ./...
<!-- SECTION:FINAL_SUMMARY:END -->
