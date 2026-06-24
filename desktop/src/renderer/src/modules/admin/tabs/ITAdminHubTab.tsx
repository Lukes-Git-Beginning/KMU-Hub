/**
 * ITAdminHubTab — Admin-Hub IT-Tab.
 *
 * Rendert den bestehenden ITAdminTab-Inhalt (E-Mail, Sicherheits-Policy,
 * Berechtigungen, Signatur). Branding wurde in den dedizierten Branding-Tab
 * ausgelagert (A-4, keine Dublette mehr). Die „Berechtigungen"-Sektion in
 * ITAdminTab ist durch den Rollen-Tab (A-2) abgelöst — die settings-Lane sollte
 * sie entfernen.
 */
import { ITAdminTab } from '@/modules/settings/tabs/ITAdminTab'

export default function ITAdminHubTab() {
  return (
    <div className="px-6 py-6">
      <ITAdminTab />
    </div>
  )
}
