/**
 * I18nProvider -- wraps react-i18next I18nextProvider with locale sync.
 *
 * Initializes i18next on first render, synchronizes the Zustand locale
 * store with i18next's language, and provides the i18n context.
 */
import { useEffect, useRef, type ReactNode } from 'react'
import { I18nextProvider } from 'react-i18next'
import { useLocale } from '@/hooks/useLocale'
import { i18n, initI18n } from '@/i18n/i18n'

interface I18nProviderProps {
  children: ReactNode
}

export function I18nProvider({ children }: I18nProviderProps) {
  const { locale } = useLocale()
  const initialized = useRef(false)

  if (!initialized.current) {
    initI18n(locale)
    initialized.current = true
  }

  useEffect(() => {
    if (i18n.language !== locale) {
      i18n.changeLanguage(locale)
    }
  }, [locale])

  return (
    <I18nextProvider i18n={i18n}>
      {children}
    </I18nextProvider>
  )
}
