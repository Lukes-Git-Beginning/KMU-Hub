// One-off: add CRM-settings-panel i18n keys to all 4 locales (append-only).
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const __dirname = dirname(fileURLToPath(import.meta.url))
const MSG_DIR = join(__dirname, '..', 'src', 'renderer', 'src', 'i18n', 'messages')

const additions = {
  de: {
    'moduleSettings.entries.crm': 'CRM',
    'crm.settings.title': 'CRM-Einstellungen',
    'crm.settings.subtitle': 'Pipeline, eigene Felder und CRM-Konfiguration für das ganze Team.',
    'crm.settings.pipeline.title': 'Pipeline-Phasen',
    'crm.settings.pipeline.desc': 'Phasen, Reihenfolge, Abschlusswahrscheinlichkeit und Farben der Verkaufspipeline.',
    'crm.settings.pipeline.newStage': 'Neue Phase',
    'crm.settings.pipeline.namePlaceholder': 'Phasenname',
    'crm.settings.pipeline.addButton': 'Phase hinzufügen',
    'crm.settings.pipeline.added': 'Phase „{name}" hinzugefügt',
    'crm.settings.pipeline.deleted': 'Phase „{name}" gelöscht',
    'crm.settings.pipeline.deleteTitle': 'Phase löschen?',
    'crm.settings.pipeline.deleteDescription': 'Soll die Phase „{name}" wirklich gelöscht werden? Deals in dieser Phase verlieren ihre Zuordnung.',
    'crm.settings.pipeline.moveUp': 'Nach oben',
    'crm.settings.pipeline.moveDown': 'Nach unten',
    'crm.settings.pipeline.won': 'Gewonnen-Phase',
    'crm.settings.pipeline.lost': 'Verloren-Phase',
    'crm.settings.pipeline.dealCount': '{count} Deals',
    'crm.settings.customFields.title': 'Eigene Felder',
    'crm.settings.customFields.desc': 'Zusätzliche Felder für Kontakte definieren und verwalten.',
  },
  en: {
    'moduleSettings.entries.crm': 'CRM',
    'crm.settings.title': 'CRM settings',
    'crm.settings.subtitle': 'Pipeline, custom fields and CRM configuration for the whole team.',
    'crm.settings.pipeline.title': 'Pipeline stages',
    'crm.settings.pipeline.desc': 'Stages, order, win probability and colours of the sales pipeline.',
    'crm.settings.pipeline.newStage': 'New stage',
    'crm.settings.pipeline.namePlaceholder': 'Stage name',
    'crm.settings.pipeline.addButton': 'Add stage',
    'crm.settings.pipeline.added': 'Stage "{name}" added',
    'crm.settings.pipeline.deleted': 'Stage "{name}" deleted',
    'crm.settings.pipeline.deleteTitle': 'Delete stage?',
    'crm.settings.pipeline.deleteDescription': 'Really delete the stage "{name}"? Deals in this stage lose their assignment.',
    'crm.settings.pipeline.moveUp': 'Move up',
    'crm.settings.pipeline.moveDown': 'Move down',
    'crm.settings.pipeline.won': 'Won stage',
    'crm.settings.pipeline.lost': 'Lost stage',
    'crm.settings.pipeline.dealCount': '{count} deals',
    'crm.settings.customFields.title': 'Custom fields',
    'crm.settings.customFields.desc': 'Define and manage additional contact fields.',
  },
  fr: {
    'moduleSettings.entries.crm': 'CRM',
    'crm.settings.title': 'Paramètres CRM',
    'crm.settings.subtitle': "Pipeline, champs personnalisés et configuration CRM pour toute l'équipe.",
    'crm.settings.pipeline.title': 'Étapes du pipeline',
    'crm.settings.pipeline.desc': 'Étapes, ordre, probabilité de réussite et couleurs du pipeline commercial.',
    'crm.settings.pipeline.newStage': 'Nouvelle étape',
    'crm.settings.pipeline.namePlaceholder': "Nom de l'étape",
    'crm.settings.pipeline.addButton': 'Ajouter une étape',
    'crm.settings.pipeline.added': 'Étape « {name} » ajoutée',
    'crm.settings.pipeline.deleted': 'Étape « {name} » supprimée',
    'crm.settings.pipeline.deleteTitle': "Supprimer l'étape ?",
    'crm.settings.pipeline.deleteDescription': "Supprimer vraiment l'étape « {name} » ? Les affaires de cette étape perdent leur affectation.",
    'crm.settings.pipeline.moveUp': 'Monter',
    'crm.settings.pipeline.moveDown': 'Descendre',
    'crm.settings.pipeline.won': 'Étape gagnée',
    'crm.settings.pipeline.lost': 'Étape perdue',
    'crm.settings.pipeline.dealCount': '{count} affaires',
    'crm.settings.customFields.title': 'Champs personnalisés',
    'crm.settings.customFields.desc': 'Définir et gérer des champs de contact supplémentaires.',
  },
  it: {
    'moduleSettings.entries.crm': 'CRM',
    'crm.settings.title': 'Impostazioni CRM',
    'crm.settings.subtitle': 'Pipeline, campi personalizzati e configurazione CRM per tutto il team.',
    'crm.settings.pipeline.title': 'Fasi della pipeline',
    'crm.settings.pipeline.desc': 'Fasi, ordine, probabilità di successo e colori della pipeline di vendita.',
    'crm.settings.pipeline.newStage': 'Nuova fase',
    'crm.settings.pipeline.namePlaceholder': 'Nome della fase',
    'crm.settings.pipeline.addButton': 'Aggiungi fase',
    'crm.settings.pipeline.added': 'Fase «{name}» aggiunta',
    'crm.settings.pipeline.deleted': 'Fase «{name}» eliminata',
    'crm.settings.pipeline.deleteTitle': 'Eliminare la fase?',
    'crm.settings.pipeline.deleteDescription': 'Eliminare davvero la fase «{name}»? Le trattative in questa fase perdono l\'assegnazione.',
    'crm.settings.pipeline.moveUp': 'Su',
    'crm.settings.pipeline.moveDown': 'Giù',
    'crm.settings.pipeline.won': 'Fase vinta',
    'crm.settings.pipeline.lost': 'Fase persa',
    'crm.settings.pipeline.dealCount': '{count} trattative',
    'crm.settings.customFields.title': 'Campi personalizzati',
    'crm.settings.customFields.desc': 'Definisci e gestisci campi di contatto aggiuntivi.',
  },
}

for (const [locale, keys] of Object.entries(additions)) {
  const file = join(MSG_DIR, `${locale}.json`)
  const data = JSON.parse(readFileSync(file, 'utf8'))
  let added = 0
  for (const [k, v] of Object.entries(keys)) {
    if (!(k in data)) added++
    if (!(k in data)) data[k] = v
  }
  writeFileSync(file, JSON.stringify(data, null, 2) + '\n', 'utf8')
  console.log(`${locale}.json: +${added} keys (total ${Object.keys(data).length})`)
}
