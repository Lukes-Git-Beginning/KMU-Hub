/**
 * i18n batch for RBAC R-3 batch 5 (close-out: berichte/formulare/
 * automatisierung + standard-module mini catalogues kommunikation/kalender/
 * zeiterfassung/infrastructure): new capability subjects + one action for the
 * freshly curated catalogue entries, plus one module-scoped subject override
 * (infrastructure_service — plain `service` is fuhrpark's „Wartung"; see
 * capabilityLabel module-first lookup in lib/rbac-format.ts). Subject names
 * mirror Luke's permission seeds where they exist: berichte `reports` PLURAL
 * (000080), formulare `schemas`/`submissions` (000129), automatisierung
 * `automations` (000129, BE resource without module prefix).
 *
 * Inserts new keys at their alphabetical position relative to the existing
 * order (same as i18n-rbac-r3b4.mjs — no global re-sort). Keys already
 * present are skipped, so listing safety duplicates is harmless.
 *
 * Run: node scripts/i18n-rbac-r3b5.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))
const MESSAGES_DIR = join(here, '..', 'src', 'renderer', 'src', 'i18n', 'messages')
const LOCALES = ['de', 'en', 'fr', 'it']

/** @type {Record<string, [string, string, string, string]>} key → [de, en, fr, it] */
const ADD = {
  // ── Subjects ──────────────────────────────────────────────────────────────
  'rbac.subject.automations': ['Automatisierungen', 'Automations', 'Automatisations', 'Automazioni'],
  'rbac.subject.backup': ['Backups', 'Backups', 'Sauvegardes', 'Backup'],
  'rbac.subject.booking_page': ['Buchungsseiten', 'Booking pages', 'Pages de réservation', 'Pagine di prenotazione'],
  'rbac.subject.channel': ['Kanäle', 'Channels', 'Canaux', 'Canali'],
  'rbac.subject.datev': ['DATEV', 'DATEV', 'DATEV', 'DATEV'],
  'rbac.subject.executions': ['Ausführungsprotokoll', 'Execution log', "Journal d'exécution", 'Registro esecuzioni'],
  'rbac.subject.infrastructure_service': ['Dienste', 'Services', 'Services', 'Servizi'],
  'rbac.subject.logs': ['System-Logs', 'System logs', 'Journaux système', 'Log di sistema'],
  'rbac.subject.reports': ['Berichte', 'Reports', 'Rapports', 'Report'],
  'rbac.subject.routing': ['Routing-Regeln', 'Routing rules', 'Règles de routage', 'Regole di instradamento'],
  'rbac.subject.schedule': ['Zeitpläne', 'Schedules', 'Planifications', 'Pianificazioni'],
  'rbac.subject.schemas': ['Formulare', 'Forms', 'Formulaires', 'Moduli'],
  'rbac.subject.security': ['Sicherheitseinstellungen', 'Security settings', 'Paramètres de sécurité', 'Impostazioni di sicurezza'],
  'rbac.subject.submissions': ['Eingänge', 'Submissions', 'Soumissions', 'Invii'],
  'rbac.subject.team': ['Teamübersicht', 'Team overview', "Vue d'équipe", 'Panoramica team'],
  'rbac.subject.team_inbox': ['Team-Postfächer', 'Team inboxes', "Boîtes d'équipe", 'Caselle di team'],
  'rbac.subject.updates': ['Updates', 'Updates', 'Mises à jour', 'Aggiornamenti'],
  'rbac.subject.webhook': ['Webhooks', 'Webhooks', 'Webhooks', 'Webhook'],
  'rbac.subject.week': ['Wochenabschlüsse', 'Week sign-offs', 'Clôtures hebdomadaires', 'Chiusure settimanali'],
  // ── Actions ───────────────────────────────────────────────────────────────
  'rbac.action.toggle': ['Aktivieren & Deaktivieren', 'Enable & disable', 'Activer & désactiver', 'Attivare e disattivare'],
}

for (const [li, locale] of LOCALES.entries()) {
  const file = join(MESSAGES_DIR, `${locale}.json`)
  const data = JSON.parse(readFileSync(file, 'utf8'))

  // Insert new keys at their alphabetical position relative to existing order.
  const pending = Object.keys(ADD)
    .filter((k) => !(k in data))
    .sort()
  const out = {}
  const existing = Object.keys(data)
  let pi = 0
  for (const key of existing) {
    while (pi < pending.length && pending[pi] < key) {
      out[pending[pi]] = ADD[pending[pi]][li]
      pi++
    }
    out[key] = data[key]
  }
  while (pi < pending.length) {
    out[pending[pi]] = ADD[pending[pi]][li]
    pi++
  }

  writeFileSync(file, JSON.stringify(out, null, 2) + '\n', 'utf8')
  console.log(`${locale}: +${pending.length} keys`)
}
console.log('done')
