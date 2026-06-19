import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
const dir = resolve('src/renderer/src/i18n/messages')
const groups = [
  { anchor: 'crm.nav.leads', keys: {
    'crm.nav.reports': { de:'Auswertungen', en:'Reports', fr:'Analyses', it:'Analisi' },
  }},
  { anchor: 'crm.nav.search', keys: {
    'crm.reports.activitiesByType': { de:'Aktivitäten nach Typ', en:'Activities by type', fr:'Activités par type', it:'Attività per tipo' },
    'crm.reports.conversion': { de:'Conversion (Gewonnen/Verloren)', en:'Conversion (won/lost)', fr:'Conversion (gagné/perdu)', it:'Conversione (vinti/persi)' },
    'crm.reports.empty': { de:'Sobald Deals und Leads vorhanden sind, erscheinen hier Kennzahlen und Diagramme.', en:'Once you have deals and leads, KPIs and charts appear here.', fr:'Dès que vous aurez des opportunités et des leads, les indicateurs et graphiques apparaîtront ici.', it:'Non appena ci saranno deal e lead, qui appariranno KPI e grafici.' },
    'crm.reports.emptyTitle': { de:'Noch keine Auswertungen', en:'No analytics yet', fr:"Pas encore d'analyses", it:'Ancora nessuna analisi' },
    'crm.reports.funnel': { de:'Pipeline-Funnel', en:'Pipeline funnel', fr:'Entonnoir du pipeline', it:'Funnel della pipeline' },
    'crm.reports.kpi.dueActivities': { de:'Fällige Aktivitäten', en:'Due activities', fr:'Activités à échéance', it:'Attività in scadenza' },
    'crm.reports.kpi.openLeads': { de:'Offene Leads', en:'Open leads', fr:'Leads ouverts', it:'Lead aperti' },
    'crm.reports.kpi.pipelineValue': { de:'Offenes Pipeline-Volumen', en:'Open pipeline value', fr:'Volume du pipeline ouvert', it:'Volume pipeline aperto' },
    'crm.reports.kpi.weightedForecast': { de:'Gewichtete Prognose', en:'Weighted forecast', fr:'Prévision pondérée', it:'Previsione ponderata' },
    'crm.reports.kpi.winRate': { de:'Gewinnrate', en:'Win rate', fr:'Taux de réussite', it:'Tasso di vincita' },
    'crm.reports.leadSources': { de:'Lead-Quellen', en:'Lead sources', fr:'Sources de leads', it:'Origini lead' },
    'crm.reports.noData': { de:'Keine Daten', en:'No data', fr:'Aucune donnée', it:'Nessun dato' },
    'crm.reports.value': { de:'Wert', en:'Value', fr:'Valeur', it:'Valore' },
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
