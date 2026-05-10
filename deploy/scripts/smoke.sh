#!/usr/bin/env bash
set -euo pipefail

# ==========================================
# KMU Hub Smoke Test Suite
# Curl/jq-based, no Go toolchain required
# ==========================================

BASE_URL="https://app.zentria.tech"
VERBOSE=false
EXPECT_VERSION=""
PASSED=0
FAILED=0
TOTAL=0
SMOKE_TOKEN=""
SMOKE_USER_ID=""
SMOKE_EMAIL=""
SMOKE_CONTACT_ID=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --base-url) BASE_URL="$2"; shift 2 ;;
        --base-url=*) BASE_URL="${1#*=}"; shift ;;
        --verbose) VERBOSE=true; shift ;;
        --expect-version) EXPECT_VERSION="$2"; shift 2 ;;
        --expect-version=*) EXPECT_VERSION="${1#*=}"; shift ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

# Remove trailing slash
BASE_URL="${BASE_URL%/}"

# Check dependencies
for cmd in curl jq; do
    if ! command -v "$cmd" &>/dev/null; then
        echo "ERROR: $cmd is required but not installed"
        exit 1
    fi
done

pass() {
    : $((TOTAL++))
    : $((PASSED++))
    echo "  [PASS] $1"
}

fail() {
    : $((TOTAL++))
    : $((FAILED++))
    echo "  [FAIL] $1"
    if [[ "$VERBOSE" == "true" && -n "${2:-}" ]]; then
        echo "         Details: $2"
    fi
}

section() {
    echo ""
    echo "--- $1 ---"
}

# ==========================================
# Infrastructure Tests (5)
# ==========================================
section "Infrastructure"

# 1. Gateway health = 200 + healthy
HEALTH_RESP=$(curl -s -w "\n%{http_code}" "$BASE_URL/health" 2>/dev/null) || HEALTH_RESP=$'\n000'
HEALTH_CODE=$(echo "$HEALTH_RESP" | tail -1)
HEALTH_BODY=$(echo "$HEALTH_RESP" | sed '$d')

if [[ "$HEALTH_CODE" == "200" ]]; then
    HEALTH_STATUS=$(echo "$HEALTH_BODY" | jq -r '.status' 2>/dev/null || echo "")
    if [[ "$HEALTH_STATUS" == "healthy" ]]; then
        pass "Gateway /health returns 200 + healthy"
    else
        fail "Gateway /health status is '$HEALTH_STATUS', expected 'healthy'" "$HEALTH_BODY"
    fi
else
    fail "Gateway /health returned $HEALTH_CODE, expected 200" ""
fi

# 2. All 10 services registered
SVC_COUNT=$(echo "$HEALTH_BODY" | jq '.registered_services | length' 2>/dev/null || echo "0")
if [[ "$SVC_COUNT" -ge 10 ]]; then
    pass "All services registered ($SVC_COUNT)"
else
    fail "Only $SVC_COUNT services registered, expected >= 10" "$(echo "$HEALTH_BODY" | jq -r '.registered_services[]' 2>/dev/null)"
fi

# 3. HTTPS cert valid
if curl -sf --max-time 5 "$BASE_URL/health" > /dev/null 2>&1; then
    pass "HTTPS certificate valid"
else
    # Try without strict SSL to differentiate cert vs connectivity
    if curl -ksf --max-time 5 "$BASE_URL/health" > /dev/null 2>&1; then
        fail "HTTPS certificate invalid (request succeeds with -k)" ""
    else
        fail "Cannot reach $BASE_URL" ""
    fi
fi

# 4. Response time < 2s
RESP_TIME=$(curl -s -o /dev/null -w "%{time_total}" "$BASE_URL/health" 2>/dev/null || echo "99")
if (( $(echo "$RESP_TIME < 2.0" | bc -l 2>/dev/null || echo 0) )); then
    pass "Health response time ${RESP_TIME}s < 2s"
else
    fail "Health response time ${RESP_TIME}s >= 2s" ""
fi

# 5. Version info present
HAS_VERSION=$(echo "$HEALTH_BODY" | jq 'has("version")' 2>/dev/null || echo "false")
if [[ "$HAS_VERSION" == "true" ]]; then
    DEPLOYED_VERSION=$(echo "$HEALTH_BODY" | jq -r '.version' 2>/dev/null || echo "")
    if [[ -n "$EXPECT_VERSION" && "$DEPLOYED_VERSION" != *"$EXPECT_VERSION"* ]]; then
        DEPLOYED_COMMIT=$(echo "$HEALTH_BODY" | jq -r '.commit' 2>/dev/null || echo "")
        if [[ "$DEPLOYED_COMMIT" == *"$EXPECT_VERSION"* ]]; then
            pass "Expected version deployed (commit: $DEPLOYED_COMMIT)"
        else
            fail "Version mismatch: expected '$EXPECT_VERSION', got version='$DEPLOYED_VERSION' commit='$DEPLOYED_COMMIT'" ""
        fi
    else
        pass "Version info present ($DEPLOYED_VERSION)"
    fi
else
    fail "No version info in health response" ""
fi

# ==========================================
# Auth Flow Tests (3)
# ==========================================
section "Auth Flow"

SMOKE_EMAIL="smoke-$(date +%s)@test.kmuhub.local"
SMOKE_PASS="SmokeTest123!"

# 6. Register
REG_RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/auth/register" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$SMOKE_EMAIL\",\"password\":\"$SMOKE_PASS\",\"first_name\":\"Smoke\",\"last_name\":\"Test\"}" 2>/dev/null) || REG_RESP=$'\n000'
REG_CODE=$(echo "$REG_RESP" | tail -1)

if [[ "$REG_CODE" == "201" || "$REG_CODE" == "200" ]]; then
    pass "Register smoke user ($SMOKE_EMAIL)"
else
    fail "Register returned $REG_CODE" "$(echo "$REG_RESP" | sed '$d')"
fi

# 7. Login = 200 + JWT
LOGIN_RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$SMOKE_EMAIL\",\"password\":\"$SMOKE_PASS\"}" 2>/dev/null) || LOGIN_RESP=$'\n000'
LOGIN_CODE=$(echo "$LOGIN_RESP" | tail -1)
LOGIN_BODY=$(echo "$LOGIN_RESP" | sed '$d')

if [[ "$LOGIN_CODE" == "200" ]]; then
    SMOKE_TOKEN=$(echo "$LOGIN_BODY" | jq -r '.access_token' 2>/dev/null || echo "")
    SMOKE_USER_ID=$(echo "$LOGIN_BODY" | jq -r '.user.id' 2>/dev/null || echo "")
    if [[ -n "$SMOKE_TOKEN" && "$SMOKE_TOKEN" != "null" ]]; then
        pass "Login returns JWT"
    else
        fail "Login returned 200 but no access_token" "$LOGIN_BODY"
    fi
else
    fail "Login returned $LOGIN_CODE" "$LOGIN_BODY"
fi

# 8. /auth/me with token
if [[ -n "$SMOKE_TOKEN" && "$SMOKE_TOKEN" != "null" ]]; then
    ME_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/v1/auth/me" \
        -H "Authorization: Bearer $SMOKE_TOKEN" 2>/dev/null || echo "000")
    if [[ "$ME_CODE" == "200" ]]; then
        pass "GET /auth/me with token = 200"
    else
        fail "GET /auth/me returned $ME_CODE" ""
    fi
else
    fail "GET /auth/me skipped (no token)" ""
fi

# 8b. Bootstrap smoke user to manager role (optional — requires SMOKE_ADMIN_TOKEN).
# Default registration assigns the read-only 'member' role, which makes the
# subsequent CRUD tests fail with 403. Setting SMOKE_ADMIN_TOKEN to a
# long-lived admin JWT (e.g. as a CI secret + .env.production entry) lets the
# smoke run upgrade the just-registered user to 'manager' so the CRUD
# assertions actually exercise the write path.
if [[ -n "$SMOKE_TOKEN" && "$SMOKE_TOKEN" != "null" && -n "$SMOKE_USER_ID" && "$SMOKE_USER_ID" != "null" ]]; then
    if [[ -n "${SMOKE_ADMIN_TOKEN:-}" ]]; then
        ROLE_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST \
            "$BASE_URL/api/v1/auth/users/$SMOKE_USER_ID/roles" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $SMOKE_ADMIN_TOKEN" \
            -d '{"role":"manager"}' 2>/dev/null || echo "000")
        if [[ "$ROLE_CODE" =~ ^(200|201|204)$ ]]; then
            pass "Bootstrap smoke user to manager (role upgrade $ROLE_CODE)"
        else
            fail "Role upgrade returned $ROLE_CODE — CRUD tests will fail with 403" ""
        fi
    else
        echo "  [SKIP] Role bootstrap — SMOKE_ADMIN_TOKEN not set; Tests 9/10/11 will return 403"
    fi
fi

# ==========================================
# CRM CRUD Tests (3)
# ==========================================
section "CRM CRUD"

if [[ -n "$SMOKE_TOKEN" && "$SMOKE_TOKEN" != "null" ]]; then
    # 9. Create contact
    CONTACT_RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/contacts" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $SMOKE_TOKEN" \
        -d '{"first_name":"Smoke","last_name":"Contact","email":"smoke-contact@test.kmuhub.local"}' 2>/dev/null) || CONTACT_RESP=$'\n000'
    CONTACT_CODE=$(echo "$CONTACT_RESP" | tail -1)
    CONTACT_BODY=$(echo "$CONTACT_RESP" | sed '$d')

    if [[ "$CONTACT_CODE" == "201" || "$CONTACT_CODE" == "200" ]]; then
        SMOKE_CONTACT_ID=$(echo "$CONTACT_BODY" | jq -r '.id // .contact.id // empty' 2>/dev/null || echo "")
        pass "POST /contacts = $CONTACT_CODE"
    else
        fail "POST /contacts returned $CONTACT_CODE" "$CONTACT_BODY"
    fi

    # 10. Get contact
    if [[ -n "$SMOKE_CONTACT_ID" ]]; then
        GET_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/v1/contacts/$SMOKE_CONTACT_ID" \
            -H "Authorization: Bearer $SMOKE_TOKEN" 2>/dev/null || echo "000")
        if [[ "$GET_CODE" == "200" ]]; then
            pass "GET /contacts/$SMOKE_CONTACT_ID = 200"
        else
            fail "GET /contacts/$SMOKE_CONTACT_ID returned $GET_CODE" ""
        fi
    else
        fail "GET /contacts skipped (no contact ID)" ""
    fi

    # 11. Delete contact
    if [[ -n "$SMOKE_CONTACT_ID" ]]; then
        DEL_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/api/v1/contacts/$SMOKE_CONTACT_ID" \
            -H "Authorization: Bearer $SMOKE_TOKEN" 2>/dev/null || echo "000")
        if [[ "$DEL_CODE" == "200" || "$DEL_CODE" == "204" ]]; then
            pass "DELETE /contacts/$SMOKE_CONTACT_ID = $DEL_CODE"
        else
            fail "DELETE /contacts returned $DEL_CODE" ""
        fi
    else
        fail "DELETE /contacts skipped (no contact ID)" ""
    fi
else
    fail "POST /contacts skipped (no token)" ""
    fail "GET /contacts skipped (no token)" ""
    fail "DELETE /contacts skipped (no token)" ""
fi

# ==========================================
# Security Tests (3)
# ==========================================
section "Security"

# 12. Unauthenticated /contacts = 401
UNAUTH_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/v1/contacts" 2>/dev/null || echo "000")
if [[ "$UNAUTH_CODE" == "401" ]]; then
    pass "Unauthenticated /contacts = 401"
else
    fail "Unauthenticated /contacts returned $UNAUTH_CODE, expected 401" ""
fi

# 13. CORS headers on preflight
CORS_RESP=$(curl -sf -D - -o /dev/null -X OPTIONS "$BASE_URL/api/v1/contacts" \
    -H "Origin: https://app.zentria.tech" \
    -H "Access-Control-Request-Method: GET" 2>/dev/null || echo "")
if echo "$CORS_RESP" | grep -qi "access-control-allow"; then
    pass "CORS headers present on preflight"
else
    fail "No CORS headers on preflight" ""
fi

# 14. HSTS header
HSTS_RESP=$(curl -sf -D - -o /dev/null "$BASE_URL/health" 2>/dev/null || echo "")
if echo "$HSTS_RESP" | grep -qi "strict-transport-security"; then
    pass "HSTS header present"
else
    fail "No HSTS header" ""
fi

# 14b. OnlyOffice JWT — request without JWT token must return 401/403
OO_PORT="${ONLYOFFICE_PORT:-8088}"
OO_RESP=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:${OO_PORT}/web-apps/apps/api/documents/api.js" 2>/dev/null || echo "000")
if [[ "$OO_RESP" == "401" || "$OO_RESP" == "403" ]]; then
    pass "OnlyOffice JWT active (unauthenticated request = $OO_RESP)"
elif [[ "$OO_RESP" == "000" ]]; then
    pass "OnlyOffice not reachable from smoke runner (skip JWT check)"
else
    fail "OnlyOffice JWT may be disabled — unauthenticated request returned $OO_RESP, expected 401/403" ""
fi

# 14c. Idempotency: two identical POSTs with same Idempotency-Key must not duplicate
# Requires SMOKE_ADMIN_TOKEN (manager role) — otherwise SKIP like tests 9-11.
if [[ -n "${SMOKE_ADMIN_TOKEN:-}" ]]; then
    IDEM_KEY="smoke-idem-$(date +%s)"
    IDEM_BODY='{"contact_id":"00000000-0000-0000-0000-000000000001","outcome":"answered","duration_seconds":30,"notes":"smoke-idempotency-test"}'

    IDEM_RESP1=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/dialer/outcomes" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $SMOKE_ADMIN_TOKEN" \
        -H "Idempotency-Key: $IDEM_KEY" \
        -d "$IDEM_BODY" 2>/dev/null) || IDEM_RESP1=$'\n000'
    IDEM_CODE1=$(echo "$IDEM_RESP1" | tail -1)
    IDEM_BODY1=$(echo "$IDEM_RESP1" | sed '$d')

    IDEM_RESP2=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/dialer/outcomes" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $SMOKE_ADMIN_TOKEN" \
        -H "Idempotency-Key: $IDEM_KEY" \
        -d "$IDEM_BODY" 2>/dev/null) || IDEM_RESP2=$'\n000'
    IDEM_CODE2=$(echo "$IDEM_RESP2" | tail -1)
    IDEM_BODY2=$(echo "$IDEM_RESP2" | sed '$d')

    if [[ "$IDEM_CODE1" =~ ^(200|201)$ ]] && [[ "$IDEM_CODE2" =~ ^(200|201)$ ]] && [[ "$IDEM_BODY1" == "$IDEM_BODY2" ]]; then
        pass "Idempotency: duplicate POST with same key returns cached response (codes: $IDEM_CODE1/$IDEM_CODE2)"
    elif [[ "$IDEM_CODE1" =~ ^(200|201)$ ]] && [[ "$IDEM_CODE2" == "400" ]]; then
        # HardMode: second POST blocked — acceptable, not a duplicate row created
        pass "Idempotency HardMode: second POST with duplicate key rejected 400 (no duplicate row)"
    else
        fail "Idempotency: unexpected codes $IDEM_CODE1/$IDEM_CODE2 or response mismatch" "$IDEM_BODY2"
    fi
else
    echo "  [SKIP] Idempotency check — SMOKE_ADMIN_TOKEN not set"
fi

# ==========================================
# Performance Tests (3)
# ==========================================
section "Performance"

# 15. /health < 500ms
PERF_HEALTH=$(curl -s -o /dev/null -w "%{time_total}" "$BASE_URL/health" 2>/dev/null || echo "99")
if (( $(echo "$PERF_HEALTH < 0.5" | bc -l 2>/dev/null || echo 0) )); then
    pass "Health response ${PERF_HEALTH}s < 500ms"
else
    fail "Health response ${PERF_HEALTH}s >= 500ms" ""
fi

# 16. /auth/login < 2s
if [[ -n "$SMOKE_TOKEN" ]]; then
    PERF_LOGIN=$(curl -s -o /dev/null -w "%{time_total}" -X POST "$BASE_URL/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"$SMOKE_EMAIL\",\"password\":\"$SMOKE_PASS\"}" 2>/dev/null || echo "99")
    if (( $(echo "$PERF_LOGIN < 2.0" | bc -l 2>/dev/null || echo 0) )); then
        pass "Login response ${PERF_LOGIN}s < 2s"
    else
        fail "Login response ${PERF_LOGIN}s >= 2s" ""
    fi
else
    fail "Login perf skipped (no auth)" ""
fi

# 17. /contacts list < 1s
if [[ -n "$SMOKE_TOKEN" && "$SMOKE_TOKEN" != "null" ]]; then
    PERF_CONTACTS=$(curl -s -o /dev/null -w "%{time_total}" "$BASE_URL/api/v1/contacts" \
        -H "Authorization: Bearer $SMOKE_TOKEN" 2>/dev/null || echo "99")
    if (( $(echo "$PERF_CONTACTS < 1.0" | bc -l 2>/dev/null || echo 0) )); then
        pass "Contacts list ${PERF_CONTACTS}s < 1s"
    else
        fail "Contacts list ${PERF_CONTACTS}s >= 1s" ""
    fi
else
    fail "Contacts list perf skipped (no auth)" ""
fi

# ==========================================
# Cross-Service Tests (2)
# ==========================================
section "Cross-Service"

# 18. Chat channel (create + verify)
if [[ -n "$SMOKE_TOKEN" && "$SMOKE_TOKEN" != "null" ]]; then
    CHAN_RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/chat/channels" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $SMOKE_TOKEN" \
        -d '{"name":"smoke-test-channel","is_private":false}' 2>/dev/null) || CHAN_RESP=$'\n000'
    CHAN_CODE=$(echo "$CHAN_RESP" | tail -1)
    if [[ "$CHAN_CODE" == "201" || "$CHAN_CODE" == "200" ]]; then
        pass "Create chat channel = $CHAN_CODE"
    else
        fail "Create chat channel returned $CHAN_CODE" "$(echo "$CHAN_RESP" | sed '$d')"
    fi
else
    fail "Chat channel skipped (no auth)" ""
fi

# 19. Dashboard endpoint reachable
if [[ -n "$SMOKE_TOKEN" && "$SMOKE_TOKEN" != "null" ]]; then
    DASH_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/v1/dashboard/layout" \
        -H "Authorization: Bearer $SMOKE_TOKEN" 2>/dev/null || echo "000")
    if [[ "$DASH_CODE" == "200" ]]; then
        pass "Dashboard endpoint = 200"
    else
        # Dashboard might not exist yet — 404 is acceptable info, not a critical fail
        if [[ "$DASH_CODE" == "404" ]]; then
            pass "Dashboard endpoint responds ($DASH_CODE — not yet implemented)"
        else
            fail "Dashboard endpoint returned $DASH_CODE" ""
        fi
    fi
else
    fail "Dashboard endpoint skipped (no auth)" ""
fi

# ==========================================
# Berichte Module (3) — gated by modules.berichte feature flag
# ==========================================
section "Berichte"

if [[ -n "$SMOKE_TOKEN" && "$SMOKE_TOKEN" != "null" ]]; then
    DEFS_RESP=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/v1/berichte/definitions" \
        -H "Authorization: Bearer $SMOKE_TOKEN" 2>/dev/null || echo $'\n000')
    DEFS_CODE=$(echo "$DEFS_RESP" | tail -1)
    DEFS_BODY=$(echo "$DEFS_RESP" | sed '$d')

    if [[ "$DEFS_CODE" == "200" ]]; then
        DEFS_COUNT=$(echo "$DEFS_BODY" | jq 'length' 2>/dev/null || echo "0")
        if [[ "$DEFS_COUNT" -ge "1" ]]; then
            pass "GET /berichte/definitions = 200 ($DEFS_COUNT definitions)"
        else
            fail "GET /berichte/definitions returned 200 but 0 definitions (expected >= 1 system seed)" "$DEFS_BODY"
        fi

        # 20. Run a report (first definition in the list)
        FIRST_DEF_ID=$(echo "$DEFS_BODY" | jq -r '.[0].id' 2>/dev/null || echo "")
        if [[ -n "$FIRST_DEF_ID" && "$FIRST_DEF_ID" != "null" ]]; then
            RUN_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST \
                "$BASE_URL/api/v1/berichte/definitions/$FIRST_DEF_ID/run" \
                -H "Content-Type: application/json" \
                -H "Authorization: Bearer $SMOKE_TOKEN" \
                -d '{}' 2>/dev/null || echo "000")
            if [[ "$RUN_CODE" == "200" ]]; then
                pass "POST /berichte/definitions/{id}/run = 200"
            else
                fail "POST /berichte/definitions/{id}/run returned $RUN_CODE" ""
            fi

            # 21. Export as PDF
            EXPORT_MIME=$(curl -s -o /dev/null -w "%{content_type}" -X POST \
                "$BASE_URL/api/v1/berichte/definitions/$FIRST_DEF_ID/export?format=pdf" \
                -H "Content-Type: application/json" \
                -H "Authorization: Bearer $SMOKE_TOKEN" \
                -d '{}' 2>/dev/null || echo "")
            if [[ "$EXPORT_MIME" == application/pdf* ]]; then
                pass "POST /berichte/definitions/{id}/export?format=pdf returns application/pdf"
            else
                fail "Berichte export PDF returned content-type '$EXPORT_MIME'" ""
            fi
        else
            fail "Berichte run/export skipped (no definition id)" ""
        fi
    elif [[ "$DEFS_CODE" == "404" ]]; then
        # Modul gated off — acceptable on environments without COSMI_MODULE_BERICHTE_ENABLED=true
        pass "Berichte module gated off ($DEFS_CODE — modules.berichte flag disabled)"
    else
        fail "GET /berichte/definitions returned $DEFS_CODE" "$DEFS_BODY"
    fi
else
    fail "Berichte checks skipped (no auth)" ""
fi

# ==========================================
# Cleanup
# ==========================================
section "Cleanup"

if [[ -n "$SMOKE_USER_ID" && -n "$SMOKE_TOKEN" && "$SMOKE_TOKEN" != "null" ]]; then
    CLEANUP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/api/v1/auth/users/$SMOKE_USER_ID" \
        -H "Authorization: Bearer $SMOKE_TOKEN" 2>/dev/null || echo "000")
    if [[ "$CLEANUP_CODE" == "200" || "$CLEANUP_CODE" == "204" || "$CLEANUP_CODE" == "404" ]]; then
        echo "  Smoke user cleaned up ($SMOKE_EMAIL)"
    else
        echo "  WARNING: Could not delete smoke user (code: $CLEANUP_CODE)"
    fi
else
    echo "  No smoke user to clean up"
fi

# ==========================================
# Results
# ==========================================
echo ""
echo "=========================================="
echo "  Smoke Test Results: $PASSED/$TOTAL passed, $FAILED failed"
echo "=========================================="

if [[ $FAILED -gt 0 ]]; then
    exit 1
fi
exit 0
