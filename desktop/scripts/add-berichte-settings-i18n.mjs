// Untracked helper: insert the berichte module-settings structural i18n keys
// (rendered without defaultValue → must exist in all 4 languages) into the
// flat message JSONs. Internal labels stay defaultValue (migrated in B-5).
import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'

const DIR = resolve('src/renderer/src/i18n/messages')

const KEYS = {
  de: {
    'moduleSettings.entries.berichte': 'Berichte',
    'berichte.settings.title': 'Berichte-Einstellungen',
    'berichte.settings.subtitle': 'Standard-Format, Zeitraum und Versand-Regeln',
    'berichte.settings.personal.title': 'Persönliche Voreinstellungen',
    'berichte.settings.personal.desc': 'Standard-Format und Zeitraum für neue Berichte',
    'berichte.settings.tenant.title': 'Versand & Formate',
    'berichte.settings.tenant.desc':
      'Erlaubte Export-Formate und zulässige E-Mail-Domains für geplante Berichte',
  },
  en: {
    'moduleSettings.entries.berichte': 'Reports',
    'berichte.settings.title': 'Reports settings',
    'berichte.settings.subtitle': 'Default format, period and delivery rules',
    'berichte.settings.personal.title': 'Personal defaults',
    'berichte.settings.personal.desc': 'Default format and period for new reports',
    'berichte.settings.tenant.title': 'Delivery & formats',
    'berichte.settings.tenant.desc':
      'Allowed export formats and permitted email domains for scheduled reports',
  },
  fr: {
    'moduleSettings.entries.berichte': 'Rapports',
    'berichte.settings.title': 'Paramètres des rapports',
    'berichte.settings.subtitle': "Format, période et règles d'envoi par défaut",
    'berichte.settings.personal.title': 'Préférences personnelles',
    'berichte.settings.personal.desc': 'Format et période par défaut pour les nouveaux rapports',
    'berichte.settings.tenant.title': 'Envoi et formats',
    'berichte.settings.tenant.desc':
      "Formats d'export autorisés et domaines e-mail admis pour les rapports planifiés",
  },
  it: {
    'moduleSettings.entries.berichte': 'Report',
    'berichte.settings.title': 'Impostazioni report',
    'berichte.settings.subtitle': 'Formato, periodo e regole di invio predefiniti',
    'berichte.settings.personal.title': 'Preferenze personali',
    'berichte.settings.personal.desc': 'Formato e periodo predefiniti per i nuovi report',
    'berichte.settings.tenant.title': 'Invio e formati',
    'berichte.settings.tenant.desc':
      'Formati di esportazione consentiti e domini e-mail ammessi per i report pianificati',
  },
}

function insertAfter(text, anchorKey, line) {
  const re = new RegExp(`("${anchorKey.replace(/\./g, '\\.')}":[^\\n]*\\n)`)
  if (!re.test(text)) return null
  return text.replace(re, `$1${line}`)
}

for (const lang of ['de', 'en', 'fr', 'it']) {
  const file = resolve(DIR, `${lang}.json`)
  let text = readFileSync(file, 'utf8')
  const dict = KEYS[lang]
  let added = 0
  for (const [key, value] of Object.entries(dict)) {
    if (text.includes(`"${key}":`)) continue // already present
    const line = `  "${key}": ${JSON.stringify(value)},\n`
    const anchor = key.startsWith('moduleSettings.')
      ? 'moduleSettings.entries.automatisierung'
      : 'berichte.chart.vorjahr'
    let next = insertAfter(text, anchor, line)
    if (next === null) {
      // fallback anchors
      next =
        insertAfter(text, 'moduleSettings.entries.helpdesk', line) ||
        insertAfter(text, 'berichte.chart.umsatzverlauf', line)
    }
    if (next) {
      text = next
      added++
    } else {
      console.error(`[${lang}] no anchor for ${key}`)
    }
  }
  // validate JSON
  JSON.parse(text)
  writeFileSync(file, text)
  console.log(`[${lang}] +${added} keys, valid JSON`)
}
