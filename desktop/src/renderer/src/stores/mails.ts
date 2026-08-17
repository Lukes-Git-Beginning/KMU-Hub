/**
 * Compose-draft handover to the Electron pop-out window.
 *
 * This used to be a Zustand store, which could not work: the pop-out is a second
 * BrowserWindow, i.e. its own renderer process with its own JS heap, so it never
 * saw a value held in module state. Anyone who clicked "open as window" with text
 * already typed lost it.
 *
 * localStorage is shared across windows of the same origin, so it is the channel
 * that actually reaches the other window -- the same approach the employee wizard
 * (cosmi-employee-wizard-draft) and the customization editor already use.
 *
 * The entry is consumed on read and carries a TTL, so a window that never opened
 * cannot resurrect a stale draft into an unrelated compose session later.
 */

export type ComposeMode = 'compose' | 'reply' | 'reply-all' | 'forward'

export interface ComposeDraft {
  to: string[]
  cc: string[]
  bcc: string[]
  subject: string
  body: string
  mode: ComposeMode
  /** Original message ID for reply/forward threading */
  replyToMessageId?: string
  /** Account ID to send from */
  accountId?: string
}

const HANDOVER_KEY = 'cosmi:mails:compose-handover'

/** How long a stashed draft stays valid. The window opens immediately; anything
 *  older than this is leftover from a window that never came up. */
const HANDOVER_TTL_MS = 30_000

interface StashEnvelope {
  at: number
  draft: ComposeDraft
}

/** Hand the current draft to a compose window that is about to open. */
export function stashComposeDraft(draft: ComposeDraft): void {
  try {
    const envelope: StashEnvelope = { at: Date.now(), draft }
    localStorage.setItem(HANDOVER_KEY, JSON.stringify(envelope))
  } catch {
    // Quota or storage disabled -- the window then opens with an empty form,
    // which is the old behaviour and no worse.
  }
}

/**
 * Read and consume a stashed draft. Returns null when there is none, when it is
 * older than the TTL, or when the payload does not look like a draft.
 */
export function takeComposeDraft(): ComposeDraft | null {
  try {
    const raw = localStorage.getItem(HANDOVER_KEY)
    if (!raw) return null
    localStorage.removeItem(HANDOVER_KEY)

    const parsed = JSON.parse(raw) as Partial<StashEnvelope>
    if (typeof parsed?.at !== 'number' || Date.now() - parsed.at > HANDOVER_TTL_MS) {
      return null
    }

    // localStorage is user-writable: a hand-edited entry must read as "no draft",
    // never as a draft whose array fields are undefined -- the compose form calls
    // .length on them straight away.
    const draft = parsed.draft
    if (
      !draft ||
      !Array.isArray(draft.to) ||
      !Array.isArray(draft.cc) ||
      !Array.isArray(draft.bcc) ||
      typeof draft.subject !== 'string' ||
      typeof draft.body !== 'string'
    ) {
      return null
    }
    return draft
  } catch {
    return null
  }
}

/** Drop a stashed draft without consuming it into a form. */
export function clearComposeDraft(): void {
  try {
    localStorage.removeItem(HANDOVER_KEY)
  } catch {
    // Non-critical.
  }
}
