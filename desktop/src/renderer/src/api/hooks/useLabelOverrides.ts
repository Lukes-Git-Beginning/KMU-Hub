/**
 * useLabelOverrides — React Query hooks für den Label-Override-Editor (v1.2).
 *
 * GET /api/v1/customization/labels?locale=de[&base=1]
 * PUT /api/v1/customization/labels
 *
 * Nach jedem PUT: invalidateQueries → Re-Fetch → applyLabelOverlay()
 * → sofortiger Live-Preview in der gesamten App ohne Reload.
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { API_BASE_URL } from '@/lib/constants'
import type {
  LabelOverridesResponse,
  UpdateLabelOverridesInput,
} from '@/api/customization-types'
import { applyLabelOverlay } from '@/i18n/useLabelOverlay'

const API = API_BASE_URL

// ── Query keys ──────────────────────────────────────────────────────────────

export const labelOverridesKey = (locale: string, base = false) =>
  ['customization', 'labels', locale, base ? 'base' : 'resolved'] as const

// ── Fetch helpers ────────────────────────────────────────────────────────────

async function fetchLabels(locale: string, base = false): Promise<LabelOverridesResponse> {
  const url = new URL(`${API}/api/v1/customization/labels`)
  url.searchParams.set('locale', locale)
  if (base) url.searchParams.set('base', '1')
  const res = await fetch(url.toString())
  if (!res.ok) throw new Error(`fetchLabels failed: ${res.status}`)
  return res.json() as Promise<LabelOverridesResponse>
}

async function putLabels(input: UpdateLabelOverridesInput): Promise<LabelOverridesResponse> {
  const res = await fetch(`${API}/api/v1/customization/labels`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  if (!res.ok) throw new Error(`putLabels failed: ${res.status}`)
  return res.json() as Promise<LabelOverridesResponse>
}

async function resetLabel(locale: string, key: string): Promise<LabelOverridesResponse> {
  // Reset = PUT mit value: '' für den einzelnen Key (Handler löscht bei leerem Wert)
  return putLabels({ locale, layer: 'tenant', overrides: { [key]: '' } })
}

async function resetAllLabels(locale: string): Promise<void> {
  // Reset-All: PUT mit allen LABEL_WHITELIST-Keys auf '' (Handler entfernt alle)
  // Wir schicken einen leeren overrides-Record mit einem Sentinel-Flag.
  // Einfachste Lösung: alle Keys auf '' setzen → Handler löscht alle Tenant-Einträge.
  // Wir fragen zuerst die aktuellen Labels ab um die Keys zu kennen.
  const current = await fetchLabels(locale)
  const overrides: Record<string, string> = {}
  for (const key of Object.keys(current.labels)) {
    overrides[key] = ''
  }
  await putLabels({ locale, layer: 'tenant', overrides })
}

// ── Hooks ────────────────────────────────────────────────────────────────────

/**
 * Lädt die aufgelösten Label-Overrides (default/vendor/tenant) für eine Locale.
 * Wird vom Editor genutzt, um die Liste aller Whitelist-Keys mit Provenance darzustellen.
 */
export function useLabelOverrides(locale: string) {
  return useQuery({
    queryKey: labelOverridesKey(locale),
    queryFn: () => fetchLabels(locale),
  })
}

/**
 * Lädt den reinen Code-Default (base=true) für eine Locale.
 * Wird vom Editor genutzt, um den „Cosmi-Standard"-Wert anzuzeigen
 * (analog ?base=1 aus R-6 für die Baseline-Ansicht).
 */
export function useLabelDefaults(locale: string) {
  return useQuery({
    queryKey: labelOverridesKey(locale, true),
    queryFn: () => fetchLabels(locale, true),
  })
}

/**
 * Speichert einen oder mehrere Label-Overrides.
 * Nach Erfolg: invalidate → Re-Fetch → applyLabelOverlay() (Live-Preview).
 */
export function useSetLabelOverrides() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: putLabels,
    onSuccess: (data) => {
      const locale = data.locale
      // Invalidate both resolved + base queries für diese Locale
      void qc.invalidateQueries({ queryKey: labelOverridesKey(locale) })
      void qc.invalidateQueries({ queryKey: labelOverridesKey(locale, true) })
      // Live-Preview: Overlay sofort einspiegeln (kein Reload nötig)
      applyLabelOverlay(locale, data.labels)
    },
  })
}

/**
 * Setzt einen einzelnen Key zurück (entfernt den Tenant-Override).
 */
export function useResetLabelOverride() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ locale, key }: { locale: string; key: string }) => resetLabel(locale, key),
    onSuccess: (data) => {
      const locale = data.locale
      void qc.invalidateQueries({ queryKey: labelOverridesKey(locale) })
      void qc.invalidateQueries({ queryKey: labelOverridesKey(locale, true) })
      applyLabelOverlay(locale, data.labels)
    },
  })
}

/**
 * Setzt alle Tenant-Label-Overrides für eine Locale zurück.
 */
export function useResetAllLabelOverrides() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (locale: string) => resetAllLabels(locale),
    onSuccess: async (_data, locale) => {
      void qc.invalidateQueries({ queryKey: labelOverridesKey(locale) })
      void qc.invalidateQueries({ queryKey: labelOverridesKey(locale, true) })
      // Re-Fetch und neu einspiegeln
      try {
        const fresh = await fetchLabels(locale)
        applyLabelOverlay(locale, fresh.labels)
      } catch {
        // silent
      }
    },
  })
}
