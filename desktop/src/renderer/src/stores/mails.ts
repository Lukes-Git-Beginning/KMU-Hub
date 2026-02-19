/**
 * Ephemeral mail UI state (compose draft only).
 *
 * Email data (messages, folders, accounts) now comes from TanStack Query hooks
 * in api/hooks/useEmail.ts. This store only manages cross-component compose
 * state needed for the Electron pop-out compose window.
 */
import { create } from 'zustand'

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

interface MailsState {
  composeDraft: ComposeDraft | null
  setComposeDraft: (draft: ComposeDraft | null) => void
}

export const useMailsStore = create<MailsState>()((set) => ({
  composeDraft: null,
  setComposeDraft: (draft) => set({ composeDraft: draft }),
}))
