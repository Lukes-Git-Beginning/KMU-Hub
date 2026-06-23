/**
 * Normalizes protobuf-Timestamp wire values to ISO-8601 strings.
 *
 * Services that serialize via `response.JSON` (encoding/json over protobuf
 * structs — e.g. the work service) emit `google.protobuf.Timestamp` as
 * `{ "seconds": <int>, "nanos": <int> }` instead of an RFC3339 string. The
 * frontend reads these fields as ISO strings (`parseISO`, `new Date(...)`,
 * `.split('T')`), so an unconverted object breaks date rendering, sorting and
 * relative-time formatting (the board/gantt fall back to skeleton/NaN).
 *
 * The `{seconds, nanos}` shape is unambiguous (a plain object whose only keys
 * are `seconds` and optionally `nanos`, both numeric), so a recursive walk can
 * safely rewrite every timestamp at any depth without colliding with real data.
 * Mock-mode responses already use ISO strings and pass through untouched.
 */

interface ProtoTimestamp {
  seconds: number | string
  nanos?: number | string
}

function isProtoTimestamp(v: unknown): v is ProtoTimestamp {
  if (v === null || typeof v !== 'object' || Array.isArray(v)) return false
  const keys = Object.keys(v as Record<string, unknown>)
  if (keys.length === 0 || keys.some((k) => k !== 'seconds' && k !== 'nanos')) return false
  const sec = (v as ProtoTimestamp).seconds
  return typeof sec === 'number' || (typeof sec === 'string' && sec !== '' && !Number.isNaN(Number(sec)))
}

function protoTsToISO(ts: ProtoTimestamp): string {
  const seconds = Number(ts.seconds)
  const nanos = ts.nanos != null ? Number(ts.nanos) : 0
  return new Date(seconds * 1000 + Math.round(nanos / 1e6)).toISOString()
}

/**
 * Deep-clone `value`, rewriting any `{seconds, nanos}` proto-Timestamp to an ISO
 * string. Arrays and nested objects are walked; primitives pass through. Returns
 * the input unchanged when there is nothing to convert.
 */
export function normalizeWireTimestamps<T>(value: T): T {
  if (Array.isArray(value)) {
    return value.map((v) => normalizeWireTimestamps(v)) as unknown as T
  }
  if (value !== null && typeof value === 'object') {
    if (isProtoTimestamp(value)) {
      return protoTsToISO(value as ProtoTimestamp) as unknown as T
    }
    const out: Record<string, unknown> = {}
    for (const [k, v] of Object.entries(value as Record<string, unknown>)) {
      out[k] = normalizeWireTimestamps(v)
    }
    return out as T
  }
  return value
}
