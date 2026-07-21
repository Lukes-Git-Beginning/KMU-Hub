/**
 * RBAC R-5 i18n pass (audit UI · vendor access · templates · view-as).
 *
 * 1. ADD: 35 keys x4 languages (audit action labels, detail panel, delta
 *    block, retention note, roles-builder sub-tabs) — agent A used them in
 *    code, the JSON entries land here centrally.
 * 2. FIX de: Du-Form statt Sie-Form (Cosmi-Standard), echte Umlaute statt
 *    ASCII-Substitution, zwei Tippfehler.
 * 3. FIX fr/it: template/viewAs blocks were English copies — real FR/IT.
 *
 * Ordering: overrides replace in place, new keys append at the end of the
 * object (same spot the R-5 blocks already live). No global sort.
 *
 * Run once from desktop/: node scripts/i18n-rbac-r5.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))
const msgDir = join(here, '..', 'src', 'renderer', 'src', 'i18n', 'messages')

const ADD = {
  'rbac.audit.action.role_assigned': {
    de: 'Rolle zugewiesen', en: 'Role assigned', fr: 'Rôle attribué', it: 'Ruolo assegnato',
  },
  'rbac.audit.action.role_revoked': {
    de: 'Rolle entzogen', en: 'Role revoked', fr: 'Rôle retiré', it: 'Ruolo revocato',
  },
  'rbac.audit.action.role_definition_created': {
    de: 'Rolle erstellt', en: 'Role created', fr: 'Rôle créé', it: 'Ruolo creato',
  },
  'rbac.audit.action.role_definition_updated': {
    de: 'Rolle bearbeitet', en: 'Role updated', fr: 'Rôle modifié', it: 'Ruolo modificato',
  },
  'rbac.audit.action.role_definition_deleted': {
    de: 'Rolle gelöscht', en: 'Role deleted', fr: 'Rôle supprimé', it: 'Ruolo eliminato',
  },
  'rbac.audit.action.user_invited': {
    de: 'Benutzer eingeladen', en: 'User invited', fr: 'Utilisateur invité', it: 'Utente invitato',
  },
  'rbac.audit.action.user_deactivated': {
    de: 'Benutzer deaktiviert', en: 'User deactivated', fr: 'Utilisateur désactivé', it: 'Utente disattivato',
  },
  'rbac.audit.action.user_reactivated': {
    de: 'Benutzer reaktiviert', en: 'User reactivated', fr: 'Utilisateur réactivé', it: 'Utente riattivato',
  },
  'rbac.audit.action.user_offboarded': {
    de: 'Benutzer-Austritt durchgeführt', en: 'User offboarded', fr: 'Départ utilisateur exécuté', it: 'Uscita utente eseguita',
  },
  'rbac.audit.action.user_view_as': {
    de: 'Als Benutzer angezeigt', en: 'Viewed as user', fr: 'Affiché en tant qu’utilisateur', it: 'Visualizzato come utente',
  },
  'rbac.audit.action.vendor_access_requested': {
    de: 'Anbieter-Zugang beantragt', en: 'Vendor access requested', fr: 'Accès fournisseur demandé', it: 'Accesso fornitore richiesto',
  },
  'rbac.audit.action.vendor_access_approved': {
    de: 'Anbieter-Zugang genehmigt', en: 'Vendor access approved', fr: 'Accès fournisseur approuvé', it: 'Accesso fornitore approvato',
  },
  'rbac.audit.action.vendor_access_declined': {
    de: 'Anbieter-Zugang abgelehnt', en: 'Vendor access declined', fr: 'Accès fournisseur refusé', it: 'Accesso fornitore rifiutato',
  },
  'rbac.audit.action.vendor_access_counter_proposed': {
    de: 'Terminvorschlag für Anbieter-Zugang gesendet', en: 'Vendor access date counter-proposed', fr: 'Contre-proposition de date pour l’accès fournisseur', it: 'Controproposta di data per l’accesso fornitore',
  },
  'rbac.audit.action.vendor_access_granted': {
    de: 'Anbieter-Zugang erteilt', en: 'Vendor access granted', fr: 'Accès fournisseur accordé', it: 'Accesso fornitore concesso',
  },
  'rbac.audit.action.vendor_access_revoked': {
    de: 'Anbieter-Zugang entzogen', en: 'Vendor access revoked', fr: 'Accès fournisseur révoqué', it: 'Accesso fornitore revocato',
  },
  'rbac.audit.action.vendor_access_expired': {
    de: 'Anbieter-Zugang abgelaufen', en: 'Vendor access expired', fr: 'Accès fournisseur expiré', it: 'Accesso fornitore scaduto',
  },
  'rbac.audit.action.vendor_access_completed': {
    de: 'Anbieter-Zugang abgeschlossen', en: 'Vendor access completed', fr: 'Accès fournisseur clôturé', it: 'Accesso fornitore concluso',
  },
  'rbac.audit.action.permission_override_set': {
    de: 'Berechtigungs-Ausnahme gesetzt', en: 'Permission override set', fr: 'Exception de permission définie', it: 'Eccezione di permesso impostata',
  },
  'rbac.audit.action.permission_override_removed': {
    de: 'Berechtigungs-Ausnahme entfernt', en: 'Permission override removed', fr: 'Exception de permission supprimée', it: 'Eccezione di permesso rimossa',
  },
  'rbac.audit.action.setting_changed': {
    de: 'Einstellung geändert', en: 'Setting changed', fr: 'Paramètre modifié', it: 'Impostazione modificata',
  },
  'rbac.audit.detail.actor': { de: 'Akteur', en: 'Actor', fr: 'Acteur', it: 'Autore' },
  'rbac.audit.detail.target': { de: 'Ziel', en: 'Target', fr: 'Cible', it: 'Destinazione' },
  'rbac.audit.detail.event': { de: 'Ereignis', en: 'Event', fr: 'Événement', it: 'Evento' },
  'rbac.audit.detail.ip': { de: 'IP-Adresse', en: 'IP address', fr: 'Adresse IP', it: 'Indirizzo IP' },
  'rbac.audit.detail.userAgent': { de: 'User-Agent', en: 'User agent', fr: 'Agent utilisateur', it: 'User agent' },
  'rbac.audit.detail.sequenceNum': { de: 'Sequenznummer', en: 'Sequence number', fr: 'Numéro de séquence', it: 'Numero di sequenza' },
  'rbac.audit.detail.additionalContext': { de: 'Weitere Details', en: 'Additional details', fr: 'Détails supplémentaires', it: 'Ulteriori dettagli' },
  'rbac.audit.deltaTitle': { de: 'Änderung', en: 'Change', fr: 'Modification', it: 'Modifica' },
  'rbac.audit.deltaBefore': { de: 'Vorher', en: 'Before', fr: 'Avant', it: 'Prima' },
  'rbac.audit.deltaAfter': { de: 'Nachher', en: 'After', fr: 'Après', it: 'Dopo' },
  'rbac.audit.retentionNote': {
    de: 'Einträge werden 24 Monate aufbewahrt und können nicht verändert werden.',
    en: 'Entries are retained for 24 months and cannot be modified.',
    fr: 'Les entrées sont conservées 24 mois et ne peuvent pas être modifiées.',
    it: 'Le voci vengono conservate per 24 mesi e non possono essere modificate.',
  },
  'rbac.audit.protocolEmpty': {
    de: 'Noch keine Rechteänderungen protokolliert',
    en: 'No permission changes recorded yet',
    fr: 'Aucune modification de droits enregistrée pour l’instant',
    it: 'Nessuna modifica dei permessi registrata finora',
  },
  'rbac.builder.subTab.overview': { de: 'Übersicht', en: 'Overview', fr: 'Aperçu', it: 'Panoramica' },
  'rbac.builder.subTab.protocol': { de: 'Protokoll', en: 'Audit trail', fr: 'Journal', it: 'Protocollo' },
}

// Du-Form (Cosmi-Standard), echte Umlaute, Tippfehler.
const SET_DE = {
  'rbac.vendorAccess.pageDescription': 'Verwalte zeitlich befristete Zugänge von Zentria auf dein System.',
  'rbac.vendorAccess.sensitiveWarning': 'Diese Anfrage enthält Zugriff auf sensible Personal- oder Lohndaten. Prüfe den Umfang sorgfältig, bevor du genehmigst.',
  'rbac.vendorAccess.approveDialog.description': 'Du genehmigst den Anbieter-Zugang für: {reason}',
  'rbac.vendorAccess.counterDialog.description': 'Schlage Zentria einen alternativen Starttermin vor für: {reason}',
  'rbac.vendorAccess.empty.active.description': 'Zentria hat derzeit keinen aktiven Zugriff auf dein System.',
  'rbac.template.dialogSubtitle': 'Wähle ein vorkonfiguriertes Rollen-Set und passe es an.',
  'rbac.template.setPrompt': 'Für welche Branche soll die Rolle gelten?',
  'rbac.template.backToSets': 'Zurück zu Branchen',
  'rbac.template.backToRoles': 'Zurück zur Rollenauswahl',
  'rbac.template.role.handel_lager.description': 'Bucht Warenbewegungen, führt Inventuren durch und nimmt Bestellungen an. Kein Finanzzugang.',
  'rbac.template.role.handel_lager.highlight.1': 'Inventar vollständig operativ (ohne Stammdaten-Änderung)',
}

// template/viewAs blocks in fr/it were English copies — real translations.
const SET_FR = {
  'rbac.template.tabLabel': 'À partir d’un modèle',
  'rbac.template.dialogTitle': 'Créer un rôle à partir d’un modèle',
  'rbac.template.dialogSubtitle': 'Choisissez un ensemble de rôles préconfiguré et adaptez-le.',
  'rbac.template.setPrompt': 'Pour quel secteur ce rôle est-il destiné ?',
  'rbac.template.rolesCount': 'rôles',
  'rbac.template.backToSets': 'Retour aux secteurs',
  'rbac.template.backToRoles': 'Retour à la sélection des rôles',
  'rbac.template.set.handwerk': 'Artisanat & BTP',
  'rbac.template.set.dienstleister': 'Services & IT',
  'rbac.template.set.handel': 'Commerce & Logistique',
  'rbac.template.role.handwerk_buero.label': 'Bureau & gestion des commandes',
  'rbac.template.role.handwerk_buero.description': 'Accès complet aux commandes, CRM, finances et achats. Pour le personnel de bureau qui coordonne l’activité quotidienne.',
  'rbac.template.role.handwerk_buero.highlight.1': 'Accès complet au CRM, aux finances et aux achats',
  'rbac.template.role.handwerk_buero.highlight.2': 'Temps de travail : vue d’équipe (sans approbation)',
  'rbac.template.role.handwerk_buero.highlight.3': 'Pas de données RH, pas d’accès admin',
  'rbac.template.role.handwerk_bauleiter.label': 'Chef de chantier / chef de projet',
  'rbac.template.role.handwerk_bauleiter.description': 'Dirige les projets et les équipes sur le chantier. Approuve les rapports et les corrections de temps. Aucun accès aux finances ni aux achats.',
  'rbac.template.role.handwerk_bauleiter.highlight.1': 'Accès complet aux projets, rapports et plannings',
  'rbac.template.role.handwerk_bauleiter.highlight.2': 'Temps de travail incl. approbation',
  'rbac.template.role.handwerk_bauleiter.highlight.3': 'Aucune donnée financière ou d’achat',
  'rbac.template.role.handwerk_monteur.label': 'Monteur / compagnon',
  'rbac.template.role.handwerk_monteur.description': 'Saisit rapports et temps, consulte le planning et peut demander un échange. Enregistre les mouvements de stock. Aucun accès financier.',
  'rbac.template.role.handwerk_monteur.highlight.1': 'Créer des rapports et modifier les siens',
  'rbac.template.role.handwerk_monteur.highlight.2': 'Demander un échange de poste',
  'rbac.template.role.handwerk_monteur.highlight.3': 'Aucune donnée CRM, financière ou RH',
  'rbac.template.role.handwerk_azubi.label': 'Apprenti',
  'rbac.template.role.handwerk_azubi.description': 'Accès minimal : saisie des temps, formulaires, lecture des rapports, consultation du wiki et des documents.',
  'rbac.template.role.handwerk_azubi.highlight.1': 'Remplir des formulaires et lire les rapports',
  'rbac.template.role.handwerk_azubi.highlight.2': 'Uniquement ses propres données',
  'rbac.template.role.handwerk_azubi.highlight.3': 'Pas de stock, pas de finances, pas de CRM',
  'rbac.template.role.dl_projektleiter.label': 'Chef de projet / senior',
  'rbac.template.role.dl_projektleiter.description': 'Dirige les projets et la relation client. Consulte les données financières en lecture. Pas de salaires RH, pas d’admin.',
  'rbac.template.role.dl_projektleiter.highlight.1': 'Accès complet au CRM, aux projets, au wiki et aux contrats',
  'rbac.template.role.dl_projektleiter.highlight.2': 'Finances : lecture seule (ni export ni envoi)',
  'rbac.template.role.dl_projektleiter.highlight.3': 'Pas de données salariales, pas d’admin',
  'rbac.template.role.dl_consultant.label': 'Consultant / collaborateur',
  'rbac.template.role.dl_consultant.description': 'Travaille sur les projets et alimente le wiki. Consulte les contacts en lecture. Aucun accès financier.',
  'rbac.template.role.dl_consultant.highlight.1': 'Projets et tâches personnelles',
  'rbac.template.role.dl_consultant.highlight.2': 'Créer et modifier le wiki',
  'rbac.template.role.dl_consultant.highlight.3': 'Pas de finances, pas de rapports',
  'rbac.template.role.dl_backoffice.label': 'Back-office / administration',
  'rbac.template.role.dl_backoffice.description': 'Gère factures, achats et contrats. Voit les temps de l’équipe. Pas d’export financier, pas de salaires RH.',
  'rbac.template.role.dl_backoffice.highlight.1': 'Finances opérationnelles (créer/comptabiliser des factures)',
  'rbac.template.role.dl_backoffice.highlight.2': 'Vue d’équipe des temps de travail',
  'rbac.template.role.dl_backoffice.highlight.3': 'Pas de pipeline commercial, pas de données salariales',
  'rbac.template.role.dl_freelancer.label': 'Freelance / externe',
  'rbac.template.role.dl_freelancer.description': 'Traite les tâches attribuées, remplit des formulaires et saisit ses temps. Aucun accès au CRM, aux finances ni aux données internes.',
  'rbac.template.role.dl_freelancer.highlight.1': 'Uniquement les tâches attribuées',
  'rbac.template.role.dl_freelancer.highlight.2': 'Formulaires et saisie des temps',
  'rbac.template.role.dl_freelancer.highlight.3': 'Pas de CRM, pas de finances, pas de wiki',
  'rbac.template.role.handel_filialleiter.label': 'Responsable de magasin',
  'rbac.template.role.handel_filialleiter.description': 'Dirige le magasin : CRM, plannings, stock et temps de travail. Finances en lecture. Pas d’achats (prix d’achat). Remarque : pas de filtre par site en v1.0.',
  'rbac.template.role.handel_filialleiter.highlight.1': 'Gérer entièrement plannings et stock',
  'rbac.template.role.handel_filialleiter.highlight.2': 'Finances : lecture seule',
  'rbac.template.role.handel_filialleiter.highlight.3': 'Pas d’achats (prix d’achat protégés)',
  'rbac.template.role.handel_verkauf.label': 'Vente / caisse',
  'rbac.template.role.handel_verkauf.description': 'Personnel de vente et de caisse : temps de travail, planning, création de contacts et consultation des articles.',
  'rbac.template.role.handel_verkauf.highlight.1': 'Demander un échange de poste',
  'rbac.template.role.handel_verkauf.highlight.2': 'Créer des contacts',
  'rbac.template.role.handel_verkauf.highlight.3': 'Pas de finances, pas d’achats',
  'rbac.template.role.handel_lager.label': 'Entrepôt / logistique',
  'rbac.template.role.handel_lager.description': 'Enregistre les mouvements de stock, réalise les inventaires et réceptionne les commandes. Aucun accès financier.',
  'rbac.template.role.handel_lager.highlight.1': 'Stock entièrement opérationnel (sans modification des données de base)',
  'rbac.template.role.handel_lager.highlight.2': 'Enregistrer les réceptions de marchandises',
  'rbac.template.role.handel_lager.highlight.3': 'Pas de finances, pas de CRM',
  'rbac.template.role.handel_einkauf.label': 'Achats',
  'rbac.template.role.handel_einkauf.description': 'Gère fournisseurs, commandes et contrats. Entretient les données de base du stock. Aucun accès financier, aucune donnée RH.',
  'rbac.template.role.handel_einkauf.highlight.1': 'Achats et stock complets',
  'rbac.template.role.handel_einkauf.highlight.2': 'Gérer fournisseurs et contrats',
  'rbac.template.role.handel_einkauf.highlight.3': 'Pas de finances, pas de composeur',
  'rbac.viewAs.action': 'Afficher en tant qu’utilisateur',
  'rbac.viewAs.banner': 'Vous voyez Cosmi en tant que {name}',
  'rbac.viewAs.exit': 'Quitter',
  'rbac.viewAs.auditLabel': 'Affiché en tant qu’utilisateur',
}

const SET_IT = {
  'rbac.template.tabLabel': 'Da modello',
  'rbac.template.dialogTitle': 'Crea ruolo da modello',
  'rbac.template.dialogSubtitle': 'Scegli un set di ruoli preconfigurato e adattalo.',
  'rbac.template.setPrompt': 'Per quale settore è destinato il ruolo?',
  'rbac.template.rolesCount': 'ruoli',
  'rbac.template.backToSets': 'Torna ai settori',
  'rbac.template.backToRoles': 'Torna alla selezione dei ruoli',
  'rbac.template.set.handwerk': 'Artigianato & Edilizia',
  'rbac.template.set.dienstleister': 'Servizi & IT',
  'rbac.template.set.handel': 'Commercio & Logistica',
  'rbac.template.role.handwerk_buero.label': 'Ufficio & gestione ordini',
  'rbac.template.role.handwerk_buero.description': 'Accesso completo a ordini, CRM, finanze e acquisti. Per il personale d’ufficio che coordina l’attività quotidiana.',
  'rbac.template.role.handwerk_buero.highlight.1': 'Accesso completo a CRM, finanze e acquisti',
  'rbac.template.role.handwerk_buero.highlight.2': 'Ore di lavoro: vista team (senza approvazione)',
  'rbac.template.role.handwerk_buero.highlight.3': 'Nessun dato HR, nessun accesso admin',
  'rbac.template.role.handwerk_bauleiter.label': 'Capocantiere / capo progetto',
  'rbac.template.role.handwerk_bauleiter.description': 'Dirige progetti e squadre in cantiere. Approva rapporti e correzioni orarie. Nessun accesso a finanze o acquisti.',
  'rbac.template.role.handwerk_bauleiter.highlight.1': 'Accesso completo a progetti, rapporti e turni',
  'rbac.template.role.handwerk_bauleiter.highlight.2': 'Ore di lavoro incl. approvazione',
  'rbac.template.role.handwerk_bauleiter.highlight.3': 'Nessun dato finanziario o di acquisto',
  'rbac.template.role.handwerk_monteur.label': 'Montatore / operaio specializzato',
  'rbac.template.role.handwerk_monteur.description': 'Registra rapporti e ore, consulta il piano turni e può richiedere scambi. Registra movimenti di magazzino. Nessun accesso finanziario.',
  'rbac.template.role.handwerk_monteur.highlight.1': 'Creare rapporti e modificare i propri',
  'rbac.template.role.handwerk_monteur.highlight.2': 'Richiedere scambio turno',
  'rbac.template.role.handwerk_monteur.highlight.3': 'Nessun dato CRM, finanziario o HR',
  'rbac.template.role.handwerk_azubi.label': 'Apprendista',
  'rbac.template.role.handwerk_azubi.description': 'Accesso minimo: registrazione ore, moduli, lettura rapporti, consultazione wiki e documenti.',
  'rbac.template.role.handwerk_azubi.highlight.1': 'Compilare moduli e leggere rapporti',
  'rbac.template.role.handwerk_azubi.highlight.2': 'Solo i propri dati',
  'rbac.template.role.handwerk_azubi.highlight.3': 'Niente magazzino, finanze o CRM',
  'rbac.template.role.dl_projektleiter.label': 'Capo progetto / senior',
  'rbac.template.role.dl_projektleiter.description': 'Guida progetti e rapporti con i clienti. Vede i dati finanziari in sola lettura. Nessuno stipendio HR, nessun admin.',
  'rbac.template.role.dl_projektleiter.highlight.1': 'Accesso completo a CRM, progetti, wiki e contratti',
  'rbac.template.role.dl_projektleiter.highlight.2': 'Finanze: sola lettura (senza export né invio)',
  'rbac.template.role.dl_projektleiter.highlight.3': 'Nessun dato salariale, nessun admin',
  'rbac.template.role.dl_consultant.label': 'Consulente / collaboratore',
  'rbac.template.role.dl_consultant.description': 'Lavora sui progetti e cura il wiki. Vede i contatti in sola lettura. Nessun accesso finanziario.',
  'rbac.template.role.dl_consultant.highlight.1': 'Progetti e attività personali',
  'rbac.template.role.dl_consultant.highlight.2': 'Creare e modificare il wiki',
  'rbac.template.role.dl_consultant.highlight.3': 'Niente finanze, niente report',
  'rbac.template.role.dl_backoffice.label': 'Back office / amministrazione',
  'rbac.template.role.dl_backoffice.description': 'Gestisce fatture, acquisti e contratti. Vede le ore del team. Nessun export finanziario, nessuno stipendio HR.',
  'rbac.template.role.dl_backoffice.highlight.1': 'Finanze operative (creare/contabilizzare fatture)',
  'rbac.template.role.dl_backoffice.highlight.2': 'Vista team delle ore di lavoro',
  'rbac.template.role.dl_backoffice.highlight.3': 'Nessuna pipeline commerciale, nessun dato salariale',
  'rbac.template.role.dl_freelancer.label': 'Freelance / esterno',
  'rbac.template.role.dl_freelancer.description': 'Elabora le attività assegnate, compila moduli e registra le ore. Nessun accesso a CRM, finanze o dati interni.',
  'rbac.template.role.dl_freelancer.highlight.1': 'Solo attività assegnate',
  'rbac.template.role.dl_freelancer.highlight.2': 'Moduli e registrazione ore',
  'rbac.template.role.dl_freelancer.highlight.3': 'Niente CRM, finanze o wiki',
  'rbac.template.role.handel_filialleiter.label': 'Responsabile di filiale',
  'rbac.template.role.handel_filialleiter.description': 'Dirige la filiale: CRM, turni, magazzino e ore di lavoro. Finanze in sola lettura. Nessun acquisto (prezzi d’acquisto). Nota: nessun filtro per sede nella v1.0.',
  'rbac.template.role.handel_filialleiter.highlight.1': 'Gestione completa di turni e magazzino',
  'rbac.template.role.handel_filialleiter.highlight.2': 'Finanze: sola lettura',
  'rbac.template.role.handel_filialleiter.highlight.3': 'Nessun acquisto (prezzi d’acquisto protetti)',
  'rbac.template.role.handel_verkauf.label': 'Vendita / cassa',
  'rbac.template.role.handel_verkauf.description': 'Personale di vendita e cassa: ore di lavoro, piano turni, creazione contatti e consultazione articoli.',
  'rbac.template.role.handel_verkauf.highlight.1': 'Richiedere scambio turno',
  'rbac.template.role.handel_verkauf.highlight.2': 'Creare contatti',
  'rbac.template.role.handel_verkauf.highlight.3': 'Niente finanze, niente acquisti',
  'rbac.template.role.handel_lager.label': 'Magazzino / logistica',
  'rbac.template.role.handel_lager.description': 'Registra movimenti di merce, esegue inventari e riceve gli ordini. Nessun accesso finanziario.',
  'rbac.template.role.handel_lager.highlight.1': 'Magazzino completamente operativo (senza modifica anagrafiche)',
  'rbac.template.role.handel_lager.highlight.2': 'Registrare entrate merce',
  'rbac.template.role.handel_lager.highlight.3': 'Niente finanze, niente CRM',
  'rbac.template.role.handel_einkauf.label': 'Acquisti',
  'rbac.template.role.handel_einkauf.description': 'Gestisce fornitori, ordini e contratti. Cura le anagrafiche di magazzino. Nessun accesso finanziario, nessun dato HR.',
  'rbac.template.role.handel_einkauf.highlight.1': 'Acquisti e magazzino completi',
  'rbac.template.role.handel_einkauf.highlight.2': 'Gestire fornitori e contratti',
  'rbac.template.role.handel_einkauf.highlight.3': 'Niente finanze, niente dialer',
  'rbac.viewAs.action': 'Visualizza come utente',
  'rbac.viewAs.banner': 'Stai vedendo Cosmi come {name}',
  'rbac.viewAs.exit': 'Esci',
  'rbac.viewAs.auditLabel': 'Visualizzato come utente',
}

const OVERRIDES = { de: SET_DE, en: {}, fr: SET_FR, it: SET_IT }

for (const lang of ['de', 'en', 'fr', 'it']) {
  const file = join(msgDir, `${lang}.json`)
  const messages = JSON.parse(readFileSync(file, 'utf8'))

  let replaced = 0
  for (const [key, value] of Object.entries(OVERRIDES[lang])) {
    if (!(key in messages)) throw new Error(`${lang}: override target missing: ${key}`)
    if (messages[key] !== value) replaced += 1
    messages[key] = value
  }

  let added = 0
  for (const [key, byLang] of Object.entries(ADD)) {
    if (key in messages) throw new Error(`${lang}: ADD key already exists: ${key}`)
    messages[key] = byLang[lang]
    added += 1
  }

  writeFileSync(file, JSON.stringify(messages, null, 2) + '\n', 'utf8')
  console.log(`${lang}.json — +${added} added, ${replaced} replaced`)
}
console.log('done')
