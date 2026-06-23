/**
 * Tolerant casing helpers for the OpenAPI ↔ gRPC-gateway drift (X-3).
 *
 * The Go gateway serialises responses via `encoding/json` over protobuf structs,
 * whose json tags are **snake_case** (`first_name`, `company_name`, `created_at`).
 * The TypeScript types generated from the OpenAPI spec, however, are **camelCase**
 * (`firstName`, `companyName`, `createdAt`). Reading an OpenAPI-typed value against
 * the real backend therefore yields `undefined` for every multi-word field.
 *
 * Most hand-written domain clients (`*-client.ts`) already declare snake_case
 * interfaces and match the wire — they do NOT need this. Only modules typed off
 * `components['schemas'][...]` / `paths` are affected. Until the spec is
 * consolidated on snake_case (a coordinated backend+regen change), those modules
 * read fields through these helpers so a single value works against BOTH the
 * camelCase mock and the snake_case backend.
 *
 * Deliberately field-level, not a blanket recursive key transform: the codebase
 * is mixed-casing (thousands of existing snake_case reads), and free-form maps
 * (custom_fields, settings) carry arbitrary keys that must never be rewritten.
 */

/** Convert a camelCase identifier to its snake_case equivalent. */
export function snakeOf(camel: string): string {
  return camel.replace(/[A-Z]/g, (m) => '_' + m.toLowerCase())
}

/**
 * Read a single field tolerating both camelCase and its snake_case form.
 *
 * Pass the camelCase name; the snake_case fallback is derived automatically
 * (override `snake` for irregular cases). A present camelCase value wins even
 * when falsy (`0`, `''`, `false`) — only `undefined` falls through to snake.
 */
export function dual<T = unknown>(
  obj: unknown,
  camel: string,
  snake: string = snakeOf(camel),
): T | undefined {
  if (obj == null || typeof obj !== 'object') return undefined
  const rec = obj as Record<string, unknown>
  const v = rec[camel]
  return (v !== undefined ? v : rec[snake]) as T | undefined
}

/**
 * Read several fields at once, returning an object keyed by the camelCase names.
 * Convenience over repeated `dual()` calls when adapting a flat DTO.
 */
export function dualAll<K extends string>(
  obj: unknown,
  camelKeys: readonly K[],
): { [P in K]: unknown } {
  const out = {} as { [P in K]: unknown }
  for (const k of camelKeys) out[k] = dual(obj, k)
  return out
}
