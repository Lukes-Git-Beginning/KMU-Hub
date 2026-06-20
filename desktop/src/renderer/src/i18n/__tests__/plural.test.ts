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

// i18next is configured with the ICU plugin and WITHOUT native suffix-plural
// resolution (no compatibilityJSON). A key like "foo.bar_one" therefore never
// resolves on a t('foo.bar', { count }) call and renders the raw key string.
// Every plural MUST be a single ICU base key ("{count, plural, ...}").
const PLURAL_SUFFIX = /_(zero|one|two|few|many|other)$/
function getSuffixPluralKeys(messages: Messages): string[] {
  return Object.keys(messages).filter((k) => PLURAL_SUFFIX.test(k))
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

describe('no native suffix-plural keys (ICU base keys only)', () => {
  for (const [lang, messages] of Object.entries(locales)) {
    it(`${lang}.json has no _one/_other-style suffix keys`, () => {
      const suffixKeys = getSuffixPluralKeys(messages)
      expect(
        suffixKeys,
        `These keys use native plural suffixes and will render raw — use a single ICU "{count, plural, ...}" base key instead: ${suffixKeys.join(', ')}`,
      ).toEqual([])
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

  it('previously-raw plural keys now resolve (E1 regression)', async () => {
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

    expect(instance.t('admin.workflows.ruleCount', { count: 1 })).toBe('1 Workflow-Regel')
    expect(instance.t('admin.workflows.ruleCount', { count: 5 })).toBe('5 Workflow-Regeln')
    expect(instance.t('admin.validation.ruleCount', { count: 1 })).toBe('1 Validierungsregel')
    expect(instance.t('security.dsar.result.entryCount', { count: 1 })).toBe('1 Eintrag')
    expect(instance.t('security.dsar.result.entryCount', { count: 9 })).toBe('9 Einträge')
    expect(instance.t('kontakte.groups.memberCount', { count: 2 })).toBe('2 Kontakte')
    expect(instance.t('shared.editor.wordCount', { count: 1 })).toBe('1 Wort')
  })
})
