---
phase: 09-security-compliance
plan: 07
subsystem: ui
tags: [i18n, react-intl, icu, translations, locale]
requires:
  - phase: 05-desktop-app-shell
    provides: "App.tsx, Zustand store patterns, React app shell"
provides:
  - "I18nProvider with react-intl IntlProvider"
  - "4-language translation files (DE/FR/IT/EN) with 206 keys each"
  - "Zustand locale store with localStorage persistence"
  - "useLocale hook with browser detection and fallback chain"
  - "Locale-specific format configs (date, number, currency, time)"
affects: [09-08, 09-09, future-phases-ui]
tech-stack:
  added: [react-intl, "@formatjs/cli"]
  patterns: [i18n-provider-pattern, locale-store-pattern, icu-message-format]
key-files:
  created:
    - desktop/src/renderer/src/i18n/index.tsx
    - desktop/src/renderer/src/i18n/formats.ts
    - desktop/src/renderer/src/i18n/messages/de.json
    - desktop/src/renderer/src/i18n/messages/en.json
    - desktop/src/renderer/src/i18n/messages/fr.json
    - desktop/src/renderer/src/i18n/messages/it.json
    - desktop/src/renderer/src/stores/locale.ts
    - desktop/src/renderer/src/hooks/useLocale.ts
  modified:
    - desktop/src/renderer/src/App.tsx
    - desktop/package.json
key-decisions:
  - "react-intl over i18next for native ICU message format support (no plugin needed)"
  - "Static imports for all 4 locale bundles (small JSON files, no async loading complexity)"
  - "Zustand with persist middleware for locale store (consistent with existing store patterns)"
  - "Fallback chain: user explicit choice -> browser navigator.language -> DE default"
  - "MISSING_TRANSLATION errors suppressed in dev mode only (incremental translation workflow)"
  - "206 translation keys covering common UI, nav, auth, 2FA, audit, GDPR, sessions, vault, password, settings, roles, relative time"
  - "14 ICU plural patterns for locale-correct number/count formatting"
  - "CHF and EUR currency formats per locale for Swiss market"
duration: 6min
completed: 2026-02-11
---

# Phase 9 Plan 7: i18n Framework Setup Summary

Complete i18n infrastructure with react-intl, 4-language translations (DE/FR/IT/EN), ICU message format with pluralization, Zustand locale store with browser detection and persistence.

## What Was Built

### I18n Provider (`i18n/index.tsx`)
- `I18nProvider` component wrapping react-intl's `IntlProvider`
- Statically imports all 4 locale message bundles
- Uses `useLocale()` hook for effective locale resolution
- Custom error handler suppresses MISSING_TRANSLATION in dev mode
- Integrated into `App.tsx` wrapping the entire app above TooltipProvider

### Locale Store (`stores/locale.ts`)
- Zustand store with `persist` middleware (localStorage key: `kmuhub-locale`)
- `SupportedLocale` type union: `'de' | 'en' | 'fr' | 'it'`
- `SUPPORTED_LOCALES` const array and `DEFAULT_LOCALE = 'de'`
- `setLocale()` and `resetLocale()` (back to auto-detect) actions

### useLocale Hook (`hooks/useLocale.ts`)
- Provides effective locale: user choice -> browser detection -> DE fallback
- Browser detection via `navigator.language.split('-')[0]`
- `AVAILABLE_LOCALES` array with code + display name for locale pickers
- `isAutoDetected` boolean flag for UI display
- `resetToAuto()` convenience method

### Format Configurations (`i18n/formats.ts`)
- Per-locale date formats: short (DD.MM.YYYY), medium, long with weekday
- Per-locale number formats: decimal (2 fraction digits), CHF, EUR currencies
- Per-locale time formats: short (HH:mm), medium (HH:mm:ss), respecting 12h/24h conventions
- `getFormats(locale)` function returns react-intl `CustomFormats` object

### Translation Files (206 keys each)
All 4 locale files have identical key sets covering:
- **Common UI** (31 keys): save, cancel, delete, search, edit, create, etc.
- **Navigation** (9 keys): dashboard, CRM, chat, work, calendar, meetings, etc.
- **Auth** (7 keys): login, logout, email, password, error messages
- **2FA** (22 keys): enable/disable, QR scan, recovery codes, enforcement, grace period, wizard steps
- **Audit** (19 keys): table headers, filters, export, integrity verification, action messages
- **Sessions** (15 keys): device types, terminate, browser/OS/location info
- **GDPR** (19 keys): data export workflow, erasure preview, module actions, article references
- **Vault** (14 keys): add/edit/delete secrets, show/hide values, categories
- **Password** (15 keys): policy settings, strength indicators, change workflow
- **Settings** (8 keys): language, security, privacy, general sections
- **Roles** (5 keys): admin, manager, member, HR, IT support
- **Time** (5 keys): relative time expressions with pluralization

14 ICU plural patterns ensure correct grammar across all 4 languages (e.g., German one/other, French one/other).

## Task Commits

| Task | Name | Commit | Key Files |
|------|------|--------|-----------|
| 1 | Install react-intl and create i18n infrastructure | `8876432` | locale.ts, formats.ts, useLocale.ts, i18n/index.tsx, package.json |
| 2 | Translation files and App.tsx integration | `116a6f5` | de/en/fr/it.json, App.tsx |

## Deviations from Plan

None -- plan executed exactly as written.

## Verification Results

- All 4 JSON files have exactly 206 keys (verified programmatically)
- All locale files have identical key sets (zero missing/extra keys)
- 14 ICU pluralization patterns confirmed in de.json
- No TypeScript errors from any i18n files (pre-existing errors in other files unchanged)
- I18nProvider wraps the app in App.tsx above TooltipProvider
- react-intl@8.1.3 and @formatjs/cli@6.12.2 installed in package.json

## Next Phase Readiness

**09-08 and 09-09** can now use the i18n infrastructure:
- Import `useIntl` from `react-intl` and call `intl.formatMessage({ id: 'key' })`
- Or use `<FormattedMessage id="key" values={{ count: 5 }} />`
- Locale switching works at runtime via `useLocale().setLocale('fr')`
- Components with hardcoded German strings can be migrated incrementally

**No blockers.** The i18n infrastructure is fully operational and ready for consumption.

## Self-Check: PASSED
