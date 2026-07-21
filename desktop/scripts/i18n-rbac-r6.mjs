/**
 * RBAC R-6 i18n pass (per-user permission overrides).
 *
 * ADD: 40 keys x4 languages (override editor, tri-state pills, badge/filter,
 * effective-view provenance for override/deny, role-change confirm, the new
 * `rbac.subject.user_override` capability subject label).
 *
 * Du-Form (Cosmi standard), real umlauts, real fr/it — the R-5 lesson.
 * ICU single braces `{var}`, plural as `{count, plural, ...}`.
 *
 * Run once from desktop/: node scripts/i18n-rbac-r6.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))
const msgDir = join(here, '..', 'src', 'renderer', 'src', 'i18n', 'messages')

const ADD = {
  'rbac.subject.user_override': {
    de: 'Benutzer-Anpassungen', en: 'Per-user overrides', fr: 'Ajustements par utilisateur', it: 'Personalizzazioni per utente',
  },
  'rbac.override.title': {
    de: 'Berechtigungen von {name}', en: 'Permissions for {name}', fr: 'Autorisations de {name}', it: 'Autorizzazioni di {name}',
  },
  'rbac.override.editAction': {
    de: 'Berechtigungen anpassen', en: 'Adjust permissions', fr: 'Ajuster les autorisations', it: 'Modifica autorizzazioni',
  },
  'rbac.override.rolesLine': {
    de: 'Rollen: {roles}', en: 'Roles: {roles}', fr: 'Rôles : {roles}', it: 'Ruoli: {roles}',
  },
  'rbac.override.count': {
    de: '{count, plural, one {# Anpassung} other {# Anpassungen}}',
    en: '{count, plural, one {# override} other {# overrides}}',
    fr: '{count, plural, one {# ajustement} other {# ajustements}}',
    it: '{count, plural, one {# personalizzazione} other {# personalizzazioni}}',
  },
  'rbac.override.customize': {
    de: 'Benutzerdefiniert', en: 'Customize', fr: 'Personnaliser', it: 'Personalizza',
  },
  'rbac.override.customHint': {
    de: 'Bearbeitungsmodus aktiv — jeder Schalter, den du umlegst, wird eine persönliche Abweichung nur für diesen Benutzer.',
    en: 'Edit mode active — every switch you flip becomes a personal override for this user only.',
    fr: 'Mode édition actif — chaque commutateur que vous modifiez devient un ajustement personnel pour cet utilisateur uniquement.',
    it: 'Modalità modifica attiva — ogni interruttore che cambi diventa una personalizzazione solo per questo utente.',
  },
  'rbac.override.readonlyHint': {
    de: 'Die Schalter zeigen den geerbten Stand aus den Rollen. Klicke auf „Benutzerdefiniert", um einzelne Rechte für diesen Benutzer abweichend zu setzen.',
    en: 'The switches show the state inherited from the roles. Click "Customize" to set individual permissions for this user.',
    fr: 'Les commutateurs affichent l’état hérité des rôles. Cliquez sur « Personnaliser » pour définir des autorisations individuelles pour cet utilisateur.',
    it: 'Gli interruttori mostrano lo stato ereditato dai ruoli. Clicca su «Personalizza» per impostare autorizzazioni individuali per questo utente.',
  },
  'rbac.override.selfLocked': {
    de: 'Du kannst deine eigenen Berechtigungen nicht anpassen.',
    en: 'You cannot adjust your own permissions.',
    fr: 'Vous ne pouvez pas ajuster vos propres autorisations.',
    it: 'Non puoi modificare le tue autorizzazioni.',
  },
  'rbac.override.paneHint': {
    de: 'Grau = geerbt aus den Rollen. Grün = zusätzlich erlaubt. Rot = entzogen.',
    en: 'Gray = inherited from roles. Green = additionally granted. Red = revoked.',
    fr: 'Gris = hérité des rôles. Vert = accordé en plus. Rouge = retiré.',
    it: 'Grigio = ereditato dai ruoli. Verde = concesso in aggiunta. Rosso = revocato.',
  },
  'rbac.override.state.inheritedOn': {
    de: 'Geerbt', en: 'Inherited', fr: 'Hérité', it: 'Ereditato',
  },
  'rbac.override.state.inheritedOff': {
    de: 'Geerbt', en: 'Inherited', fr: 'Hérité', it: 'Ereditato',
  },
  'rbac.override.state.allow': {
    de: 'Erlaubt', en: 'Granted', fr: 'Accordé', it: 'Concesso',
  },
  'rbac.override.state.deny': {
    de: 'Entzogen', en: 'Revoked', fr: 'Retiré', it: 'Revocato',
  },
  'rbac.override.cycleLabel': {
    de: '{label} umschalten', en: 'Toggle {label}', fr: 'Basculer {label}', it: 'Attiva/disattiva {label}',
  },
  'rbac.override.resetKey': {
    de: 'Auf Rollen-Stand zurücksetzen', en: 'Reset to role state', fr: 'Rétablir l’état du rôle', it: 'Ripristina lo stato del ruolo',
  },
  'rbac.override.resetAll': {
    de: 'Alle zurücksetzen', en: 'Reset all', fr: 'Tout rétablir', it: 'Ripristina tutto',
  },
  'rbac.override.preview': {
    de: 'Als Benutzer anzeigen', en: 'View as user', fr: 'Afficher en tant qu’utilisateur', it: 'Visualizza come utente',
  },
  'rbac.override.previewHint': {
    de: 'Zeigt Cosmi mit den effektiven Rechten dieses Benutzers (inkl. Anpassungen).',
    en: 'Shows Cosmi with this user’s effective permissions (incl. overrides).',
    fr: 'Affiche Cosmi avec les autorisations effectives de cet utilisateur (ajustements inclus).',
    it: 'Mostra Cosmi con le autorizzazioni effettive di questo utente (incluse le personalizzazioni).',
  },
  'rbac.override.previewLabel': {
    de: '{name} (mit Anpassungen)', en: '{name} (with overrides)', fr: '{name} (avec ajustements)', it: '{name} (con personalizzazioni)',
  },
  'rbac.override.apply': {
    de: 'Übernehmen', en: 'Apply', fr: 'Appliquer', it: 'Applica',
  },
  'rbac.override.applyAction': {
    de: 'Anpassungen speichern', en: 'Save overrides', fr: 'Enregistrer les ajustements', it: 'Salva le personalizzazioni',
  },
  'rbac.override.applyTitle': {
    de: 'Anpassungen für {name} speichern?', en: 'Save overrides for {name}?', fr: 'Enregistrer les ajustements pour {name} ?', it: 'Salvare le personalizzazioni per {name}?',
  },
  'rbac.override.applyBody': {
    de: '{count, plural, one {Eine persönliche Abweichung wird} other {# persönliche Abweichungen werden}} für diesen Benutzer gespeichert. Nicht angepasste Rechte folgen weiterhin den Rollen.',
    en: '{count, plural, one {One personal override will be} other {# personal overrides will be}} saved for this user. Permissions not overridden keep following the roles.',
    fr: '{count, plural, one {Un ajustement personnel sera} other {# ajustements personnels seront}} enregistré pour cet utilisateur. Les autorisations non ajustées continuent de suivre les rôles.',
    it: '{count, plural, one {Verrà salvata una personalizzazione} other {Verranno salvate # personalizzazioni}} per questo utente. Le autorizzazioni non modificate continuano a seguire i ruoli.',
  },
  'rbac.override.applyClearBody': {
    de: 'Alle persönlichen Abweichungen werden entfernt. Der Benutzer folgt danach wieder vollständig seinen Rollen.',
    en: 'All personal overrides will be removed. The user then fully follows their roles again.',
    fr: 'Tous les ajustements personnels seront supprimés. L’utilisateur suit alors de nouveau entièrement ses rôles.',
    it: 'Tutte le personalizzazioni verranno rimosse. L’utente torna quindi a seguire completamente i suoi ruoli.',
  },
  'rbac.override.escalationBlocked': {
    de: 'Du kannst keine Rechte vergeben, die du selbst nicht hast:',
    en: 'You cannot grant permissions you don’t hold yourself:',
    fr: 'Vous ne pouvez pas accorder des autorisations que vous ne détenez pas vous-même :',
    it: 'Non puoi concedere autorizzazioni che tu stesso non possiedi:',
  },
  'rbac.override.adminLockWarning': {
    de: 'Achtung: Diese Anpassung entzieht einem Administrator zentrale Verwaltungsrechte. Prüfe, dass mindestens ein Vollzugriff-Konto erhalten bleibt.',
    en: 'Warning: this override revokes core administration rights from an administrator. Make sure at least one full-access account remains.',
    fr: 'Attention : cet ajustement retire des droits d’administration essentiels à un administrateur. Assurez-vous qu’au moins un compte à accès complet subsiste.',
    it: 'Attenzione: questa personalizzazione revoca diritti di amministrazione fondamentali a un amministratore. Assicurati che rimanga almeno un account con accesso completo.',
  },
  'rbac.override.stagedCount': {
    de: '{count, plural, one {# Abweichung} other {# Abweichungen}}',
    en: '{count, plural, one {# override} other {# overrides}}',
    fr: '{count, plural, one {# ajustement} other {# ajustements}}',
    it: '{count, plural, one {# personalizzazione} other {# personalizzazioni}}',
  },
  'rbac.override.noneStaged': {
    de: 'Keine Abweichungen — der Benutzer folgt seinen Rollen.',
    en: 'No overrides — the user follows their roles.',
    fr: 'Aucun ajustement — l’utilisateur suit ses rôles.',
    it: 'Nessuna personalizzazione — l’utente segue i suoi ruoli.',
  },
  'rbac.override.userNotFound': {
    de: 'Benutzer nicht gefunden.', en: 'User not found.', fr: 'Utilisateur introuvable.', it: 'Utente non trovato.',
  },
  'rbac.override.badge': {
    de: 'Angepasst', en: 'Customized', fr: 'Ajusté', it: 'Personalizzato',
  },
  'rbac.override.badgeHint': {
    de: 'Dieser Benutzer hat persönliche Berechtigungs-Abweichungen von seinen Rollen.',
    en: 'This user has personal permission overrides on top of their roles.',
    fr: 'Cet utilisateur a des ajustements d’autorisations personnels par rapport à ses rôles.',
    it: 'Questo utente ha personalizzazioni delle autorizzazioni rispetto ai suoi ruoli.',
  },
  'rbac.override.filterOnly': {
    de: 'Nur angepasste', en: 'Customized only', fr: 'Ajustés uniquement', it: 'Solo personalizzati',
  },
  'rbac.override.source': {
    de: 'Persönlich', en: 'Personal', fr: 'Personnel', it: 'Personale',
  },
  'rbac.override.deniedSource': {
    de: 'Persönlich entzogen', en: 'Personally revoked', fr: 'Retiré personnellement', it: 'Revocato personalmente',
  },
  'rbac.override.saveDone': {
    de: 'Anpassungen gespeichert', en: 'Overrides saved', fr: 'Ajustements enregistrés', it: 'Personalizzazioni salvate',
  },
  'rbac.override.saveError': {
    de: 'Anpassungen konnten nicht gespeichert werden', en: 'Could not save overrides', fr: 'Impossible d’enregistrer les ajustements', it: 'Impossibile salvare le personalizzazioni',
  },
  'rbac.override.roleChangeTitle': {
    de: 'Rolle ändern trotz Anpassungen?', en: 'Change role despite overrides?', fr: 'Changer de rôle malgré les ajustements ?', it: 'Cambiare ruolo nonostante le personalizzazioni?',
  },
  'rbac.override.roleChangeBody': {
    de: '{name} hat persönliche Berechtigungs-Anpassungen. Diese bleiben nach der Rollen-Änderung bestehen — prüfe danach die effektiven Rechte.',
    en: '{name} has personal permission overrides. They remain after the role change — review the effective permissions afterwards.',
    fr: '{name} a des ajustements d’autorisations personnels. Ils sont conservés après le changement de rôle — vérifiez ensuite les autorisations effectives.',
    it: '{name} ha personalizzazioni delle autorizzazioni. Rimangono dopo il cambio di ruolo — verifica poi le autorizzazioni effettive.',
  },
  'rbac.override.roleChangeConfirm': {
    de: 'Rolle ändern', en: 'Change role', fr: 'Changer de rôle', it: 'Cambia ruolo',
  },
}

for (const lang of ['de', 'en', 'fr', 'it']) {
  const file = join(msgDir, `${lang}.json`)
  const messages = JSON.parse(readFileSync(file, 'utf8'))
  let added = 0
  for (const [key, byLang] of Object.entries(ADD)) {
    if (key in messages) throw new Error(`${lang}: ADD key already exists: ${key}`)
    messages[key] = byLang[lang]
    added += 1
  }
  writeFileSync(file, JSON.stringify(messages, null, 2) + '\n', 'utf8')
  console.log(`${lang}.json — +${added} added`)
}
console.log('done')
