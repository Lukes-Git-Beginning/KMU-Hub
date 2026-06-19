import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
const dir = resolve('src/renderer/src/i18n/messages')
const groups = [
  { anchor: 'automatisierung.tabs.templates', keys: {
    'automatisierung.settings.title': { de:'Automatisierung', en:'Automation', fr:'Automatisation', it:'Automazione' },
    'automatisierung.settings.subtitle': { de:'Standard-Ansicht und Protokoll-Einstellungen', en:'Default view and log settings', fr:'Vue par défaut et paramètres du journal', it:'Vista predefinita e impostazioni del registro' },
    'automatisierung.settings.personal.title': { de:'Persönliche Vorgaben', en:'Personal defaults', fr:'Préférences personnelles', it:'Preferenze personali' },
    'automatisierung.settings.personal.desc': { de:'Gilt nur für dich.', en:'Applies only to you.', fr:'Ne concerne que vous.', it:'Vale solo per te.' },
    'automatisierung.settings.personal.startTab': { de:'Standard-Ansicht', en:'Default view', fr:'Vue par défaut', it:'Vista predefinita' },
    'automatisierung.settings.personal.startTabHint': { de:'Welche Ansicht beim Öffnen des Moduls erscheint.', en:'Which view opens when you enter the module.', fr:"Quelle vue s'ouvre à l'entrée du module.", it:"Quale vista si apre all'apertura del modulo." },
    'automatisierung.settings.retention.title': { de:'Protokoll-Aufbewahrung', en:'Log retention', fr:'Conservation du journal', it:'Conservazione del registro' },
    'automatisierung.settings.retention.desc': { de:'Wie lange Ausführungs-Protokolle aufbewahrt werden.', en:'How long execution logs are kept.', fr:"Durée de conservation des journaux d'exécution.", it:'Per quanto tempo vengono conservati i registri di esecuzione.' },
    'automatisierung.settings.retention.label': { de:'Aufbewahrungsdauer', en:'Retention period', fr:'Durée de conservation', it:'Periodo di conservazione' },
    'automatisierung.settings.retention.hint': { de:'Ältere Protokolleinträge werden automatisch entfernt.', en:'Older log entries are removed automatically.', fr:'Les entrées plus anciennes sont supprimées automatiquement.', it:'Le voci più vecchie vengono rimosse automaticamente.' },
    'automatisierung.settings.retention.days': { de:'{days} Tage', en:'{days} days', fr:'{days} jours', it:'{days} giorni' },
    'automatisierung.settings.failure.title': { de:'Fehler-Benachrichtigung', en:'Failure notification', fr:"Notification d'échec", it:'Notifica di errore' },
    'automatisierung.settings.failure.desc': { de:'Benachrichtigung bei fehlgeschlagenen Ausführungen.', en:'Notify on failed executions.', fr:'Notifier en cas d’échec.', it:'Notifica in caso di esecuzioni non riuscite.' },
    'automatisierung.settings.failure.label': { de:'Bei Fehler benachrichtigen', en:'Notify on failure', fr:"Notifier en cas d'échec", it:'Notifica in caso di errore' },
    'automatisierung.settings.failure.hint': { de:'Das Team wird informiert, wenn eine Automatisierung fehlschlägt.', en:'The team is informed when an automation fails.', fr:'L’équipe est informée en cas d’échec.', it:'Il team viene informato quando un’automazione fallisce.' },
  }},
  { anchor: 'moduleSettings.entries.work', keys: {
    'moduleSettings.entries.automatisierung': { de:'Automatisierung', en:'Automation', fr:'Automatisation', it:'Automazione' },
  }},
]
const report={}
for (const loc of ['de','en','fr','it']) {
  const file=resolve(dir,`${loc}.json`); const obj=JSON.parse(readFileSync(file,'utf8'))
  let lines=readFileSync(file,'utf8').split('\n'); let added=0
  for (const g of groups) {
    const nk=Object.keys(g.keys).filter(k=>!(k in obj)).sort()
    if(!nk.length) continue
    const block=nk.map(k=>`  ${JSON.stringify(k)}: ${JSON.stringify(g.keys[k][loc])},`)
    const idx=lines.findIndex(l=>l.trimStart().startsWith(`"${g.anchor}":`))
    if(idx===-1) throw new Error(`anchor ${g.anchor} missing ${loc}`)
    lines=[...lines.slice(0,idx+1),...block,...lines.slice(idx+1)]; added+=block.length
  }
  const out=lines.join('\n'); JSON.parse(out); writeFileSync(file,out,'utf8'); report[loc]=added
}
console.log(JSON.stringify(report))
