// One-off: add Module-Settings overlay i18n keys to all 4 locales (append-only).
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const __dirname = dirname(fileURLToPath(import.meta.url))
const MSG_DIR = join(__dirname, '..', 'src', 'renderer', 'src', 'i18n', 'messages')

const additions = {
  de: {
    'moduleSettings.title': 'Einstellungen',
    'moduleSettings.groups.module': 'Module',
    'moduleSettings.groups.cosmi': 'Cosmi (Allgemein)',
    'moduleSettings.activeBadge': 'Aktiv',
    'moduleSettings.empty': 'Keine Einstellungen verfügbar.',
    'moduleSettings.entries.finance': 'Finanzen',
    'moduleSettings.entries.calendar': 'Kalender',
    'moduleSettings.entries.mail': 'E-Mail',
    'moduleSettings.entries.team': 'Team',
    'moduleSettings.entries.company': 'Firma',
    'moduleSettings.entries.billing': 'Abrechnung',
    'moduleSettings.entries.integrations': 'Integrationen',
    'moduleSettings.entries.it': 'IT & System',
    'common.close': 'Schließen',
  },
  en: {
    'moduleSettings.title': 'Settings',
    'moduleSettings.groups.module': 'Modules',
    'moduleSettings.groups.cosmi': 'Cosmi (General)',
    'moduleSettings.activeBadge': 'Active',
    'moduleSettings.empty': 'No settings available.',
    'moduleSettings.entries.finance': 'Finance',
    'moduleSettings.entries.calendar': 'Calendar',
    'moduleSettings.entries.mail': 'Email',
    'moduleSettings.entries.team': 'Team',
    'moduleSettings.entries.company': 'Company',
    'moduleSettings.entries.billing': 'Billing',
    'moduleSettings.entries.integrations': 'Integrations',
    'moduleSettings.entries.it': 'IT & System',
    'common.close': 'Close',
  },
  fr: {
    'moduleSettings.title': 'Paramètres',
    'moduleSettings.groups.module': 'Modules',
    'moduleSettings.groups.cosmi': 'Cosmi (Général)',
    'moduleSettings.activeBadge': 'Actif',
    'moduleSettings.empty': 'Aucun paramètre disponible.',
    'moduleSettings.entries.finance': 'Finances',
    'moduleSettings.entries.calendar': 'Calendrier',
    'moduleSettings.entries.mail': 'E-mail',
    'moduleSettings.entries.team': 'Équipe',
    'moduleSettings.entries.company': 'Entreprise',
    'moduleSettings.entries.billing': 'Facturation',
    'moduleSettings.entries.integrations': 'Intégrations',
    'moduleSettings.entries.it': 'IT et système',
    'common.close': 'Fermer',
  },
  it: {
    'moduleSettings.title': 'Impostazioni',
    'moduleSettings.groups.module': 'Moduli',
    'moduleSettings.groups.cosmi': 'Cosmi (Generale)',
    'moduleSettings.activeBadge': 'Attivo',
    'moduleSettings.empty': 'Nessuna impostazione disponibile.',
    'moduleSettings.entries.finance': 'Finanze',
    'moduleSettings.entries.calendar': 'Calendario',
    'moduleSettings.entries.mail': 'E-mail',
    'moduleSettings.entries.team': 'Team',
    'moduleSettings.entries.company': 'Azienda',
    'moduleSettings.entries.billing': 'Fatturazione',
    'moduleSettings.entries.integrations': 'Integrazioni',
    'moduleSettings.entries.it': 'IT e sistema',
    'common.close': 'Chiudi',
  },
}

for (const [locale, keys] of Object.entries(additions)) {
  const file = join(MSG_DIR, `${locale}.json`)
  const data = JSON.parse(readFileSync(file, 'utf8'))
  let added = 0
  for (const [k, v] of Object.entries(keys)) {
    if (!(k in data)) added++
    if (!(k in data)) data[k] = v // do not overwrite existing (e.g. common.close)
  }
  writeFileSync(file, JSON.stringify(data, null, 2) + '\n', 'utf8')
  console.log(`${locale}.json: +${added} keys (total ${Object.keys(data).length})`)
}
