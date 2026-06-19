import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
const dir = resolve('src/renderer/src/i18n/messages')
const groups = [
  { anchor: 'automatisierung.action.type', keys: {
    'automatisierung.detail.trigger': { de:'Auslöser', en:'Trigger', fr:'Déclencheur', it:'Attivatore' },
    'automatisierung.detail.conditions': { de:'Bedingungen', en:'Conditions', fr:'Conditions', it:'Condizioni' },
    'automatisierung.detail.actions': { de:'Aktionen', en:'Actions', fr:'Actions', it:'Azioni' },
    'automatisierung.detail.details': { de:'Details', en:'Details', fr:'Détails', it:'Dettagli' },
    'automatisierung.detail.recentRuns': { de:'Letzte Ausführungen', en:'Recent runs', fr:'Dernières exécutions', it:'Esecuzioni recenti' },
    'automatisierung.detail.noConditions': { de:'Keine Bedingung — wird immer ausgeführt', en:'No condition — always runs', fr:"Aucune condition — s'exécute toujours", it:'Nessuna condizione — eseguita sempre' },
    'automatisierung.detail.noRuns': { de:'Noch keine Ausführungen', en:'No runs yet', fr:'Aucune exécution', it:'Nessuna esecuzione' },
    'automatisierung.detail.owner': { de:'Eigentümer', en:'Owner', fr:'Propriétaire', it:'Proprietario' },
    'automatisierung.detail.created': { de:'Erstellt', en:'Created', fr:'Créé', it:'Creato' },
    'automatisierung.detail.updated': { de:'Aktualisiert', en:'Updated', fr:'Mis à jour', it:'Aggiornato' },
    'automatisierung.detail.edit': { de:'Bearbeiten', en:'Edit', fr:'Modifier', it:'Modifica' },
    'automatisierung.detail.active': { de:'Aktiv', en:'Active', fr:'Actif', it:'Attivo' },
    'automatisierung.detail.inactive': { de:'Inaktiv', en:'Inactive', fr:'Inactif', it:'Inattivo' },
    'automatisierung.detail.duplicate': { de:'Duplizieren', en:'Duplicate', fr:'Dupliquer', it:'Duplica' },
    'automatisierung.detail.delete': { de:'Löschen', en:'Delete', fr:'Supprimer', it:'Elimina' },
    'automatisierung.detail.deleteTitle': { de:'Automatisierung löschen?', en:'Delete automation?', fr:"Supprimer l'automatisation ?", it:'Eliminare automazione?' },
    'automatisierung.detail.deleteBody': { de:'„{name}" wird dauerhaft entfernt. Diese Aktion kann nicht rückgängig gemacht werden.', en:'"{name}" will be permanently removed. This action cannot be undone.', fr:'« {name} » sera supprimé définitivement. Cette action est irréversible.', it:'"{name}" verrà rimosso definitivamente. Questa azione non può essere annullata.' },
    'automatisierung.detail.duplicateSuffix': { de:'(Kopie)', en:'(Copy)', fr:'(Copie)', it:'(Copia)' },
  }},
  { anchor: 'automatisierung.modules.work', keys: {
    'automatisierung.modules.support': { de:'Helpdesk', en:'Helpdesk', fr:'Assistance', it:'Helpdesk' },
    'automatisierung.modules.system': { de:'System', en:'System', fr:'Système', it:'Sistema' },
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
