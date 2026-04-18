import { describe, it, expect } from 'vitest'
import messagesDE from '@/i18n/messages/de.json'
import messagesEN from '@/i18n/messages/en.json'
import messagesFR from '@/i18n/messages/fr.json'
import messagesIT from '@/i18n/messages/it.json'

type Messages = Record<string, string>

const locales: Record<string, Messages> = {
  de: messagesDE as Messages,
  en: messagesEN as Messages,
  fr: messagesFR as Messages,
  it: messagesIT as Messages,
}

function getICUPluralEntries(messages: Messages): [string, string][] {
  return Object.entries(messages).filter(
    ([, v]) => typeof v === 'string' && v.includes(', plural,'),
  ) as [string, string][]
}

function bracesBalanced(s: string): boolean {
  return (s.match(/\{/g) || []).length === (s.match(/\}/g) || []).length
}

describe('ICU plural syntax — brace balance', () => {
  for (const [lang, messages] of Object.entries(locales)) {
    const pluralEntries = getICUPluralEntries(messages)

    describe(`${lang}.json (${pluralEntries.length} plural keys)`, () => {
      it('has at least 1 plural key', () => {
        expect(pluralEntries.length).toBeGreaterThan(0)
      })

      for (const [key, value] of pluralEntries) {
        it(`"${key}" has balanced braces`, () => {
          expect(bracesBalanced(value), `Unbalanced braces in: ${value}`).toBe(true)
        })
      }
    })
  }
})

describe('ICU plural rendering via i18next-icu', () => {
  it('common.results renders correctly for count=0, 1, 5', async () => {
    const i18next = await import('i18next')
    const ICU = await import('i18next-icu')
    const instance = i18next.createInstance()
    await instance.use(ICU.default).init({
      lng: 'de',
      keySeparator: false,
      nsSeparator: false,
      interpolation: { escapeValue: false },
      resources: { de: { translation: messagesDE as Messages } },
    })

    expect(instance.t('common.results', { count: 0 })).toBe('Keine Ergebnisse')
    expect(instance.t('common.results', { count: 1 })).toBe('1 Ergebnis')
    expect(instance.t('common.results', { count: 5 })).toBe('5 Ergebnisse')
  })

  it('audit.retentionYears renders correctly for count=1 and count=2', async () => {
    const i18next = await import('i18next')
    const ICU = await import('i18next-icu')
    const instance = i18next.createInstance()
    await instance.use(ICU.default).init({
      lng: 'de',
      keySeparator: false,
      nsSeparator: false,
      interpolation: { escapeValue: false },
      resources: { de: { translation: messagesDE as Messages } },
    })

    expect(instance.t('audit.retentionYears', { years: 1 })).toBe('1 Jahr')
    expect(instance.t('audit.retentionYears', { years: 3 })).toBe('3 Jahre')
  })
})
