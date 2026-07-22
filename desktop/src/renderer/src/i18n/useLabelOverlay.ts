/**
 * useLabelOverlay — Runtime-Label-Overlay-Hook (v1.2).
 *
 * Lädt die tenant/vendor-Label-Overrides für alle aktiven Locales
 * und spielt sie via i18next.addResourceBundle() in-memory ein.
 * Danach reagiert jeder t()-Call in der App auf die gemergten Werte
 * ohne Rebuild oder Reload.
 *
 * Andock-Punkt:
 *   i18next init (i18n/index.tsx) → I18nProvider → useEffect nach Locale-Sync
 *
 * Merge-Strategie:
 *   addResourceBundle(locale, 'translation', overrides, deep=true, overwrite=true)
 *   Setzt nur die Keys aus der Whitelist, lässt alle anderen Keys unberührt.
 *   Idempotent: erneuter Aufruf (z.B. nach Sprach-Wechsel oder Save) überschreibt
 *   sauber.
 *
 * Live-Preview nach Speichern:
 *   Nach dem PUT /api/v1/customization/labels refetcht React Query die Labels
 *   (invalidateQueries), der useEffect feuert neu und ruft applyOverlay().
 *   i18next.emit('languageChanged') triggert ein Re-Render aller useTranslation()-
 *   Konsumenten, ohne die Sprache tatsächlich zu wechseln.
 *
 * Namespace-Kompatibilität:
 *   nsSeparator: false + keySeparator: false → alle Keys liegen im Namespace
 *   'translation'. addResourceBundle mergt in denselben Namespace.
 *
 * Fallback:
 *   Bei Fehler (MSW nicht bereit, Auth nicht gesetzt) bleibt i18n unverändert.
 *   Kein Crash, kein leerer UI-State.
 */
import { useEffect, useRef } from 'react'
import { i18n } from '@/i18n/i18n'
import { API_BASE_URL } from '@/lib/constants'
import type { LabelOverridesResponse, ResolvedLabelMap } from '@/api/customization-types'

const API = API_BASE_URL

/**
 * Snapshot der Code-Default-Werte je Locale, gezogen BEVOR der erste
 * addResourceBundle-Merge das Bundle in-place überschreibt. Ohne diesen
 * Snapshot liefert i18n.getResourceBundle() nach dem Merge den Override-Wert
 * statt des echten Cosmi-Standards — der Editor könnte den Original-Begriff
 * nicht mehr anzeigen (v1.2-Fix). Nur die Whitelist-Keys (aus der API-Antwort),
 * damit kein mocks-Import in den i18n-Layer nötig ist.
 */
const defaultsCache: Record<string, Record<string, string>> = {}

/**
 * Friert die aktuellen Bundle-Werte der gegebenen Keys als Default ein — nur
 * beim ersten Mal pro Locale (idempotent), also bevor ein Merge stattfand.
 */
function captureDefaults(locale: string, keys: string[]): void {
  if (defaultsCache[locale]) return
  const bundle = i18n.getResourceBundle(locale, 'translation') as Record<string, string> | undefined
  const snap: Record<string, string> = {}
  for (const key of keys) snap[key] = bundle?.[key] ?? key
  defaultsCache[locale] = snap
}

/**
 * Liefert den echten Code-Default eines Label-Keys. Für gebootstrappte Locales
 * aus dem Snapshot (kollisionsfrei mit dem Overlay); für nie-gemergte Locales
 * fällt es auf das unveränderte Bundle zurück.
 */
export function getLabelDefault(locale: string, key: string): string {
  const cached = defaultsCache[locale]?.[key]
  if (cached !== undefined) return cached
  const bundle = i18n.getResourceBundle(locale, 'translation') as Record<string, string> | undefined
  return bundle?.[key] ?? key
}

/**
 * Wendet eine aufgelöste Label-Map als i18next-ResourceBundle-Override an.
 * Schreibt nur Keys mit tatsächlichem Override-Wert (provenance != 'default').
 * Ruft i18next.emit('languageChanged') für ein globales Re-Render auf.
 */
export function applyLabelOverlay(locale: string, labels: ResolvedLabelMap): void {
  const overrides: Record<string, string> = {}

  for (const [key, entry] of Object.entries(labels)) {
    if (entry.provenance !== 'default' && entry.value !== '') {
      overrides[key] = entry.value
    }
  }

  // addResourceBundle(locale, namespace, resources, deep, overwrite)
  // deep=true mergt statt zu ersetzen; overwrite=true lässt Override gewinnen.
  i18n.addResourceBundle(locale, 'translation', overrides, true, true)

  // Erzwingt Re-Render aller useTranslation()-Konsumenten für die aktive Locale.
  // Wir feuern nur für die aktive Locale, damit keine unnötigen Renders passieren.
  if (i18n.language === locale) {
    // changeLanguage auf dieselbe Sprache triggert das Event ohne echten Wechsel.
    // Der Trick ist dokumentiert: i18next emits 'languageChanged' intern.
    void i18n.changeLanguage(locale)
  }
}

/**
 * Editor-Only: entfernt Draft-Overrides wieder aus dem globalen i18n-Bundle.
 *
 * Der Editor schreibt Draft-Labels via applyLabelOverlay() ins GLOBALE Bundle
 * für die Live-Vorschau (R-1 in IST-EDITOR: kontaminiert den Live-Hintergrund,
 * solange der Editor offen ist). Beim Schließen ohne Commit müssen genau die
 * angefassten Draft-Keys auf ihren echten Code-Default zurückgesetzt werden —
 * ein reines Re-Apply der tenant-Labels würde einen draft-only-Key (nicht im
 * tenant-Overlay) NICHT löschen, weil addResourceBundle nur mitgelieferte Keys
 * setzt. Deshalb erst die Scrub-Keys auf ihren Default zurückschreiben, dann
 * das persistente (tenant/vendor) Overlay erneut anwenden.
 */
export function restoreLabelOverlay(
  locale: string,
  scrubKeys: string[],
  resolved: ResolvedLabelMap,
): void {
  if (scrubKeys.length > 0) {
    const resets: Record<string, string> = {}
    for (const key of scrubKeys) resets[key] = getLabelDefault(locale, key)
    i18n.addResourceBundle(locale, 'translation', resets, true, true)
  }
  applyLabelOverlay(locale, resolved)
}

/**
 * Lädt Label-Overrides für eine Locale vom Mock-Endpoint und spielt sie ein.
 * Gibt bei Fehler still auf (kein throw).
 */
async function fetchAndApply(locale: string): Promise<void> {
  try {
    const res = await fetch(`${API}/api/v1/customization/labels?locale=${locale}`)
    if (!res.ok) return
    const data = (await res.json()) as LabelOverridesResponse
    // Defaults einfrieren, BEVOR addResourceBundle das Bundle überschreibt.
    captureDefaults(locale, Object.keys(data.labels))
    applyLabelOverlay(locale, data.labels)
  } catch {
    // Kein MSW, nicht authenticated — still fail
  }
}

/**
 * Hook: Bootstrapped den Label-Overlay beim Mount und bei Locale-Wechsel.
 *
 * Zu verwenden in I18nProvider (nach dem useEffect für i18n.changeLanguage).
 * Ein zweiter Aufruf nach Speichern im Editor erfolgt durch
 * React-Query-invalidateQueries → erneuter Fetch → applyLabelOverlay().
 */
export function useLabelOverlayBootstrap(locale: string): void {
  // Verhindert Doppel-Fetch beim StrictMode-Doppel-Mount (Dev only)
  const appliedRef = useRef<string | null>(null)

  useEffect(() => {
    if (appliedRef.current === locale) return
    appliedRef.current = locale
    void fetchAndApply(locale)
  }, [locale])
}
