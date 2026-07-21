/**
 * Customization v1.1 i18n pass — Custom Fields Editor.
 *
 * ADD: Keys für:
 *   - Anpassungen-Hub (Titel, Subtitle, Tab-Namen)
 *   - Custom-Fields-Editor (Entity-Labels, Feldtyp-Labels, UI-Strings)
 *   - Basis-Schicht: Label, Typ, Pflichtfeld
 *   - Fortgeschritten-Schicht: Optionen, Validierung, Standardwert, Sichtbarkeit
 *   - Soft-Delete-Dialoge (inUse-Konsequenz + normales Löschen)
 *   - Audit-Action-Labels für field_created / field_updated / field_deleted
 *
 * Konventionen (projekt-verbindlich):
 *   - Du-Form (Cosmi duzt)
 *   - `{var}` NICHT `{{var}}` (ICU-Plugin)
 *   - Plural als ICU: `{count, plural, one {# …} other {# …}}`
 *   - ECHTE fr+it-Übersetzungen (keine EN-Kopien)
 *   - Typografischer Apostroph U+2019 in FR (kein gerader Apostroph in Strings)
 *
 * Ausführen (einmalig) aus desktop/: node scripts/i18n-customization-v11.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))
const msgDir = join(here, '..', 'src', 'renderer', 'src', 'i18n', 'messages')

// Typografischer Apostroph (U+2019) — gerade Apostrophe brechen ICU-Parser
const AP = '’'

const ADD = {

  // ── Anpassungen-Hub ────────────────────────────────────────────────────────

  'customization.hub.title': {
    de: 'Anpassungen',
    en: 'Customization',
    fr: 'Personnalisation',
    it: 'Personalizzazione',
  },
  'customization.hub.subtitle': {
    de: 'Passe Felder, Begriffe und Wertelisten deines Cosmi an.',
    en: 'Customize fields, terms and value lists of your Cosmi.',
    fr: `Personnalise les champs, termes et listes de valeurs de ton Cosmi.`,
    it: 'Personalizza i campi, i termini e gli elenchi di valori del tuo Cosmi.',
  },
  'customization.hub.tabs.felder': {
    de: 'Felder',
    en: 'Fields',
    fr: 'Champs',
    it: 'Campi',
  },

  // ── Entity-Labels ──────────────────────────────────────────────────────────

  'customization.fields.entity.work_task': {
    de: 'Aufgaben',
    en: 'Tasks',
    fr: 'Tâches',
    it: 'Attività',
  },
  'customization.fields.entity.crm_contact': {
    de: 'Kontakte',
    en: 'Contacts',
    fr: 'Contacts',
    it: 'Contatti',
  },
  'customization.fields.entity.crm_company': {
    de: 'Firmen',
    en: 'Companies',
    fr: 'Entreprises',
    it: 'Aziende',
  },
  'customization.fields.entity.crm_deal': {
    de: 'Deals',
    en: 'Deals',
    fr: 'Affaires',
    it: 'Trattative',
  },
  'customization.fields.entity.crm_activity': {
    de: 'Aktivitäten',
    en: 'Activities',
    fr: 'Activités',
    it: 'Attività CRM',
  },

  // ── Feldtyp-Labels ─────────────────────────────────────────────────────────

  'customization.fields.type.text': {
    de: 'Text',
    en: 'Text',
    fr: 'Texte',
    it: 'Testo',
  },
  'customization.fields.type.number': {
    de: 'Zahl',
    en: 'Number',
    fr: 'Nombre',
    it: 'Numero',
  },
  'customization.fields.type.date': {
    de: 'Datum',
    en: 'Date',
    fr: 'Date',
    it: 'Data',
  },
  'customization.fields.type.boolean': {
    de: 'Ja/Nein',
    en: 'Yes/No',
    fr: 'Oui/Non',
    it: 'Sì/No',
  },
  'customization.fields.type.select': {
    de: 'Auswahl',
    en: 'Select',
    fr: 'Sélection',
    it: 'Selezione',
  },
  'customization.fields.type.multi_select': {
    de: 'Mehrfach',
    en: 'Multi-select',
    fr: 'Multi-sélection',
    it: 'Multiselezione',
  },
  'customization.fields.type.url': {
    de: 'URL',
    en: 'URL',
    fr: 'URL',
    it: 'URL',
  },
  'customization.fields.type.email': {
    de: 'E-Mail',
    en: 'E-mail',
    fr: 'E-mail',
    it: 'E-mail',
  },
  'customization.fields.type.phone': {
    de: 'Telefon',
    en: 'Phone',
    fr: 'Téléphone',
    it: 'Telefono',
  },

  // ── Feldliste / Entity-Selektor ────────────────────────────────────────────

  'customization.fields.entityLabel': {
    de: 'Entität',
    en: 'Entity',
    fr: 'Entité',
    it: 'Entità',
  },
  'customization.fields.fieldCount': {
    de: '{count, plural, one {# Feld} other {# Felder}}',
    en: '{count, plural, one {# field} other {# fields}}',
    fr: '{count, plural, one {# champ} other {# champs}}',
    it: '{count, plural, one {# campo} other {# campi}}',
  },
  'customization.fields.newField': {
    de: 'Feld anlegen',
    en: 'Add field',
    fr: 'Ajouter un champ',
    it: 'Aggiungi campo',
  },
  'customization.fields.requiredBadge': {
    de: 'Pflicht',
    en: 'Required',
    fr: 'Obligatoire',
    it: 'Obbligatorio',
  },
  'customization.fields.inUseHint': {
    de: 'Dieses Feld hat gespeicherte Werte',
    en: 'This field has stored values',
    fr: `Ce champ contient des valeurs enregistrées`,
    it: 'Questo campo contiene valori salvati',
  },
  'customization.fields.hidden': {
    de: 'ausgeblendet',
    en: 'hidden',
    fr: 'masqué',
    it: 'nascosto',
  },
  'customization.fields.emptyTitle': {
    de: 'Noch keine benutzerdefinierten Felder',
    en: 'No custom fields yet',
    fr: 'Aucun champ personnalisé pour le moment',
    it: 'Nessun campo personalizzato ancora',
  },
  'customization.fields.emptyHint': {
    de: 'Lege das erste Feld für diese Entität an.',
    en: 'Create the first field for this entity.',
    fr: `Crée le premier champ pour cette entité.`,
    it: 'Crea il primo campo per questa entità.',
  },

  // ── FieldEditorModal: Basis-Schicht ────────────────────────────────────────

  'customization.fields.newFieldTitle': {
    de: 'Neues Feld anlegen',
    en: 'New field',
    fr: 'Nouveau champ',
    it: 'Nuovo campo',
  },
  'customization.fields.editFieldTitle': {
    de: '{label} bearbeiten',
    en: 'Edit {label}',
    fr: 'Modifier {label}',
    it: 'Modifica {label}',
  },
  'customization.fields.labelField': {
    de: 'Name / Bezeichnung',
    en: 'Name / Label',
    fr: 'Nom / Étiquette',
    it: 'Nome / Etichetta',
  },
  'customization.fields.required': {
    de: '(Pflicht)',
    en: '(required)',
    fr: '(obligatoire)',
    it: '(obbligatorio)',
  },
  'customization.fields.labelPlaceholder': {
    de: 'z.B. Kundennummer ERP',
    en: 'e.g. ERP customer number',
    fr: `p. ex. Numéro client ERP`,
    it: 'es. Numero cliente ERP',
  },
  'customization.fields.typeField': {
    de: 'Feldtyp',
    en: 'Field type',
    fr: 'Type de champ',
    it: 'Tipo di campo',
  },
  'customization.fields.requiredField': {
    de: 'Pflichtfeld',
    en: 'Required field',
    fr: 'Champ obligatoire',
    it: 'Campo obbligatorio',
  },

  // ── FieldEditorModal: Fortgeschrittene Schicht ─────────────────────────────

  'customization.fields.advancedOptions': {
    de: 'Weitere Optionen',
    en: 'More options',
    fr: `Plus d${AP}options`,
    it: 'Altre opzioni',
  },
  'customization.fields.options': {
    de: 'Auswahloptionen',
    en: 'Options',
    fr: `Options de sélection`,
    it: 'Opzioni di selezione',
  },
  'customization.fields.optionsPlaceholder': {
    de: 'Option 1, Option 2, Option 3',
    en: 'Option 1, Option 2, Option 3',
    fr: 'Option 1, Option 2, Option 3',
    it: 'Opzione 1, Opzione 2, Opzione 3',
  },
  'customization.fields.optionsHint': {
    de: 'Kommagetrennte Werte. Reihenfolge wird beibehalten.',
    en: 'Comma-separated values. Order is preserved.',
    fr: `Valeurs séparées par des virgules. L${AP}ordre est conservé.`,
    it: `Valori separati da virgola. L${AP}ordine viene mantenuto.`,
  },
  'customization.fields.validation': {
    de: 'Validierung',
    en: 'Validation',
    fr: 'Validation',
    it: 'Validazione',
  },
  'customization.fields.validationMin': {
    de: 'Min',
    en: 'Min',
    fr: 'Min',
    it: 'Min',
  },
  'customization.fields.validationMax': {
    de: 'Max',
    en: 'Max',
    fr: 'Max',
    it: 'Max',
  },
  'customization.fields.validationPattern': {
    de: 'Muster (Regex)',
    en: 'Pattern (Regex)',
    fr: 'Modèle (Regex)',
    it: 'Schema (Regex)',
  },
  'customization.fields.defaultValue': {
    de: 'Standardwert',
    en: 'Default value',
    fr: 'Valeur par défaut',
    it: 'Valore predefinito',
  },
  'customization.fields.defaultValuePlaceholder': {
    de: 'Optional',
    en: 'Optional',
    fr: 'Facultatif',
    it: 'Facoltativo',
  },
  'customization.fields.visibleField': {
    de: 'In Listen und Detailansichten anzeigen',
    en: 'Show in lists and detail views',
    fr: 'Afficher dans les listes et les vues de détail',
    it: 'Mostra in elenchi e viste di dettaglio',
  },

  // ── Modal-Actions ──────────────────────────────────────────────────────────

  'customization.fields.create': {
    de: 'Feld anlegen',
    en: 'Create field',
    fr: 'Créer le champ',
    it: 'Crea campo',
  },
  'customization.fields.save': {
    de: 'Speichern',
    en: 'Save',
    fr: 'Enregistrer',
    it: 'Salva',
  },

  // ── Toast-Meldungen ────────────────────────────────────────────────────────

  'customization.fields.created': {
    de: 'Feld „{label}" angelegt',
    en: 'Field "{label}" created',
    fr: `Champ « {label} » créé`,
    it: 'Campo «{label}» creato',
  },
  'customization.fields.updated': {
    de: 'Feld gespeichert',
    en: 'Field saved',
    fr: 'Champ enregistré',
    it: 'Campo salvato',
  },
  'customization.fields.deleted': {
    de: 'Feld „{label}" gelöscht',
    en: 'Field "{label}" deleted',
    fr: `Champ « {label} » supprimé`,
    it: 'Campo «{label}» eliminato',
  },

  // ── Soft-Delete-Dialoge ────────────────────────────────────────────────────

  'customization.fields.deleteTitle': {
    de: 'Feld löschen?',
    en: 'Delete field?',
    fr: 'Supprimer le champ ?',
    it: 'Eliminare il campo?',
  },
  'customization.fields.deleteBody': {
    de: 'Das Feld „{label}" wird unwiderruflich entfernt.',
    en: 'The field "{label}" will be permanently removed.',
    fr: `Le champ « {label} » sera définitivement supprimé.`,
    it: 'Il campo «{label}» verrà rimosso definitivamente.',
  },
  'customization.fields.deleteInUseTitle': {
    de: 'Feld mit gespeicherten Daten löschen?',
    en: 'Delete field with stored data?',
    fr: 'Supprimer un champ avec des données enregistrées ?',
    it: 'Eliminare un campo con dati salvati?',
  },
  'customization.fields.deleteInUseBody': {
    de: 'Das Feld „{label}" enthält bereits gespeicherte Werte in bestehenden Datensätzen. Beim Löschen gehen diese Werte verloren.',
    en: 'The field "{label}" already has stored values in existing records. Deleting it will permanently remove those values.',
    fr: `Le champ « {label} » contient déjà des valeurs dans des enregistrements existants. En le supprimant, ces valeurs seront définitivement perdues.`,
    it: 'Il campo «{label}» contiene già valori in record esistenti. Eliminandolo, questi valori andranno definitivamente persi.',
  },

  // ── Audit-Action-Labels ────────────────────────────────────────────────────

  'audit.action.customization.field_created': {
    de: 'Benutzerdefiniertes Feld angelegt',
    en: 'Custom field created',
    fr: 'Champ personnalisé créé',
    it: 'Campo personalizzato creato',
  },
  'audit.action.customization.field_updated': {
    de: 'Benutzerdefiniertes Feld geändert',
    en: 'Custom field updated',
    fr: 'Champ personnalisé modifié',
    it: 'Campo personalizzato modificato',
  },
  'audit.action.customization.field_deleted': {
    de: 'Benutzerdefiniertes Feld gelöscht',
    en: 'Custom field deleted',
    fr: 'Champ personnalisé supprimé',
    it: 'Campo personalizzato eliminato',
  },
}

// ── JSON-Dateien patchen ───────────────────────────────────────────────────────

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
console.log('done — v1.1 i18n pass complete')
