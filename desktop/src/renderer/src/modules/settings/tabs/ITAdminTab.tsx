/**
 * IT-Admin Settings Tab
 *
 * Per-employee email config, security policies (PW expiry, 2FA),
 * permission management, and email signature editor.
 */
import { useState } from 'react'
import {
  Server,
  Shield,
  Users,
  Mail,
  Save,
  ChevronDown,
  ChevronRight,
  Eye,
  Pencil,
} from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Separator } from '@/components/ui/separator'
import { toast } from 'sonner'
import { EMPLOYEES } from '@/mocks/mock-db'

// Mock employee list from mock-db
const employees = EMPLOYEES.slice(0, 8).map((e) => ({
  id: e.id,
  name: `${e.firstName} ${e.lastName}`,
  email: e.email,
  initials: e.initials,
}))

// Mock signature template
const DEFAULT_SIGNATURE_TEMPLATE = `<div style="font-family: Arial, sans-serif; font-size: 13px; color: #333;">
  <p style="margin: 0; font-weight: 600;">{{name}}</p>
  <p style="margin: 0; color: #666;">{{position}} | {{department}}</p>
  <p style="margin: 8px 0 0; font-size: 12px; color: #888;">
    {{company}} · {{phone}} · {{email}}
  </p>
  <p style="margin: 4px 0 0; font-size: 11px; color: #aaa;">
    {{street}}, {{plz}} {{city}}
  </p>
</div>`

type SubTab = 'email' | 'security' | 'permissions' | 'signature'

export function ITAdminTab() {
  const [subTab, setSubTab] = useState<SubTab>('email')

  const subTabs: { key: SubTab; label: string; icon: typeof Server }[] = [
    { key: 'email', label: 'E-Mail-Config', icon: Server },
    { key: 'security', label: 'Sicherheit', icon: Shield },
    { key: 'permissions', label: 'Berechtigungen', icon: Users },
    { key: 'signature', label: 'E-Mail-Signatur', icon: Mail },
  ]

  return (
    <div className="max-w-3xl">
      <h2 className="text-foreground mb-1">IT-Administration</h2>
      <p className="text-sm text-muted-foreground mb-6">
        E-Mail-Konfiguration, Sicherheitsrichtlinien und Berechtigungen verwalten
      </p>

      {/* Sub-tabs */}
      <div className="flex gap-1 mb-6 border-b border-border">
        {subTabs.map((tab) => {
          const Icon = tab.icon
          return (
            <button
              key={tab.key}
              onClick={() => setSubTab(tab.key)}
              className={`flex items-center gap-1.5 px-3 py-2 text-sm border-b-2 transition-colors ${
                subTab === tab.key
                  ? 'border-primary text-primary font-medium'
                  : 'border-transparent text-muted-foreground hover:text-foreground'
              }`}
            >
              <Icon className="h-3.5 w-3.5" />
              {tab.label}
            </button>
          )
        })}
      </div>

      {subTab === 'email' && <EmailConfigSection />}
      {subTab === 'security' && <SecurityPolicySection />}
      {subTab === 'permissions' && <PermissionsSection />}
      {subTab === 'signature' && <SignatureSection />}
    </div>
  )
}

// ============================================================
// E-Mail Config — per-employee IMAP/SMTP settings
// ============================================================
function EmailConfigSection() {
  const [expandedId, setExpandedId] = useState<string | null>(null)

  return (
    <div>
      <p className="text-xs text-muted-foreground mb-4">
        E-Mail-Server-Konfiguration pro Mitarbeiter. Zugangsdaten werden verschluesselt gespeichert.
      </p>
      <div className="space-y-2">
        {employees.map((emp) => {
          const isExpanded = expandedId === emp.id
          return (
            <div key={emp.id} className="rounded-lg border border-border bg-card">
              <button
                onClick={() => setExpandedId(isExpanded ? null : emp.id)}
                className="flex w-full items-center gap-3 p-3 text-left"
              >
                <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary-light text-xs font-medium text-primary">
                  {emp.initials}
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-foreground">{emp.name}</p>
                  <p className="text-xs text-muted-foreground">{emp.email}</p>
                </div>
                <span className="rounded-full bg-success-light px-2 py-0.5 text-[10px] text-success font-medium">
                  Konfiguriert
                </span>
                {isExpanded ? (
                  <ChevronDown className="h-4 w-4 text-muted-foreground" />
                ) : (
                  <ChevronRight className="h-4 w-4 text-muted-foreground" />
                )}
              </button>

              {isExpanded && (
                <div className="border-t border-border p-4 space-y-3">
                  <div className="grid grid-cols-2 gap-3">
                    <div className="space-y-1.5">
                      <Label className="text-xs">IMAP Server</Label>
                      <Input defaultValue="imap.techvision.de" className="h-8 text-xs" />
                    </div>
                    <div className="space-y-1.5">
                      <Label className="text-xs">IMAP Port</Label>
                      <Input defaultValue="993" className="h-8 text-xs" />
                    </div>
                  </div>
                  <div className="grid grid-cols-2 gap-3">
                    <div className="space-y-1.5">
                      <Label className="text-xs">SMTP Server</Label>
                      <Input defaultValue="smtp.techvision.de" className="h-8 text-xs" />
                    </div>
                    <div className="space-y-1.5">
                      <Label className="text-xs">SMTP Port</Label>
                      <Input defaultValue="587" className="h-8 text-xs" />
                    </div>
                  </div>
                  <div className="grid grid-cols-2 gap-3">
                    <div className="space-y-1.5">
                      <Label className="text-xs">Benutzername</Label>
                      <Input defaultValue={emp.email} className="h-8 text-xs" />
                    </div>
                    <div className="space-y-1.5">
                      <Label className="text-xs">Passwort</Label>
                      <Input type="password" defaultValue="••••••••" className="h-8 text-xs" />
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <Switch defaultChecked />
                    <span className="text-xs text-foreground">SSL/TLS verwenden</span>
                  </div>
                  <Button size="sm" className="text-xs" onClick={() => toast.success(`E-Mail-Config fuer ${emp.name} gespeichert`)}>
                    <Save className="mr-1 h-3 w-3" />
                    Speichern
                  </Button>
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}

// ============================================================
// Security Policy — password policy, 2FA requirements
// ============================================================
function SecurityPolicySection() {
  const [pwExpiryDays, setPwExpiryDays] = useState(90)
  const [pwMinLength, setPwMinLength] = useState(8)
  const [requireUppercase, setRequireUppercase] = useState(true)
  const [requireNumber, setRequireNumber] = useState(true)
  const [requireSpecial, setRequireSpecial] = useState(false)
  const [enforce2FA, setEnforce2FA] = useState(false)
  const [sessionTimeoutMinutes, setSessionTimeoutMinutes] = useState(480)

  return (
    <div>
      <p className="text-xs text-muted-foreground mb-4">
        Sicherheitsrichtlinien fuer alle Mitarbeiter. Aenderungen werden beim naechsten Login wirksam.
      </p>

      {/* Password Policy */}
      <section className="mb-6">
        <h3 className="text-sm font-medium text-foreground mb-3">Passwort-Richtlinie</h3>
        <div className="space-y-3">
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Ablauf nach (Tagen)</Label>
              <Input
                type="number"
                min={0}
                max={365}
                value={pwExpiryDays}
                onChange={(e) => setPwExpiryDays(Number(e.target.value))}
              />
              <p className="text-[10px] text-muted-foreground">0 = kein Ablauf</p>
            </div>
            <div className="space-y-1.5">
              <Label>Mindestlaenge</Label>
              <Input
                type="number"
                min={6}
                max={64}
                value={pwMinLength}
                onChange={(e) => setPwMinLength(Number(e.target.value))}
              />
            </div>
          </div>

          <div className="space-y-2">
            <div className="flex items-center justify-between rounded-md border border-border p-2.5">
              <span className="text-xs text-foreground">Grossbuchstaben erforderlich</span>
              <Switch checked={requireUppercase} onCheckedChange={setRequireUppercase} />
            </div>
            <div className="flex items-center justify-between rounded-md border border-border p-2.5">
              <span className="text-xs text-foreground">Zahl erforderlich</span>
              <Switch checked={requireNumber} onCheckedChange={setRequireNumber} />
            </div>
            <div className="flex items-center justify-between rounded-md border border-border p-2.5">
              <span className="text-xs text-foreground">Sonderzeichen erforderlich</span>
              <Switch checked={requireSpecial} onCheckedChange={setRequireSpecial} />
            </div>
          </div>
        </div>
      </section>

      <Separator className="mb-6" />

      {/* 2FA & Session */}
      <section className="mb-6">
        <h3 className="text-sm font-medium text-foreground mb-3">Authentifizierung</h3>
        <div className="space-y-2">
          <div className="flex items-center justify-between rounded-lg border border-border bg-card p-3">
            <div>
              <p className="text-sm text-foreground">2FA fuer alle erzwingen</p>
              <p className="text-xs text-muted-foreground">Alle Mitarbeiter muessen 2FA aktivieren</p>
            </div>
            <Switch checked={enforce2FA} onCheckedChange={setEnforce2FA} />
          </div>
          <div className="space-y-1.5">
            <Label>Sitzungs-Timeout (Minuten)</Label>
            <Input
              type="number"
              min={15}
              max={1440}
              value={sessionTimeoutMinutes}
              onChange={(e) => setSessionTimeoutMinutes(Number(e.target.value))}
            />
            <p className="text-[10px] text-muted-foreground">Automatische Abmeldung nach Inaktivitaet</p>
          </div>
        </div>
      </section>

      <Button size="sm" onClick={() => toast.success('Sicherheitsrichtlinien gespeichert')}>
        <Save className="mr-1.5 h-4 w-4" />
        Richtlinien speichern
      </Button>
    </div>
  )
}

// ============================================================
// Permissions — module access per role
// ============================================================
function PermissionsSection() {
  const roles = ['Admin', 'Manager', 'Mitarbeiter', 'Extern']
  const modules = [
    'Dashboard', 'CRM', 'Projekte', 'Aufgaben', 'Kalender',
    'Chat', 'E-Mail', 'Kontakte', 'Team', 'Buchhaltung',
    'Kommunikation', 'Dokumente',
  ]

  // Mock permission matrix (admin has all, others have some)
  const [permissions, setPermissions] = useState<Record<string, Record<string, boolean>>>(() => {
    const matrix: Record<string, Record<string, boolean>> = {}
    for (const role of roles) {
      matrix[role] = {}
      for (const mod of modules) {
        matrix[role][mod] = role === 'Admin' || (role === 'Manager' && mod !== 'Buchhaltung') || ['Dashboard', 'Chat', 'Kalender', 'Aufgaben'].includes(mod)
      }
    }
    return matrix
  })

  const toggle = (role: string, mod: string) => {
    if (role === 'Admin') return // Admin always has all
    setPermissions((prev) => ({
      ...prev,
      [role]: { ...prev[role], [mod]: !prev[role][mod] },
    }))
  }

  return (
    <div>
      <p className="text-xs text-muted-foreground mb-4">
        Modul-Zugriff pro Rolle verwalten. Admins haben immer vollen Zugriff.
      </p>

      <div className="rounded-lg border border-border overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-xs">
            <thead>
              <tr className="border-b border-border bg-secondary/50">
                <th className="px-3 py-2.5 text-left font-medium text-muted-foreground w-28">Modul</th>
                {roles.map((role) => (
                  <th key={role} className="px-3 py-2.5 text-center font-medium text-muted-foreground">{role}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {modules.map((mod) => (
                <tr key={mod} className="border-b border-border-muted">
                  <td className="px-3 py-2 text-foreground font-medium">{mod}</td>
                  {roles.map((role) => (
                    <td key={role} className="px-3 py-2 text-center">
                      <button
                        onClick={() => toggle(role, mod)}
                        disabled={role === 'Admin'}
                        className={`h-4 w-4 rounded border transition-colors ${
                          permissions[role]?.[mod]
                            ? 'bg-primary border-primary'
                            : 'border-border hover:border-muted-foreground'
                        } ${role === 'Admin' ? 'cursor-not-allowed opacity-60' : 'cursor-pointer'}`}
                      >
                        {permissions[role]?.[mod] && (
                          <svg viewBox="0 0 16 16" fill="none" className="h-4 w-4 text-primary-foreground">
                            <path d="M4 8l3 3 5-6" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                          </svg>
                        )}
                      </button>
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <Button size="sm" className="mt-4" onClick={() => toast.success('Berechtigungen gespeichert')}>
        <Save className="mr-1.5 h-4 w-4" />
        Berechtigungen speichern
      </Button>
    </div>
  )
}

// ============================================================
// Email Signature Editor
// ============================================================
function SignatureSection() {
  const [mode, setMode] = useState<'template' | 'preview'>('template')
  const [template, setTemplate] = useState(DEFAULT_SIGNATURE_TEMPLATE)
  const [personalName, setPersonalName] = useState('Max Mustermann')
  const [personalPosition, setPersonalPosition] = useState('Projektleiter')
  const [personalDept, setPersonalDept] = useState('Engineering')
  const [personalPhone, setPersonalPhone] = useState('+49 89 123 456 78')
  const [personalEmail, setPersonalEmail] = useState('m.mustermann@techvision.de')

  // Replace template variables for preview
  const previewHtml = template
    .replace(/\{\{name\}\}/g, personalName)
    .replace(/\{\{position\}\}/g, personalPosition)
    .replace(/\{\{department\}\}/g, personalDept)
    .replace(/\{\{phone\}\}/g, personalPhone)
    .replace(/\{\{email\}\}/g, personalEmail)
    .replace(/\{\{company\}\}/g, 'TechVision GmbH')
    .replace(/\{\{street\}\}/g, 'Maximilianstrasse 35')
    .replace(/\{\{plz\}\}/g, '80539')
    .replace(/\{\{city\}\}/g, 'Muenchen')

  return (
    <div>
      <p className="text-xs text-muted-foreground mb-4">
        IT erstellt eine Firmen-Signaturvorlage mit Platzhaltern. Mitarbeiter passen ihre persoenlichen Felder an.
      </p>

      {/* Mode toggle */}
      <div className="flex gap-1 mb-4">
        <button
          onClick={() => setMode('template')}
          className={`flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs transition-colors ${
            mode === 'template' ? 'bg-primary text-primary-foreground' : 'bg-secondary text-foreground hover:bg-secondary/80'
          }`}
        >
          <Pencil className="h-3 w-3" />
          Template bearbeiten
        </button>
        <button
          onClick={() => setMode('preview')}
          className={`flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs transition-colors ${
            mode === 'preview' ? 'bg-primary text-primary-foreground' : 'bg-secondary text-foreground hover:bg-secondary/80'
          }`}
        >
          <Eye className="h-3 w-3" />
          Vorschau
        </button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Editor / Template */}
        <div>
          {mode === 'template' ? (
            <div className="space-y-3">
              <div className="space-y-1.5">
                <Label>HTML-Signaturvorlage</Label>
                <textarea
                  value={template}
                  onChange={(e) => setTemplate(e.target.value)}
                  rows={12}
                  className="w-full rounded-lg border border-border bg-input-background px-3 py-2 text-xs font-mono text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring resize-y"
                />
                <p className="text-[10px] text-muted-foreground">
                  Platzhalter: {'{{name}}'}, {'{{position}}'}, {'{{department}}'}, {'{{phone}}'}, {'{{email}}'}, {'{{company}}'}, {'{{street}}'}, {'{{plz}}'}, {'{{city}}'}
                </p>
              </div>
            </div>
          ) : (
            <div className="space-y-3">
              <h4 className="text-xs font-medium text-foreground">Persoenliche Felder</h4>
              <div className="space-y-2">
                <div className="space-y-1">
                  <Label className="text-xs">Name</Label>
                  <Input value={personalName} onChange={(e) => setPersonalName(e.target.value)} className="h-8 text-xs" />
                </div>
                <div className="grid grid-cols-2 gap-2">
                  <div className="space-y-1">
                    <Label className="text-xs">Position</Label>
                    <Input value={personalPosition} onChange={(e) => setPersonalPosition(e.target.value)} className="h-8 text-xs" />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs">Abteilung</Label>
                    <Input value={personalDept} onChange={(e) => setPersonalDept(e.target.value)} className="h-8 text-xs" />
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-2">
                  <div className="space-y-1">
                    <Label className="text-xs">Telefon</Label>
                    <Input value={personalPhone} onChange={(e) => setPersonalPhone(e.target.value)} className="h-8 text-xs" />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs">E-Mail</Label>
                    <Input value={personalEmail} onChange={(e) => setPersonalEmail(e.target.value)} className="h-8 text-xs" />
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Preview */}
        <div>
          <h4 className="text-xs font-medium text-foreground mb-2">Vorschau</h4>
          <div className="rounded-lg border border-border bg-white p-4 min-h-[120px]">
            <div dangerouslySetInnerHTML={{ __html: previewHtml }} />
          </div>
        </div>
      </div>

      <div className="flex gap-2 mt-4">
        <Button size="sm" onClick={() => toast.success('Signatur-Vorlage gespeichert')}>
          <Save className="mr-1.5 h-4 w-4" />
          Vorlage speichern
        </Button>
        <Button size="sm" variant="outline" onClick={() => setTemplate(DEFAULT_SIGNATURE_TEMPLATE)}>
          Zuruecksetzen
        </Button>
      </div>
    </div>
  )
}
