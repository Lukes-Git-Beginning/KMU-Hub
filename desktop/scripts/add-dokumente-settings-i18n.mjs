import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'

const dir = resolve('src/renderer/src/i18n/messages')

/** key → { de, en, fr, it } — inserted alphabetically into the flat-key files. */
const KEYS = {
  'moduleSettings.entries.dokumente': {
    de: 'Dokumente', en: 'Documents', fr: 'Documents', it: 'Documenti',
  },
  'dokumente.settings.title': {
    de: 'Dokumente', en: 'Documents', fr: 'Documents', it: 'Documenti',
  },
  'dokumente.settings.subtitle': {
    de: 'Ansicht, Sortierung und Richtlinien für das Dokumente-Modul.',
    en: 'View, sorting and policies for the documents module.',
    fr: 'Affichage, tri et règles pour le module Documents.',
    it: 'Visualizzazione, ordinamento e regole per il modulo Documenti.',
  },
  'dokumente.settings.personal.title': {
    de: 'Ansicht & Verhalten', en: 'View & behaviour', fr: 'Affichage et comportement', it: 'Visualizzazione e comportamento',
  },
  'dokumente.settings.personal.desc': {
    de: 'Wie Dateien für dich angezeigt werden.',
    en: 'How files are displayed for you.',
    fr: 'Comment les fichiers s’affichent pour vous.',
    it: 'Come vengono visualizzati i file per te.',
  },
  'dokumente.settings.personal.defaultView': {
    de: 'Standard-Ansicht', en: 'Default view', fr: 'Vue par défaut', it: 'Vista predefinita',
  },
  'dokumente.settings.personal.defaultViewHint': {
    de: 'Gilt für Ordner ohne eigene Ansichtswahl.',
    en: 'Applies to folders without their own view choice.',
    fr: 'S’applique aux dossiers sans choix de vue propre.',
    it: 'Vale per le cartelle senza una scelta di vista propria.',
  },
  'dokumente.settings.personal.viewGrid': {
    de: 'Kacheln', en: 'Grid', fr: 'Grille', it: 'Griglia',
  },
  'dokumente.settings.personal.viewList': {
    de: 'Liste', en: 'List', fr: 'Liste', it: 'Elenco',
  },
  'dokumente.settings.personal.defaultSort': {
    de: 'Standard-Sortierung', en: 'Default sorting', fr: 'Tri par défaut', it: 'Ordinamento predefinito',
  },
  'dokumente.settings.personal.density': {
    de: 'Dichte', en: 'Density', fr: 'Densité', it: 'Densità',
  },
  'dokumente.settings.personal.densityComfortable': {
    de: 'Komfortabel', en: 'Comfortable', fr: 'Confortable', it: 'Comoda',
  },
  'dokumente.settings.personal.densityCompact': {
    de: 'Kompakt', en: 'Compact', fr: 'Compacte', it: 'Compatta',
  },
  'dokumente.settings.storage.title': {
    de: 'Speicher & Aufbewahrung', en: 'Storage & retention', fr: 'Stockage et conservation', it: 'Archiviazione e conservazione',
  },
  'dokumente.settings.storage.desc': {
    de: 'Speicherkontingent und Papierkorb-Regeln.',
    en: 'Storage quota and trash rules.',
    fr: 'Quota de stockage et règles de corbeille.',
    it: 'Quota di archiviazione e regole del cestino.',
  },
  'dokumente.settings.storage.quota': {
    de: 'Speicher-Quota pro Tarif', en: 'Storage quota per plan', fr: 'Quota de stockage par offre', it: 'Quota di archiviazione per piano',
  },
  'dokumente.settings.storage.quotaValue': {
    de: '{gb} GB', en: '{gb} GB', fr: '{gb} Go', it: '{gb} GB',
  },
  'dokumente.settings.storage.quotaHint': {
    de: 'Die Quota ergibt sich aus dem gebuchten Tarif und dient hier nur der Übersicht.',
    en: 'The quota is defined by the booked plan and shown here for reference only.',
    fr: 'Le quota découle de l’offre souscrite et n’est affiché ici qu’à titre indicatif.',
    it: 'La quota deriva dal piano sottoscritto ed è mostrata qui solo a titolo informativo.',
  },
  'dokumente.settings.storage.tier.starter': {
    de: 'Starter', en: 'Starter', fr: 'Starter', it: 'Starter',
  },
  'dokumente.settings.storage.tier.business': {
    de: 'Business', en: 'Business', fr: 'Business', it: 'Business',
  },
  'dokumente.settings.storage.tier.enterprise': {
    de: 'Enterprise', en: 'Enterprise', fr: 'Enterprise', it: 'Enterprise',
  },
  'dokumente.settings.storage.trashDays': {
    de: 'Aufbewahrung im Papierkorb (Tage)', en: 'Trash retention (days)', fr: 'Conservation dans la corbeille (jours)', it: 'Conservazione nel cestino (giorni)',
  },
  'dokumente.settings.storage.trashDaysHint': {
    de: 'Gelöschte Dateien werden nach Ablauf endgültig entfernt.',
    en: 'Deleted files are permanently removed after this period.',
    fr: 'Les fichiers supprimés sont définitivement effacés après ce délai.',
    it: 'I file eliminati vengono rimossi definitivamente dopo questo periodo.',
  },
  'dokumente.settings.fileTypes.title': {
    de: 'Erlaubte Dateitypen', en: 'Allowed file types', fr: 'Types de fichiers autorisés', it: 'Tipi di file consentiti',
  },
  'dokumente.settings.fileTypes.desc': {
    de: 'Welche Dateigruppen hochgeladen werden dürfen.',
    en: 'Which file groups may be uploaded.',
    fr: 'Quels groupes de fichiers peuvent être téléversés.',
    it: 'Quali gruppi di file possono essere caricati.',
  },
  'dokumente.settings.fileTypes.hint': {
    de: 'Gilt für neue Uploads; bestehende Dateien bleiben unberührt.',
    en: 'Applies to new uploads; existing files are not affected.',
    fr: 'S’applique aux nouveaux téléversements ; les fichiers existants ne sont pas concernés.',
    it: 'Vale per i nuovi caricamenti; i file esistenti non vengono toccati.',
  },
  'dokumente.settings.fileTypes.group.documents': {
    de: 'Dokumente', en: 'Documents', fr: 'Documents', it: 'Documenti',
  },
  'dokumente.settings.fileTypes.group.spreadsheets': {
    de: 'Tabellen', en: 'Spreadsheets', fr: 'Tableurs', it: 'Fogli di calcolo',
  },
  'dokumente.settings.fileTypes.group.presentations': {
    de: 'Präsentationen', en: 'Presentations', fr: 'Présentations', it: 'Presentazioni',
  },
  'dokumente.settings.fileTypes.group.images': {
    de: 'Bilder', en: 'Images', fr: 'Images', it: 'Immagini',
  },
  'dokumente.settings.fileTypes.group.media': {
    de: 'Audio & Video', en: 'Audio & video', fr: 'Audio et vidéo', it: 'Audio e video',
  },
  'dokumente.settings.fileTypes.group.archives': {
    de: 'Archive', en: 'Archives', fr: 'Archives', it: 'Archivi',
  },
  'dokumente.settings.fileTypes.group.other': {
    de: 'Sonstige', en: 'Other', fr: 'Autres', it: 'Altri',
  },
  'dokumente.settings.sharing.title': {
    de: 'Freigabe & Bearbeitung', en: 'Sharing & editing', fr: 'Partage et édition', it: 'Condivisione e modifica',
  },
  'dokumente.settings.sharing.desc': {
    de: 'Standard-Sichtbarkeit und Online-Bearbeitung.',
    en: 'Default visibility and online editing.',
    fr: 'Visibilité par défaut et édition en ligne.',
    it: 'Visibilità predefinita e modifica online.',
  },
  'dokumente.settings.sharing.defaultScope': {
    de: 'Standard-Freigabe neuer Dateien', en: 'Default sharing for new files', fr: 'Partage par défaut des nouveaux fichiers', it: 'Condivisione predefinita dei nuovi file',
  },
  'dokumente.settings.sharing.defaultScopeHint': {
    de: 'Neue Uploads sind zunächst privat oder für das Team sichtbar.',
    en: 'New uploads start out private or visible to the team.',
    fr: 'Les nouveaux téléversements sont d’abord privés ou visibles par l’équipe.',
    it: 'I nuovi caricamenti sono inizialmente privati o visibili al team.',
  },
  'dokumente.settings.sharing.scopePrivate': {
    de: 'Privat', en: 'Private', fr: 'Privé', it: 'Privato',
  },
  'dokumente.settings.sharing.scopeTeam': {
    de: 'Team', en: 'Team', fr: 'Équipe', it: 'Team',
  },
  'dokumente.settings.sharing.onlyOffice': {
    de: 'OnlyOffice-Editor', en: 'OnlyOffice editor', fr: 'Éditeur OnlyOffice', it: 'Editor OnlyOffice',
  },
  'dokumente.settings.sharing.onlyOfficeHint': {
    de: 'Office-Dateien direkt in Cosmi bearbeiten.',
    en: 'Edit office files directly in Cosmi.',
    fr: 'Modifier les fichiers Office directement dans Cosmi.',
    it: 'Modifica i file Office direttamente in Cosmi.',
  },
}

const lineKey = (line) => {
  const m = line.match(/^\s*"([^"]+)":/)
  return m ? m[1] : null
}

const report = {}
for (const loc of ['de', 'en', 'fr', 'it']) {
  const file = resolve(dir, `${loc}.json`)
  const obj = JSON.parse(readFileSync(file, 'utf8'))
  let lines = readFileSync(file, 'utf8').split('\n')
  let added = 0
  for (const key of Object.keys(KEYS).sort()) {
    if (key in obj) continue
    // insert before the first existing key that sorts after the new key
    let idx = lines.findIndex((l) => {
      const k = lineKey(l)
      return k !== null && k > key
    })
    if (idx === -1) throw new Error(`no insertion point for ${key} in ${loc}`)
    lines = [...lines.slice(0, idx), `  ${JSON.stringify(key)}: ${JSON.stringify(KEYS[key][loc])},`, ...lines.slice(idx)]
    added++
  }
  const out = lines.join('\n')
  JSON.parse(out) // validate before writing
  writeFileSync(file, out, 'utf8')
  report[loc] = added
}
console.log(JSON.stringify(report))
