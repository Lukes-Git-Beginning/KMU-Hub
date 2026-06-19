import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
const dir = resolve('src/renderer/src/i18n/messages')
const groups = [
  { anchor: 'crm.deals.expectedCloseShort', keys: {
    'crm.bulk.clearSelection': { de:'Auswahl aufheben', en:'Clear selection', fr:'Effacer la sélection', it:'Deseleziona' },
    'crm.bulk.deleteConfirm': { de:'Möchtest du {count} Einträge wirklich löschen?', en:'Delete {count} items?', fr:'Supprimer {count} éléments ?', it:'Eliminare {count} elementi?' },
    'crm.bulk.deleteError': { de:'Löschen fehlgeschlagen', en:'Delete failed', fr:'Échec de la suppression', it:'Eliminazione non riuscita' },
    'crm.bulk.deleteTitle': { de:'Auswahl löschen', en:'Delete selection', fr:'Supprimer la sélection', it:'Elimina selezione' },
    'crm.bulk.deleted': { de:'{count} gelöscht', en:'{count} deleted', fr:'{count} supprimé(s)', it:'{count} eliminati' },
    'crm.bulk.selected': { de:'{count} ausgewählt', en:'{count} selected', fr:'{count} sélectionné(s)', it:'{count} selezionati' },
    'crm.deals.forecast.byMonth': { de:'Erwarteter Abschluss nach Monat', en:'Expected close by month', fr:'Clôture prévue par mois', it:'Chiusura prevista per mese' },
    'crm.deals.forecast.byStage': { de:'Prognose nach Phase', en:'Forecast by stage', fr:'Prévision par étape', it:'Previsione per fase' },
    'crm.deals.forecast.empty': { de:'Keine offenen Deals für eine Prognose vorhanden.', en:'No open deals to forecast.', fr:'Aucune opportunité ouverte à prévoir.', it:'Nessun deal aperto da prevedere.' },
    'crm.deals.forecast.openVolume': { de:'Offenes Volumen', en:'Open volume', fr:'Volume ouvert', it:'Volume aperto' },
    'crm.deals.forecast.view': { de:'Prognose', en:'Forecast', fr:'Prévision', it:'Previsione' },
    'crm.deals.forecast.weighted': { de:'Gewichtete Prognose', en:'Weighted forecast', fr:'Prévision pondérée', it:'Previsione ponderata' },
    'crm.deals.forecast.wonValue': { de:'Gewonnen', en:'Won', fr:'Gagnées', it:'Vinti' },
    'crm.deals.inactiveDays': { de:'{days} T. inaktiv', en:'{days}d inactive', fr:'{days} j inactif', it:'{days}g inattivo' },
    'crm.deals.markLost': { de:'Verloren', en:'Lost', fr:'Perdue', it:'Perso' },
    'crm.deals.markWon': { de:'Gewonnen', en:'Won', fr:'Gagnée', it:'Vinto' },
    'crm.deals.moveError': { de:'Verschieben fehlgeschlagen', en:'Failed to move deal', fr:'Échec du déplacement', it:'Spostamento non riuscito' },
    'crm.deals.movedTo': { de:'Deal nach {stage} verschoben', en:'Deal moved to {stage}', fr:'Opportunité déplacée vers {stage}', it:'Deal spostato in {stage}' },
  }},
  { anchor: 'kontakte.tag.add', keys: {
    'kontakte.tag.added': { de:'Tag hinzugefügt', en:'Tag added', fr:'Tag ajouté', it:'Tag aggiunto' },
    'kontakte.tag.remove': { de:'Entfernen', en:'Remove', fr:'Retirer', it:'Rimuovi' },
    'kontakte.tag.removed': { de:'Tag entfernt', en:'Tag removed', fr:'Tag retiré', it:'Tag rimosso' },
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
