/**
 * Modul-Editor E-5 i18n pass — Deploy-Dialog (jetzt / terminiert / entwurf).
 * Ausführen aus desktop/: node scripts/i18n-editor-e5.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))
const msgDir = join(here, '..', 'src', 'renderer', 'src', 'i18n', 'messages')

const ADD = {
  'customization.editor.deploy.title': {
    de: 'Änderungen übernehmen',
    en: 'Apply changes',
    fr: 'Appliquer les modifications',
    it: 'Applica le modifiche',
  },
  'customization.editor.deploy.affects': {
    de: 'Betrifft alle Nutzer eurer Firma.',
    en: 'Affects everyone in your company.',
    fr: 'Concerne tous les utilisateurs de ton entreprise.',
    it: 'Riguarda tutti gli utenti della tua azienda.',
  },
  'customization.editor.deploy.modeNow': { de: 'Jetzt', en: 'Now', fr: 'Maintenant', it: 'Adesso' },
  'customization.editor.deploy.modeNowHint': {
    de: 'Sofort für alle live.',
    en: 'Live for everyone right away.',
    fr: 'En ligne pour tous immédiatement.',
    it: 'Attivo subito per tutti.',
  },
  'customization.editor.deploy.modeScheduled': {
    de: 'Terminiert',
    en: 'Scheduled',
    fr: 'Planifié',
    it: 'Pianificato',
  },
  'customization.editor.deploy.modeScheduledHint': {
    de: 'An einem festen Tag ausrollen.',
    en: 'Roll out on a set day.',
    fr: 'Déployer un jour précis.',
    it: 'Distribuisci in un giorno preciso.',
  },
  'customization.editor.deploy.scheduleLabel': {
    de: 'Datum & Uhrzeit',
    en: 'Date & time',
    fr: 'Date et heure',
    it: 'Data e ora',
  },
  'customization.editor.deploy.announceLabel': {
    de: 'Ankündigung (optional)',
    en: 'Announcement (optional)',
    fr: 'Annonce (facultatif)',
    it: 'Annuncio (facoltativo)',
  },
  'customization.editor.deploy.announcePlaceholder': {
    de: 'z. B. Neue Begriffe ab Montag',
    en: 'e.g. New terms from Monday',
    fr: 'p. ex. Nouveaux termes dès lundi',
    it: 'es. Nuovi termini da lunedì',
  },
  'customization.editor.deploy.cancel': {
    de: 'Abbrechen',
    en: 'Cancel',
    fr: 'Annuler',
    it: 'Annulla',
  },
  'customization.editor.deploy.confirmNow': {
    de: 'Jetzt übernehmen',
    en: 'Apply now',
    fr: 'Appliquer maintenant',
    it: 'Applica adesso',
  },
  'customization.editor.deploy.confirmScheduled': {
    de: 'Rollout planen',
    en: 'Schedule rollout',
    fr: 'Planifier le déploiement',
    it: 'Pianifica il rilascio',
  },
  'customization.editor.deploy.toastScheduled': {
    de: 'Rollout geplant.',
    en: 'Rollout scheduled.',
    fr: 'Déploiement planifié.',
    it: 'Rilascio pianificato.',
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
