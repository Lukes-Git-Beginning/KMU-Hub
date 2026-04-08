/**
 * Standalone page for the employee wizard pop-out Electron BrowserWindow.
 * Reads initial form data from localStorage (saved by the inline wizard).
 */
import { useState, useEffect } from 'react'
import { CreateEmployeeWizard } from './CreateEmployeeWizard'

export default function EmployeeWizardWindowPage() {
  const [initialData, setInitialData] = useState<Record<string, unknown> | undefined>(undefined)
  const [ready, setReady] = useState(false)

   
  useEffect(() => {
    try {
      const raw = localStorage.getItem('cosmi-employee-wizard-draft')
      if (raw) {
        // eslint-disable-next-line react-hooks/set-state-in-effect -- sync form fields from prop/API data
        setInitialData(JSON.parse(raw))
        localStorage.removeItem('cosmi-employee-wizard-draft')
      }
    } catch { /* ignore parse errors */ }
    setReady(true)
  }, [])

  if (!ready) return null

  return (
    <CreateEmployeeWizard
      onClose={() => window.electronAPI?.window.close()}
      isWindow
      initialData={initialData as never}
    />
  )
}
