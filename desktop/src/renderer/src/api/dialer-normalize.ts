/**
 * Wire-shape normalizers for the Dialer protojson endpoints.
 *
 * The gateway encodes dialer responses with protojson (`UseProtoNames`,
 * `UseEnumNumbers`) and **`EmitUnpopulated: false`** — so proto3 default
 * values are OMITTED from the JSON entirely:
 *   - int32 == 0          → field absent  (e.g. calls_today, duration_seconds)
 *   - bool == false       → field absent  (e.g. is_positive)
 *   - empty repeated      → field absent  (e.g. agents, recent_calls, calls)
 *   - message all-default → field absent  (e.g. totals when every counter is 0)
 *   - enum == 0           → field absent  (status)
 *
 * The MSW mocks return fully-populated objects, so these gaps only surface
 * against the real backend (classic mock-masked bug). A quiet/empty dialer
 * would otherwise crash the supervisor page (`data.totals` undefined →
 * `totals.active_agents` throws) or render `NaN`/`undefined` in the KPIs.
 *
 * These helpers fill the omitted defaults; they NEVER overwrite present values,
 * so running mock responses through them is a no-op.
 */
import type {
  SupervisorOverview,
  SupervisorAgent,
  RecentCallEntry,
  ContactCallEntry,
} from './dialer-client'

/** Neutral slate used when an outcome has no colour (omitted empty string). */
const FALLBACK_OUTCOME_COLOR = '#94a3b8'

function num(v: unknown): number {
  const n = typeof v === 'string' ? Number(v) : (v as number)
  return Number.isFinite(n) ? n : 0
}

function str(v: unknown): string {
  return typeof v === 'string' ? v : ''
}

function normalizeAgent(a: Partial<SupervisorAgent>): SupervisorAgent {
  return {
    user_id: str(a.user_id),
    first_name: str(a.first_name),
    last_name: str(a.last_name),
    status: num(a.status),
    campaign_id: a.campaign_id ?? null,
    since: str(a.since),
    calls_today: num(a.calls_today),
    avg_duration_seconds: num(a.avg_duration_seconds),
    active_campaign_name: a.active_campaign_name ?? null,
  }
}

function normalizeRecentCall(c: Partial<RecentCallEntry>): RecentCallEntry {
  return {
    id: str(c.id),
    contact_name: str(c.contact_name),
    contact_company: c.contact_company ?? null,
    outcome_label: c.outcome_label ?? null,
    outcome_color: str(c.outcome_color) || FALLBACK_OUTCOME_COLOR,
    is_positive: c.is_positive === true,
    duration_seconds: num(c.duration_seconds),
    agent_name: str(c.agent_name),
    campaign_name: c.campaign_name ?? null,
    created_at: str(c.created_at),
  }
}

function normalizeContactCall(c: Partial<ContactCallEntry>): ContactCallEntry {
  return {
    id: str(c.id),
    outcome_label: c.outcome_label ?? null,
    outcome_color: str(c.outcome_color) || FALLBACK_OUTCOME_COLOR,
    is_positive: c.is_positive === true,
    duration_seconds: num(c.duration_seconds),
    notes: c.notes ?? null,
    agent_name: str(c.agent_name),
    created_at: str(c.created_at),
  }
}

/** Fills protojson-omitted defaults so the supervisor page never sees `undefined`. */
export function normalizeSupervisorOverview(
  raw: Partial<SupervisorOverview> | null | undefined,
): SupervisorOverview {
  const t: Partial<SupervisorOverview['totals']> = raw?.totals ?? {}
  return {
    agents: Array.isArray(raw?.agents) ? raw!.agents.map(normalizeAgent) : [],
    recent_calls: Array.isArray(raw?.recent_calls)
      ? raw!.recent_calls.map(normalizeRecentCall)
      : [],
    totals: {
      active_agents: num(t.active_agents),
      on_call: num(t.on_call),
      calls_today: num(t.calls_today),
      appointments_today: num(t.appointments_today),
    },
  }
}

/** Fills protojson-omitted defaults for the contact call-history feed. */
export function normalizeContactCalls(
  raw: { calls?: Partial<ContactCallEntry>[] } | null | undefined,
): { calls: ContactCallEntry[] } {
  return {
    calls: Array.isArray(raw?.calls) ? raw!.calls.map(normalizeContactCall) : [],
  }
}
