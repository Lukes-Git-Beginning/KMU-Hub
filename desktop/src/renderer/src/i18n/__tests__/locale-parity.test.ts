import { describe, it, expect } from 'vitest'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'
import messagesDE from '@/i18n/messages/de.json'
import messagesEN from '@/i18n/messages/en.json'
import messagesFR from '@/i18n/messages/fr.json'
import messagesIT from '@/i18n/messages/it.json'

type Messages = Record<string, string>

const LOCALES = ['de', 'en', 'fr', 'it'] as const
type Locale = (typeof LOCALES)[number]

const messages: Record<Locale, Messages> = {
  de: messagesDE as Messages,
  en: messagesEN as Messages,
  fr: messagesFR as Messages,
  it: messagesIT as Messages,
}

const messagesDir = join(dirname(fileURLToPath(import.meta.url)), '..', 'messages')

// de is the source language: every key originates there and the others follow.
const REFERENCE: Locale = 'de'

describe('locale parity', () => {
  const referenceKeys = Object.keys(messages[REFERENCE])

  it.each(LOCALES.filter((l) => l !== REFERENCE))(
    '%s covers every key in de',
    (locale) => {
      const keys = new Set(Object.keys(messages[locale]))
      const missing = referenceKeys.filter((k) => !keys.has(k))

      // Name them: a bare count tells you nothing about what the user sees,
      // and untranslated keys render as the raw key in the UI.
      expect(missing, `${locale}.json is missing ${missing.length} key(s)`).toEqual([])
    },
  )

  it.each(LOCALES.filter((l) => l !== REFERENCE))(
    '%s carries no keys that de does not have',
    (locale) => {
      const reference = new Set(referenceKeys)
      const orphans = Object.keys(messages[locale]).filter((k) => !reference.has(k))

      // An orphan is either a typo or a leftover from a removed feature —
      // both are dead weight that never renders.
      expect(orphans, `${locale}.json has ${orphans.length} orphan key(s)`).toEqual([])
    },
  )

  it.each(LOCALES)('%s has no empty values', (locale) => {
    const empty = Object.entries(messages[locale])
      .filter(([, v]) => typeof v === 'string' && v.trim() === '')
      .map(([k]) => k)

    expect(empty, `${locale}.json has ${empty.length} empty value(s)`).toEqual([])
  })
})

describe('locale encoding', () => {
  // A UTF-8 BOM survives naive parsing into the first key name, so a lookup of
  // that key silently misses. All four files carried one until 2026-08-11.
  it.each(LOCALES)('%s.json has no UTF-8 BOM', (locale) => {
    const bytes = readFileSync(join(messagesDir, `${locale}.json`))
    const hasBOM = bytes[0] === 0xef && bytes[1] === 0xbb && bytes[2] === 0xbf

    expect(hasBOM, `${locale}.json starts with a UTF-8 BOM`).toBe(false)
  })
})
