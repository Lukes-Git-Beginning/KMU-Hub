/**
 * Customization v1.2 i18n pass — Label-Override-Editor (Begriffe-Tab).
 *
 * ADD: Keys für:
 *   - Sub-Tab-Name „Begriffe"
 *   - Sprach-Auswahl im Editor
 *   - Gruppen-Überschriften (Modulnamen / Objekte / Navigation)
 *   - Inline-Bearbeiten: Eingabe, Speichern, Abbrechen, Reset pro Key
 *   - Reset-All-Bestätigungs-Dialog
 *   - Empty-State (keine Overrides)
 *   - Toast-Meldungen (gespeichert, zurückgesetzt, Fehler)
 *   - Provenance-Badge-Labels (ergänzt bestehende customization.provenance.*)
 *   - Live-Preview-Hinweis-Banner
 *
 * Konventionen (projekt-verbindlich):
 *   - Du-Form (Cosmi duzt)
 *   - `{var}` NICHT `{{var}}` (ICU-Plugin, keySeparator:false)
 *   - Plural als ICU: `{count, plural, one {# …} other {# …}}`
 *   - ECHTE fr+it-Übersetzungen (keine EN-Kopien)
 *   - Typografischer Apostroph U+2019 in FR (kein gerader Apostroph in Strings)
 *
 * Ausführen (einmalig) aus desktop/: node scripts/i18n-customization-v12.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))
const msgDir = join(here, '..', 'src', 'renderer', 'src', 'i18n', 'messages')

// Typografischer Apostroph (U+2019) — gerade Apostrophe brechen ICU-Parser
const AP = '’'

const ADD = {

  // ── Sub-Tab-Name ────────────────────────────────────────────────────────────

  'customization.hub.tabs.begriffe': {
    de: 'Begriffe',
    en: 'Terms',
    fr: 'Termes',
    it: 'Termini',
  },

  // ── Seiten-Header ───────────────────────────────────────────────────────────

  'customization.labels.localeLabel': {
    de: 'Sprache bearbeiten',
    en: 'Edit language',
    fr: 'Langue à modifier',
    it: 'Lingua da modificare',
  },
  'customization.labels.localeHint': {
    de: 'Änderungen gelten für die gewählte Sprache.',
    en: 'Changes apply to the selected language.',
    fr: `Les modifications s${AP}appliquent à la langue sélectionnée.`,
    it: 'Le modifiche si applicano alla lingua selezionata.',
  },

  // ── Gruppen-Überschriften ───────────────────────────────────────────────────

  'customization.labels.group.modules': {
    de: 'Modulnamen',
    en: 'Module names',
    fr: 'Noms de modules',
    it: 'Nomi dei moduli',
  },
  'customization.labels.group.objects': {
    de: 'Objekt-Bezeichnungen',
    en: 'Object names',
    fr: `Noms d${AP}objets`,
    it: 'Nomi degli oggetti',
  },
  'customization.labels.group.navigation': {
    de: 'Navigation',
    en: 'Navigation',
    fr: 'Navigation',
    it: 'Navigazione',
  },

  // ── Zeilen-Inhalte ──────────────────────────────────────────────────────────

  'customization.labels.defaultValue': {
    de: 'Cosmi-Standard',
    en: 'Cosmi default',
    fr: 'Valeur Cosmi par défaut',
    it: 'Valore predefinito Cosmi',
  },
  'customization.labels.currentValue': {
    de: 'Aktueller Begriff',
    en: 'Current term',
    fr: 'Terme actuel',
    it: 'Termine attuale',
  },
  'customization.labels.editPlaceholder': {
    de: 'Neuer Begriff …',
    en: 'New term …',
    fr: 'Nouveau terme …',
    it: 'Nuovo termine …',
  },
  'customization.labels.saveKey': {
    de: 'Speichern',
    en: 'Save',
    fr: 'Enregistrer',
    it: 'Salva',
  },
  'customization.labels.cancelEdit': {
    de: 'Abbrechen',
    en: 'Cancel',
    fr: 'Annuler',
    it: 'Annulla',
  },

  // ── Reset einzelner Key ─────────────────────────────────────────────────────

  'customization.labels.resetKeyTitle': {
    de: 'Begriff zurücksetzen',
    en: 'Reset term',
    fr: 'Réinitialiser le terme',
    it: 'Ripristina il termine',
  },
  'customization.labels.resetKeyHint': {
    de: 'Fällt auf die darunter liegende Schicht zurück (Zentria-Einrichtung oder Cosmi-Standard).',
    en: 'Falls back to the layer below (Zentria setup or Cosmi default).',
    fr: `Revient à la couche inférieure (configuration Zentria ou valeur Cosmi par défaut).`,
    it: 'Torna al livello sottostante (configurazione Zentria o valore predefinito Cosmi).',
  },

  // ── Reset-All-Bestätigung ───────────────────────────────────────────────────

  'customization.labels.resetAllTitle': {
    de: 'Alle eigenen Begriffe zurücksetzen?',
    en: 'Reset all custom terms?',
    fr: `Réinitialiser tous les termes personnalisés ?`,
    it: 'Ripristinare tutti i termini personalizzati?',
  },
  'customization.labels.resetAllBody': {
    de: 'Alle eigenen Begriffsänderungen werden entfernt. Zentria-Einstellungen und Cosmi-Standards bleiben erhalten.',
    en: 'All your custom term changes will be removed. Zentria settings and Cosmi defaults remain intact.',
    fr: `Toutes tes modifications de termes personnalisés seront supprimées. Les réglages Zentria et les valeurs Cosmi par défaut restent inchangés.`,
    it: 'Tutte le tue modifiche ai termini personalizzati verranno rimosse. Le impostazioni Zentria e i valori predefiniti di Cosmi rimangono invariati.',
  },
  'customization.labels.resetAllConfirm': {
    de: 'Zurücksetzen',
    en: 'Reset',
    fr: 'Réinitialiser',
    it: 'Ripristina',
  },

  // ── Toast-Meldungen ─────────────────────────────────────────────────────────

  'customization.labels.savedSingle': {
    de: 'Begriff „{key}" gespeichert',
    en: 'Term "{key}" saved',
    fr: `Terme « {key} » enregistré`,
    it: 'Termine «{key}» salvato',
  },
  'customization.labels.resetSingle': {
    de: 'Begriff zurückgesetzt',
    en: 'Term reset',
    fr: 'Terme réinitialisé',
    it: 'Termine ripristinato',
  },
  'customization.labels.resetAllDone': {
    de: 'Alle eigenen Begriffe zurückgesetzt',
    en: 'All custom terms reset',
    fr: 'Tous les termes personnalisés réinitialisés',
    it: 'Tutti i termini personalizzati ripristinati',
  },

  // ── Empty-State ─────────────────────────────────────────────────────────────

  'customization.labels.emptyTitle': {
    de: 'Noch keine eigenen Begriffe',
    en: 'No custom terms yet',
    fr: 'Aucun terme personnalisé pour le moment',
    it: 'Nessun termine personalizzato ancora',
  },
  'customization.labels.emptyHint': {
    de: 'Klicke auf einen Begriff, um ihn umzubenennen. Die Änderung ist sofort in der gesamten App sichtbar.',
    en: 'Click a term to rename it. The change is immediately visible across the entire app.',
    fr: `Clique sur un terme pour le renommer. La modification est immédiatement visible dans toute l${AP}application.`,
    it: `Clicca su un termine per rinominarlo. La modifica è immediatamente visibile in tutta l${AP}applicazione.`,
  },

  // ── Live-Preview-Banner ─────────────────────────────────────────────────────

  'customization.labels.livePreviewBanner': {
    de: 'Änderungen sind sofort in der gesamten App sichtbar — ohne Neustart.',
    en: 'Changes are immediately visible across the entire app — no restart needed.',
    fr: `Les modifications sont immédiatement visibles dans toute l${AP}application — sans redémarrage.`,
    it: `Le modifiche sono immediatamente visibili in tutta l${AP}applicazione — senza riavvio.`,
  },

  // ── Provenance-Badge-Ergänzungen ────────────────────────────────────────────
  // (customization.provenance.default/vendor/tenant existieren bereits aus v1.0)

  'customization.labels.provenance.vendor': {
    de: 'Von Zentria',
    en: 'From Zentria',
    fr: 'De Zentria',
    it: 'Da Zentria',
  },
  'customization.labels.provenance.tenant': {
    de: 'Angepasst',
    en: 'Custom',
    fr: 'Personnalisé',
    it: 'Personalizzato',
  },
  'customization.labels.provenance.default': {
    de: 'Standard',
    en: 'Default',
    fr: 'Par défaut',
    it: 'Predefinito',
  },

  // ── Zähler ──────────────────────────────────────────────────────────────────

  'customization.labels.overrideCount': {
    de: '{count, plural, one {# eigener Begriff} other {# eigene Begriffe}}',
    en: '{count, plural, one {# custom term} other {# custom terms}}',
    fr: `{count, plural, one {# terme personnalisé} other {# termes personnalisés}}`,
    it: `{count, plural, one {# termine personalizzato} other {# termini personalizzati}}`,
  },

  // ── Vendor-Hinweis-Tooltip ──────────────────────────────────────────────────

  'customization.labels.vendorTooltip': {
    de: 'Beim Onboarding von Zentria eingerichtet. Du kannst es weiter anpassen.',
    en: 'Set up by Zentria during onboarding. You can further customize it.',
    fr: `Configuré par Zentria lors de l${AP}intégration. Tu peux le personnaliser davantage.`,
    it: `Configurato da Zentria durante l${AP}onboarding. Puoi personalizzarlo ulteriormente.`,
  },

  // ── Audit-Action-Labels ─────────────────────────────────────────────────────
  // (customization.label_set/label_removed existieren bereits aus v1.0)

}

// ── JSON-Dateien patchen ────────────────────────────────────────────────────────

for (const lang of ['de', 'en', 'fr', 'it']) {
  const file = join(msgDir, `${lang}.json`)
  const messages = JSON.parse(readFileSync(file, 'utf8'))
  let added = 0
  for (const [key, byLang] of Object.entries(ADD)) {
    if (key in messages) throw new Error(`${lang}: ADD key already exists: ${key} — entferne es aus dem Skript`)
    messages[key] = byLang[lang]
    added += 1
  }
  writeFileSync(file, JSON.stringify(messages, null, 2) + '\n', 'utf8')
  console.log(`${lang}.json — +${added} hinzugefügt`)
}
console.log('done — v1.2 i18n pass complete')
