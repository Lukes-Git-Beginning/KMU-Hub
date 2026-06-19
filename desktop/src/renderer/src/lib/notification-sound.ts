/**
 * Subtle notification chime via the Web Audio API — no audio asset required.
 *
 * A short two-note pluck (C6 → G6) kept quiet enough to be a hint, not an
 * interruption. Safe to call anywhere: it no-ops when Web Audio is unavailable
 * or blocked (e.g. before the first user gesture) and never throws.
 */

let audioCtx: AudioContext | null = null

function getCtx(): AudioContext | null {
  try {
    const Ctor =
      window.AudioContext ||
      (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext
    if (!Ctor) return null
    if (!audioCtx) audioCtx = new Ctor()
    return audioCtx
  } catch {
    return null
  }
}

/** Play the notification chime. No-op if audio is unavailable/blocked. */
export function playNotificationSound(): void {
  const ctx = getCtx()
  if (!ctx) return
  try {
    if (ctx.state === 'suspended') void ctx.resume()
    const now = ctx.currentTime
    const notes = [
      { freq: 1046.5, start: 0, dur: 0.13 }, // C6
      { freq: 1567.98, start: 0.08, dur: 0.18 }, // G6
    ]
    for (const note of notes) {
      const osc = ctx.createOscillator()
      const gain = ctx.createGain()
      osc.type = 'sine'
      osc.frequency.value = note.freq
      const t0 = now + note.start
      gain.gain.setValueAtTime(0, t0)
      gain.gain.linearRampToValueAtTime(0.07, t0 + 0.012)
      gain.gain.exponentialRampToValueAtTime(0.0001, t0 + note.dur)
      osc.connect(gain)
      gain.connect(ctx.destination)
      osc.start(t0)
      osc.stop(t0 + note.dur + 0.02)
    }
  } catch {
    // Audio not available — ignore silently.
  }
}

/** Whether the user asked to reduce motion (we suppress the chime too). */
export function prefersReducedMotion(): boolean {
  return (
    typeof window !== 'undefined' &&
    typeof window.matchMedia === 'function' &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches
  )
}
