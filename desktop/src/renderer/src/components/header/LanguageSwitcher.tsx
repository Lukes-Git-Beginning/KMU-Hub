import { useState, useRef, useEffect } from 'react'
import { Globe } from 'lucide-react'
import { useUIStore } from '@/stores/ui'
import { cn } from '@/lib/cn'
import { useTranslation } from 'react-i18next'

interface Language {
  code: string
  name: string
  flag: string
  disabled?: boolean
}

const languages: Language[] = [
  { code: 'de', name: 'Deutsch', flag: '\uD83C\uDDE9\uD83C\uDDEA' },
  { code: 'fr', name: 'Fran\u00E7ais', flag: '\uD83C\uDDEB\uD83C\uDDF7', disabled: true },
  { code: 'it', name: 'Italiano', flag: '\uD83C\uDDEE\uD83C\uDDF9' },
  { code: 'en', name: 'English', flag: '\uD83C\uDDEC\uD83C\uDDE7', disabled: true },
]

export function LanguageSwitcher() {
  const [isOpen, setIsOpen] = useState(false)
  const dropdownRef = useRef<HTMLDivElement>(null)
  const { t } = useTranslation()
  const locale = useUIStore((s) => s.locale)
  const setLocale = useUIStore((s) => s.setLocale)

  const currentLang = languages.find((l) => l.code === locale) ?? languages[0]

   
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node)
      ) {
        setIsOpen(false)
      }
    }

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside)
      return () =>
        document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [isOpen])

  const handleSelect = (lang: Language) => {
    if (lang.disabled) return
    setLocale(lang.code)
    setIsOpen(false)
  }

  return (
    <div className="relative" ref={dropdownRef}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center gap-2 px-3 py-2 hover:bg-accent rounded-lg transition-colors"
      >
        <Globe className="h-4 w-4 text-muted-foreground" />
        <span className="text-sm text-foreground hidden lg:inline">
          {currentLang.flag}
        </span>
      </button>

      {isOpen && (
        <div className="absolute right-0 top-full mt-2 w-48 bg-card border border-border rounded-lg shadow-lg z-50 overflow-hidden">
          {languages.map((lang) => (
            <button
              key={lang.code}
              onClick={() => handleSelect(lang)}
              disabled={lang.disabled}
              className={cn(
                'w-full flex items-center gap-2 px-4 py-2.5 text-sm text-left transition-colors',
                lang.disabled
                  ? 'opacity-50 cursor-not-allowed'
                  : 'hover:bg-accent cursor-pointer',
                lang.code === locale && 'bg-accent font-medium',
              )}
            >
              <span>{lang.flag}</span>
              <span className="text-foreground">{lang.name}</span>
              {lang.disabled && (
                <span className="ml-auto text-xs text-muted-foreground">
                  {t('header.languageSwitcher.comingSoon')}
                </span>
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
