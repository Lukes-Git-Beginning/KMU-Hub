import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'

const dir = resolve('src/renderer/src/i18n/messages')

const KEYS = {
  'dokumente.moveCopy.titleMove': {
    de: 'Datei verschieben', en: 'Move file', fr: 'Déplacer le fichier', it: 'Sposta file',
  },
  'dokumente.moveCopy.titleCopy': {
    de: 'Datei kopieren', en: 'Copy file', fr: 'Copier le fichier', it: 'Copia file',
  },
  'dokumente.moveCopy.subtitle': {
    de: 'Zielordner für „{name}" wählen',
    en: 'Choose a target folder for “{name}”',
    fr: 'Choisir un dossier cible pour « {name} »',
    it: 'Scegli una cartella di destinazione per “{name}”',
  },
  'dokumente.moveCopy.currentFolder': {
    de: 'Aktueller Ordner', en: 'Current folder', fr: 'Dossier actuel', it: 'Cartella attuale',
  },
  'dokumente.moveCopy.noFolders': {
    de: 'Keine Ordner vorhanden', en: 'No folders available', fr: 'Aucun dossier disponible', it: 'Nessuna cartella disponibile',
  },
  'dokumente.moveCopy.actionMove': {
    de: 'Verschieben', en: 'Move', fr: 'Déplacer', it: 'Sposta',
  },
  'dokumente.moveCopy.actionCopy': {
    de: 'Kopieren', en: 'Copy', fr: 'Copier', it: 'Copia',
  },
  'dokumente.moveCopy.movedToast': {
    de: '„{name}" wurde verschoben',
    en: '“{name}” has been moved',
    fr: '« {name} » a été déplacé',
    it: '“{name}” è stato spostato',
  },
  'dokumente.moveCopy.copiedToast': {
    de: '„{name}" wurde kopiert',
    en: '“{name}” has been copied',
    fr: '« {name} » a été copié',
    it: '“{name}” è stato copiato',
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
    let idx = lines.findIndex((l) => {
      const k = lineKey(l)
      return k !== null && k > key
    })
    if (idx === -1) throw new Error(`no insertion point for ${key} in ${loc}`)
    lines = [...lines.slice(0, idx), `  ${JSON.stringify(key)}: ${JSON.stringify(KEYS[key][loc])},`, ...lines.slice(idx)]
    added++
  }
  const out = lines.join('\n')
  JSON.parse(out)
  writeFileSync(file, out, 'utf8')
  report[loc] = added
}
console.log(JSON.stringify(report))
