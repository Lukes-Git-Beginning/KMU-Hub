/**
 * Modul-Editor E-4 (Galerie) + E-5b (Rollout-/Entwurfs-Liste) i18n pass.
 * Ausführen aus desktop/: node scripts/i18n-editor-e4-e5b.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))
const msgDir = join(here, '..', 'src', 'renderer', 'src', 'i18n', 'messages')

const ADD = {
  // E-4 gallery — dimension chips (what is customizable)
  'customization.editor.gallery.terms': {
    de: '{count, plural, one {# Begriff} other {# Begriffe}}',
    en: '{count, plural, one {# term} other {# terms}}',
    fr: '{count, plural, one {# terme} other {# termes}}',
    it: '{count, plural, one {# termine} other {# termini}}',
  },
  'customization.editor.gallery.valueSets': {
    de: '{count, plural, one {# Werteliste} other {# Wertelisten}}',
    en: '{count, plural, one {# value list} other {# value lists}}',
    fr: '{count, plural, one {# liste de valeurs} other {# listes de valeurs}}',
    it: '{count, plural, one {# elenco valori} other {# elenchi valori}}',
  },
  'customization.editor.gallery.fields': {
    de: '{count, plural, one {# Feld-Typ} other {# Feld-Typen}}',
    en: '{count, plural, one {# field type} other {# field types}}',
    fr: '{count, plural, one {# type de champ} other {# types de champ}}',
    it: '{count, plural, one {# tipo di campo} other {# tipi di campo}}',
  },
  // E-4 module-card status
  'customization.editor.status.customized': { de: 'Angepasst', en: 'Customized', fr: 'Personnalisé', it: 'Personalizzato' },
  'customization.editor.status.scheduled': { de: 'Rollout geplant', en: 'Rollout scheduled', fr: 'Déploiement planifié', it: 'Rollout pianificato' },
  'customization.editor.status.standard': { de: 'Standard', en: 'Default', fr: 'Par défaut', it: 'Predefinito' },
  // E-5b rollouts list
  'customization.editor.rollouts.title': { de: 'Rollouts & Entwürfe', en: 'Rollouts & drafts', fr: 'Déploiements et brouillons', it: 'Rollout e bozze' },
  'customization.editor.rollouts.empty': { de: 'Noch keine Entwürfe oder Rollouts.', en: 'No drafts or rollouts yet.', fr: 'Aucun brouillon ni déploiement pour l’instant.', it: 'Ancora nessuna bozza o rollout.' },
  'customization.editor.rollouts.status.draft': { de: 'Entwurf', en: 'Draft', fr: 'Brouillon', it: 'Bozza' },
  'customization.editor.rollouts.status.scheduled': { de: 'Geplant', en: 'Scheduled', fr: 'Planifié', it: 'Pianificato' },
  'customization.editor.rollouts.status.live': { de: 'Live', en: 'Live', fr: 'En ligne', it: 'Attivo' },
  'customization.editor.rollouts.status.superseded': { de: 'Ersetzt', en: 'Superseded', fr: 'Remplacé', it: 'Sostituito' },
  'customization.editor.rollouts.scheduledFor': { de: 'Geplant für {date}', en: 'Scheduled for {date}', fr: 'Planifié pour {date}', it: 'Pianificato per {date}' },
  'customization.editor.rollouts.rollback': { de: 'Zurückrollen', en: 'Roll back', fr: 'Annuler', it: 'Ripristina' },
  'customization.editor.rollouts.reopen': { de: 'Öffnen', en: 'Open', fr: 'Ouvrir', it: 'Apri' },
  'customization.editor.rollouts.rolledBack': { de: 'Zurückgerollt.', en: 'Rolled back.', fr: 'Annulé.', it: 'Ripristinato.' },
  'customization.editor.rollouts.deleted': { de: 'Entwurf gelöscht.', en: 'Draft deleted.', fr: 'Brouillon supprimé.', it: 'Bozza eliminata.' },
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
