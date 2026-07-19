/**
 * i18n batch for RBAC R-3 batch 1 (enforcement sweep): the shared gate
 * building blocks (rbac.gate.* — header chip, no-access page, exception
 * tooltips, amount masking) plus the new wiki category subject.
 *
 * Inserts new keys alphabetically into the existing flat JSON without
 * reordering existing keys.
 *
 * Run: node scripts/i18n-rbac-r3.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))
const MESSAGES_DIR = join(here, '..', 'src', 'renderer', 'src', 'i18n', 'messages')
const LOCALES = ['de', 'en', 'fr', 'it']

const REMOVE = []
const REMOVE_PREFIXES = []

/** @type {Record<string, [string, string, string, string]>} key → [de, en, fr, it] */
const ADD = {
  // ── shared gate building blocks ─────────────────────────────────────────────
  'rbac.gate.readOnlyChip': ['Nur Ansicht', 'View only', 'Lecture seule', 'Sola visualizzazione'],
  'rbac.gate.limitedChip': ['Eingeschränkt', 'Limited', 'Restreint', 'Limitato'],
  'rbac.gate.readOnlyHint': [
    'Deine Rolle kann dieses Modul ansehen, aber nichts ändern.',
    'Your role can view this module but not change anything.',
    'Votre rôle peut consulter ce module mais ne rien modifier.',
    'Il tuo ruolo può visualizzare questo modulo ma non modificarlo.',
  ],
  'rbac.gate.limitedHint': [
    'Deine Rolle erlaubt hier nur einzelne Aktionen.',
    'Your role allows only some actions here.',
    'Votre rôle n’autorise ici que certaines actions.',
    'Il tuo ruolo consente qui solo alcune azioni.',
  ],
  'rbac.gate.noAccessTitle': [
    'Kein Zugriff auf {module}',
    'No access to {module}',
    'Pas d’accès à {module}',
    'Nessun accesso a {module}',
  ],
  'rbac.gate.noAccessBody': [
    'Deine Rolle hat keinen Zugriff auf dieses Modul. Wende dich an deine Verwaltung, wenn du Zugriff brauchst.',
    'Your role has no access to this module. Contact your administration if you need access.',
    'Votre rôle n’a pas accès à ce module. Adressez-vous à votre administration si vous avez besoin d’un accès.',
    'Il tuo ruolo non ha accesso a questo modulo. Rivolgiti alla tua amministrazione se ti serve l’accesso.',
  ],
  'rbac.gate.noAccessCta': ['Zum Dashboard', 'Go to dashboard', 'Vers le tableau de bord', 'Al dashboard'],
  'rbac.gate.noPermission': [
    'Dir fehlt das Recht dafür — wende dich an deine Verwaltung.',
    'You lack the permission for this — contact your administration.',
    'Ce droit vous manque — adressez-vous à votre administration.',
    'Ti manca il permesso necessario — rivolgiti alla tua amministrazione.',
  ],
  'rbac.gate.importDisabled': [
    'Import erfordert zusätzliche Rechte — wende dich an deine Verwaltung.',
    'Importing requires additional permissions — contact your administration.',
    'L’import requiert des droits supplémentaires — adressez-vous à votre administration.',
    'L’importazione richiede permessi aggiuntivi — rivolgiti alla tua amministrazione.',
  ],
  'rbac.gate.downloadDisabled': [
    'Download ist für deine Rolle deaktiviert.',
    'Downloads are disabled for your role.',
    'Le téléchargement est désactivé pour votre rôle.',
    'Il download è disattivato per il tuo ruolo.',
  ],
  'rbac.gate.sendDisabled': [
    'Versenden ist für deine Rolle nicht freigegeben.',
    'Sending is not enabled for your role.',
    'L’envoi n’est pas autorisé pour votre rôle.',
    'L’invio non è abilitato per il tuo ruolo.',
  ],
  'rbac.gate.amountsHidden': [
    'Beträge sind für deine Rolle verborgen.',
    'Amounts are hidden for your role.',
    'Les montants sont masqués pour votre rôle.',
    'Gli importi sono nascosti per il tuo ruolo.',
  ],

  // ── new capability building block (wiki:category:manage) ────────────────────
  'rbac.subject.category': ['Kategorien', 'Categories', 'Catégories', 'Categorie'],
}

// ── apply ───────────────────────────────────────────────────────────────────
for (const [li, locale] of LOCALES.entries()) {
  const file = join(MESSAGES_DIR, `${locale}.json`)
  const data = JSON.parse(readFileSync(file, 'utf8'))

  let removed = 0
  for (const key of Object.keys(data)) {
    if (REMOVE.includes(key) || REMOVE_PREFIXES.some((p) => key.startsWith(p))) {
      delete data[key]
      removed++
    }
  }

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
  console.log(`${locale}: +${pending.length} keys, -${removed} removed`)
}
console.log('done')
