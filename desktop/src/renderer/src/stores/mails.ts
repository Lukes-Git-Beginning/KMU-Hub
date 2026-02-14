import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export interface EmailAttachment {
  name: string
  size: string
}

export interface Email {
  id: string
  from: { name: string; email: string; initials: string }
  to: string[]
  cc: string[]
  bcc: string[]
  subject: string
  preview: string
  body: string
  date: string
  time: string
  isRead: boolean
  isStarred: boolean
  folderId: string
  attachments: EmailAttachment[]
  signature: string
}

export interface MailFolder {
  id: string
  name: string
  type: 'system' | 'custom'
  unread: number
}

export interface ComposeDraft {
  to: string[]
  cc: string[]
  bcc: string[]
  subject: string
  body: string
  mode: 'compose' | 'reply' | 'reply-all' | 'forward'
}

interface MailsState {
  emails: Email[]
  folders: MailFolder[]
  composeDraft: ComposeDraft | null
  setComposeDraft: (draft: ComposeDraft | null) => void
  addEmail: (email: Omit<Email, 'id'>) => void
  deleteEmail: (id: string) => void
  markRead: (id: string) => void
  markUnread: (id: string) => void
  toggleStar: (id: string) => void
  archiveEmail: (id: string) => void
  moveToFolder: (id: string, folderId: string) => void
  sendEmail: (email: Omit<Email, 'id' | 'date' | 'time' | 'isRead' | 'isStarred' | 'folderId'>) => void
  saveDraft: (email: Omit<Email, 'id' | 'date' | 'time' | 'isRead' | 'isStarred' | 'folderId'>) => void
  addFolder: (name: string) => void
  renameFolder: (id: string, name: string) => void
  deleteFolder: (id: string) => void
  emptyTrash: () => void
}

const me = { name: 'Du', email: 'darien@kmuhub.ch', initials: 'DK' }

const mockEmails: Email[] = [
  {
    id: 'e1',
    from: { name: 'Sarah Klein', email: 'sarah@designstudio.ch', initials: 'SK' },
    to: ['darien@kmuhub.ch'], cc: [], bcc: [],
    subject: 'Design Review: Neue Landingpage',
    preview: 'Hallo, ich habe die neuen Designs fuer die Landingpage fertig. Koenntest du dir das anschauen?',
    body: 'Hallo,\n\nich habe die neuen Designs fuer die Landingpage fertig. Koenntest du dir das anschauen und Feedback geben?\n\nDie Mockups findest du im angehaengten PDF. Besonders wichtig ist mir deine Meinung zum Hero-Bereich und den CTA-Buttons.\n\nLiebe Gruesse,\nSarah',
    date: '2026-02-09', time: '09:45', isRead: false, isStarred: true, folderId: 'inbox',
    attachments: [{ name: 'Landingpage_v3.pdf', size: '4.2 MB' }], signature: '',
  },
  {
    id: 'e2',
    from: { name: 'Michael Berg', email: 'michael.berg@kmuhub.ch', initials: 'MB' },
    to: ['darien@kmuhub.ch'], cc: ['anna.mueller@kmuhub.ch'], bcc: [],
    subject: 'Sprint Retro Ergebnisse',
    preview: 'Hier die Zusammenfassung der Retrospektive vom letzten Sprint.',
    body: 'Hier die Zusammenfassung der Retrospektive vom letzten Sprint.\n\nAction Items:\n- API Performance verbessern (Michael)\n- Test Coverage erhoehen (Jonas)\n- Dokumentation aktualisieren (Sarah)\n\nBitte bis Freitag umsetzen.\n\nGruss, Michael',
    date: '2026-02-09', time: '08:30', isRead: false, isStarred: false, folderId: 'inbox',
    attachments: [], signature: '',
  },
  {
    id: 'e3',
    from: { name: 'Anna Mueller', email: 'anna.mueller@kmuhub.ch', initials: 'AM' },
    to: ['darien@kmuhub.ch'], cc: [], bcc: [],
    subject: 'Kundentermin morgen - Vorbereitung',
    preview: 'Nicht vergessen: Morgen um 14:00 haben wir den Termin mit der ABC GmbH.',
    body: 'Nicht vergessen: Morgen um 14:00 haben wir den Termin mit der ABC GmbH.\n\nBitte die Praesentation vorbereiten und die neuesten Zahlen einpflegen.\n\nTeilnehmer: Anna, Peter, Michael\n\nVG Anna',
    date: '2026-02-08', time: '16:20', isRead: true, isStarred: true, folderId: 'inbox',
    attachments: [{ name: 'Praesentation_ABC.pptx', size: '8.1 MB' }], signature: '',
  },
  {
    id: 'e4',
    from: { name: 'Jonas Diaz', email: 'jonas.diaz@kmuhub.ch', initials: 'JD' },
    to: ['darien@kmuhub.ch'], cc: [], bcc: [],
    subject: 'Bug Report: Login-Fehler auf iOS',
    preview: 'Habe einen kritischen Bug im Login-Flow auf iOS gefunden.',
    body: 'Habe einen kritischen Bug im Login-Flow auf iOS gefunden.\n\nDetails:\n- iOS 17.3, iPhone 15\n- Login mit Social Auth schlaegt fehl\n- Error: OAuth callback timeout\n\nTicket: PROJ-342\nPrioritaet: Hoch\n\nJonas',
    date: '2026-02-08', time: '14:15', isRead: true, isStarred: false, folderId: 'inbox',
    attachments: [], signature: '',
  },
  {
    id: 'e5',
    from: { name: 'Peter Koch', email: 'peter.koch@kmuhub.ch', initials: 'PK' },
    to: ['darien@kmuhub.ch', 'team@kmuhub.ch'], cc: [], bcc: [],
    subject: 'Neue Sicherheitsrichtlinien',
    preview: 'Ab sofort gelten neue Sicherheitsrichtlinien fuer alle Projekte.',
    body: 'Ab sofort gelten neue Sicherheitsrichtlinien fuer alle Projekte.\n\nWichtige Aenderungen:\n1. 2FA ist jetzt Pflicht\n2. Passwoerter muessen alle 90 Tage geaendert werden\n3. Neue VPN-Policy fuer Remote-Arbeit\n\nBitte lesen und per Antwort bestaetigen.\n\nPeter',
    date: '2026-02-07', time: '11:00', isRead: true, isStarred: false, folderId: 'inbox',
    attachments: [{ name: 'Sicherheitsrichtlinien_2026.pdf', size: '1.2 MB' }], signature: '',
  },
  {
    id: 'e6',
    from: { name: 'Thomas Weber', email: 'thomas.weber@abc-gmbh.ch', initials: 'TW' },
    to: ['darien@kmuhub.ch'], cc: ['anna.mueller@kmuhub.ch'], bcc: [],
    subject: 'RE: Angebot CRM-Integration Phase 2',
    preview: 'Vielen Dank fuer das Angebot. Wir sind grundsaetzlich einverstanden.',
    body: 'Vielen Dank fuer das Angebot. Wir sind grundsaetzlich einverstanden.\n\nKoennten wir noch einen Punkt besprechen: die Schulungskosten scheinen uns etwas hoch. Gibt es da Spielraum?\n\nAnsonsten wuerden wir gerne naechste Woche starten.\n\nBeste Gruesse,\nThomas Weber\nABC GmbH',
    date: '2026-02-07', time: '09:30', isRead: false, isStarred: false, folderId: 'inbox',
    attachments: [], signature: '',
  },
  {
    id: 'e7',
    from: { name: 'Lisa Schmidt', email: 'lisa.schmidt@kmuhub.ch', initials: 'LS' },
    to: ['darien@kmuhub.ch'], cc: [], bcc: [],
    subject: 'Newsletter-Entwurf Februar',
    preview: 'Hier der Entwurf fuer den Februar-Newsletter. Bitte draufschauen.',
    body: 'Hier der Entwurf fuer den Februar-Newsletter.\n\nThemen:\n- Neue Features Q1\n- Kundenstory: ABC GmbH\n- Team-Vorstellung: Jonas Diaz\n- Tipp: Keyboard Shortcuts\n\nBitte bis Mittwoch Feedback geben, Versand ist Donnerstag.\n\nLG Lisa',
    date: '2026-02-06', time: '15:45', isRead: true, isStarred: false, folderId: 'inbox',
    attachments: [{ name: 'Newsletter_Feb_Draft.html', size: '340 KB' }], signature: '',
  },
  {
    id: 'e8',
    from: { name: 'Eva Brunner', email: 'eva@brunner-partner.ch', initials: 'EB' },
    to: ['darien@kmuhub.ch'], cc: [], bcc: [],
    subject: 'DSGVO Gutachten fertig',
    preview: 'Das Datenschutz-Gutachten ist fertig. Keine kritischen Maengel gefunden.',
    body: 'Das Datenschutz-Gutachten ist fertig.\n\nErgebnis: Keine kritischen Maengel gefunden. Zwei kleinere Empfehlungen:\n1. Cookie-Banner Text anpassen\n2. Aufbewahrungsfristen fuer Log-Daten dokumentieren\n\nDas vollstaendige Gutachten im Anhang.\n\nMit freundlichen Gruessen,\nEva Brunner\nBrunner & Partner',
    date: '2026-02-05', time: '10:15', isRead: true, isStarred: true, folderId: 'inbox',
    attachments: [{ name: 'DSGVO_Gutachten_2026.pdf', size: '2.8 MB' }], signature: '',
  },
  {
    id: 'e9',
    from: me,
    to: ['sarah@designstudio.ch'], cc: [], bcc: [],
    subject: 'RE: Design Review: Neue Landingpage',
    preview: 'Hi Sarah, sieht super aus! Nur beim Hero-Bereich...',
    body: 'Hi Sarah,\n\nsieht super aus! Nur beim Hero-Bereich wuerde ich den Text etwas groesser machen und den CTA-Button in der Primaerfarbe statt Weiss.\n\nAnsonsten Freigabe von meiner Seite.\n\nGruss',
    date: '2026-02-08', time: '10:30', isRead: true, isStarred: false, folderId: 'sent',
    attachments: [], signature: 'Mit freundlichen Gruessen\nDarien\nKMU Hub AG',
  },
  {
    id: 'e10',
    from: me,
    to: ['thomas.weber@abc-gmbh.ch'], cc: ['anna.mueller@kmuhub.ch'], bcc: [],
    subject: 'Angebot CRM-Integration Phase 2',
    preview: 'Sehr geehrter Herr Weber, anbei das Angebot fuer Phase 2...',
    body: 'Sehr geehrter Herr Weber,\n\nanbei das Angebot fuer Phase 2 der CRM-Integration.\n\nUmfang:\n- Erweiterte Kontaktverwaltung\n- Automatisierte Workflows\n- Reporting-Dashboard\n\nGesamtkosten: CHF 48\'000.-\nZeitraum: 8 Wochen\n\nBei Fragen stehe ich gerne zur Verfuegung.\n\nMit freundlichen Gruessen',
    date: '2026-02-06', time: '14:00', isRead: true, isStarred: false, folderId: 'sent',
    attachments: [{ name: 'Angebot_Phase2.pdf', size: '560 KB' }], signature: 'Mit freundlichen Gruessen\nDarien\nKMU Hub AG',
  },
  {
    id: 'e11',
    from: me,
    to: ['claudia.frei@techventures.at'], cc: [], bcc: [],
    subject: 'Demo-Unterlagen KMU Hub',
    preview: 'Entwurf: Demo-Unterlagen fuer TechVentures...',
    body: 'Sehr geehrte Frau Frei,\n\nanbei die Demo-Unterlagen fuer Ihr Team.\n\n(Entwurf)',
    date: '2026-02-08', time: '17:00', isRead: true, isStarred: false, folderId: 'drafts',
    attachments: [], signature: '',
  },
  {
    id: 'e12',
    from: { name: 'Newsletter Spam', email: 'promo@spamsite.com', initials: 'NS' },
    to: ['darien@kmuhub.ch'], cc: [], bcc: [],
    subject: 'Gewinne jetzt ein iPhone 20!!!',
    preview: 'Klicke hier um dein gratis iPhone abzuholen...',
    body: 'HERZLICHEN GLUECKWUNSCH!\n\nSie wurden ausgewaehlt...',
    date: '2026-02-07', time: '03:22', isRead: false, isStarred: false, folderId: 'spam',
    attachments: [], signature: '',
  },
  {
    id: 'e13',
    from: { name: 'Crypto Scam', email: 'invest@scam.xyz', initials: 'CS' },
    to: ['darien@kmuhub.ch'], cc: [], bcc: [],
    subject: '1000% Rendite garantiert!',
    preview: 'Investieren Sie nur CHF 100 und erhalten Sie...',
    body: 'Investieren Sie nur CHF 100...',
    date: '2026-02-06', time: '05:41', isRead: false, isStarred: false, folderId: 'spam',
    attachments: [], signature: '',
  },
]

const mockFolders: MailFolder[] = [
  { id: 'inbox', name: 'Posteingang', type: 'system', unread: 3 },
  { id: 'drafts', name: 'Entwuerfe', type: 'system', unread: 1 },
  { id: 'sent', name: 'Gesendet', type: 'system', unread: 0 },
  { id: 'archive', name: 'Archiv', type: 'system', unread: 0 },
  { id: 'spam', name: 'Spam', type: 'system', unread: 2 },
  { id: 'trash', name: 'Papierkorb', type: 'system', unread: 0 },
]

let nextId = 14
let nextFolderId = 1

export const useMailsStore = create<MailsState>()(
  persist(
    (set, get) => ({
      emails: mockEmails,
      folders: mockFolders,
      composeDraft: null,
      setComposeDraft: (draft) => set({ composeDraft: draft }),

      addEmail: (email) =>
        set((state) => ({
          emails: [{ ...email, id: `e${nextId++}` }, ...state.emails],
        })),

      deleteEmail: (id) =>
        set((state) => {
          const email = state.emails.find((e) => e.id === id)
          if (!email) return state
          if (email.folderId === 'trash') {
            return { emails: state.emails.filter((e) => e.id !== id) }
          }
          return {
            emails: state.emails.map((e) =>
              e.id === id ? { ...e, folderId: 'trash' } : e
            ),
          }
        }),

      markRead: (id) =>
        set((state) => ({
          emails: state.emails.map((e) =>
            e.id === id ? { ...e, isRead: true } : e
          ),
        })),

      markUnread: (id) =>
        set((state) => ({
          emails: state.emails.map((e) =>
            e.id === id ? { ...e, isRead: false } : e
          ),
        })),

      toggleStar: (id) =>
        set((state) => ({
          emails: state.emails.map((e) =>
            e.id === id ? { ...e, isStarred: !e.isStarred } : e
          ),
        })),

      archiveEmail: (id) =>
        set((state) => ({
          emails: state.emails.map((e) =>
            e.id === id ? { ...e, folderId: 'archive' } : e
          ),
        })),

      moveToFolder: (id, folderId) =>
        set((state) => ({
          emails: state.emails.map((e) =>
            e.id === id ? { ...e, folderId } : e
          ),
        })),

      sendEmail: (email) => {
        const now = new Date()
        set((state) => ({
          emails: [
            {
              ...email,
              id: `e${nextId++}`,
              date: now.toISOString().split('T')[0],
              time: `${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}`,
              isRead: true,
              isStarred: false,
              folderId: 'sent',
            },
            ...state.emails,
          ],
        }))
      },

      saveDraft: (email) => {
        const now = new Date()
        set((state) => ({
          emails: [
            {
              ...email,
              id: `e${nextId++}`,
              date: now.toISOString().split('T')[0],
              time: `${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}`,
              isRead: true,
              isStarred: false,
              folderId: 'drafts',
            },
            ...state.emails,
          ],
        }))
      },

      addFolder: (name) =>
        set((state) => ({
          folders: [
            ...state.folders,
            { id: `custom-${nextFolderId++}`, name, type: 'custom', unread: 0 },
          ],
        })),

      renameFolder: (id, name) =>
        set((state) => ({
          folders: state.folders.map((f) => (f.id === id ? { ...f, name } : f)),
        })),

      deleteFolder: (id) =>
        set((state) => ({
          folders: state.folders.filter((f) => f.id !== id),
          emails: state.emails.map((e) =>
            e.folderId === id ? { ...e, folderId: 'inbox' } : e
          ),
        })),

      emptyTrash: () =>
        set((state) => ({
          emails: state.emails.filter((e) => e.folderId !== 'trash'),
        })),
    }),
    { name: 'kmuhub-mails' }
  )
)
