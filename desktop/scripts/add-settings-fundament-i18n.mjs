// One-off: add Settings-Fundament i18n keys to all 4 locales.
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const __dirname = dirname(fileURLToPath(import.meta.url))
const MSG_DIR = join(__dirname, '..', 'src', 'renderer', 'src', 'i18n', 'messages')

const additions = {
  de: {
    'settings.scope.personal': 'Persönlich',
    'settings.scope.tenant': 'Für alle',
    'settings.scope.lockedHint':
      'Diese Einstellungen gelten für das ganze Team und können nur von der Modulleitung geändert werden. Du siehst sie schreibgeschützt.',
    'settings.calendar.section.personalTitle': 'Persönliche Ansicht',
    'settings.calendar.section.personalDesc':
      'Gilt nur für dich — Standardansicht, Wochenstart und Erinnerung.',
    'settings.calendar.section.tenantTitle': 'Arbeitszeiten & Feiertage',
    'settings.calendar.section.tenantDesc':
      'Gilt für das ganze Team — Arbeitszeiten und Feiertagsregion.',
    'team.member.moduleLead.title': 'Erweiterte Moduleinstellungen',
    'team.member.moduleLead.hint':
      'Aktiviere Module, für die dieser Mitarbeiter die teamweiten Einstellungen verwalten darf (Modulleitung).',
  },
  en: {
    'settings.scope.personal': 'Personal',
    'settings.scope.tenant': 'Org-wide',
    'settings.scope.lockedHint':
      'These settings apply to the whole team and can only be changed by the module lead. They are shown read-only.',
    'settings.calendar.section.personalTitle': 'Personal view',
    'settings.calendar.section.personalDesc':
      'Applies only to you — default view, week start and reminder.',
    'settings.calendar.section.tenantTitle': 'Working hours & holidays',
    'settings.calendar.section.tenantDesc':
      'Applies to the whole team — working hours and holiday region.',
    'team.member.moduleLead.title': 'Advanced module settings',
    'team.member.moduleLead.hint':
      'Enable modules for which this employee may manage the team-wide settings (module lead).',
  },
  fr: {
    'settings.scope.personal': 'Personnel',
    'settings.scope.tenant': 'Pour tous',
    'settings.scope.lockedHint':
      "Ces paramètres s'appliquent à toute l'équipe et ne peuvent être modifiés que par le responsable du module. Ils sont affichés en lecture seule.",
    'settings.calendar.section.personalTitle': 'Affichage personnel',
    'settings.calendar.section.personalDesc':
      "Ne s'applique qu'à vous — affichage par défaut, début de semaine et rappel.",
    'settings.calendar.section.tenantTitle': 'Heures de travail et jours fériés',
    'settings.calendar.section.tenantDesc':
      "S'applique à toute l'équipe — heures de travail et région des jours fériés.",
    'team.member.moduleLead.title': 'Paramètres de module avancés',
    'team.member.moduleLead.hint':
      "Activez les modules pour lesquels cet employé peut gérer les paramètres de toute l'équipe (responsable du module).",
  },
  it: {
    'settings.scope.personal': 'Personale',
    'settings.scope.tenant': 'Per tutti',
    'settings.scope.lockedHint':
      'Queste impostazioni valgono per tutto il team e possono essere modificate solo dal responsabile del modulo. Sono mostrate in sola lettura.',
    'settings.calendar.section.personalTitle': 'Vista personale',
    'settings.calendar.section.personalDesc':
      'Vale solo per te — vista predefinita, inizio settimana e promemoria.',
    'settings.calendar.section.tenantTitle': 'Orario di lavoro e festività',
    'settings.calendar.section.tenantDesc':
      'Vale per tutto il team — orario di lavoro e regione delle festività.',
    'team.member.moduleLead.title': 'Impostazioni avanzate del modulo',
    'team.member.moduleLead.hint':
      'Attiva i moduli per cui questo dipendente può gestire le impostazioni di tutto il team (responsabile del modulo).',
  },
}

for (const [locale, keys] of Object.entries(additions)) {
  const file = join(MSG_DIR, `${locale}.json`)
  const data = JSON.parse(readFileSync(file, 'utf8'))
  let added = 0
  for (const [k, v] of Object.entries(keys)) {
    if (!(k in data)) added++
    data[k] = v
  }
  // Preserve original key order; new keys append at the end (i18next is order-agnostic).
  writeFileSync(file, JSON.stringify(data, null, 2) + '\n', 'utf8')
  console.log(`${locale}.json: +${added} keys (total ${Object.keys(data).length})`)
}
