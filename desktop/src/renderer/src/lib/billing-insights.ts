/**
 * billing-insights.ts — Pure-Funktion fuer Optimierungs-Vorschläge.
 * Keine Seiteneffekte, kein React. Deterministisch für gleiche Inputs.
 */
import {
  MODULE_PRICES,
  type InsightSettings,
  type ModuleUsageStats,
  type Recommendation,
} from './pricing'

/** Modul-Display-Namen (Deutsch) für Recommendations. */
const MODULE_DISPLAY_NAMES: Record<string, string> = {
  crm: 'CRM',
  tasks: 'Aufgaben',
  finance: 'Buchhaltung',
  calendar: 'Kalender',
  documents: 'Dokumente',
  chat: 'Chat',
  meetings: 'Meetings',
  mail: 'Mail',
  dialer: 'Dialer',
  team: 'Team',
  zeiterfassung: 'Zeiterfassung',
  schichten: 'Schichten',
  projects: 'Projekte',
  inventar: 'Inventar',
  einkauf: 'Einkauf',
  helpdesk: 'Helpdesk',
  fuhrpark: 'Fuhrpark',
  vertraege: 'Verträge',
  produktion: 'Produktion',
  berichte: 'Berichte',
  formulare: 'Formulare',
  wiki: 'Wiki',
  rapporte: 'Rapporte',
  vermietung: 'Vermietung',
}

/**
 * Berechnet Optimierungs-Vorschläge aus Modul-Nutzungsdaten.
 *
 * Deterministisch: gleiche Inputs → gleiche Outputs + Hashes.
 * Gibt max 10 Recommendations zurück (UI zeigt 3, Rest für späteres "mehr"-Feature).
 *
 * Edge-Cases (werden geprüft, bevor Hauptlogik läuft):
 *   1. recommendationsEnabled === false → sofort leeres Array zurück (kein Compute).
 *   2. assignedUserCount === 0 für ein Modul → wird übersprungen (Empfehlung wäre sinnlos).
 *   3. utilizationThreshold >= 100 → wird auf 99 geclampt (100% würde alle Module melden).
 *   4. utilizationThreshold <= 0 → wird auf 0 geclampt (niemand wird empfohlen).
 *   5. observationDays < 1 → wird auf 1 geclampt.
 *   6. minInactivityDays < 0 → wird auf 0 geclampt.
 */
export function calculateRecommendations(
  usage: ModuleUsageStats[],
  settings: InsightSettings,
  dismissedHashes: Set<string>,
): Recommendation[] {
  // Edge-Case 1: Feature deaktiviert — kein Compute, kein Performance-Hit.
  if (!settings.recommendationsEnabled) return []

  // Edge-Cases 3–6: Settings clampen (defensive, falls ungültige Werte aus localStorage).
  const threshold = Math.min(Math.max(settings.utilizationThreshold, 0), 99)
  const observationDays = Math.max(settings.observationDays, 1)
  const minInactivityDays = Math.max(settings.minInactivityDays, 0)

  const now = Date.now()
  const minInactivityMs = minInactivityDays * 24 * 60 * 60 * 1000

  const recommendations: Recommendation[] = []

  for (const stat of usage) {
    // Edge-Case 2: Modul hat keine zugeteilten User → überspringen (Empfehlung wäre absurd).
    if (stat.assignedUserCount === 0) continue
    // Nutze geclampten Threshold (Edge-Cases 3+4).
    if (stat.utilizationPercent >= threshold) continue

    // Inaktive User filtern: lastActiveAt älter als minInactivityDays (geclampt) oder null.
    const qualifyingInactive = stat.inactiveUsers.filter((u) => {
      if (u.lastActiveAt === null) return true
      const lastActive = new Date(u.lastActiveAt).getTime()
      return now - lastActive >= minInactivityMs
    })

    if (qualifyingInactive.length === 0) continue

    const inactiveUserIds = qualifyingInactive.map((u) => u.userId)
    const hash = `${stat.moduleId}:${[...inactiveUserIds].sort().join(',')}`

    if (dismissedHashes.has(hash)) continue

    const pricePerSeat = MODULE_PRICES[stat.moduleId] ?? 0
    const monthlySavings = pricePerSeat * qualifyingInactive.length
    const yearlySavings = monthlySavings * 12

    recommendations.push({
      hash,
      moduleId: stat.moduleId,
      moduleName: MODULE_DISPLAY_NAMES[stat.moduleId] ?? stat.moduleId,
      inactiveUsers: qualifyingInactive.map((u) => ({ userId: u.userId, userName: u.userName })),
      monthlySavings,
      yearlySavings,
      observationDays,
    })
  }

  // Sortiere nach Ersparnis absteigend, slice auf max 10
  return recommendations.sort((a, b) => b.monthlySavings - a.monthlySavings).slice(0, 10)
}
