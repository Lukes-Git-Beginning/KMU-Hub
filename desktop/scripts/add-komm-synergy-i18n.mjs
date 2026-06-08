import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
const dir = resolve('src/renderer/src/i18n/messages')
const groups = [
  { anchor: 'kommunikation.bereich.posteingang', keys: {
    'kommunikation.call.audio': { de:'Audioanruf', en:'Audio call', fr:'Appel audio', it:'Chiamata audio' },
    'kommunikation.call.video': { de:'Videoanruf', en:'Video call', fr:'Appel vidéo', it:'Videochiamata' },
    'kommunikation.call.startingAudio': { de:'Audioanruf mit {name} wird gestartet…', en:'Starting audio call with {name}…', fr:'Appel audio avec {name}…', it:'Avvio chiamata audio con {name}…' },
    'kommunikation.call.startingVideo': { de:'Videoanruf mit {name} wird gestartet…', en:'Starting video call with {name}…', fr:'Appel vidéo avec {name}…', it:'Avvio videochiamata con {name}…' },
    'kommunikation.collision.editing': { de:'{name} bearbeitet diese Konversation gerade', en:'{name} is working on this conversation', fr:'{name} traite cette conversation', it:'{name} sta lavorando su questa conversazione' },
    'kommunikation.slash.title': { de:'Befehle', en:'Commands', fr:'Commandes', it:'Comandi' },
    'kommunikation.slash.giphy': { de:'GIF einfügen', en:'Insert GIF', fr:'Insérer un GIF', it:'Inserisci GIF' },
    'kommunikation.slash.giphyDesc': { de:'Ein GIF suchen und senden', en:'Search and send a GIF', fr:'Rechercher et envoyer un GIF', it:'Cerca e invia una GIF' },
    'kommunikation.slash.poll': { de:'Umfrage', en:'Poll', fr:'Sondage', it:'Sondaggio' },
    'kommunikation.slash.pollDesc': { de:'Eine Umfrage erstellen', en:'Create a poll', fr:'Créer un sondage', it:'Crea un sondaggio' },
    'kommunikation.slash.reminder': { de:'Erinnerung', en:'Reminder', fr:'Rappel', it:'Promemoria' },
    'kommunikation.slash.reminderDesc': { de:'Eine Erinnerung setzen', en:'Set a reminder', fr:'Définir un rappel', it:'Imposta un promemoria' },
    'kommunikation.slash.executed': { de:'Befehl /{command} (Demo)', en:'Command /{command} (demo)', fr:'Commande /{command} (démo)', it:'Comando /{command} (demo)' },
    'kommunikation.canned.title': { de:'Textbausteine', en:'Canned responses', fr:'Réponses types', it:'Risposte predefinite' },
    'kommunikation.canned.new': { de:'Neuer Baustein', en:'New response', fr:'Nouvelle réponse', it:'Nuova risposta' },
    'kommunikation.canned.empty': { de:'Noch keine Textbausteine.', en:'No canned responses yet.', fr:'Aucune réponse type.', it:'Nessuna risposta predefinita.' },
    'kommunikation.canned.titlePlaceholder': { de:'Titel', en:'Title', fr:'Titre', it:'Titolo' },
    'kommunikation.canned.shortcutPlaceholder': { de:'Kürzel (z.B. /hi)', en:'Shortcut (e.g. /hi)', fr:'Raccourci (p. ex. /hi)', it:'Scorciatoia (es. /hi)' },
    'kommunikation.canned.contentPlaceholder': { de:'Inhalt des Bausteins…', en:'Response content…', fr:'Contenu de la réponse…', it:'Contenuto della risposta…' },
    'kommunikation.webhook.title': { de:'Webhooks', en:'Webhooks', fr:'Webhooks', it:'Webhook' },
    'kommunikation.webhook.empty': { de:'Noch keine Webhooks konfiguriert.', en:'No webhooks configured yet.', fr:'Aucun webhook configuré.', it:'Nessun webhook configurato.' },
    'kommunikation.webhook.urlPlaceholder': { de:'https://…', en:'https://…', fr:'https://…', it:'https://…' },
    'kommunikation.webhook.hint': { de:'Ausgehende Webhooks senden Ereignisse an externe Dienste (Demo).', en:'Outgoing webhooks send events to external services (demo).', fr:'Les webhooks sortants envoient des événements à des services externes (démo).', it:'I webhook in uscita inviano eventi a servizi esterni (demo).' },
    'kommunikation.settings.presence.title': { de:'Eigener Status', en:'Your status', fr:'Votre statut', it:'Il tuo stato' },
    'kommunikation.settings.presence.desc': { de:'Lege fest, wie du für dein Team erscheinst.', en:'Choose how you appear to your team.', fr:'Choisissez comment vous apparaissez à votre équipe.', it:'Scegli come appari al tuo team.' },
    'kommunikation.settings.presence.online': { de:'Verfügbar', en:'Available', fr:'Disponible', it:'Disponibile' },
    'kommunikation.settings.presence.away': { de:'Abwesend', en:'Away', fr:'Absent', it:'Assente' },
    'kommunikation.settings.presence.dnd': { de:'Bitte nicht stören', en:'Do not disturb', fr:'Ne pas déranger', it:'Non disturbare' },
    'kommunikation.settings.notifications.title': { de:'Benachrichtigungen', en:'Notifications', fr:'Notifications', it:'Notifiche' },
    'kommunikation.settings.notifications.desc': { de:'Steuere, welche Kanäle dich benachrichtigen.', en:'Control which channels notify you.', fr:'Choisissez quels canaux vous notifient.', it:'Controlla quali canali ti notificano.' },
    'kommunikation.settings.notifications.hint': { de:'Deaktiviere einen Kanal, um seine Benachrichtigungen stummzuschalten.', en:'Turn a channel off to mute its notifications.', fr:'Désactivez un canal pour couper ses notifications.', it:'Disattiva un canale per silenziarne le notifiche.' },
    'kommunikation.settings.notifications.chat': { de:'Team-Chat', en:'Team chat', fr:'Chat d’équipe', it:'Chat del team' },
    'kommunikation.settings.notifications.system': { de:'Systembenachrichtigungen', en:'System notifications', fr:'Notifications système', it:'Notifiche di sistema' },
    'kommunikation.settings.channels.title': { de:'Kanäle', en:'Channels', fr:'Canaux', it:'Canali' },
    'kommunikation.settings.channels.desc': { de:'E-Mail, WhatsApp und Widget-Kanäle verbinden.', en:'Connect email, WhatsApp and widget channels.', fr:'Connecter les canaux e-mail, WhatsApp et widget.', it:'Collega i canali e-mail, WhatsApp e widget.' },
    'kommunikation.settings.channels.hint': { de:'Verbinde externe Kanäle, damit deren Nachrichten im Posteingang landen.', en:'Connect external channels so their messages land in the inbox.', fr:'Connectez des canaux externes pour recevoir leurs messages dans la boîte de réception.', it:'Collega canali esterni perché i loro messaggi arrivino in posta in arrivo.' },
    'kommunikation.settings.channels.manage': { de:'Kanäle verwalten', en:'Manage channels', fr:'Gérer les canaux', it:'Gestisci canali' },
    'kommunikation.settings.canned.title': { de:'Textbausteine', en:'Canned responses', fr:'Réponses types', it:'Risposte predefinite' },
    'kommunikation.settings.canned.desc': { de:'Vorlagen für schnelle Antworten verwalten.', en:'Manage quick-reply templates.', fr:'Gérer les modèles de réponse rapide.', it:'Gestisci i modelli di risposta rapida.' },
    'kommunikation.settings.webhooks.title': { de:'Webhooks', en:'Webhooks', fr:'Webhooks', it:'Webhook' },
    'kommunikation.settings.webhooks.desc': { de:'Ereignisse an externe Dienste senden.', en:'Send events to external services.', fr:'Envoyer des événements à des services externes.', it:'Invia eventi a servizi esterni.' },
    'kommunikation.settings.retention.title': { de:'Aufbewahrung', en:'Retention', fr:'Conservation', it:'Conservazione' },
    'kommunikation.settings.retention.desc': { de:'Gesetzliche Aufbewahrungsfristen für Korrespondenz.', en:'Legal retention periods for correspondence.', fr:'Délais légaux de conservation de la correspondance.', it:'Periodi di conservazione legale della corrispondenza.' },
    'kommunikation.settings.retention.notice': { de:'Geschäftliche Korrespondenz (Handelsbriefe) unterliegt einer gesetzlichen Aufbewahrungsfrist von 6 Jahren (§257 HGB, §147 AO). Nachrichten im Posteingang werden entsprechend archiviert und nicht vorzeitig gelöscht.', en:'Business correspondence (commercial letters) is subject to a 6-year legal retention period (§257 HGB, §147 AO). Inbox messages are archived accordingly and not deleted early.', fr:'La correspondance commerciale est soumise à une conservation légale de 6 ans (§257 HGB, §147 AO). Les messages sont archivés en conséquence et non supprimés prématurément.', it:'La corrispondenza commerciale è soggetta a un periodo di conservazione legale di 6 anni (§257 HGB, §147 AO). I messaggi vengono archiviati di conseguenza e non eliminati in anticipo.' },
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
