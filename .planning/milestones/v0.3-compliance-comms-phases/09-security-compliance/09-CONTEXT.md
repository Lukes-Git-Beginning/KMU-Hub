# Phase 9: Security & Compliance - Context

**Gathered:** 2026-02-11
**Status:** Ready for planning

<domain>
## Phase Boundary

The Hub meets enterprise security requirements and DSGVO compliance obligations, with multi-language support for the Swiss market. This phase adds 2FA, audit logging, DSGVO data export/deletion, session management, an encrypted secret vault, and i18n (DE/FR/IT/EN) to the existing application. No new user-facing modules -- this hardens and internationalizes what exists.

**Design branch assets:** `InfrastrukturPage.tsx` (security tab) and `VaultSettings.tsx` on `design/brainstorm` -- integrate via cherry-pick like Phase 7.

</domain>

<decisions>
## Implementation Decisions

### 2FA Setup & Enforcement
- Guided wizard flow: multi-step modal (show QR code -> scan with authenticator app -> enter verification code -> show recovery codes)
- 8 single-use recovery codes generated at setup, with option to download as text file or copy to clipboard
- Users can regenerate recovery codes anytime (invalidates old set)
- Per-role enforcement with configurable grace period (e.g., admin can mandate 2FA for admin/manager roles, give users X days to set up)
- Account recovery when authenticator + codes lost: admin can reset 2FA for a user, action is logged in audit log with mandatory reason field
- Login flow: after password verification, prompt for TOTP code (or recovery code) if 2FA is enabled

### Audit Log Scope & Presentation
- Comprehensive logging: security events + admin actions + data access events (who viewed sensitive data, bulk exports)
- Security events: login/logout, 2FA changes, password changes, role changes, session terminations, data exports/deletions
- Admin events: user invite/deactivate, settings changes, vault access, enforcement policy changes
- Data access events: sensitive data views, bulk data exports
- Presentation: searchable/filterable table with columns: timestamp, user, action, target, IP, result
- Filters: date range, action type, user, result (success/failure)
- Export: both CSV (for Steuerberater/auditors) and JSON (for programmatic analysis)
- Retention: admin-configurable between 1 and 10 years, default 3 years
- Tamper-evident: append-only log with integrity verification

### DSGVO User Workflows
- Data export format: ZIP archive with structured JSON files per module (contacts, messages, tasks, etc.) -- metadata only for uploaded files (file list with names/sizes but files not included)
- Export trigger: user submits request from settings, admin approves/denies, user gets download link when ready
- Right-to-erasure flow: two-step with preview -- admin clicks delete -> sees preview of what will be anonymized per module -> confirms with password -> executes
- Anonymization approach: configurable per module by admin -- choose between anonymize (replace with "Geloeschter Benutzer #NNN", keep structural data) vs. full content deletion per module
- Examples: keep anonymized chat messages (for thread context) but fully delete CRM activities

### Session Management
- Admin can view all active sessions with device, IP, location info
- Admin can terminate individual sessions or all sessions for a user
- User can view and terminate their own sessions from settings

### Secret Vault
- Encrypted storage for sensitive config (API keys, SMTP passwords, integration credentials)
- Integrate Darien's VaultSettings.tsx from design branch

### i18n Framework
- Languages: DE, FR, IT, EN
- Language picker in user profile/settings (persists per user)
- Login page defaults to browser-detected language
- Auto-detection: browser language on first use, user can override
- Fallback chain: user choice -> browser language -> DE
- Translation scope: UI labels + system content (email templates, notifications, system messages) + default values (status names, priority labels, role names)
- Full ICU message format: pluralization, gender, ordinals
- Locale-aware formatting: dates (31.01.2026 vs 01/31/2026), numbers (1.000,00 vs 1,000.00), currencies
- Org-level default not needed -- browser detection + per-user override is sufficient

### Claude's Discretion
- Specific TOTP library choice (backend and frontend)
- Audit log database schema and indexing strategy
- ICU/i18n library choice (react-intl, i18next, etc.)
- Exact tamper-evidence implementation (hash chaining, etc.)
- Session storage and cleanup strategy
- Vault encryption algorithm and key management approach
- Password policy specifics (min length, complexity rules)

</decisions>

<specifics>
## Specific Ideas

- Design branch has `InfrastrukturPage.tsx` with a security tab and `VaultSettings.tsx` -- cherry-pick and wire to backend like Phase 7/8 integration pattern
- Backend currently has 3 roles (admin, manager, member), frontend has 5 (adds hr, it_support) -- role expansion may be needed for this phase's enforcement features
- DSGVO anonymization placeholder text should be "Geloeschter Benutzer" (German) with a numeric ID, not user's real data

</specifics>

<deferred>
## Deferred Ideas

None -- discussion stayed within phase scope

</deferred>

---

*Phase: 09-security-compliance*
*Context gathered: 2026-02-11*
