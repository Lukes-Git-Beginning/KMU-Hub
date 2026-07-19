/**
 * i18n batch for RBAC R-3 batch 4 (industry modules schichten/fuhrpark/
 * vermietung/rapporte/dialer): new capability subjects + actions for the
 * freshly curated catalogue entries and one module-scoped subject override
 * (rapporte_report — see capabilityLabel in lib/rbac-format.ts). Dialer
 * subjects are PLURAL (campaigns/calls/outcomes/agent) mirroring Luke's
 * 000068 permission seed.
 *
 * Inserts new keys at their alphabetical position relative to the existing
 * order (same as i18n-rbac-r3b3.mjs — no global re-sort). Keys already
 * present are skipped, so listing safety duplicates is harmless.
 *
 * Run: node scripts/i18n-rbac-r3b4.mjs
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
  'rbac.subject.agent': ['Agenten', 'Agents', 'Agents', 'Agenti'],
  'rbac.subject.assignment': ['Zuweisungen', 'Assignments', 'Affectations', 'Assegnazioni'],
  'rbac.subject.calls': ['Anrufe', 'Calls', 'Appels', 'Chiamate'],
  'rbac.subject.campaigns': ['Kampagnen', 'Campaigns', 'Campagnes', 'Campagne'],
  'rbac.subject.damage': ['Schadensmeldungen', 'Damage reports', 'Déclarations de dommages', 'Segnalazioni danni'],
  'rbac.subject.fuel': ['Tankprotokoll', 'Fuel log', 'Carnet de carburant', 'Registro carburante'],
  'rbac.subject.gps': ['GPS-Tracking', 'GPS tracking', 'Suivi GPS', 'Tracciamento GPS'],
  'rbac.subject.inspection': ['Zustandsprotokolle', 'Condition reports', 'États des lieux', 'Verbali di consegna'],
  'rbac.subject.measurement': ['Aufmaße', 'Measurements', 'Métrés', 'Misurazioni'],
  'rbac.subject.object': ['Mietobjekte', 'Rental objects', 'Objets de location', 'Oggetti a noleggio'],
  'rbac.subject.outcomes': ['Anruf-Ergebnisse', 'Call outcomes', "Résultats d'appel", 'Esiti chiamate'],
  'rbac.subject.rapporte_report': ['Tagesberichte', 'Daily reports', 'Rapports journaliers', 'Rapporti giornalieri'],
  'rbac.subject.rental': ['Reservierungen', 'Reservations', 'Réservations', 'Prenotazioni'],
  'rbac.subject.report': ['Berichte', 'Reports', 'Rapports', 'Rapporti'],
  'rbac.subject.service': ['Wartung', 'Maintenance', 'Entretien', 'Manutenzione'],
  'rbac.subject.shift': ['Schichten', 'Shifts', 'Postes', 'Turni'],
  'rbac.subject.swap': ['Tauschanfragen', 'Swap requests', "Demandes d'échange", 'Richieste di scambio'],
  'rbac.subject.trip': ['Fahrten', 'Trips', 'Trajets', 'Viaggi'],
  'rbac.subject.vehicle': ['Fahrzeuge', 'Vehicles', 'Véhicules', 'Veicoli'],
  // ── Actions ───────────────────────────────────────────────────────────────
  'rbac.action.handover': ['Ausgabe & Rücknahme', 'Hand over & return', 'Remise & retour', 'Consegna e ritiro'],
  'rbac.action.publish': ['Veröffentlichen', 'Publish', 'Publier', 'Pubblicare'],
  'rbac.action.write': ['Erfassen & bearbeiten', 'Record & edit', 'Saisir et modifier', 'Registrare e modificare'],
  // ── rapporte approve UI (built alongside the gating — endpoint existed, UI did not) ──
  'rapporte.detail.approve': ['Genehmigen', 'Approve', 'Approuver', 'Approvare'],
  'rapporte.detail.approveError': ['Genehmigen fehlgeschlagen', 'Approval failed', "Échec de l'approbation", 'Approvazione non riuscita'],
  'rapporte.detail.approveSuccess': ['Rapport genehmigt', 'Report approved', 'Rapport approuvé', 'Rapporto approvato'],
  'rapporte.detail.reject': ['Ablehnen', 'Reject', 'Refuser', 'Respingere'],
  'rapporte.detail.rejectConfirm': ['Ablehnung senden', 'Send rejection', 'Envoyer le refus', 'Invia rifiuto'],
  'rapporte.detail.rejectError': ['Ablehnen fehlgeschlagen', 'Rejection failed', 'Échec du refus', 'Rifiuto non riuscito'],
  'rapporte.detail.rejectNoteLabel': ['Grund der Ablehnung', 'Reason for rejection', 'Motif du refus', 'Motivo del rifiuto'],
  'rapporte.detail.rejectNotePlaceholder': ['Für den Ersteller sichtbar …', 'Visible to the author …', "Visible pour l'auteur …", "Visibile all'autore …"],
  'rapporte.detail.rejectSuccess': ['Rapport abgelehnt', 'Report rejected', 'Rapport refusé', 'Rapporto respinto'],
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
