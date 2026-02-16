import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export interface Ticket {
  id: string
  ticketNr: string
  subject: string
  description: string
  priority: 'low' | 'medium' | 'high' | 'critical'
  status: 'open' | 'in_progress' | 'waiting' | 'resolved' | 'closed'
  assignedTo: string
  contactName: string
  slaDueAt: string
  slaOverdue: boolean
  slaRemaining: string
  createdAt: string
  updatedAt: string
}

export interface KBArticle {
  id: string
  title: string
  category: string
  excerpt: string
  views: number
  published: boolean
  updatedAt: string
}

export interface HelpdeskStats {
  openTickets: number
  avgResponseTime: string
  resolvedThisWeek: number
  customerSatisfaction: string
  weeklyBreakdown: { label: string; count: number }[]
}

interface HelpdeskStore {
  tickets: Ticket[]
  kbArticles: KBArticle[]
  stats: HelpdeskStats
}

function computeSla(slaDueAt: string): { overdue: boolean; remaining: string } {
  const due = new Date(slaDueAt)
  const now = new Date('2026-02-15T11:00:00')
  const diffMs = due.getTime() - now.getTime()
  if (diffMs < 0) {
    const hours = Math.abs(Math.floor(diffMs / 3600000))
    return { overdue: true, remaining: `${hours}h überfällig` }
  }
  const hours = Math.floor(diffMs / 3600000)
  if (hours < 24) return { overdue: false, remaining: `${hours}h übrig` }
  const days = Math.floor(hours / 24)
  return { overdue: false, remaining: `${days}d ${hours % 24}h übrig` }
}

function makeTicket(
  id: string, ticketNr: string, subject: string, description: string,
  priority: Ticket['priority'], status: Ticket['status'],
  assignedTo: string, contactName: string,
  slaDueAt: string, createdAt: string, updatedAt: string,
): Ticket {
  const sla = computeSla(slaDueAt)
  return { id, ticketNr, subject, description, priority, status, assignedTo, contactName, slaDueAt, slaOverdue: sla.overdue, slaRemaining: sla.remaining, createdAt, updatedAt }
}

const MOCK_TICKETS: Ticket[] = [
  makeTicket('tk-1', 'HD-2026-0301', 'Drucker im 2. OG druckt nicht', 'Seit heute Morgen funktioniert der Netzwerkdrucker HP LaserJet im 2. OG nicht mehr. Fehlermeldung: Offline.', 'high', 'in_progress', 'Marco Hartmann', 'Brigitte Schärer', '2026-02-15T16:00:00', '2026-02-15T08:30:00', '2026-02-15T09:15:00'),
  makeTicket('tk-2', 'HD-2026-0302', 'VPN-Verbindung bricht ab', 'VPN-Verbindung zum Firmennetz trennt sich alle 10 Minuten. Betrifft Home-Office Zugang.', 'critical', 'in_progress', 'Marco Hartmann', 'Stefan Wenger', '2026-02-15T12:00:00', '2026-02-14T17:45:00', '2026-02-15T08:00:00'),
  makeTicket('tk-3', 'HD-2026-0303', 'Neuer Mitarbeiter - Zugänge einrichten', 'Ab 01.03.2026 neuer Mitarbeiter: Lukas Meier, Abteilung Buchhaltung. Bitte alle Standardzugänge einrichten.', 'medium', 'open', 'Sandra Bürki', 'Karin Pfister', '2026-02-28T17:00:00', '2026-02-14T14:00:00', '2026-02-14T14:00:00'),
  makeTicket('tk-4', 'HD-2026-0304', 'Outlook synchronisiert Kalender nicht', 'Kalendereinträge werden nicht zwischen Outlook Desktop und Mobile synchronisiert.', 'medium', 'waiting', 'Marco Hartmann', 'Andreas Müller', '2026-02-17T12:00:00', '2026-02-14T11:30:00', '2026-02-14T16:20:00'),
  makeTicket('tk-5', 'HD-2026-0305', 'Bildschirm flackert', 'Der externe Monitor (Dell 27") an Arbeitsplatz B-12 flackert sporadisch.', 'low', 'open', 'Sandra Bürki', 'Regula Vogt', '2026-02-20T17:00:00', '2026-02-14T09:00:00', '2026-02-14T09:00:00'),
  makeTicket('tk-6', 'HD-2026-0306', 'ERP-System Fehlermeldung bei Rechnungserstellung', 'Beim Erstellen von Rechnungen erscheint Fehler 500. Betrifft alle Mitarbeiter in der Buchhaltung.', 'critical', 'resolved', 'Marco Hartmann', 'Thomas Kunz', '2026-02-14T14:00:00', '2026-02-14T08:00:00', '2026-02-14T12:30:00'),
  makeTicket('tk-7', 'HD-2026-0307', 'WLAN im Sitzungszimmer zu schwach', 'Im Sitzungszimmer "Pilatus" ist das WLAN-Signal zu schwach für Videokonferenzen.', 'medium', 'in_progress', 'Sandra Bürki', 'Daniel Roth', '2026-02-18T17:00:00', '2026-02-13T15:30:00', '2026-02-15T10:00:00'),
  makeTicket('tk-8', 'HD-2026-0308', 'Software-Lizenz Adobe CC abgelaufen', 'Die Adobe Creative Cloud Lizenz für Marketing ist seit gestern abgelaufen.', 'high', 'waiting', 'Sandra Bürki', 'Nicole Berger', '2026-02-16T12:00:00', '2026-02-13T11:00:00', '2026-02-14T10:00:00'),
  makeTicket('tk-9', 'HD-2026-0309', 'Backup-Fehler Fileserver', 'Nächtliches Backup des Fileservers ist seit 3 Tagen fehlgeschlagen. Fehlermeldung: Speicherplatz unzureichend.', 'critical', 'resolved', 'Marco Hartmann', 'System Alert', '2026-02-13T10:00:00', '2026-02-12T06:00:00', '2026-02-13T09:00:00'),
  makeTicket('tk-10', 'HD-2026-0310', 'Zutrittskarte funktioniert nicht', 'Zutrittskarte für Büro 3. OG funktioniert seit heute Morgen nicht mehr. Badge-Nr: Z-4412.', 'medium', 'open', 'Sandra Bürki', 'Eveline Stauffer', '2026-02-16T17:00:00', '2026-02-15T07:50:00', '2026-02-15T07:50:00'),
  makeTicket('tk-11', 'HD-2026-0311', 'Telefon-Anlage: Weiterleitung einrichten', 'Neue Rufweiterleitung für Empfang auf +41 44 555 12 99 einrichten (Ferienabwesenheit).', 'low', 'closed', 'Marco Hartmann', 'Ruth Eberle', '2026-02-14T17:00:00', '2026-02-12T14:00:00', '2026-02-13T11:00:00'),
  makeTicket('tk-12', 'HD-2026-0312', 'Laptop für Aussendienst bestellen', 'Neuer Laptop (ThinkPad T14) für Aussendienst-Mitarbeiter Reto Graf benötigt. Inkl. Dockingstation.', 'low', 'open', 'Sandra Bürki', 'Beat Kuhn', '2026-02-25T17:00:00', '2026-02-11T16:00:00', '2026-02-11T16:00:00'),
  makeTicket('tk-13', 'HD-2026-0313', 'Sharepoint-Berechtigung für Projekt X', 'Team Projekt X braucht Zugriff auf Sharepoint-Ordner /Projekte/X. Benutzer: Meier, Huber, Keller.', 'medium', 'resolved', 'Sandra Bürki', 'Patricia Hofer', '2026-02-15T12:00:00', '2026-02-13T09:00:00', '2026-02-14T15:30:00'),
  makeTicket('tk-14', 'HD-2026-0314', 'Virenwarnung auf Arbeitsplatz C-03', 'Sophos meldet verdächtige Datei auf Arbeitsplatz C-03 (Abt. Einkauf). Quarantäne aktiv.', 'high', 'in_progress', 'Marco Hartmann', 'System Alert', '2026-02-15T14:00:00', '2026-02-15T06:30:00', '2026-02-15T08:45:00'),
  makeTicket('tk-15', 'HD-2026-0315', 'Teams-Raum "Säntis" Kamera defekt', 'Die Konferenzkamera im Teams-Raum Säntis zeigt kein Bild mehr. Modell: Logitech Rally.', 'medium', 'open', 'Marco Hartmann', 'Yvonne Stocker', '2026-02-19T17:00:00', '2026-02-14T13:00:00', '2026-02-14T13:00:00'),
]

const MOCK_KB_ARTICLES: KBArticle[] = [
  { id: 'kb-1', title: 'VPN-Verbindung einrichten (Windows)', category: 'Netzwerk', excerpt: 'Schritt-für-Schritt Anleitung zur Einrichtung der VPN-Verbindung unter Windows 10/11 mit dem Cisco AnyConnect Client.', views: 342, published: true, updatedAt: '2026-01-20T10:00:00' },
  { id: 'kb-2', title: 'Drucker hinzufügen im Netzwerk', category: 'Hardware', excerpt: 'So fügen Sie einen Netzwerkdrucker unter Windows oder macOS hinzu. Inkl. Treiber-Download Links.', views: 218, published: true, updatedAt: '2026-01-15T14:00:00' },
  { id: 'kb-3', title: 'Passwort zurücksetzen (Self-Service)', category: 'Sicherheit', excerpt: 'Anleitung zum selbstständigen Zurücksetzen des Active-Directory Passworts über das Self-Service Portal.', views: 567, published: true, updatedAt: '2026-02-01T09:00:00' },
  { id: 'kb-4', title: 'Outlook E-Mail Signatur einrichten', category: 'E-Mail', excerpt: 'Einheitliche E-Mail Signatur gemäß CI/CD-Richtlinien einrichten. Vorlagen und Anleitungen.', views: 189, published: true, updatedAt: '2025-12-10T11:00:00' },
  { id: 'kb-5', title: 'Home-Office Checkliste IT', category: 'Allgemein', excerpt: 'Checkliste für die IT-Ausstattung im Home-Office: VPN, Telefonie, Hardware-Anforderungen.', views: 445, published: true, updatedAt: '2026-01-08T16:00:00' },
]

const MOCK_STATS: HelpdeskStats = {
  openTickets: 8,
  avgResponseTime: '2.4 Std.',
  resolvedThisWeek: 5,
  customerSatisfaction: '4.6 / 5.0',
  weeklyBreakdown: [
    { label: 'Mo', count: 12 },
    { label: 'Di', count: 8 },
    { label: 'Mi', count: 15 },
    { label: 'Do', count: 10 },
    { label: 'Fr', count: 7 },
    { label: 'Sa', count: 2 },
    { label: 'So', count: 1 },
  ],
}

export const useHelpdeskStore = create<HelpdeskStore>()(
  persist(
    () => ({
      tickets: MOCK_TICKETS,
      kbArticles: MOCK_KB_ARTICLES,
      stats: MOCK_STATS,
    }),
    { name: 'kmuhub-helpdesk' },
  ),
)
