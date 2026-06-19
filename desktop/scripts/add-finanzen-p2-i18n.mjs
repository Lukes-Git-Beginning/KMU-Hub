import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
const dir = resolve('src/renderer/src/i18n/messages')
const groups = [
  { anchor: 'buchhaltung.newExpense', keys: {
    'buchhaltung.editExpense': { de:`Ausgabe bearbeiten`, en:`Edit expense`, fr:`Modifier la dépense`, it:`Modifica spesa` },
    'buchhaltung.receiptPreview.serverNote': { de:`Eine Vorschau erscheint hier, sobald das Backend angebunden ist. Der Beleg ist erfasst und im Beleg-Archiv hinterlegt.`, en:`A preview appears here once the backend is connected. The receipt is recorded and stored in the receipt archive.`, fr:`Un aperçu s'affichera ici une fois le backend connecté. Le justificatif est enregistré et archivé.`, it:`Un'anteprima apparirà qui dopo il collegamento al backend. Il giustificativo è registrato e archiviato.` },
  }},
  { anchor: 'buchhaltung.toast.expenseRecorded', keys: {
    'buchhaltung.toast.expenseUpdated': { de:`Ausgabe aktualisiert`, en:`Expense updated`, fr:`Dépense mise à jour`, it:`Spesa aggiornata` },
  }},
  { anchor: 'buchhaltung.form.amount', keys: {
    'buchhaltung.form.account': { de:`Sachkonto`, en:`Account`, fr:`Compte`, it:`Conto` },
    'buchhaltung.form.accountHint': { de:`(Kontierung für {framework})`, en:`(account assignment for {framework})`, fr:`(imputation pour {framework})`, it:`(imputazione per {framework})` },
    'buchhaltung.form.accountNone': { de:`Kein Konto`, en:`No account`, fr:`Aucun compte`, it:`Nessun conto` },
    'buchhaltung.form.accountPlaceholder': { de:`Sachkonto wählen`, en:`Select account`, fr:`Choisir un compte`, it:`Seleziona conto` },
    'buchhaltung.form.receipt': { de:`Beleg`, en:`Receipt`, fr:`Justificatif`, it:`Giustificativo` },
    'buchhaltung.form.receiptRemove': { de:`Beleg entfernen`, en:`Remove receipt`, fr:`Supprimer le justificatif`, it:`Rimuovi giustificativo` },
    'buchhaltung.form.receiptUpload': { de:`Beleg hochladen (Bild oder PDF)`, en:`Upload receipt (image or PDF)`, fr:`Téléverser un justificatif (image ou PDF)`, it:`Carica giustificativo (immagine o PDF)` },
  }},
  { anchor: 'buchhaltung.table.supplier', keys: {
    'buchhaltung.table.account': { de:`Konto`, en:`Account`, fr:`Compte`, it:`Conto` },
    'buchhaltung.table.uncategorized': { de:`Kontieren`, en:`Assign account`, fr:`Imputer`, it:`Imputa` },
  }},
  { anchor: 'buchhaltung.actions.viewDetails', keys: {
    'buchhaltung.actions.editExpense': { de:`Bearbeiten`, en:`Edit`, fr:`Modifier`, it:`Modifica` },
    'buchhaltung.actions.viewReceipt': { de:`Beleg ansehen`, en:`View receipt`, fr:`Voir le justificatif`, it:`Vedi giustificativo` },
  }},
  { anchor: 'buchhaltung.reports.expensesByCategory', keys: {
    'buchhaltung.reports.emptyTitle': { de:`Noch keine Daten`, en:`No data yet`, fr:`Pas encore de données`, it:`Ancora nessun dato` },
    'buchhaltung.reports.emptyDesc': { de:`Sobald Einnahmen und Ausgaben erfasst sind, erscheinen hier Auswertungen.`, en:`Once income and expenses are recorded, reports appear here.`, fr:`Dès que des revenus et des dépenses seront enregistrés, les analyses apparaîtront ici.`, it:`Una volta registrati ricavi e spese, qui appariranno le analisi.` },
  }},
  { anchor: 'buchhaltung.categories.Beratung', keys: {
    'buchhaltung.categories.Bewirtung': { de:`Bewirtung`, en:`Entertainment`, fr:`Frais de représentation`, it:`Rappresentanza` },
    'buchhaltung.categories.Büromaterial': { de:`Büromaterial`, en:`Office supplies`, fr:`Fournitures de bureau`, it:`Materiale per ufficio` },
    'buchhaltung.categories.IT-Infrastruktur': { de:`IT-Infrastruktur`, en:`IT infrastructure`, fr:`Infrastructure informatique`, it:`Infrastruttura IT` },
    'buchhaltung.categories.Marketing': { de:`Marketing`, en:`Marketing`, fr:`Marketing`, it:`Marketing` },
    'buchhaltung.categories.Reisekosten': { de:`Reisekosten`, en:`Travel expenses`, fr:`Frais de déplacement`, it:`Spese di viaggio` },
    'buchhaltung.categories.Weiterbildung': { de:`Weiterbildung`, en:`Training`, fr:`Formation`, it:`Formazione` },
  }},
  { anchor: 'finanzen.settings.invoicing.title', keys: {
    'finanzen.settings.kontierung.title': { de:`Kontierung & Kontenrahmen`, en:`Account assignment & chart`, fr:`Imputation et plan comptable`, it:`Imputazione e piano dei conti` },
    'finanzen.settings.kontierung.desc': { de:`Kontenrahmen für die Sachkonto-Zuordnung und den DATEV-Export.`, en:`Chart of accounts for account assignment and DATEV export.`, fr:`Plan comptable pour l'imputation et l'export DATEV.`, it:`Piano dei conti per l'imputazione e l'export DATEV.` },
    'finanzen.settings.kontierung.frameworkLabel': { de:`Kontenrahmen`, en:`Chart of accounts`, fr:`Plan comptable`, it:`Piano dei conti` },
    'finanzen.settings.kontierung.skr03Desc': { de:`SKR03 — Standard-Kontenrahmen (Prozessgliederung), DATEV-Default für die meisten KMU.`, en:`SKR03 — standard chart (process-based), the DATEV default for most SMEs.`, fr:`SKR03 — plan standard (par processus), valeur DATEV par défaut pour la plupart des PME.`, it:`SKR03 — piano standard (per processo), predefinito DATEV per la maggior parte delle PMI.` },
    'finanzen.settings.kontierung.skr04Desc': { de:`SKR04 — Kontenrahmen nach Abschlussgliederung (Bilanz/GuV).`, en:`SKR04 — chart based on financial-statement structure (balance sheet / P&L).`, fr:`SKR04 — plan basé sur la structure des comptes annuels (bilan / compte de résultat).`, it:`SKR04 — piano basato sulla struttura del bilancio (stato patrimoniale / conto economico).` },
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
