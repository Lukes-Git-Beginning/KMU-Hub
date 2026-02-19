/**
 * Language settings tab: locale picker with format previews.
 *
 * Provides locale switching (DE/EN/FR/IT), auto-detect option,
 * and real-time date/number format previews using react-intl.
 * Wired to useLocale hook for persisted locale state.
 */
import { Check, Globe } from 'lucide-react'
import { FormattedMessage, FormattedDate, FormattedNumber } from 'react-intl'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useLocale, LOCALE_DISPLAY_NAMES, AVAILABLE_LOCALES } from '@/hooks/useLocale'
import type { SupportedLocale } from '@/stores/locale'

/** Flag labels for locale picker. */
const LOCALE_FLAGS: Record<SupportedLocale, string> = {
  de: 'DE',
  en: 'EN',
  fr: 'FR',
  it: 'IT',
}

export function LanguageSettingsTab() {
  const { locale, setLocale, resetToAuto, isAutoDetected } = useLocale()

  /** Detect the browser's preferred language for display. */
  const browserLang = (() => {
    try {
      const raw = navigator.language
      const code = raw.split('-')[0].toLowerCase()
      return LOCALE_DISPLAY_NAMES[code as SupportedLocale] ?? raw
    } catch {
      return 'Unknown'
    }
  })()

  const now = new Date()

  return (
    <div className="mx-auto max-w-2xl px-6 py-6 space-y-8">
      <div>
        <h2 className="text-2xl font-semibold text-foreground">
          <FormattedMessage id="settings.language.title" />
        </h2>
        <p className="mt-1 text-sm text-muted-foreground">
          <FormattedMessage id="settings.language.subtitle" />
        </p>
      </div>

      {/* Language Picker */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-lg">
            <Globe className="h-5 w-5" />
            <FormattedMessage id="settings.language.title" />
          </CardTitle>
          <CardDescription>
            <FormattedMessage
              id="settings.language.current"
              values={{ language: LOCALE_DISPLAY_NAMES[locale] }}
            />
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          {/* Auto-detect option */}
          <button
            onClick={resetToAuto}
            className={`flex w-full items-center gap-3 rounded-lg border p-3 transition-colors ${
              isAutoDetected
                ? 'border-primary bg-primary/5'
                : 'border-border hover:bg-accent'
            }`}
          >
            <span className="flex h-8 w-8 items-center justify-center rounded-full bg-muted text-xs font-medium text-muted-foreground">
              A
            </span>
            <span className="flex-1 text-left text-sm text-foreground">
              <FormattedMessage id="settings.language.auto" />
            </span>
            {isAutoDetected && <Check className="h-4 w-4 text-primary" />}
          </button>

          {/* Explicit locale choices */}
          {AVAILABLE_LOCALES.map((loc) => {
            const isSelected = !isAutoDetected && locale === loc.code
            return (
              <button
                key={loc.code}
                onClick={() => setLocale(loc.code as SupportedLocale)}
                className={`flex w-full items-center gap-3 rounded-lg border p-3 transition-colors ${
                  isSelected
                    ? 'border-primary bg-primary/5'
                    : 'border-border hover:bg-accent'
                }`}
              >
                <span className="flex h-8 w-8 items-center justify-center rounded-full bg-muted text-xs font-medium text-foreground">
                  {LOCALE_FLAGS[loc.code as SupportedLocale]}
                </span>
                <span className="flex-1 text-left text-sm text-foreground">
                  {loc.name}
                </span>
                {isSelected && <Check className="h-4 w-4 text-primary" />}
              </button>
            )
          })}
        </CardContent>
      </Card>

      {/* Format Previews */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">
            <FormattedMessage id="settings.language.datePreview" />
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="rounded-lg border border-border p-3">
              <p className="text-xs text-muted-foreground mb-1">
                <FormattedMessage id="settings.language.datePreview" />
              </p>
              <p className="text-sm font-medium text-foreground">
                <FormattedDate
                  value={now}
                  year="numeric"
                  month="long"
                  day="numeric"
                  weekday="long"
                />
              </p>
            </div>
            <div className="rounded-lg border border-border p-3">
              <p className="text-xs text-muted-foreground mb-1">
                <FormattedMessage id="settings.language.numberPreview" />
              </p>
              <p className="text-sm font-medium text-foreground">
                <FormattedNumber value={1234567.89} style="decimal" />
              </p>
            </div>
          </div>

          <div className="rounded-lg border border-border p-3">
            <p className="text-xs text-muted-foreground mb-1">
              <FormattedMessage id="settings.language.browserLanguage" />
            </p>
            <p className="text-sm text-foreground">{browserLang}</p>
          </div>
        </CardContent>
      </Card>

      {/* Reset */}
      {!isAutoDetected && (
        <Button variant="outline" onClick={resetToAuto}>
          <FormattedMessage id="settings.language.auto" />
        </Button>
      )}
    </div>
  )
}
