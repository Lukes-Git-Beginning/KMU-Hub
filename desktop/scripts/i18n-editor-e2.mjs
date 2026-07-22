/**
 * Modul-Editor E-2 i18n pass — EditorFrame shell strings.
 *
 * Du-Form (Cosmi-Standard), echte Umlaute, ECHTE fr+it-Übersetzungen.
 * ICU single braces {var}, Plural als {count, plural, ...}.
 *
 * Ausführen (einmalig) aus desktop/: node scripts/i18n-editor-e2.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))
const msgDir = join(here, '..', 'src', 'renderer', 'src', 'i18n', 'messages')

const AP = '’' // typografischer Apostroph (U+2019) — gerade Apostrophe brechen den Parser

const ADD = {
  'customization.editor.a11yTitle': {
    de: 'Modul-Editor',
    en: 'Module editor',
    fr: 'Éditeur de module',
    it: 'Editor del modulo',
  },
  'customization.editor.titleBar': {
    de: '{module} bearbeiten',
    en: 'Edit {module}',
    fr: 'Modifier {module}',
    it: 'Modifica {module}',
  },
  'customization.editor.subtitle': {
    de: 'Sandbox-Vorschau · nicht live',
    en: 'Sandbox preview · not live',
    fr: 'Aperçu sandbox · hors production',
    it: 'Anteprima sandbox · non attiva',
  },
  'customization.editor.close': {
    de: 'Editor schließen',
    en: 'Close editor',
    fr: `Fermer l${AP}éditeur`,
    it: `Chiudi l${AP}editor`,
  },
  'customization.editor.undo': { de: 'Rückgängig', en: 'Undo', fr: 'Annuler', it: 'Annulla' },
  'customization.editor.redo': { de: 'Wiederholen', en: 'Redo', fr: 'Rétablir', it: 'Ripeti' },
  'customization.editor.preview': { de: 'Vorschau', en: 'Preview', fr: 'Aperçu', it: 'Anteprima' },
  'customization.editor.previewLabel': {
    de: 'Vorschau',
    en: 'Preview',
    fr: 'Aperçu',
    it: 'Anteprima',
  },
  'customization.editor.previewUnavailable': {
    de: 'Vorschau nicht verfügbar',
    en: 'Preview unavailable',
    fr: 'Aperçu indisponible',
    it: 'Anteprima non disponibile',
  },
  'customization.editor.sandboxBanner': {
    de: 'Entwurf — du bearbeitest eine Kopie. Änderungen werden erst nach dem Übernehmen für alle live.',
    en: `Draft — you${AP}re editing a copy. Changes go live for everyone only after you apply them.`,
    fr: `Brouillon — tu modifies une copie. Les changements ne seront actifs pour tous qu${AP}après leur application.`,
    it: 'Bozza — stai modificando una copia. Le modifiche diventano attive per tutti solo dopo averle applicate.',
  },
  'customization.editor.nav.label': {
    de: 'Anpassen',
    en: 'Customize',
    fr: 'Personnaliser',
    it: 'Personalizza',
  },
  'customization.editor.nav.fields': { de: 'Felder', en: 'Fields', fr: 'Champs', it: 'Campi' },
  'customization.editor.nav.terms': { de: 'Begriffe', en: 'Terms', fr: 'Termes', it: 'Termini' },
  'customization.editor.nav.valueSets': {
    de: 'Wertelisten',
    en: 'Value lists',
    fr: 'Listes de valeurs',
    it: 'Elenchi di valori',
  },
  'customization.editor.props.empty': {
    de: 'Wähle links Felder, Begriffe oder Wertelisten, um dieses Modul anzupassen.',
    en: 'Pick Fields, Terms or Value lists on the left to customize this module.',
    fr: 'Choisis Champs, Termes ou Listes de valeurs à gauche pour personnaliser ce module.',
    it: 'Scegli Campi, Termini o Elenchi di valori a sinistra per personalizzare questo modulo.',
  },
  'customization.editor.props.pickElement': {
    de: 'Wähle ein Element, um es anzupassen.',
    en: 'Select an element to customize it.',
    fr: 'Sélectionne un élément pour le personnaliser.',
    it: 'Seleziona un elemento per personalizzarlo.',
  },
  'customization.editor.props.fieldsDesc': {
    de: 'Eigene Felder für dieses Modul anlegen und verwalten.',
    en: 'Create and manage custom fields for this module.',
    fr: 'Crée et gère des champs personnalisés pour ce module.',
    it: 'Crea e gestisci campi personalizzati per questo modulo.',
  },
  'customization.editor.props.termsDesc': {
    de: 'Begriffe dieses Moduls in eure Sprache umbenennen.',
    en: `Rename this module${AP}s terms to your own wording.`,
    fr: 'Renomme les termes de ce module dans ton vocabulaire.',
    it: 'Rinomina i termini di questo modulo con le tue parole.',
  },
  'customization.editor.props.valueSetsDesc': {
    de: 'Auswahllisten und Status-Werte dieses Moduls anpassen.',
    en: `Adjust this module${AP}s pick lists and status values.`,
    fr: 'Ajuste les listes de choix et les statuts de ce module.',
    it: 'Regola gli elenchi di scelta e gli stati di questo modulo.',
  },
  'customization.editor.footer.changes': {
    de: '{count, plural, =0 {Keine Änderungen} one {# Änderung} other {# Änderungen}}',
    en: '{count, plural, =0 {No changes} one {# change} other {# changes}}',
    fr: '{count, plural, =0 {Aucune modification} one {# modification} other {# modifications}}',
    it: '{count, plural, =0 {Nessuna modifica} one {# modifica} other {# modifiche}}',
  },
  'customization.editor.footer.saveDraft': {
    de: 'Als Entwurf speichern',
    en: 'Save as draft',
    fr: 'Enregistrer comme brouillon',
    it: 'Salva come bozza',
  },
  'customization.editor.footer.apply': {
    de: 'Übernehmen',
    en: 'Apply',
    fr: 'Appliquer',
    it: 'Applica',
  },
  'customization.editor.draftName': {
    de: '{module} — Anpassung',
    en: '{module} — customization',
    fr: '{module} — personnalisation',
    it: '{module} — personalizzazione',
  },
  'customization.editor.toast.draftSaved': {
    de: 'Als Entwurf gespeichert.',
    en: 'Saved as draft.',
    fr: 'Enregistré comme brouillon.',
    it: 'Salvato come bozza.',
  },
  'customization.editor.toast.applied': {
    de: 'Änderungen übernommen.',
    en: 'Changes applied.',
    fr: 'Modifications appliquées.',
    it: 'Modifiche applicate.',
  },
  'customization.editor.launch.title': {
    de: 'Modul-Editor',
    en: 'Module editor',
    fr: 'Éditeur de module',
    it: 'Editor del modulo',
  },
  'customization.editor.launch.beta': { de: 'Beta', en: 'Beta', fr: 'Bêta', it: 'Beta' },
  'customization.editor.launch.subtitle': {
    de: 'Öffne ein Modul im Editor, um seine Felder, Begriffe und Wertelisten anzupassen — als Vorschau, ohne das Live-System zu berühren.',
    en: 'Open a module in the editor to customize its fields, terms and value lists — as a preview, without touching the live system.',
    fr: `Ouvre un module dans l${AP}éditeur pour personnaliser ses champs, termes et listes de valeurs — en aperçu, sans toucher au système en production.`,
    it: `Apri un modulo nell${AP}editor per personalizzare campi, termini ed elenchi di valori — in anteprima, senza toccare il sistema attivo.`,
  },
  'customization.editor.launch.open': {
    de: 'Editor öffnen',
    en: 'Open editor',
    fr: `Ouvrir l${AP}éditeur`,
    it: `Apri l${AP}editor`,
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
