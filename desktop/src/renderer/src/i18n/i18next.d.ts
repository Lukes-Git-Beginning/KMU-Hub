import 'i18next'

// NOTE: We deliberately do NOT type `resources` against `typeof de.json`.
// Typing it against the full ~8.5k-key JSON makes tsc's overload resolver crash
// with an internal "Debug Failure. No error for last overload signature" on
// whole-program checks — a known, still-open compiler bug triggered by the huge
// resource type × the multi-overload `t()` (microsoft/TypeScript#63195). Even
// `keyof typeof de` still materialises that type and crashes, and an index-
// signature (`Record<string, string>`) pushes i18next onto a return path that
// types plain `t('key')` as `() => string`. So we omit `resources` entirely:
// i18next's default `TFunction` then types `t()` as `(key: string) => string`.
// Trade-off: no compile-time key-name validation — the project's i18n gate is
// Playwright screenshot-QA (raw-key detection), not tsc. Keeps tsc usable again.
declare module 'i18next' {
  interface CustomTypeOptions {
    defaultNS: 'translation'
    returnNull: false
  }
}
