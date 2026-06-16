/**
 * Password-reset API client (unauthenticated).
 *
 * forgot-password / reset-password are public endpoints (security: [] in
 * openapi.yaml). They are not yet in the generated api/types.ts, so — like
 * security-client.ts — this is a hand-written fetch helper until the types are
 * regenerated. MSW intercepts these fetches in demo mode.
 */
import { API_BASE_URL } from '@/lib/constants'

/**
 * Request a password-reset link. Enumeration-safe: the backend always returns
 * 200 regardless of whether the email is registered, so this resolves on any
 * 2xx and only throws on a network / 5xx failure.
 */
export async function requestPasswordReset(email: string): Promise<void> {
  const res = await fetch(`${API_BASE_URL}/api/v1/auth/forgot-password`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email }),
  })
  if (!res.ok) {
    throw new Error(`Request failed: ${res.status}`)
  }
}

/**
 * Reset the password using the single-use token from the email link.
 * Throws with the backend error message on 400 (invalid/expired token, weak
 * password) or 404.
 */
export async function resetPassword(token: string, newPassword: string): Promise<void> {
  const res = await fetch(`${API_BASE_URL}/api/v1/auth/reset-password`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ token, new_password: newPassword }),
  })
  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as { error?: string }
    throw new Error(body.error || `Request failed: ${res.status}`)
  }
}
