/**
 * Customization v1.0 i18n pass — Datenschicht-Fundament.
 *
 * ADD: Keys für:
 *   - RBAC-Capability `admin:customization:manage` (subject-Label)
 *   - Feature/Modul-Name „Anpassungen"
 *   - Overlay-Provenance-Labels (Standard / Zentria / Eigene)
 *   - Value-Set-Namen aus den Seeds (deal_stages, ticket_priority, project_status)
 *   - Audit-Action-Labels für die drei customization.* Events
 *
 * Du-Form (Cosmi-Standard), echte Umlaute, ECHTE fr+it-Übersetzungen.
 * ICU single braces `{var}`, Plural als `{count, plural, ...}`.
 *
 * Ausführen (einmalig) aus desktop/: node scripts/i18n-customization-v10.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))
const msgDir = join(here, '..', 'src', 'renderer', 'src', 'i18n', 'messages')

// Typografischer Apostroph (U+2019) — R-5-Lektion: gerade Apostrophe in
// JS-Single-Quote-Strings brechen den Parser; r6 nutzt dasselbe Zeichen.
const AP = '’'

const ADD = {
  // ── RBAC capability label (subject half — action 'manage' already exists) ──
  'rbac.subject.customization': {
    de: 'Anpassungen',
    en: 'Customization',
    fr: 'Personnalisation',
    it: 'Personalizzazione',
  },

  // ── Feature / module name ─────────────────────────────────────────────────
  'customization.title': {
    de: 'Anpassungen',
    en: 'Customization',
    fr: 'Personnalisation',
    it: 'Personalizzazione',
  },
  'customization.subtitle': {
    de: 'Sprache, Begriffe und Wertelisten deines Cosmi anpassen',
    en: 'Adapt the language, terms and value lists of your Cosmi',
    fr: 'Adapte la langue, les termes et les listes de valeurs de ton Cosmi',
    it: 'Adatta la lingua, i termini e gli elenchi di valori del tuo Cosmi',
  },

  // ── Overlay provenance labels (shown in editor cells) ─────────────────────
  'customization.provenance.default': {
    de: 'Cosmi-Standard',
    en: 'Cosmi default',
    fr: 'Valeur par défaut Cosmi',
    it: 'Predefinito Cosmi',
  },
  'customization.provenance.vendor': {
    de: 'Von Zentria eingerichtet',
    en: 'Set by Zentria',
    fr: 'Configuré par Zentria',
    it: 'Configurato da Zentria',
  },
  'customization.provenance.tenant': {
    de: 'Selbst geändert',
    en: 'Changed by you',
    fr: 'Modifié par vous',
    it: 'Modificato da te',
  },
  'customization.provenance.hint': {
    de: 'Herkunft: {provenance}',
    en: 'Origin: {provenance}',
    fr: 'Origine : {provenance}',
    it: 'Origine: {provenance}',
  },

  // ── Label-Override section ─────────────────────────────────────────────────
  'customization.labels.title': {
    de: 'Begriffe & Bezeichnungen',
    en: 'Terms & Labels',
    fr: 'Termes et étiquettes',
    it: 'Termini ed etichette',
  },
  'customization.labels.subtitle': {
    de: 'Passe Modul- und Objektbezeichnungen an deine Branche an.',
    en: 'Adapt module and object names to your industry.',
    fr: `Adapte les noms de modules et d${AP}objets à ton secteur.`,
    it: 'Adatta i nomi di moduli e oggetti al tuo settore.',
  },
  'customization.labels.resetKey': {
    de: 'Auf Standard zurücksetzen',
    en: 'Reset to default',
    fr: 'Rétablir la valeur par défaut',
    it: 'Ripristina il valore predefinito',
  },
  'customization.labels.resetAll': {
    de: 'Alle Begriffe zurücksetzen',
    en: 'Reset all terms',
    fr: 'Rétablir tous les termes',
    it: 'Ripristina tutti i termini',
  },
  'customization.labels.saved': {
    de: 'Begriffe gespeichert',
    en: 'Terms saved',
    fr: 'Termes enregistrés',
    it: 'Termini salvati',
  },
  'customization.labels.saveError': {
    de: 'Begriffe konnten nicht gespeichert werden',
    en: 'Could not save terms',
    fr: `Impossible d${AP}enregistrer les termes`,
    it: 'Impossibile salvare i termini',
  },

  // ── Value-Sets section ─────────────────────────────────────────────────────
  'customization.valueSets.title': {
    de: 'Wertelisten',
    en: 'Value Sets',
    fr: 'Listes de valeurs',
    it: 'Elenchi di valori',
  },
  'customization.valueSets.subtitle': {
    de: 'Passe Status-Sets, Prioritäten und Phasen-Listen an.',
    en: 'Adapt status sets, priorities and pipeline stages.',
    fr: 'Adapte les ensembles de statuts, priorités et étapes.',
    it: 'Adatta i set di stato, priorità e fasi.',
  },
  'customization.valueSets.addOption': {
    de: 'Option hinzufügen',
    en: 'Add option',
    fr: 'Ajouter une option',
    it: 'Aggiungi opzione',
  },
  'customization.valueSets.deactivate': {
    de: 'Deaktivieren (bestehende Daten bleiben erhalten)',
    en: 'Deactivate (existing data is preserved)',
    fr: 'Désactiver (les données existantes sont conservées)',
    it: 'Disattiva (i dati esistenti vengono conservati)',
  },
  'customization.valueSets.saved': {
    de: 'Werteliste gespeichert',
    en: 'Value set saved',
    fr: 'Liste de valeurs enregistrée',
    it: 'Elenco di valori salvato',
  },
  'customization.valueSets.saveError': {
    de: 'Werteliste konnte nicht gespeichert werden',
    en: 'Could not save value set',
    fr: `Impossible d${AP}enregistrer la liste de valeurs`,
    it: `Impossibile salvare l${AP}elenco di valori`,
  },
  'customization.valueSets.optionCount': {
    de: '{count, plural, one {# Option} other {# Optionen}}',
    en: '{count, plural, one {# option} other {# options}}',
    fr: '{count, plural, one {# option} other {# options}}',
    it: '{count, plural, one {# opzione} other {# opzioni}}',
  },

  // ── Seed value-set names ───────────────────────────────────────────────────
  'customization.valueSets.deal_stages.name': {
    de: 'Deal-Phasen',
    en: 'Deal Stages',
    fr: 'Phases des offres',
    it: 'Fasi delle offerte',
  },
  'customization.valueSets.ticket_priority.name': {
    de: 'Ticket-Priorität',
    en: 'Ticket Priority',
    fr: 'Priorité des tickets',
    it: 'Priorità dei ticket',
  },
  'customization.valueSets.project_status.name': {
    de: 'Projekt-Status',
    en: 'Project Status',
    fr: 'Statut du projet',
    it: 'Stato del progetto',
  },

  // ── Audit action labels (for the audit-log display in Security > Audit) ───
  'audit.action.customization.label_set': {
    de: 'Begriff überschrieben',
    en: 'Label overridden',
    fr: 'Étiquette remplacée',
    it: 'Etichetta sovrascritta',
  },
  'audit.action.customization.label_removed': {
    de: 'Begriff-Überschreibung entfernt',
    en: 'Label override removed',
    fr: `Remplacement d${AP}étiquette supprimé`,
    it: 'Override etichetta rimosso',
  },
  'audit.action.customization.valueset_updated': {
    de: 'Werteliste aktualisiert',
    en: 'Value set updated',
    fr: 'Liste de valeurs mise à jour',
    it: 'Elenco di valori aggiornato',
  },

  // ── Generic UI strings ────────────────────────────────────────────────────
  'customization.noCustomizations': {
    de: 'Noch keine Anpassungen — alles zeigt Cosmi-Standards.',
    en: 'No customizations yet — everything shows Cosmi defaults.',
    fr: 'Aucune personnalisation — tout affiche les valeurs Cosmi par défaut.',
    it: 'Nessuna personalizzazione — tutto mostra i valori predefiniti di Cosmi.',
  },
  'customization.resetLayer.vendor': {
    de: 'Zentria-Einrichtung zurücksetzen',
    en: 'Reset Zentria setup',
    fr: 'Réinitialiser la configuration Zentria',
    it: 'Ripristina la configurazione Zentria',
  },
  'customization.resetLayer.tenant': {
    de: 'Eigene Anpassungen zurücksetzen',
    en: 'Reset your customizations',
    fr: 'Réinitialiser tes personnalisations',
    it: 'Ripristina le tue personalizzazioni',
  },
  'customization.resetConfirmTitle': {
    de: 'Anpassungen zurücksetzen?',
    en: 'Reset customizations?',
    fr: 'Réinitialiser les personnalisations ?',
    it: 'Ripristinare le personalizzazioni?',
  },
  'customization.resetConfirmBody': {
    de: 'Alle eigenen Anpassungen werden entfernt. Zentria-Einstellungen und Cosmi-Standards bleiben erhalten.',
    en: 'All your customizations will be removed. Zentria settings and Cosmi defaults remain intact.',
    fr: 'Toutes tes personnalisations seront supprimées. Les paramètres Zentria et les valeurs par défaut de Cosmi restent intacts.',
    it: 'Tutte le tue personalizzazioni verranno rimosse. Le impostazioni Zentria e i valori predefiniti di Cosmi rimangono invariati.',
  },
  'customization.accessDenied': {
    de: 'Du brauchst die Berechtigung „Anpassungen verwalten", um diese Einstellungen zu ändern.',
    en: 'You need the "Manage customization" permission to change these settings.',
    fr: `Tu as besoin de l${AP}autorisation « Gérer la personnalisation » pour modifier ces paramètres.`,
    it: `Hai bisogno dell${AP}autorizzazione «Gestisci personalizzazione» per modificare queste impostazioni.`,
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
