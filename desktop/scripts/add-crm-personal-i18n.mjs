import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'
const __dirname = dirname(fileURLToPath(import.meta.url))
const MSG = join(__dirname, '..', 'src', 'renderer', 'src', 'i18n', 'messages')
const add = {
  de: {
    'crm.settings.personal.title': 'Persönliche Ansicht',
    'crm.settings.personal.desc': 'Passt das CRM an deinen Arbeitsablauf an — gilt nur für dich.',
    'crm.settings.personal.defaultView': 'Standard-Ansicht',
    'crm.settings.personal.viewList': 'Liste',
    'crm.settings.personal.viewGrid': 'Raster',
    'crm.settings.personal.density': 'Dichte',
    'crm.settings.personal.densityComfortable': 'Komfortabel',
    'crm.settings.personal.densityCompact': 'Kompakt',
    'crm.settings.personal.showAvatars': 'Avatare in Listen anzeigen',
  },
  en: {
    'crm.settings.personal.title': 'Personal view',
    'crm.settings.personal.desc': 'Adapt the CRM to your workflow — applies only to you.',
    'crm.settings.personal.defaultView': 'Default view',
    'crm.settings.personal.viewList': 'List',
    'crm.settings.personal.viewGrid': 'Grid',
    'crm.settings.personal.density': 'Density',
    'crm.settings.personal.densityComfortable': 'Comfortable',
    'crm.settings.personal.densityCompact': 'Compact',
    'crm.settings.personal.showAvatars': 'Show avatars in lists',
  },
  fr: {
    'crm.settings.personal.title': 'Affichage personnel',
    'crm.settings.personal.desc': "Adaptez le CRM à votre flux de travail — ne s'applique qu'à vous.",
    'crm.settings.personal.defaultView': 'Affichage par défaut',
    'crm.settings.personal.viewList': 'Liste',
    'crm.settings.personal.viewGrid': 'Grille',
    'crm.settings.personal.density': 'Densité',
    'crm.settings.personal.densityComfortable': 'Confortable',
    'crm.settings.personal.densityCompact': 'Compact',
    'crm.settings.personal.showAvatars': 'Afficher les avatars dans les listes',
  },
  it: {
    'crm.settings.personal.title': 'Vista personale',
    'crm.settings.personal.desc': 'Adatta il CRM al tuo flusso di lavoro — vale solo per te.',
    'crm.settings.personal.defaultView': 'Vista predefinita',
    'crm.settings.personal.viewList': 'Elenco',
    'crm.settings.personal.viewGrid': 'Griglia',
    'crm.settings.personal.density': 'Densità',
    'crm.settings.personal.densityComfortable': 'Comoda',
    'crm.settings.personal.densityCompact': 'Compatta',
    'crm.settings.personal.showAvatars': 'Mostra avatar negli elenchi',
  },
}
for (const [loc, keys] of Object.entries(add)) {
  const f = join(MSG, `${loc}.json`); const d = JSON.parse(readFileSync(f, 'utf8')); let n = 0
  for (const [k, v] of Object.entries(keys)) { if (!(k in d)) { d[k] = v; n++ } }
  writeFileSync(f, JSON.stringify(d, null, 2) + '\n', 'utf8'); console.log(`${loc}: +${n}`)
}
