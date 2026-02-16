import { useEffect } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'

/**
 * Global keyboard shortcuts for the app.
 *
 * Ctrl+K  — Search (handled by SearchBar itself)
 * Ctrl+,  — Open settings
 * Escape  — Handled by individual dialogs/panels
 */
export function useKeyboardShortcuts() {
  const navigate = useNavigate()
  const location = useLocation()

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      const isCtrl = e.ctrlKey || e.metaKey
      const target = e.target as HTMLElement
      const isInput = target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable

      // Don't trigger shortcuts while typing in inputs (except Escape)
      if (isInput && e.key !== 'Escape') return

      // Ctrl+, — Settings
      if (isCtrl && e.key === ',') {
        e.preventDefault()
        if (location.pathname !== '/einstellungen') {
          navigate('/einstellungen')
        }
        return
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [navigate, location.pathname])
}
