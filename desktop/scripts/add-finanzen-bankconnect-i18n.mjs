import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
const dir = resolve('src/renderer/src/i18n/messages')
const groups = [
  { anchor: 'finanzen.banking.connect', keys: {
    'finanzen.banking.connectTitle': { de:`Bankkonto verbinden`, en:`Connect bank account`, fr:`Connecter un compte bancaire`, it:`Collega conto bancario` },
    'finanzen.banking.searchBank': { de:`Bank suchen…`, en:`Search bank…`, fr:`Rechercher une banque…`, it:`Cerca banca…` },
    'finanzen.banking.noBankFound': { de:`Keine Bank gefunden`, en:`No bank found`, fr:`Aucune banque trouvée`, it:`Nessuna banca trovata` },
    'finanzen.banking.loginName': { de:`Anmeldename`, en:`Login name`, fr:`Identifiant`, it:`Nome utente` },
    'finanzen.banking.pin': { de:`PIN`, en:`PIN`, fr:`Code PIN`, it:`PIN` },
    'finanzen.banking.psd2Hint': { de:`Sichere Verbindung über FinAPI (PSD2). Zugangsdaten werden verschlüsselt direkt an Ihre Bank übermittelt.`, en:`Secure connection via FinAPI (PSD2). Your credentials are sent encrypted directly to your bank.`, fr:`Connexion sécurisée via FinAPI (PSD2). Vos identifiants sont transmis chiffrés directement à votre banque.`, it:`Connessione sicura tramite FinAPI (PSD2). Le credenziali vengono inviate crittografate direttamente alla tua banca.` },
    'finanzen.banking.demoHint': { de:`Demo — bitte keine echten Bankzugangsdaten eingeben.`, en:`Demo — please do not enter real banking credentials.`, fr:`Démo — n'entrez pas de véritables identifiants bancaires.`, it:`Demo — non inserire credenziali bancarie reali.` },
    'finanzen.banking.secureConnect': { de:`Sicher verbinden`, en:`Connect securely`, fr:`Connexion sécurisée`, it:`Connetti in sicurezza` },
    'finanzen.banking.connecting': { de:`Verbindung wird hergestellt…`, en:`Establishing connection…`, fr:`Connexion en cours…`, it:`Connessione in corso…` },
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
