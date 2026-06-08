import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
const dir = resolve('src/renderer/src/i18n/messages')
const T = {
  de: {
    'work.labels.title': 'Labels',
    'work.labels.add': 'Label',
    'work.labels.noneDefined': 'Keine Labels definiert',
    'work.labels.manageHint': 'In den Moduleinstellungen verwalten',
    'work.filter.labels': 'Labels',
  },
  en: {
    'work.labels.title': 'Labels',
    'work.labels.add': 'Label',
    'work.labels.noneDefined': 'No labels defined',
    'work.labels.manageHint': 'Manage in module settings',
    'work.filter.labels': 'Labels',
  },
  fr: {
    'work.labels.title': 'Étiquettes',
    'work.labels.add': 'Étiquette',
    'work.labels.noneDefined': 'Aucune étiquette définie',
    'work.labels.manageHint': 'Gérer dans les réglages du module',
    'work.filter.labels': 'Étiquettes',
  },
  it: {
    'work.labels.title': 'Etichette',
    'work.labels.add': 'Etichetta',
    'work.labels.noneDefined': 'Nessuna etichetta definita',
    'work.labels.manageHint': 'Gestisci nelle impostazioni del modulo',
    'work.filter.labels': 'Etichette',
  },
}
for (const [lang, entries] of Object.entries(T)) {
  const file = resolve(dir, `${lang}.json`)
  const json = JSON.parse(readFileSync(file, 'utf8'))
  let added = 0
  for (const [k, v] of Object.entries(entries)) { if (!(k in json)) added++; json[k] = v }
  writeFileSync(file, JSON.stringify(json, null, 2) + '\n', 'utf8')
  console.log(`${lang}: +${added} keys`)
}
