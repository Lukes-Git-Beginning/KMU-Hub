/**
 * I18nProvider -- wraps react-i18next I18nextProvider with locale sync.
 *
 * Initializes i18next on first render, synchronizes the Zustand locale
 * store with i18next's language, and provides the i18n context.
 *
 * initI18n is idempotent (checks i18n.isInitialized internally), so calling
 * it during render is safe and ensures i18n is ready before the first paint.
 */
import { useEffect, useMemo, type ReactNode } from 'react'
import { I18nextProvider } from 'react-i18next'
import { useLocale } from '@/hooks/useLocale'
import { i18n, initI18n } from '@/i18n/i18n'

interface I18nProviderProps {
  children: ReactNode
}

export function I18nProvider({ children }: I18nProviderProps) {
  const { locale } = useLocale()

  // Initialize i18n exactly once before the first paint.
  // useMemo runs during render but avoids the ref-during-render lint error.
  // initI18n is idempotent, so subsequent renders are no-ops.
  useMemo(() => {
    initI18n(locale)
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

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
