import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'

const dir = resolve('src/renderer/src/i18n/messages')

const KEYS = {
  'dokumente.viewer.info': { de: 'Info', en: 'Info', fr: 'Infos', it: 'Info' },
  'dokumente.viewer.details': { de: 'Details', en: 'Details', fr: 'Détails', it: 'Dettagli' },
  'dokumente.viewer.type': { de: 'Typ', en: 'Type', fr: 'Type', it: 'Tipo' },
  'dokumente.viewer.size': { de: 'Größe', en: 'Size', fr: 'Taille', it: 'Dimensione' },
  'dokumente.viewer.created': { de: 'Erstellt', en: 'Created', fr: 'Créé', it: 'Creato' },
  'dokumente.viewer.modified': { de: 'Geändert', en: 'Modified', fr: 'Modifié', it: 'Modificato' },
  'dokumente.viewer.version': { de: 'Version', en: 'Version', fr: 'Version', it: 'Versione' },
  'dokumente.viewer.tags': { de: 'Tags', en: 'Tags', fr: 'Tags', it: 'Tag' },
  'dokumente.viewer.addTag': { de: 'Tag hinzufügen', en: 'Add tag', fr: 'Ajouter un tag', it: 'Aggiungi tag' },
  'dokumente.viewer.noMoreTags': {
    de: 'Keine weiteren Tags verfügbar',
    en: 'No more tags available',
    fr: 'Aucun autre tag disponible',
    it: 'Nessun altro tag disponibile',
  },
  'dokumente.viewer.activity': { de: 'Aktivität', en: 'Activity', fr: 'Activité', it: 'Attività' },
  'dokumente.viewer.noActivity': {
    de: 'Noch keine Aktivität',
    en: 'No activity yet',
    fr: 'Aucune activité pour l’instant',
    it: 'Ancora nessuna attività',
  },
  'dokumente.activity.uploaded': {
    de: 'hat die Datei hochgeladen',
    en: 'uploaded the file',
    fr: 'a téléversé le fichier',
    it: 'ha caricato il file',
  },
  'dokumente.activity.renamed': {
    de: 'hat die Datei umbenannt',
    en: 'renamed the file',
    fr: 'a renommé le fichier',
    it: 'ha rinominato il file',
  },
  'dokumente.activity.moved': {
    de: 'hat die Datei verschoben',
    en: 'moved the file',
    fr: 'a déplacé le fichier',
    it: 'ha spostato il file',
  },
  'dokumente.activity.copied': {
    de: 'hat die Datei kopiert',
    en: 'copied the file',
    fr: 'a copié le fichier',
    it: 'ha copiato il file',
  },
  'dokumente.activity.downloaded': {
    de: 'hat die Datei heruntergeladen',
    en: 'downloaded the file',
    fr: 'a téléchargé le fichier',
    it: 'ha scaricato il file',
  },
  'dokumente.activity.shared': {
    de: 'hat die Datei geteilt',
    en: 'shared the file',
    fr: 'a partagé le fichier',
    it: 'ha condiviso il file',
  },
  'dokumente.activity.versionCreated': {
    de: 'hat eine neue Version erstellt',
    en: 'created a new version',
    fr: 'a créé une nouvelle version',
    it: 'ha creato una nuova versione',
  },
  'dokumente.activity.reverted': {
    de: 'hat eine Version wiederhergestellt',
    en: 'restored a version',
    fr: 'a restauré une version',
    it: 'ha ripristinato una versione',
  },
  'dokumente.settings.personal.showPreviews': {
    de: 'Dateivorschau in Kacheln',
    en: 'File previews on tiles',
    fr: 'Aperçus de fichiers sur les vignettes',
    it: 'Anteprime dei file sui riquadri',
  },
  'dokumente.settings.personal.showPreviewsHint': {
    de: 'Zeigt eine Seitenvorschau statt nur des Dateityp-Symbols.',
    en: 'Shows a page preview instead of just the file-type icon.',
    fr: 'Affiche un aperçu de page au lieu de la seule icône de type.',
    it: 'Mostra un’anteprima della pagina invece della sola icona del tipo.',
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
    const idx = lines.findIndex((l) => {
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
