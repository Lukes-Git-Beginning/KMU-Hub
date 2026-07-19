/**
 * i18n batch for RBAC R-3 batch 3 (industry modules inventar/einkauf/
 * produktion/vertraege/helpdesk): new capability subjects + actions for the
 * freshly curated catalogue entries, one module-scoped subject override
 * (einkauf_contract — see capabilityLabel in lib/rbac-format.ts) and one gate
 * key for modules whose every tab got filtered away.
 *
 * Inserts new keys at their alphabetical position relative to the existing
 * order (same as i18n-rbac-r3b2.mjs — no global re-sort).
 *
 * Run: node scripts/i18n-rbac-r3b3.mjs
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
  'rbac.subject.attachment': ['Anhänge', 'Attachments', 'Pièces jointes', 'Allegati'],
  'rbac.subject.bom': ['Stücklisten', 'Bills of materials', 'Nomenclatures', 'Distinte base'],
  'rbac.subject.canned': ['Schnellantworten', 'Canned responses', 'Réponses types', 'Risposte rapide'],
  'rbac.subject.catalog': ['Katalog', 'Catalogue', 'Catalogue', 'Catalogo'],
  'rbac.subject.contract': ['Verträge', 'Contracts', 'Contrats', 'Contratti'],
  'rbac.subject.einkauf_contract': ['Rahmenverträge', 'Framework contracts', 'Contrats-cadres', 'Contratti quadro'],
  'rbac.subject.inventur': ['Inventuren', 'Stocktakes', 'Inventaires', 'Inventari'],
  'rbac.subject.item': ['Artikel', 'Items', 'Articles', 'Articoli'],
  'rbac.subject.kb': ['Wissensdatenbank', 'Knowledge base', 'Base de connaissances', 'Base di conoscenza'],
  'rbac.subject.location': ['Lagerorte', 'Storage locations', 'Emplacements de stockage', 'Ubicazioni'],
  'rbac.subject.machine': ['Maschinen', 'Machines', 'Machines', 'Macchine'],
  'rbac.subject.movement': ['Bestandsbewegungen', 'Stock movements', 'Mouvements de stock', 'Movimenti di magazzino'],
  'rbac.subject.order': ['Fertigungsaufträge', 'Production orders', 'Ordres de fabrication', 'Ordini di produzione'],
  'rbac.subject.po': ['Bestellungen', 'Purchase orders', 'Bons de commande', "Ordini d'acquisto"],
  'rbac.subject.quality': ['Qualitätsprüfungen', 'Quality checks', 'Contrôles qualité', 'Controlli qualità'],
  'rbac.subject.rating': ['Lieferantenbewertungen', 'Supplier ratings', 'Évaluations fournisseurs', 'Valutazioni fornitori'],
  'rbac.subject.stats': ['Statistiken', 'Statistics', 'Statistiques', 'Statistiche'],
  'rbac.subject.supplier': ['Lieferanten', 'Suppliers', 'Fournisseurs', 'Fornitori'],
  'rbac.subject.ticket': ['Tickets', 'Tickets', 'Tickets', 'Ticket'],
  'rbac.subject.workstep': ['Arbeitsschritte', 'Work steps', 'Étapes de travail', 'Fasi di lavoro'],
  // ── Actions ───────────────────────────────────────────────────────────────
  'rbac.action.adjust': ['Korrigieren', 'Adjust', 'Corriger', 'Correggere'],
  'rbac.action.call': ['Abruf anlegen', 'Create call-off', 'Créer un appel de commande', 'Creare un richiamo'],
  'rbac.action.cancel': ['Stornieren', 'Cancel', 'Annuler', 'Annullare'],
  'rbac.action.complete': ['Abschließen', 'Complete', 'Terminer', 'Completare'],
  'rbac.action.count': ['Zählen', 'Count', 'Compter', 'Contare'],
  'rbac.action.receive': ['Wareneingang buchen', 'Receive goods', 'Réceptionner', "Registrare l'entrata merci"],
  'rbac.action.reply': ['Antworten', 'Reply', 'Répondre', 'Rispondere'],
  'rbac.action.start': ['Starten', 'Start', 'Démarrer', 'Avviare'],
  'rbac.action.terminate': ['Kündigen', 'Terminate', 'Résilier', 'Disdire'],
  // ── Gate ──────────────────────────────────────────────────────────────────
  'rbac.gate.moduleEmpty': [
    'Für deine Rolle sind hier keine Inhalte sichtbar.',
    'No content is visible for your role here.',
    'Aucun contenu n’est visible ici pour votre rôle.',
    'Nessun contenuto è visibile qui per il tuo ruolo.',
  ],
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
