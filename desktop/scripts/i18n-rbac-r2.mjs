/**
 * i18n batch for RBAC R-2 (role builder: rbac.builder/editor/compare/preview/
 * assignment namespaces + team.member.roles.title).
 *
 * Inserts new keys alphabetically into the existing flat JSON without
 * reordering existing keys, and removes the dead A-2 matrix entries
 * (admin.roles.* — the legacy RolesAdminHubTab is replaced by the builder)
 * plus the single-role user-detail strings.
 *
 * Run: node scripts/i18n-rbac-r2.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))
const MESSAGES_DIR = join(here, '..', 'src', 'renderer', 'src', 'i18n', 'messages')
const LOCALES = ['de', 'en', 'fr', 'it']

// ── keys to remove (dead after A-2 → builder migration) ─────────────────────
const REMOVE_PREFIXES = ['admin.roles.']
const REMOVE = ['admin.users.detail.roleChanged', 'admin.users.detail.selfRoleLocked']

// ── new keys ────────────────────────────────────────────────────────────────
/** @type {Record<string, [string, string, string, string]>} key → [de, en, fr, it] */
const ADD = {
  'team.member.roles.title': ['Rollen & Zugriff', 'Roles & access', 'Rôles et accès', 'Ruoli e accesso'],

  // ── builder (list) ────────────────────────────────────────────────────────
  'rbac.builder.title': ['Rollen-Baukasten', 'Role builder', 'Atelier de rôles', 'Costruttore di ruoli'],
  'rbac.builder.subtitle': [
    'System-Rollen als Vorlage, eigene Rollen bis ins Detail — was jede Rolle sieht und darf.',
    'System roles as templates, custom roles down to the detail — what each role sees and may do.',
    'Rôles système comme modèles, rôles personnalisés dans le moindre détail — ce que chaque rôle voit et peut faire.',
    'Ruoli di sistema come modelli, ruoli personalizzati fin nei dettagli — cosa vede e può fare ogni ruolo.',
  ],
  'rbac.builder.compare': ['Vergleichen', 'Compare', 'Comparer', 'Confronta'],
  'rbac.builder.newRole': ['Neue Rolle', 'New role', 'Nouveau rôle', 'Nuovo ruolo'],
  'rbac.builder.limitHint': [
    'Maximal {limit} eigene Rollen erreicht — lösche eine Rolle, bevor du eine neue anlegst.',
    'Custom-role limit of {limit} reached — delete a role before creating a new one.',
    'Limite de {limit} rôles personnalisés atteinte — supprimez un rôle avant d’en créer un nouveau.',
    'Limite di {limit} ruoli personalizzati raggiunto — elimina un ruolo prima di crearne uno nuovo.',
  ],
  'rbac.builder.systemRoles': ['System-Rollen', 'System roles', 'Rôles système', 'Ruoli di sistema'],
  'rbac.builder.systemRolesHint': [
    'Unveränderlich — als Vorlage klonbar',
    'Immutable — clone as a template',
    'Immuables — clonables comme modèle',
    'Immutabili — clonabili come modello',
  ],
  'rbac.builder.customRoles': ['Eigene Rollen', 'Custom roles', 'Rôles personnalisés', 'Ruoli personalizzati'],
  'rbac.builder.customRolesHint': ['{count} von {limit}', '{count} of {limit}', '{count} sur {limit}', '{count} di {limit}'],
  'rbac.builder.systemBadge': ['System', 'System', 'Système', 'Sistema'],
  'rbac.builder.basedOnBadge': ['basiert auf {name}', 'based on {name}', 'basé sur {name}', 'basato su {name}'],
  'rbac.builder.deviationCount': [
    '{count, plural, one {# Abweichung} other {# Abweichungen}}',
    '{count, plural, one {# deviation} other {# deviations}}',
    '{count, plural, one {# écart} other {# écarts}}',
    '{count, plural, one {# differenza} other {# differenze}}',
  ],
  'rbac.builder.memberCount': [
    '{count, plural, one {# Konto} other {# Konten}}',
    '{count, plural, one {# account} other {# accounts}}',
    '{count, plural, one {# compte} other {# comptes}}',
    '{count, plural, one {# account} other {# account}}',
  ],
  'rbac.builder.capabilityCount': [
    '{count, plural, one {# Recht} other {# Rechte}}',
    '{count, plural, one {# permission} other {# permissions}}',
    '{count, plural, one {# droit} other {# droits}}',
    '{count, plural, one {# permesso} other {# permessi}}',
  ],
  'rbac.builder.cardMenu': ['Rollen-Aktionen', 'Role actions', 'Actions du rôle', 'Azioni del ruolo'],
  'rbac.builder.view': ['Ansehen', 'View', 'Afficher', 'Visualizza'],
  'rbac.builder.edit': ['Bearbeiten', 'Edit', 'Modifier', 'Modifica'],
  'rbac.builder.clone': ['Klonen', 'Clone', 'Cloner', 'Clona'],
  'rbac.builder.members': ['Mitglieder', 'Members', 'Membres', 'Membri'],
  'rbac.builder.noMembers': [
    'Niemand trägt diese Rolle.',
    'Nobody holds this role.',
    'Personne ne détient ce rôle.',
    'Nessuno ha questo ruolo.',
  ],
  'rbac.builder.delete': ['Löschen', 'Delete', 'Supprimer', 'Elimina'],
  'rbac.builder.deleteTitle': ['Rolle „{name}" löschen?', 'Delete role “{name}”?', 'Supprimer le rôle « {name} » ?', 'Eliminare il ruolo «{name}»?'],
  'rbac.builder.deleteConfirm': [
    'Die Rolle wird dauerhaft entfernt. Das lässt sich nicht rückgängig machen.',
    'The role is removed permanently. This cannot be undone.',
    'Le rôle sera supprimé définitivement. Cette action est irréversible.',
    'Il ruolo verrà rimosso definitivamente. L’operazione non può essere annullata.',
  ],
  'rbac.builder.deleteBlockedMembers': [
    '{count, plural, one {# Konto trägt} other {# Konten tragen}} diese Rolle noch — entferne zuerst die Zuweisungen.',
    '{count, plural, one {# account still holds} other {# accounts still hold}} this role — remove the assignments first.',
    '{count, plural, one {# compte détient} other {# comptes détiennent}} encore ce rôle — retirez d’abord les attributions.',
    '{count, plural, one {# account ha} other {# account hanno}} ancora questo ruolo — rimuovi prima le assegnazioni.',
  ],
  'rbac.builder.deleteAction': ['Endgültig löschen', 'Delete permanently', 'Supprimer définitivement', 'Elimina definitivamente'],
  'rbac.builder.deleteDone': ['Rolle „{name}" gelöscht', 'Role “{name}” deleted', 'Rôle « {name} » supprimé', 'Ruolo «{name}» eliminato'],
  'rbac.builder.noCustomRoles': ['Noch keine eigenen Rollen', 'No custom roles yet', 'Aucun rôle personnalisé pour l’instant', 'Ancora nessun ruolo personalizzato'],
  'rbac.builder.noCustomRolesHint': [
    'Eigene Rollen entstehen immer als Klon einer Vorlage — so startest du nie bei null und übernimmst geprüfte Grundeinstellungen.',
    'Custom roles always start as a clone of a template — you never start from zero and inherit vetted defaults.',
    'Les rôles personnalisés naissent toujours comme clone d’un modèle — vous ne partez jamais de zéro et héritez de réglages éprouvés.',
    'I ruoli personalizzati nascono sempre come clone di un modello — non parti mai da zero ed erediti impostazioni collaudate.',
  ],
  'rbac.builder.cloneFirst': ['Erste Rolle klonen', 'Clone your first role', 'Cloner un premier rôle', 'Clona il primo ruolo'],

  // clone dialog
  'rbac.builder.cloneTitle': ['Rolle klonen', 'Clone role', 'Cloner le rôle', 'Clona ruolo'],
  'rbac.builder.cloneSubtitle': [
    'Die neue Rolle startet mit allen Rechten der Vorlage — Abweichungen stellst du danach im Editor ein.',
    'The new role starts with all permissions of the template — adjust deviations in the editor afterwards.',
    'Le nouveau rôle démarre avec tous les droits du modèle — ajustez les écarts ensuite dans l’éditeur.',
    'Il nuovo ruolo parte con tutti i permessi del modello — regola le differenze dopo nell’editor.',
  ],
  'rbac.builder.cloneBase': ['Vorlage', 'Template', 'Modèle', 'Modello'],
  'rbac.builder.cloneBasePlaceholder': ['Vorlage wählen…', 'Choose a template…', 'Choisir un modèle…', 'Scegli un modello…'],
  'rbac.builder.cloneBaseHint': [
    'Rechte werden von dieser Rolle übernommen.',
    'Permissions are copied from this role.',
    'Les droits sont repris de ce rôle.',
    'I permessi vengono copiati da questo ruolo.',
  ],
  'rbac.builder.cloneName': ['Name', 'Name', 'Nom', 'Nome'],
  'rbac.builder.cloneNamePlaceholder': ['z. B. Lager & Logistik', 'e.g. Warehouse & logistics', 'p. ex. Entrepôt & logistique', 'ad es. Magazzino e logistica'],
  'rbac.builder.similarNameWarning': [
    'Ähnlich zu „{name}" — besser bestehende Rolle anpassen statt Duplikat anlegen.',
    'Similar to “{name}” — consider adjusting the existing role instead of duplicating.',
    'Proche de « {name} » — mieux vaut ajuster le rôle existant plutôt que de le dupliquer.',
    'Simile a «{name}» — meglio adattare il ruolo esistente invece di duplicarlo.',
  ],
  'rbac.builder.cloneDescription': ['Beschreibung', 'Description', 'Description', 'Descrizione'],
  'rbac.builder.cloneDescriptionPlaceholder': [
    'Wofür ist diese Rolle gedacht?',
    'What is this role for?',
    'À quoi sert ce rôle ?',
    'A cosa serve questo ruolo?',
  ],
  'rbac.builder.cloneColor': ['Farbe', 'Color', 'Couleur', 'Colore'],
  'rbac.builder.cloneAction': ['Rolle erstellen', 'Create role', 'Créer le rôle', 'Crea ruolo'],
  'rbac.builder.cloneDone': ['Rolle „{name}" erstellt', 'Role “{name}” created', 'Rôle « {name} » créé', 'Ruolo «{name}» creato'],

  // errors (code → message, mirrors the mock/BE error contract)
  'rbac.builder.errors.generic': [
    'Aktion fehlgeschlagen. Bitte erneut versuchen.',
    'Action failed. Please try again.',
    'Échec de l’action. Veuillez réessayer.',
    'Azione non riuscita. Riprova.',
  ],
  'rbac.builder.errors.preset_immutable': [
    'System-Rollen sind unveränderlich — klone die Rolle, um sie anzupassen.',
    'System roles are immutable — clone the role to adjust it.',
    'Les rôles système sont immuables — clonez le rôle pour l’ajuster.',
    'I ruoli di sistema sono immutabili — clona il ruolo per modificarlo.',
  ],
  'rbac.builder.errors.role_limit_reached': [
    'Limit für eigene Rollen erreicht.',
    'Custom-role limit reached.',
    'Limite de rôles personnalisés atteinte.',
    'Limite di ruoli personalizzati raggiunto.',
  ],
  'rbac.builder.errors.role_name_exists': [
    'Eine Rolle mit diesem Namen existiert bereits.',
    'A role with this name already exists.',
    'Un rôle portant ce nom existe déjà.',
    'Esiste già un ruolo con questo nome.',
  ],
  'rbac.builder.errors.role_has_members': [
    'Die Rolle wird noch von Konten getragen — entferne zuerst die Zuweisungen.',
    'Accounts still hold this role — remove the assignments first.',
    'Des comptes détiennent encore ce rôle — retirez d’abord les attributions.',
    'Alcuni account hanno ancora questo ruolo — rimuovi prima le assegnazioni.',
  ],
  'rbac.builder.errors.last_admin': [
    'Der letzte Vollzugriff kann nicht entfernt werden — mindestens ein Admin muss bleiben.',
    'The last full-access role cannot be removed — at least one admin must remain.',
    'Le dernier accès complet ne peut pas être retiré — au moins un admin doit rester.',
    'L’ultimo accesso completo non può essere rimosso — deve rimanere almeno un admin.',
  ],

  // ── editor ────────────────────────────────────────────────────────────────
  'rbac.editor.notFound': ['Rolle nicht gefunden.', 'Role not found.', 'Rôle introuvable.', 'Ruolo non trovato.'],
  'rbac.editor.presetReadonly': [
    'System-Rolle — hier nur ansehen. Zum Anpassen klonst du die Rolle als eigene Vorlage.',
    'System role — view only here. Clone it as your own template to adjust it.',
    'Rôle système — consultation uniquement. Clonez-le comme modèle personnel pour l’ajuster.',
    'Ruolo di sistema — solo visualizzazione. Clonalo come modello personale per modificarlo.',
  ],
  'rbac.editor.editMeta': ['Details', 'Details', 'Détails', 'Dettagli'],
  'rbac.editor.metaTitle': ['Rollen-Details bearbeiten', 'Edit role details', 'Modifier les détails du rôle', 'Modifica dettagli del ruolo'],
  'rbac.editor.metaSaved': ['Rollen-Details gespeichert', 'Role details saved', 'Détails du rôle enregistrés', 'Dettagli del ruolo salvati'],
  'rbac.editor.searchPlaceholder': ['Modul oder Recht suchen…', 'Search module or permission…', 'Rechercher un module ou un droit…', 'Cerca modulo o permesso…'],
  'rbac.editor.moduleTree': ['Module', 'Modules', 'Modules', 'Moduli'],
  'rbac.editor.category.standard': ['Standard', 'Standard', 'Standard', 'Standard'],
  'rbac.editor.category.industry': ['Branchen', 'Industry', 'Métiers', 'Settori'],
  'rbac.editor.category.verwaltung': ['Verwaltung', 'Administration', 'Administration', 'Amministrazione'],
  'rbac.editor.bulkMenu': ['Massenaktion für {category}', 'Bulk action for {category}', 'Action groupée pour {category}', 'Azione di massa per {category}'],
  'rbac.editor.bulkShowAll': ['Alle sichtbar schalten', 'Make all visible', 'Tout rendre visible', 'Rendi tutto visibile'],
  'rbac.editor.bulkHideAll': ['Alle ausblenden', 'Hide all', 'Tout masquer', 'Nascondi tutto'],
  'rbac.editor.moduleActive': ['{active} von {total} Schaltern aktiv', '{active} of {total} switches active', '{active} sur {total} interrupteurs actifs', '{active} su {total} interruttori attivi'],
  'rbac.editor.resetModule': ['Auf Vorlage zurücksetzen', 'Reset to template', 'Réinitialiser au modèle', 'Ripristina al modello'],
  'rbac.editor.visibilityHint': [
    'Steuert, ob das Modul in Navigation und Suche erscheint.',
    'Controls whether the module appears in navigation and search.',
    'Détermine si le module apparaît dans la navigation et la recherche.',
    'Determina se il modulo appare nella navigazione e nella ricerca.',
  ],
  'rbac.editor.hiddenModuleHint': [
    'Modul ist unsichtbar — die Rechte darunter bleiben gespeichert und wirken erst wieder mit der Sichtbarkeit.',
    'Module is hidden — the permissions below stay saved and only take effect once visibility is back on.',
    'Module masqué — les droits ci-dessous restent enregistrés et ne s’appliquent qu’une fois la visibilité rétablie.',
    'Modulo nascosto — i permessi sottostanti restano salvati e hanno effetto solo quando la visibilità è riattivata.',
  ],
  'rbac.editor.noFineCapabilities': [
    'Für dieses Modul ist bisher nur die Sichtbarkeit steuerbar — Fein-Schalter folgen mit dem Modul-Ausbau.',
    'Only visibility is configurable for this module so far — fine switches follow with the module rollout.',
    'Pour ce module, seule la visibilité est réglable pour l’instant — les interrupteurs fins suivront.',
    'Per questo modulo finora è regolabile solo la visibilità — gli interruttori fini arriveranno con l’estensione.',
  ],
  'rbac.editor.fineSwitches': ['Fein-Schalter', 'Fine switches', 'Interrupteurs fins', 'Interruttori fini'],
  'rbac.editor.summaryCan': ['Darf:', 'May:', 'Peut :', 'Può:'],
  'rbac.editor.summaryCannot': ['Darf nicht:', 'May not:', 'Ne peut pas :', 'Non può:'],
  'rbac.editor.summaryNothing': ['nichts in diesem Modul', 'nothing in this module', 'rien dans ce module', 'niente in questo modulo'],
  'rbac.editor.summaryMore': [
    '{count, plural, one {· # weiteres} other {· # weitere}}',
    '{count, plural, one {· # more} other {· # more}}',
    '{count, plural, one {· # de plus} other {· # de plus}}',
    '{count, plural, one {· # altro} other {· # altri}}',
  ],
  'rbac.editor.deviatingHint': ['Weicht von der Vorlage ab', 'Deviates from the template', 'S’écarte du modèle', 'Si discosta dal modello'],
  'rbac.editor.scopeLabel': ['Daten-Scope', 'Data scope', 'Portée des données', 'Ambito dati'],
  'rbac.editor.stagedCount': [
    '{count, plural, one {# Änderung} other {# Änderungen}}',
    '{count, plural, one {# change} other {# changes}}',
    '{count, plural, one {# modification} other {# modifications}}',
    '{count, plural, one {# modifica} other {# modifiche}}',
  ],
  'rbac.editor.discard': ['Verwerfen', 'Discard', 'Annuler les modifications', 'Scarta'],
  'rbac.editor.apply': ['Änderungen übernehmen', 'Apply changes', 'Appliquer les modifications', 'Applica modifiche'],
  'rbac.editor.applyTitle': ['Änderungen an „{name}" übernehmen?', 'Apply changes to “{name}”?', 'Appliquer les modifications à « {name} » ?', 'Applicare le modifiche a «{name}»?'],
  'rbac.editor.applyBody': [
    '{count} Änderungen werden sofort wirksam — {members} Konten tragen diese Rolle.',
    '{count} changes take effect immediately — {members} accounts hold this role.',
    '{count} modifications prennent effet immédiatement — {members} comptes détiennent ce rôle.',
    '{count} modifiche hanno effetto immediato — {members} account hanno questo ruolo.',
  ],
  'rbac.editor.applyAction': ['Übernehmen', 'Apply', 'Appliquer', 'Applica'],
  'rbac.editor.saveDone': ['Rollen-Rechte gespeichert', 'Role permissions saved', 'Droits du rôle enregistrés', 'Permessi del ruolo salvati'],
  'rbac.editor.selfLockoutWarning': [
    'Du trägst diese Rolle selbst und entfernst gerade Verwaltungs-Zugriff — du könntest dich aus dem Baukasten aussperren.',
    'You hold this role yourself and are removing administration access — you could lock yourself out of the builder.',
    'Vous détenez ce rôle et retirez l’accès d’administration — vous pourriez vous verrouiller hors de l’atelier.',
    'Hai tu stesso questo ruolo e stai rimuovendo l’accesso amministrativo — potresti chiuderti fuori dal costruttore.',
  ],
  'rbac.editor.escalationBlocked': [
    'Blockiert: Du kannst keine Rechte vergeben, die du selbst nicht hast.',
    'Blocked: you cannot grant permissions you do not hold yourself.',
    'Bloqué : vous ne pouvez pas accorder des droits que vous ne détenez pas vous-même.',
    'Bloccato: non puoi concedere permessi che tu stesso non possiedi.',
  ],
  'rbac.editor.escalationMore': [
    '{count, plural, one {+ # weiteres Recht} other {+ # weitere Rechte}}',
    '{count, plural, one {+ # more permission} other {+ # more permissions}}',
    '{count, plural, one {+ # droit de plus} other {+ # droits de plus}}',
    '{count, plural, one {+ # altro permesso} other {+ # altri permessi}}',
  ],

  // ── compare ───────────────────────────────────────────────────────────────
  'rbac.compare.title': ['Rollen vergleichen', 'Compare roles', 'Comparer les rôles', 'Confronta ruoli'],
  'rbac.compare.subtitle': [
    'Zwei Rollen nebeneinander — Unterschiede sind hervorgehoben.',
    'Two roles side by side — differences are highlighted.',
    'Deux rôles côte à côte — les différences sont mises en évidence.',
    'Due ruoli affiancati — le differenze sono evidenziate.',
  ],
  'rbac.compare.roleA': ['Rolle A', 'Role A', 'Rôle A', 'Ruolo A'],
  'rbac.compare.roleB': ['Rolle B', 'Role B', 'Rôle B', 'Ruolo B'],
  'rbac.compare.onlyDifferences': ['Nur Unterschiede', 'Only differences', 'Différences uniquement', 'Solo differenze'],
  'rbac.compare.identical': [
    'Keine Unterschiede — beide Rollen haben identische Rechte.',
    'No differences — both roles carry identical permissions.',
    'Aucune différence — les deux rôles ont des droits identiques.',
    'Nessuna differenza — i due ruoli hanno permessi identici.',
  ],
  'rbac.compare.empty': ['Keine Rechte vorhanden.', 'No permissions present.', 'Aucun droit présent.', 'Nessun permesso presente.'],
  'rbac.compare.diffCount': [
    '{count, plural, one {# Unterschied} other {# Unterschiede}}',
    '{count, plural, one {# difference} other {# differences}}',
    '{count, plural, one {# différence} other {# différences}}',
    '{count, plural, one {# differenza} other {# differenze}}',
  ],

  // ── preview ───────────────────────────────────────────────────────────────
  'rbac.preview.start': ['Als Rolle anzeigen', 'View as role', 'Voir comme ce rôle', 'Visualizza come ruolo'],
  'rbac.preview.startHint': [
    'Cosmi zeigt sich, wie diese Rolle es sieht — du bleibst angemeldet.',
    'Cosmi renders the way this role sees it — you stay signed in.',
    'Cosmi s’affiche comme ce rôle le voit — vous restez connecté.',
    'Cosmi si mostra come lo vede questo ruolo — resti connesso.',
  ],
  'rbac.preview.banner': ['Vorschau als „{name}"', 'Previewing as “{name}”', 'Aperçu en tant que « {name} »', 'Anteprima come «{name}»'],
  'rbac.preview.end': ['Beenden', 'End', 'Terminer', 'Termina'],

  // ── assignment (team + user detail) ───────────────────────────────────────
  'rbac.assignment.manage': ['Rollen verwalten', 'Manage roles', 'Gérer les rôles', 'Gestisci ruoli'],
  'rbac.assignment.added': ['Rolle zugewiesen', 'Role assigned', 'Rôle attribué', 'Ruolo assegnato'],
  'rbac.assignment.removed': ['Rolle entfernt', 'Role removed', 'Rôle retiré', 'Ruolo rimosso'],
  'rbac.assignment.unionHint': [
    'Mehrere Rollen addieren sich: Es gilt immer das weiteste Recht. Die Summe zeigt „Effektive Rechte".',
    'Multiple roles add up: the widest permission always wins. See the sum under “Effective rights”.',
    'Plusieurs rôles s’additionnent : le droit le plus large prévaut. La somme s’affiche sous « Droits effectifs ».',
    'Più ruoli si sommano: prevale sempre il permesso più ampio. La somma è in «Diritti effettivi».',
  ],
  'rbac.assignment.selfLocked': [
    'Eigene Rollen kannst du nicht ändern — das übernimmt eine andere Administratorin.',
    'You cannot change your own roles — another administrator does that.',
    'Vous ne pouvez pas modifier vos propres rôles — un autre administrateur s’en charge.',
    'Non puoi modificare i tuoi ruoli — se ne occupa un altro amministratore.',
  ],
  'rbac.assignment.effectiveRights': ['Effektive Rechte ansehen', 'View effective rights', 'Voir les droits effectifs', 'Vedi diritti effettivi'],
  'rbac.assignment.effectiveTitle': ['Effektive Rechte — {name}', 'Effective rights — {name}', 'Droits effectifs — {name}', 'Diritti effettivi — {name}'],
  'rbac.assignment.effectiveSubtitle': [
    'Aufgelöste Summe aller Rollen mit Herkunft je Recht',
    'Resolved union of all roles with per-permission provenance',
    'Somme résolue de tous les rôles avec provenance par droit',
    'Unione risolta di tutti i ruoli con provenienza per permesso',
  ],
}

// ── apply ───────────────────────────────────────────────────────────────────
for (const [li, locale] of LOCALES.entries()) {
  const file = join(MESSAGES_DIR, `${locale}.json`)
  const data = JSON.parse(readFileSync(file, 'utf8'))

  let removed = 0
  for (const key of Object.keys(data)) {
    if (REMOVE.includes(key) || REMOVE_PREFIXES.some((p) => key.startsWith(p))) {
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
