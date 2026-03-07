/**
 * I18nProvider -- wraps react-intl IntlProvider with locale detection.
 *
 * Loads translation messages for the effective locale (user choice ->
 * browser detection -> DE fallback) and provides them to the entire
 * app via IntlProvider.
 *
 * Usage in App.tsx:
 *   <I18nProvider>
 *     <RouterProvider router={router} />
 *   </I18nProvider>
 */
import { useMemo, type ReactNode } from 'react'
import { IntlProvider } from 'react-intl'
import { useLocale } from '@/hooks/useLocale'
import { getFormats } from '@/i18n/formats'
import type { SupportedLocale } from '@/stores/locale'

// Static imports for all locale message bundles.
// Using static imports ensures all translations are bundled and available
// immediately, avoiding async loading complexity for 4 small JSON files.
import messagesDE from '@/i18n/messages/de.json'
import messagesEN from '@/i18n/messages/en.json'
import messagesFR from '@/i18n/messages/fr.json'
import messagesIT from '@/i18n/messages/it.json'

// ---------------------------------------------------------------------------
// Message registry
// ---------------------------------------------------------------------------

const allMessages: Record<SupportedLocale, Record<string, string>> = {
  de: messagesDE,
  en: messagesEN,
  fr: messagesFR,
  it: messagesIT,
}

// ---------------------------------------------------------------------------
// Error handling
// ---------------------------------------------------------------------------

const IS_DEV = import.meta.env.DEV

/**
 * Custom error handler for react-intl.
 * Suppresses MISSING_TRANSLATION warnings in development (common during
 * incremental translation work). Logs all other errors.
 */
function handleIntlError(err: Error): void {
  if (IS_DEV && err.message?.includes('MISSING_TRANSLATION')) {
    // Suppress in development -- translations are added incrementally
    return
  }
   
  console.error('[i18n]', err.message)
}

// ---------------------------------------------------------------------------
// Provider
// ---------------------------------------------------------------------------

interface I18nProviderProps {
  children: ReactNode
}

/**
 * Provides internationalization context to the app.
 *
 * Reads the effective locale from the locale store/hook, loads the
 * corresponding message bundle, and wraps children in IntlProvider.
 * German (de) is the default locale and the reference language.
 */
export function I18nProvider({ children }: I18nProviderProps) {
  const { locale } = useLocale()

  const messages = allMessages[locale] ?? allMessages.de
  const formats = useMemo(() => getFormats(locale), [locale])

  return (
    <IntlProvider
      locale={locale}
      defaultLocale="de"
      messages={messages}
      formats={formats}
      onError={handleIntlError}
    >
      {children}
    </IntlProvider>
  )
}
