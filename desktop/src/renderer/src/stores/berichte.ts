// SUPERSEDED by api/hooks/useBerichte.ts — entfernt in Sprint 2 cleanup.
//
// The original Zustand store seeded the Berichte module with mock KPIs,
// chart data, saved/scheduled reports, and DATEV rows while the backend
// was not yet implemented. With WP-8/WP-9 of Sprint 1 the BerichtePage
// consumes the live /api/v1/berichte endpoints via TanStack Query hooks,
// so this store no longer exports anything. The file is retained in the
// tree purely as a rollback anchor until the next cleanup pass.

export {}
