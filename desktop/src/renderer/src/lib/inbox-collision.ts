/**
 * Mock-first collision detection for the Posteingang (Front/Missive pattern):
 * "X is currently working on this conversation".
 *
 * There is no real-time presence-per-conversation backend yet
 * (see backend-gaps.md). We derive a deterministic, stable indicator from the
 * conversation id so the demo shows the affordance on a subset of threads.
 * Adapter-ready: replace `getCollision` with a live presence subscription
 * (e.g. WebSocket "viewers" event) keyed by conversation id.
 */

const COLLEAGUES = ['Anna Müller', 'Peter Schmidt', 'Sarah Meier', 'Lisa Braun']

function hashId(id: string): number {
  let h = 0
  for (let i = 0; i < id.length; i++) {
    h = (h * 31 + id.charCodeAt(i)) >>> 0
  }
  return h
}

/** Returns the colleague currently viewing this conversation, or null. */
export function getCollision(conversationId: string): { name: string } | null {
  const h = hashId(conversationId)
  // ~1 in 3 conversations show a concurrent viewer (deterministic)
  if (h % 3 !== 0) return null
  return { name: COLLEAGUES[h % COLLEAGUES.length] }
}
