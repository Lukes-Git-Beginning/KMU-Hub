import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
const dir = resolve('src/renderer/src/i18n/messages')
const groups = [
  { anchor: 'kommunikation.bereich.posteingang', keys: {
    'kommunikation.assign.assign': { de:'Zuweisen', en:'Assign', fr:'Attribuer', it:'Assegna' },
    'kommunikation.assign.assignTo': { de:'Zuweisen an', en:'Assign to', fr:'Attribuer à', it:'Assegna a' },
    'kommunikation.assign.claim': { de:'Übernehmen', en:'Claim', fr:'Prendre en charge', it:'Prendi in carico' },
    'kommunikation.assign.claimHint': { de:'Mir zuweisen', en:'Assign to me', fr:'Me l’attribuer', it:'Assegna a me' },
    'kommunikation.assign.noUsers': { de:'Keine Benutzer verfügbar', en:'No users available', fr:'Aucun utilisateur disponible', it:'Nessun utente disponibile' },
    'kommunikation.bulk.archive': { de:'Archivieren', en:'Archive', fr:'Archiver', it:'Archivia' },
    'kommunikation.bulk.markRead': { de:'Gelesen', en:'Mark read', fr:'Marquer lu', it:'Segna letto' },
    'kommunikation.bulk.select': { de:'Mehrere auswählen', en:'Select multiple', fr:'Sélection multiple', it:'Selezione multipla' },
    'kommunikation.bulk.selected': {
      de:'{count, plural, one {# ausgewählt} other {# ausgewählt}}',
      en:'{count, plural, one {# selected} other {# selected}}',
      fr:'{count, plural, one {# sélectionné} other {# sélectionnés}}',
      it:'{count, plural, one {# selezionato} other {# selezionati}}',
    },
    'kommunikation.forward.title': { de:'Weiterleiten', en:'Forward', fr:'Transférer', it:'Inoltra' },
    'kommunikation.forward.description': { de:'Nachricht „{subject}" an eine Kollegin oder externe Adresse weiterleiten.', en:'Forward the message “{subject}” to a colleague or external address.', fr:'Transférer le message « {subject} » à un collègue ou une adresse externe.', it:'Inoltra il messaggio «{subject}» a un collega o a un indirizzo esterno.' },
    'kommunikation.forward.recipient': { de:'Empfänger', en:'Recipient', fr:'Destinataire', it:'Destinatario' },
    'kommunikation.forward.recipientPlaceholder': { de:'E-Mail oder Name', en:'Email or name', fr:'E-mail ou nom', it:'E-mail o nome' },
    'kommunikation.forward.note': { de:'Notiz (optional)', en:'Note (optional)', fr:'Note (facultatif)', it:'Nota (facoltativo)' },
    'kommunikation.forward.notePlaceholder': { de:'Kontext für den Empfänger…', en:'Context for the recipient…', fr:'Contexte pour le destinataire…', it:'Contesto per il destinatario…' },
    'kommunikation.forward.submit': { de:'Weiterleiten', en:'Forward', fr:'Transférer', it:'Inoltra' },
    'kommunikation.forward.sent': { de:'An {recipient} weitergeleitet', en:'Forwarded to {recipient}', fr:'Transféré à {recipient}', it:'Inoltrato a {recipient}' },
    'kommunikation.settings.routing.title': { de:'Routing-Regeln', en:'Routing rules', fr:'Règles de routage', it:'Regole di instradamento' },
    'kommunikation.settings.routing.desc': { de:'Eingehende Nachrichten automatisch zuordnen und taggen.', en:'Automatically assign and tag incoming messages.', fr:'Attribuer et étiqueter automatiquement les messages entrants.', it:'Assegna ed etichetta automaticamente i messaggi in arrivo.' },
    'kommunikation.settings.teamInboxes.title': { de:'Team-Postfächer', en:'Team inboxes', fr:'Boîtes d’équipe', it:'Caselle del team' },
    'kommunikation.settings.teamInboxes.desc': { de:'Gemeinsame Postfächer und Zuweisungsregeln für Teams verwalten.', en:'Manage shared inboxes and assignment for teams.', fr:'Gérer les boîtes partagées et l’attribution pour les équipes.', it:'Gestisci caselle condivise e assegnazioni per i team.' },
    'kommunikation.status.change': { de:'Status ändern', en:'Change status', fr:'Changer le statut', it:'Cambia stato' },
    'kommunikation.status.changed': { de:'Status: {status}', en:'Status: {status}', fr:'Statut : {status}', it:'Stato: {status}' },
    'kommunikation.tags.add': { de:'Tag hinzufügen', en:'Add tag', fr:'Ajouter un tag', it:'Aggiungi tag' },
    'kommunikation.tags.placeholder': { de:'Neues Tag…', en:'New tag…', fr:'Nouveau tag…', it:'Nuovo tag…' },
    'kommunikation.tags.remove': { de:'Tag entfernen', en:'Remove tag', fr:'Retirer le tag', it:'Rimuovi tag' },
    'kommunikation.teamInbox.create': { de:'Neues Postfach', en:'New inbox', fr:'Nouvelle boîte', it:'Nuova casella' },
    'kommunikation.teamInbox.defaultName': { de:'Neues Team-Postfach', en:'New team inbox', fr:'Nouvelle boîte d’équipe', it:'Nuova casella del team' },
    'kommunikation.teamInbox.empty': { de:'Noch keine Team-Postfächer angelegt.', en:'No team inboxes yet.', fr:'Aucune boîte d’équipe pour le moment.', it:'Nessuna casella del team.' },
    'kommunikation.teamInbox.listTitle': { de:'Team-Postfächer', en:'Team inboxes', fr:'Boîtes d’équipe', it:'Caselle del team' },
    'kommunikation.thread.you': { de:'Ich', en:'You', fr:'Moi', it:'Io' },
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
