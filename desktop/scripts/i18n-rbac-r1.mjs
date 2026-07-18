/**
 * i18n batch for RBAC R-1 (rbac.* namespace + profil.tabs.permissions).
 *
 * Inserts new keys alphabetically into the existing (mostly sorted) flat JSON
 * without reordering existing keys, and removes the dead config.roles.* +
 * team.wizard.role* entries (consumers migrated to rbac.roles.*).
 *
 * Run: node scripts/i18n-rbac-r1.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))
const MESSAGES_DIR = join(here, '..', 'src', 'renderer', 'src', 'i18n', 'messages')
const LOCALES = ['de', 'en', 'fr', 'it']

// ── keys to remove (dead after the R-1 migration) ───────────────────────────
const REMOVE = [
  ...['admin', 'hr', 'it_support', 'manager', 'member'].flatMap((r) => [
    `config.roles.${r}.label`,
    `config.roles.${r}.description`,
  ]),
  ...['Admin', 'Manager', 'Member', 'HR', 'IT'].flatMap((r) => [
    `team.wizard.role${r}`,
    `team.wizard.role${r}Desc`,
  ]),
]

// ── new keys ────────────────────────────────────────────────────────────────
/** @type {Record<string, [string, string, string, string]>} key → [de, en, fr, it] */
const ADD = {
  'profil.tabs.permissions': ['Berechtigungen', 'Permissions', 'Autorisations', 'Autorizzazioni'],

  // roles
  'rbac.roles.admin.label': ['Vollzugriff (Admin)', 'Full Access (Admin)', 'Accès complet (Admin)', 'Accesso completo (Admin)'],
  'rbac.roles.admin.description': [
    'Alle Module, alle Funktionen — Inhaber- und Administratorrechte',
    'Every module and function — owner/administrator rights',
    'Tous les modules et fonctions — droits propriétaire/administrateur',
    'Tutti i moduli e le funzioni — diritti di proprietario/amministratore',
  ],
  'rbac.roles.it_admin.label': ['IT-Admin', 'IT Admin', 'Admin IT', 'Admin IT'],
  'rbac.roles.it_admin.description': [
    'Technische Verwaltung: Rollen, System, Integrationen — ohne HR- und Gehaltsdaten',
    'Technical administration: roles, system, integrations — no HR or salary data',
    'Administration technique : rôles, système, intégrations — sans données RH ni salaires',
    'Amministrazione tecnica: ruoli, sistema, integrazioni — senza dati HR e stipendi',
  ],
  'rbac.roles.hr_admin.label': ['HR-Admin', 'HR Admin', 'Admin RH', 'Admin HR'],
  'rbac.roles.hr_admin.description': [
    'Mitarbeiterverwaltung inkl. geschützter Personaldaten, weist Rollen zu',
    'People management incl. protected HR data, assigns roles',
    'Gestion du personnel incl. données RH protégées, attribue les rôles',
    'Gestione del personale incl. dati HR protetti, assegna i ruoli',
  ],
  'rbac.roles.manager.label': ['Teamleiter', 'Team Lead', "Chef d'équipe", 'Caposquadra'],
  'rbac.roles.manager.description': [
    'Führt Projekte und Team, genehmigt Anträge — keine Systemverwaltung',
    'Leads projects and people, approves requests — no system administration',
    "Dirige projets et équipe, approuve les demandes — pas d'administration système",
    'Guida progetti e team, approva le richieste — nessuna amministrazione di sistema',
  ],
  'rbac.roles.member.label': ['Mitarbeiter', 'Employee', 'Collaborateur', 'Collaboratore'],
  'rbac.roles.member.description': [
    'Tägliche Arbeit in allen Arbeitsmodulen, bearbeitet Eigenes',
    'Day-to-day work across the work modules, edits own items',
    'Travail quotidien dans les modules, modifie ses propres éléments',
    'Lavoro quotidiano nei moduli, modifica i propri elementi',
  ],
  'rbac.roles.readonly.label': ['Nur Lesen', 'Read Only', 'Lecture seule', 'Sola lettura'],
  'rbac.roles.readonly.description': [
    'Sieht alles Relevante, ändert nichts — z. B. Steuerberater oder Audit',
    'Sees everything relevant, changes nothing — e.g. tax advisor or audit',
    'Voit tout ce qui est pertinent, ne modifie rien — p. ex. fiduciaire ou audit',
    'Vede tutto il necessario, non modifica nulla — ad es. commercialista o audit',
  ],
  'rbac.roles.extern.label': ['Aushilfe / Extern', 'Temp / External', 'Auxiliaire / Externe', 'Ausiliario / Esterno'],
  'rbac.roles.extern.description': [
    'Nur zugewiesene Aufgaben und freigegebene Dokumente',
    'Assigned tasks and shared documents only',
    'Uniquement les tâches attribuées et les documents partagés',
    'Solo attività assegnate e documenti condivisi',
  ],

  // scopes
  'rbac.scope.own': ['Eigene', 'Own', 'Propres', 'Propri'],
  'rbac.scope.team': ['Team', 'Team', 'Équipe', 'Team'],
  'rbac.scope.all': ['Alle', 'All', 'Tous', 'Tutti'],

  // effective-rights view
  'rbac.effective.title': ['Effektive Rechte', 'Effective permissions', 'Droits effectifs', 'Diritti effettivi'],
  'rbac.effective.intro': [
    'Zeigt, was dieses Konto in Cosmi darf — zusammengeführt aus allen zugewiesenen Rollen.',
    'Shows what this account may do in Cosmi — merged from all assigned roles.',
    'Montre ce que ce compte peut faire dans Cosmi — fusionné à partir de tous les rôles attribués.',
    'Mostra cosa può fare questo account in Cosmi — unione di tutti i ruoli assegnati.',
  ],
  'rbac.effective.filterPlaceholder': ['Rechte filtern …', 'Filter permissions…', 'Filtrer les droits…', 'Filtra i diritti…'],
  'rbac.effective.noMatches': ['Keine Treffer', 'No matches', 'Aucun résultat', 'Nessun risultato'],
  'rbac.effective.moduleVisible': ['Modul sichtbar', 'Module visible', 'Module visible', 'Modulo visibile'],
  'rbac.effective.grantCount': [
    '{count, plural, one {# Berechtigung} other {# Berechtigungen}}',
    '{count, plural, one {# permission} other {# permissions}}',
    '{count, plural, one {# autorisation} other {# autorisations}}',
    '{count, plural, one {# autorizzazione} other {# autorizzazioni}}',
  ],

  // modules
  'rbac.module.dashboard': ['Dashboard', 'Dashboard', 'Tableau de bord', 'Dashboard'],
  'rbac.module.work': ['Projekte & Aufgaben', 'Projects & Tasks', 'Projets et tâches', 'Progetti e attività'],
  'rbac.module.kommunikation': ['Kommunikation', 'Communication', 'Communication', 'Comunicazione'],
  'rbac.module.crm': ['Kontakte', 'Contacts', 'Contacts', 'Contatti'],
  'rbac.module.team': ['Team', 'Team', 'Équipe', 'Team'],
  'rbac.module.video': ['Meetings', 'Meetings', 'Réunions', 'Riunioni'],
  'rbac.module.kalender': ['Kalender', 'Calendar', 'Calendrier', 'Calendario'],
  'rbac.module.zeiterfassung': ['Zeiterfassung', 'Time Tracking', 'Saisie des temps', 'Rilevamento tempi'],
  'rbac.module.documents': ['Dokumente', 'Documents', 'Documents', 'Documenti'],
  'rbac.module.wiki': ['Wiki', 'Wiki', 'Wiki', 'Wiki'],
  'rbac.module.mail': ['E-Mail', 'Mail', 'E-mail', 'E-mail'],
  'rbac.module.finance': ['Buchhaltung', 'Finance', 'Comptabilité', 'Contabilità'],
  'rbac.module.infrastructure': ['Infrastruktur', 'Infrastructure', 'Infrastructure', 'Infrastruttura'],
  'rbac.module.inventar': ['Inventar', 'Inventory', 'Inventaire', 'Inventario'],
  'rbac.module.schichten': ['Schichtplanung', 'Shift Planning', 'Planification des équipes', 'Pianificazione turni'],
  'rbac.module.einkauf': ['Einkauf', 'Purchasing', 'Achats', 'Acquisti'],
  'rbac.module.helpdesk': ['Helpdesk', 'Helpdesk', 'Assistance', 'Helpdesk'],
  'rbac.module.fuhrpark': ['Fuhrpark', 'Fleet', 'Parc automobile', 'Parco veicoli'],
  'rbac.module.produktion': ['Produktion', 'Production', 'Production', 'Produzione'],
  'rbac.module.berichte': ['Berichte', 'Reports', 'Rapports', 'Report'],
  'rbac.module.vertraege': ['Verträge', 'Contracts', 'Contrats', 'Contratti'],
  'rbac.module.formulare': ['Formulare', 'Forms', 'Formulaires', 'Moduli'],
  'rbac.module.vermietung': ['Vermietung', 'Rentals', 'Location', 'Noleggio'],
  'rbac.module.rapporte': ['Rapporte', 'Work Reports', 'Rapports de travail', 'Rapporti di lavoro'],
  'rbac.module.dialer': ['Dialer', 'Dialer', 'Dialer', 'Dialer'],
  'rbac.module.automatisierung': ['Automatisierung', 'Automation', 'Automatisation', 'Automazione'],
  'rbac.module.notifications': ['Benachrichtigungen', 'Notifications', 'Notifications', 'Notifiche'],
  'rbac.module.settings': ['Einstellungen', 'Settings', 'Paramètres', 'Impostazioni'],
  'rbac.module.admin': ['Administration', 'Administration', 'Administration', 'Amministrazione'],
  'rbac.module.security': ['Sicherheit', 'Security', 'Sécurité', 'Sicurezza'],

  // subjects
  'rbac.subject.task': ['Aufgaben', 'Tasks', 'Tâches', 'Attività'],
  'rbac.subject.project': ['Projekte', 'Projects', 'Projets', 'Progetti'],
  'rbac.subject.time': ['Zeitbuchung', 'Time logging', 'Saisie du temps', 'Registrazione tempi'],
  'rbac.subject.board': ['Board', 'Board', 'Tableau', 'Bacheca'],
  'rbac.subject.file': ['Dateien', 'Files', 'Fichiers', 'File'],
  'rbac.subject.share': ['Freigaben', 'Shares', 'Partages', 'Condivisioni'],
  'rbac.subject.share_link': ['Externe Links', 'External links', 'Liens externes', 'Link esterni'],
  'rbac.subject.version': ['Versionen', 'Versions', 'Versions', 'Versioni'],
  'rbac.subject.template': ['Vorlagen', 'Templates', 'Modèles', 'Modelli'],
  'rbac.subject.contact': ['Kontakte', 'Contacts', 'Contacts', 'Contatti'],
  'rbac.subject.deal': ['Deals', 'Deals', 'Affaires', 'Trattative'],
  'rbac.subject.pipeline': ['Pipeline', 'Pipeline', 'Pipeline', 'Pipeline'],
  'rbac.subject.import': ['Import', 'Import', 'Importation', 'Importazione'],
  'rbac.subject.advisory': ['Beratungsprotokolle', 'Advisory records', 'Protocoles de conseil', 'Verbali di consulenza'],
  'rbac.subject.segment': ['Segmente', 'Segments', 'Segments', 'Segmenti'],
  'rbac.subject.invoice': ['Rechnungen', 'Invoices', 'Factures', 'Fatture'],
  'rbac.subject.dunning': ['Mahnwesen', 'Dunning', 'Relances', 'Solleciti'],
  'rbac.subject.quote': ['Angebote', 'Quotes', 'Devis', 'Preventivi'],
  'rbac.subject.amounts': ['Beträge & Umsätze', 'Amounts & revenue', 'Montants et chiffre d’affaires', 'Importi e fatturato'],
  'rbac.subject.export': ['Exporte', 'Exports', 'Exportations', 'Esportazioni'],
  'rbac.subject.incoming': ['Eingangsrechnungen', 'Incoming invoices', 'Factures fournisseurs', 'Fatture passive'],
  'rbac.subject.settings': ['Einstellungen', 'Settings', 'Paramètres', 'Impostazioni'],
  'rbac.subject.employee': ['Mitarbeiter', 'Employees', 'Collaborateurs', 'Collaboratori'],
  'rbac.subject.data_personal': ['Persönliche Daten', 'Personal data', 'Données personnelles', 'Dati personali'],
  'rbac.subject.data_job': ['Vertrags- & Jobdaten', 'Job & contract data', 'Données contractuelles', 'Dati contrattuali'],
  'rbac.subject.salary': ['Gehaltsdaten', 'Salary data', 'Données salariales', 'Dati salariali'],
  'rbac.subject.documents': ['Personaldokumente', 'HR documents', 'Documents RH', 'Documenti HR'],
  'rbac.subject.absence': ['Abwesenheiten', 'Absences', 'Absences', 'Assenze'],
  'rbac.subject.role': ['Rollen', 'Roles', 'Rôles', 'Ruoli'],
  'rbac.subject.training': ['Schulungen', 'Trainings', 'Formations', 'Formazioni'],
  'rbac.subject.payroll': ['Lohnvorbereitung', 'Payroll preparation', 'Préparation des salaires', 'Preparazione paghe'],
  'rbac.subject.corrections': ['Zeitkorrekturen', 'Time corrections', 'Corrections de temps', 'Correzioni orari'],
  'rbac.subject.onboarding': ['Onboarding', 'Onboarding', 'Intégration', 'Onboarding'],
  'rbac.subject.article': ['Artikel', 'Articles', 'Articles', 'Articoli'],
  'rbac.subject.share_token': ['Freigabe-Links', 'Share links', 'Liens de partage', 'Link di condivisione'],
  'rbac.subject.personal': ['Persönliche Einstellungen', 'Personal settings', 'Paramètres personnels', 'Impostazioni personali'],
  'rbac.subject.tenant': ['Firmeneinstellungen', 'Company settings', "Paramètres d'entreprise", 'Impostazioni aziendali'],
  'rbac.subject.user': ['Benutzerkonten', 'User accounts', 'Comptes utilisateurs', 'Account utente'],
  'rbac.subject.license': ['Lizenz & Abrechnung', 'License & billing', 'Licence et facturation', 'Licenza e fatturazione'],
  'rbac.subject.branding': ['Branding', 'Branding', 'Image de marque', 'Branding'],
  'rbac.subject.integrations': ['Integrationen', 'Integrations', 'Intégrations', 'Integrazioni'],
  'rbac.subject.company': ['Firmenprofil', 'Company profile', "Profil d'entreprise", 'Profilo aziendale'],
  'rbac.subject.modules': ['Modulzuteilung', 'Module assignment', 'Attribution des modules', 'Assegnazione moduli'],
  'rbac.subject.it': ['IT-Verwaltung', 'IT administration', 'Administration IT', 'Amministrazione IT'],
  'rbac.subject.ai': ['KI-Einstellungen', 'AI settings', 'Paramètres IA', 'Impostazioni IA'],
  'rbac.subject.impersonate': ['Als Benutzer anzeigen', 'View as user', "Voir en tant qu'utilisateur", 'Visualizza come utente'],
  'rbac.subject.audit': ['Audit-Log', 'Audit log', "Journal d'audit", 'Registro di audit'],
  'rbac.subject.policy': ['Sicherheitsrichtlinien', 'Security policies', 'Politiques de sécurité', 'Criteri di sicurezza'],
  'rbac.subject.gdpr': ['DSGVO-Aktionen', 'GDPR actions', 'Actions RGPD', 'Azioni GDPR'],

  // actions
  'rbac.action.view': ['Sehen', 'View', 'Voir', 'Vedere'],
  'rbac.action.read': ['Lesen', 'Read', 'Lire', 'Leggere'],
  'rbac.action.create': ['Erstellen', 'Create', 'Créer', 'Creare'],
  'rbac.action.edit': ['Bearbeiten', 'Edit', 'Modifier', 'Modificare'],
  'rbac.action.delete': ['Löschen', 'Delete', 'Supprimer', 'Eliminare'],
  'rbac.action.export': ['Exportieren', 'Export', 'Exporter', 'Esportare'],
  'rbac.action.download': ['Herunterladen', 'Download', 'Télécharger', 'Scaricare'],
  'rbac.action.upload': ['Hochladen', 'Upload', 'Téléverser', 'Caricare'],
  'rbac.action.send': ['Versenden', 'Send', 'Envoyer', 'Inviare'],
  'rbac.action.run': ['Ausführen', 'Run', 'Exécuter', 'Eseguire'],
  'rbac.action.manage': ['Verwalten', 'Manage', 'Gérer', 'Gestire'],
  'rbac.action.approve': ['Genehmigen', 'Approve', 'Approuver', 'Approvare'],
  'rbac.action.assign': ['Zuweisen', 'Assign', 'Attribuer', 'Assegnare'],
  'rbac.action.invite': ['Einladen', 'Invite', 'Inviter', 'Invitare'],
  'rbac.action.publish': ['Veröffentlichen', 'Publish', 'Publier', 'Pubblicare'],
  'rbac.action.restore': ['Wiederherstellen', 'Restore', 'Restaurer', 'Ripristinare'],
  'rbac.action.log': ['Erfassen', 'Log', 'Saisir', 'Registrare'],
  'rbac.action.comment': ['Kommentieren', 'Comment', 'Commenter', 'Commentare'],
  'rbac.action.be_assigned': ['Zugewiesen bekommen', 'Be assigned', 'Être assigné', 'Essere assegnato'],
  'rbac.action.book': ['Buchen', 'Book', 'Comptabiliser', 'Contabilizzare'],
  'rbac.action.review': ['Prüfen', 'Review', 'Vérifier', 'Verificare'],
  'rbac.action.deactivate': ['Deaktivieren', 'Deactivate', 'Désactiver', 'Disattivare'],
  'rbac.action.execute': ['Ausführen', 'Execute', 'Exécuter', 'Eseguire'],
  'rbac.action.write': ['Schreiben', 'Write', 'Écrire', 'Scrivere'],
}

// ── apply ───────────────────────────────────────────────────────────────────
for (const [li, locale] of LOCALES.entries()) {
  const file = join(MESSAGES_DIR, `${locale}.json`)
  const data = JSON.parse(readFileSync(file, 'utf8'))

  let removed = 0
  for (const key of REMOVE) {
    if (key in data) {
      delete data[key]
      removed++
    }
  }

  // Insert new keys at their alphabetical position relative to existing order.
  const pending = Object.keys(ADD)
    .filter((k) => !(k in data))
    .sort()
  const out = {}
  const existing = Object.keys(data)
  let pi = 0
  for (const key of existing) {
    while (pi < pending.length && pending[pi] < key) {
      out[pending[pi]] = ADD[pending[pi]][li]
      pi++
    }
    out[key] = data[key]
  }
  while (pi < pending.length) {
    out[pending[pi]] = ADD[pending[pi]][li]
    pi++
  }

  writeFileSync(file, JSON.stringify(out, null, 2) + '\n', 'utf8')
  console.log(`${locale}: +${pending.length} keys, -${removed} removed`)
}
console.log('done')
