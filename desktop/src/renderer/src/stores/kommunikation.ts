/**
 * Kommunikation (Unified Inbox) store.
 *
 * Design-branch version with mock conversations and messages.
 * When backend is ready, data will migrate to TanStack Query hooks;
 * UI state (activeView, sidebar, etc.) stays in this store.
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type {
  Conversation,
  ConversationMessage,
  CannedResponse,
  CommunicationChannel,
} from '@/types/communication'

// ---------------------------------------------------------------------------
// State interface
// ---------------------------------------------------------------------------

interface KommunikationState {
  // Data
  conversations: Conversation[]
  cannedResponses: CannedResponse[]

  // UI state
  activeView: string
  activeChannel: CommunicationChannel | 'all'
  selectedConversationId: string | null
  sidebarCollapsed: boolean
  detailPaneOpen: boolean
  searchQuery: string

  // Actions — UI
  setActiveView: (view: string) => void
  setActiveChannel: (channel: CommunicationChannel | 'all') => void
  setSelectedConversation: (id: string | null) => void
  toggleSidebar: () => void
  toggleDetailPane: () => void
  setSearchQuery: (query: string) => void

  // Actions — conversations
  markAsRead: (conversationId: string) => void
  updateStatus: (conversationId: string, status: Conversation['status']) => void
  assignTo: (conversationId: string, userId: string) => void
  addTag: (conversationId: string, tag: string) => void
  removeTag: (conversationId: string, tag: string) => void
  addMessage: (conversationId: string, message: Omit<ConversationMessage, 'id' | 'timestamp' | 'isRead'>) => void

  // Actions — canned responses
  addCannedResponse: (response: Omit<CannedResponse, 'id' | 'usageCount'>) => void
  deleteCannedResponse: (id: string) => void
  incrementCannedResponseUsage: (id: string) => void
}

// ---------------------------------------------------------------------------
// Mock data — Canned responses
// ---------------------------------------------------------------------------

const mockCannedResponses: CannedResponse[] = [
  {
    id: 'cr1',
    title: 'Begrüssung',
    content: 'Guten Tag! Vielen Dank für Ihre Nachricht. Wie kann ich Ihnen helfen?',
    category: 'allgemein',
    shortcut: '/hi',
    createdBy: 'Anna Mueller',
    usageCount: 47,
  },
  {
    id: 'cr2',
    title: 'Weiterleitung intern',
    content: 'Ich leite Ihre Anfrage an die zuständige Abteilung weiter. Sie erhalten in Kürze eine Rückmeldung.',
    category: 'allgemein',
    shortcut: '/weiter',
    createdBy: 'Sarah Meier',
    usageCount: 32,
  },
  {
    id: 'cr3',
    title: 'Terminbestätigung',
    content: 'Hiermit bestätigen wir Ihren Termin am {{datum}} um {{uhrzeit}}. Bitte bringen Sie alle relevanten Unterlagen mit.',
    category: 'termine',
    shortcut: '/termin',
    createdBy: 'Lisa Braun',
    usageCount: 28,
  },
  {
    id: 'cr4',
    title: 'Angebot nachfassen',
    content: 'Bezugnehmend auf unser Angebot vom {{datum}} wollte ich mich erkundigen, ob Sie noch Fragen haben oder weitere Informationen benötigen.',
    category: 'vertrieb',
    shortcut: '/followup',
    createdBy: 'Peter Schmidt',
    usageCount: 19,
  },
  {
    id: 'cr5',
    title: 'Ticket erstellt',
    content: 'Wir haben ein Support-Ticket (#{{ticketNr}}) für Ihre Anfrage erstellt. Unser Team kümmert sich darum und meldet sich schnellstmöglich.',
    category: 'support',
    shortcut: '/ticket',
    createdBy: 'Sarah Meier',
    usageCount: 55,
  },
]

// ---------------------------------------------------------------------------
// Mock data — Conversations
// ---------------------------------------------------------------------------

const mockConversations: Conversation[] = [
  {
    id: 'conv1',
    channel: 'email',
    subject: 'Angebot Websiteentwicklung',
    status: 'open',
    priority: 'high',
    assignedTo: 'Anna Mueller',
    contactId: 'c4',
    contactName: 'Peter Schmidt',
    contactEmail: 'peter.schmidt@techag.ch',
    contactCompany: 'Tech Solutions AG',
    lastMessage: 'Können Sie uns ein überarbeitetes Angebot mit den besprochenen Änderungen schicken?',
    lastMessageAt: '2026-02-20T09:15:00',
    unreadCount: 1,
    tags: ['angebot', 'prio'],
    crmDealIds: ['d3'],
    participants: [
      { id: 'c4', name: 'Peter Schmidt', email: 'peter.schmidt@techag.ch', isInternal: false },
      { id: 'c1', name: 'Anna Mueller', email: 'anna.mueller@kmuhub.ch', isInternal: true },
    ],
    messages: [
      {
        id: 'msg1',
        conversationId: 'conv1',
        direction: 'inbound',
        senderName: 'Peter Schmidt',
        senderId: 'c4',
        content: 'Guten Tag Frau Mueller, wir interessieren uns für eine neue Website. Könnten Sie uns ein Angebot zukommen lassen?',
        timestamp: '2026-02-18T10:30:00',
        isRead: true,
        attachments: [],
      },
      {
        id: 'msg2',
        conversationId: 'conv1',
        direction: 'outbound',
        senderName: 'Anna Mueller',
        senderId: 'c1',
        content: 'Guten Tag Herr Schmidt, vielen Dank für Ihre Anfrage! Ich habe ein Angebot basierend auf unserem Gespräch erstellt.',
        timestamp: '2026-02-18T14:00:00',
        isRead: true,
        attachments: [{ id: 'att1', name: 'Angebot-2026-047.pdf', size: '245 KB', type: 'application/pdf' }],
      },
      {
        id: 'msg3',
        conversationId: 'conv1',
        direction: 'inbound',
        senderName: 'Peter Schmidt',
        senderId: 'c4',
        content: 'Können Sie uns ein überarbeitetes Angebot mit den besprochenen Änderungen schicken?',
        timestamp: '2026-02-20T09:15:00',
        isRead: false,
        attachments: [],
      },
    ],
    createdAt: '2026-02-18T10:30:00',
    updatedAt: '2026-02-20T09:15:00',
  },
  {
    id: 'conv2',
    channel: 'whatsapp',
    subject: 'Lieferung Büromaterial',
    status: 'open',
    priority: 'normal',
    contactId: 'c8',
    contactName: 'Maria Huber',
    contactEmail: 'maria.huber@bueroplus.de',
    contactCompany: 'BüroPlus GmbH',
    lastMessage: 'Die Lieferung kommt morgen zwischen 10-12 Uhr 📦',
    lastMessageAt: '2026-02-20T08:45:00',
    unreadCount: 2,
    tags: ['lieferung'],
    participants: [
      { id: 'c8', name: 'Maria Huber', isInternal: false },
      { id: 'c6', name: 'Markus Keller', isInternal: true },
    ],
    messages: [
      {
        id: 'msg4',
        conversationId: 'conv2',
        direction: 'outbound',
        senderName: 'Markus Keller',
        senderId: 'c6',
        content: 'Hallo Frau Huber, wann können wir mit der Lieferung der Druckerpatronen rechnen?',
        timestamp: '2026-02-19T16:00:00',
        isRead: true,
        attachments: [],
      },
      {
        id: 'msg5',
        conversationId: 'conv2',
        direction: 'inbound',
        senderName: 'Maria Huber',
        senderId: 'c8',
        content: 'Guten Morgen! Ich habe gerade den Status geprüft.',
        timestamp: '2026-02-20T08:30:00',
        isRead: false,
        attachments: [],
      },
      {
        id: 'msg6',
        conversationId: 'conv2',
        direction: 'inbound',
        senderName: 'Maria Huber',
        senderId: 'c8',
        content: 'Die Lieferung kommt morgen zwischen 10-12 Uhr 📦',
        timestamp: '2026-02-20T08:45:00',
        isRead: false,
        attachments: [],
      },
    ],
    createdAt: '2026-02-19T16:00:00',
    updatedAt: '2026-02-20T08:45:00',
  },
  {
    id: 'conv3',
    channel: 'teams',
    subject: 'Projektupdate Mobile App',
    status: 'open',
    priority: 'normal',
    assignedTo: 'Thomas Weber',
    contactId: 'c9',
    contactName: 'Klaus Fischer',
    contactCompany: 'Fischer Consulting',
    lastMessage: 'Slide Deck ist angehängt. Schaut euch bitte Seite 4-6 an.',
    lastMessageAt: '2026-02-19T17:30:00',
    unreadCount: 0,
    tags: ['projekt', 'mobile'],
    crmDealIds: ['d5'],
    participants: [
      { id: 'c9', name: 'Klaus Fischer', isInternal: false },
      { id: 'c3', name: 'Thomas Weber', isInternal: true },
    ],
    messages: [
      {
        id: 'msg7',
        conversationId: 'conv3',
        direction: 'inbound',
        senderName: 'Klaus Fischer',
        senderId: 'c9',
        content: 'Hi Thomas, hier das Update zum Mobile-App-Projekt. Wir sind im Zeitplan.',
        timestamp: '2026-02-19T16:45:00',
        isRead: true,
        attachments: [],
      },
      {
        id: 'msg8',
        conversationId: 'conv3',
        direction: 'inbound',
        senderName: 'Klaus Fischer',
        senderId: 'c9',
        content: 'Slide Deck ist angehängt. Schaut euch bitte Seite 4-6 an.',
        timestamp: '2026-02-19T17:30:00',
        isRead: true,
        attachments: [{ id: 'att2', name: 'Mobile-App-Status-Feb.pptx', size: '3.2 MB', type: 'application/vnd.openxmlformats-officedocument.presentationml.presentation' }],
      },
    ],
    createdAt: '2026-02-19T16:45:00',
    updatedAt: '2026-02-19T17:30:00',
  },
  {
    id: 'conv4',
    channel: 'widget',
    subject: 'Frage zu Preisen',
    status: 'pending',
    priority: 'normal',
    contactId: 'c10',
    contactName: 'Julia Wagner',
    contactEmail: 'julia@startup-x.io',
    contactCompany: 'Startup X',
    lastMessage: 'Gibt es Rabatte für Startups unter 10 Mitarbeitern?',
    lastMessageAt: '2026-02-20T07:20:00',
    unreadCount: 1,
    tags: ['preise', 'startup'],
    participants: [
      { id: 'c10', name: 'Julia Wagner', email: 'julia@startup-x.io', isInternal: false },
    ],
    messages: [
      {
        id: 'msg9',
        conversationId: 'conv4',
        direction: 'inbound',
        senderName: 'Julia Wagner',
        senderId: 'c10',
        content: 'Hallo! Ich habe eure Website gesehen und bin interessiert. Gibt es Rabatte für Startups unter 10 Mitarbeitern?',
        timestamp: '2026-02-20T07:20:00',
        isRead: false,
        attachments: [],
      },
    ],
    createdAt: '2026-02-20T07:20:00',
    updatedAt: '2026-02-20T07:20:00',
  },
  {
    id: 'conv5',
    channel: 'email',
    subject: 'Rechnung RE-2026-0038 Rückfrage',
    status: 'open',
    priority: 'high',
    assignedTo: 'Markus Keller',
    contactId: 'c11',
    contactName: 'Stefan Brunner',
    contactEmail: 'stefan.brunner@meierag.ch',
    contactCompany: 'Meier AG',
    lastMessage: 'Die korrigierte Rechnung finden Sie im Anhang. Entschuldigung für die Unannehmlichkeiten.',
    lastMessageAt: '2026-02-19T14:20:00',
    unreadCount: 0,
    tags: ['rechnung', 'korrektur'],
    participants: [
      { id: 'c11', name: 'Stefan Brunner', email: 'stefan.brunner@meierag.ch', isInternal: false },
      { id: 'c6', name: 'Markus Keller', isInternal: true },
    ],
    messages: [
      {
        id: 'msg10',
        conversationId: 'conv5',
        direction: 'inbound',
        senderName: 'Stefan Brunner',
        senderId: 'c11',
        content: 'Guten Tag, in Ihrer Rechnung RE-2026-0038 stimmt der MWSt-Satz nicht. Wir sind in Deutschland, nicht Schweiz.',
        timestamp: '2026-02-19T10:00:00',
        isRead: true,
        attachments: [],
      },
      {
        id: 'msg11',
        conversationId: 'conv5',
        direction: 'internal',
        senderName: 'Markus Keller',
        senderId: 'c6',
        content: 'Stimmt, der Kunde ist DE-basiert. Muss mit 19% statt 8.1% ausgestellt werden. Korrektur nötig.',
        timestamp: '2026-02-19T11:00:00',
        isRead: true,
        attachments: [],
      },
      {
        id: 'msg12',
        conversationId: 'conv5',
        direction: 'outbound',
        senderName: 'Markus Keller',
        senderId: 'c6',
        content: 'Die korrigierte Rechnung finden Sie im Anhang. Entschuldigung für die Unannehmlichkeiten.',
        timestamp: '2026-02-19T14:20:00',
        isRead: true,
        attachments: [{ id: 'att3', name: 'RE-2026-0038-korrigiert.pdf', size: '198 KB', type: 'application/pdf' }],
      },
    ],
    createdAt: '2026-02-19T10:00:00',
    updatedAt: '2026-02-19T14:20:00',
  },
  {
    id: 'conv6',
    channel: 'portal',
    subject: 'Feature-Wunsch: Kalender-Sync',
    status: 'open',
    priority: 'low',
    contactId: 'c12',
    contactName: 'Andrea Zimmermann',
    contactEmail: 'a.zimmermann@bauwerk.de',
    contactCompany: 'Bauwerk GmbH',
    lastMessage: 'Es wäre super, wenn der KMU Hub Kalender mit Outlook synchronisiert werden könnte.',
    lastMessageAt: '2026-02-18T13:10:00',
    unreadCount: 0,
    tags: ['feature-request', 'kalender'],
    crmTicketIds: ['t15'],
    participants: [
      { id: 'c12', name: 'Andrea Zimmermann', email: 'a.zimmermann@bauwerk.de', isInternal: false },
    ],
    messages: [
      {
        id: 'msg13',
        conversationId: 'conv6',
        direction: 'inbound',
        senderName: 'Andrea Zimmermann',
        senderId: 'c12',
        content: 'Es wäre super, wenn der KMU Hub Kalender mit Outlook synchronisiert werden könnte. Unsere Mitarbeiter nutzen beide Systeme parallel und müssen alles doppelt eintragen.',
        timestamp: '2026-02-18T13:10:00',
        isRead: true,
        attachments: [],
      },
    ],
    createdAt: '2026-02-18T13:10:00',
    updatedAt: '2026-02-18T13:10:00',
  },
  {
    id: 'conv7',
    channel: 'email',
    subject: 'Bewerbung: Frontend-Entwickler/in',
    status: 'pending',
    priority: 'normal',
    assignedTo: 'Lisa Braun',
    contactId: 'c13',
    contactName: 'Lena Hofmann',
    contactEmail: 'lena.hofmann@gmail.com',
    lastMessage: 'Anbei meine Bewerbungsunterlagen für die ausgeschriebene Stelle als Frontend-Entwicklerin.',
    lastMessageAt: '2026-02-19T20:15:00',
    unreadCount: 1,
    tags: ['bewerbung', 'hr'],
    participants: [
      { id: 'c13', name: 'Lena Hofmann', email: 'lena.hofmann@gmail.com', isInternal: false },
      { id: 'c5', name: 'Lisa Braun', isInternal: true },
    ],
    messages: [
      {
        id: 'msg14',
        conversationId: 'conv7',
        direction: 'inbound',
        senderName: 'Lena Hofmann',
        senderId: 'c13',
        content: 'Sehr geehrte Frau Braun,\n\nanbei meine Bewerbungsunterlagen für die ausgeschriebene Stelle als Frontend-Entwicklerin. Ich bringe 4 Jahre Erfahrung mit React und TypeScript mit.\n\nMit freundlichen Grüssen\nLena Hofmann',
        timestamp: '2026-02-19T20:15:00',
        isRead: false,
        attachments: [
          { id: 'att4', name: 'Lebenslauf-Hofmann.pdf', size: '340 KB', type: 'application/pdf' },
          { id: 'att5', name: 'Zeugnisse-Hofmann.pdf', size: '1.2 MB', type: 'application/pdf' },
        ],
      },
    ],
    createdAt: '2026-02-19T20:15:00',
    updatedAt: '2026-02-19T20:15:00',
  },
  {
    id: 'conv8',
    channel: 'whatsapp',
    subject: 'Termin verschieben',
    status: 'resolved',
    priority: 'normal',
    contactId: 'c14',
    contactName: 'Rolf Steiner',
    contactCompany: 'Steiner Elektro',
    lastMessage: 'Perfekt, Donnerstag 14 Uhr passt! Bis dann 👍',
    lastMessageAt: '2026-02-19T12:00:00',
    unreadCount: 0,
    tags: ['termin'],
    participants: [
      { id: 'c14', name: 'Rolf Steiner', isInternal: false },
      { id: 'c4', name: 'Peter Schmidt', isInternal: true },
    ],
    messages: [
      {
        id: 'msg15',
        conversationId: 'conv8',
        direction: 'inbound',
        senderName: 'Rolf Steiner',
        senderId: 'c14',
        content: 'Hallo Peter, können wir den Termin am Mittwoch auf Donnerstag verschieben?',
        timestamp: '2026-02-19T10:30:00',
        isRead: true,
        attachments: [],
      },
      {
        id: 'msg16',
        conversationId: 'conv8',
        direction: 'outbound',
        senderName: 'Peter Schmidt',
        senderId: 'c4',
        content: 'Hallo Rolf, kein Problem. Donnerstag 14 Uhr?',
        timestamp: '2026-02-19T11:45:00',
        isRead: true,
        attachments: [],
      },
      {
        id: 'msg17',
        conversationId: 'conv8',
        direction: 'inbound',
        senderName: 'Rolf Steiner',
        senderId: 'c14',
        content: 'Perfekt, Donnerstag 14 Uhr passt! Bis dann 👍',
        timestamp: '2026-02-19T12:00:00',
        isRead: true,
        attachments: [],
      },
    ],
    createdAt: '2026-02-19T10:30:00',
    updatedAt: '2026-02-19T12:00:00',
  },
  {
    id: 'conv9',
    channel: 'email',
    subject: 'Wartungsvertrag Verlängerung',
    status: 'open',
    priority: 'normal',
    assignedTo: 'Peter Schmidt',
    contactId: 'c11',
    contactName: 'Stefan Brunner',
    contactEmail: 'stefan.brunner@meierag.ch',
    contactCompany: 'Meier AG',
    lastMessage: 'Wir würden den Wartungsvertrag gerne um ein weiteres Jahr verlängern. Bitte senden Sie uns die Unterlagen.',
    lastMessageAt: '2026-02-17T09:00:00',
    unreadCount: 0,
    tags: ['vertrag', 'wartung'],
    crmDealIds: ['d8'],
    participants: [
      { id: 'c11', name: 'Stefan Brunner', email: 'stefan.brunner@meierag.ch', isInternal: false },
      { id: 'c4', name: 'Peter Schmidt', isInternal: true },
    ],
    messages: [
      {
        id: 'msg18',
        conversationId: 'conv9',
        direction: 'inbound',
        senderName: 'Stefan Brunner',
        senderId: 'c11',
        content: 'Wir würden den Wartungsvertrag gerne um ein weiteres Jahr verlängern. Bitte senden Sie uns die Unterlagen.',
        timestamp: '2026-02-17T09:00:00',
        isRead: true,
        attachments: [],
      },
    ],
    createdAt: '2026-02-17T09:00:00',
    updatedAt: '2026-02-17T09:00:00',
  },
  {
    id: 'conv10',
    channel: 'teams',
    subject: 'Server-Migration Wochenende',
    status: 'resolved',
    priority: 'high',
    assignedTo: 'Thomas Weber',
    contactId: 'c7',
    contactName: 'Felix Wagner',
    contactCompany: 'Intern',
    lastMessage: 'Migration erfolgreich abgeschlossen. Alle Services laufen.',
    lastMessageAt: '2026-02-16T22:30:00',
    unreadCount: 0,
    tags: ['it', 'migration', 'server'],
    participants: [
      { id: 'c7', name: 'Felix Wagner', isInternal: true },
      { id: 'c3', name: 'Thomas Weber', isInternal: true },
    ],
    messages: [
      {
        id: 'msg19',
        conversationId: 'conv10',
        direction: 'inbound',
        senderName: 'Felix Wagner',
        senderId: 'c7',
        content: 'Server-Migration startet heute Abend 20:00. Downtime ca. 2 Stunden.',
        timestamp: '2026-02-16T18:00:00',
        isRead: true,
        attachments: [],
      },
      {
        id: 'msg20',
        conversationId: 'conv10',
        direction: 'outbound',
        senderName: 'Thomas Weber',
        senderId: 'c3',
        content: 'Alles klar, ich bin erreichbar falls es Probleme gibt.',
        timestamp: '2026-02-16T18:15:00',
        isRead: true,
        attachments: [],
      },
      {
        id: 'msg21',
        conversationId: 'conv10',
        direction: 'inbound',
        senderName: 'Felix Wagner',
        senderId: 'c7',
        content: 'Migration erfolgreich abgeschlossen. Alle Services laufen.',
        timestamp: '2026-02-16T22:30:00',
        isRead: true,
        attachments: [],
      },
    ],
    createdAt: '2026-02-16T18:00:00',
    updatedAt: '2026-02-16T22:30:00',
  },
  {
    id: 'conv11',
    channel: 'email',
    subject: 'DSGVO Auskunftsanfrage',
    status: 'open',
    priority: 'urgent',
    assignedTo: 'Thomas Weber',
    contactId: 'c15',
    contactName: 'Michael Gross',
    contactEmail: 'm.gross@webdesign.de',
    contactCompany: 'Gross Webdesign',
    lastMessage: 'Gemäss Art. 15 DSGVO bitte ich um Auskunft über alle bei Ihnen gespeicherten personenbezogenen Daten.',
    lastMessageAt: '2026-02-20T06:00:00',
    unreadCount: 1,
    tags: ['dsgvo', 'auskunft', 'dringend'],
    participants: [
      { id: 'c15', name: 'Michael Gross', email: 'm.gross@webdesign.de', isInternal: false },
      { id: 'c3', name: 'Thomas Weber', isInternal: true },
    ],
    messages: [
      {
        id: 'msg22',
        conversationId: 'conv11',
        direction: 'inbound',
        senderName: 'Michael Gross',
        senderId: 'c15',
        content: 'Sehr geehrte Damen und Herren,\n\ngemäss Art. 15 DSGVO bitte ich um Auskunft über alle bei Ihnen gespeicherten personenbezogenen Daten zu meiner Person.\n\nBitte teilen Sie mir innert 30 Tagen mit:\n- Welche Daten gespeichert sind\n- Zu welchem Zweck\n- An wen diese weitergegeben wurden\n\nMit freundlichen Grüssen\nMichael Gross',
        timestamp: '2026-02-20T06:00:00',
        isRead: false,
        attachments: [],
      },
    ],
    createdAt: '2026-02-20T06:00:00',
    updatedAt: '2026-02-20T06:00:00',
  },
  {
    id: 'conv12',
    channel: 'widget',
    subject: 'Demo anfragen',
    status: 'pending',
    priority: 'normal',
    contactId: 'c16',
    contactName: 'Sandra Berger',
    contactEmail: 'sandra@maler-berger.ch',
    contactCompany: 'Maler Berger GmbH',
    lastMessage: 'Wir sind ein Malerbetrieb mit 12 Mitarbeitern. Ist KMU Hub für Handwerksbetriebe geeignet?',
    lastMessageAt: '2026-02-19T15:45:00',
    unreadCount: 0,
    tags: ['demo', 'lead', 'handwerk'],
    participants: [
      { id: 'c16', name: 'Sandra Berger', email: 'sandra@maler-berger.ch', isInternal: false },
    ],
    messages: [
      {
        id: 'msg23',
        conversationId: 'conv12',
        direction: 'inbound',
        senderName: 'Sandra Berger',
        senderId: 'c16',
        content: 'Wir sind ein Malerbetrieb mit 12 Mitarbeitern. Ist KMU Hub für Handwerksbetriebe geeignet? Wir bräuchten vor allem Zeiterfassung, Rapporte und Kundenverwaltung.',
        timestamp: '2026-02-19T15:45:00',
        isRead: true,
        attachments: [],
      },
    ],
    createdAt: '2026-02-19T15:45:00',
    updatedAt: '2026-02-19T15:45:00',
  },
]

// ---------------------------------------------------------------------------
// ID counters
// ---------------------------------------------------------------------------

let nextConvId = 13
let nextMsgId = 24
let nextCrId = 6

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useKommunikationStore = create<KommunikationState>()(
  persist(
    (set) => ({
      // Data
      conversations: mockConversations,
      cannedResponses: mockCannedResponses,

      // UI state
      activeView: 'all',
      activeChannel: 'all',
      selectedConversationId: null,
      sidebarCollapsed: false,
      detailPaneOpen: true,
      searchQuery: '',

      // -- UI actions --

      setActiveView: (view) =>
        set({ activeView: view, selectedConversationId: null }),

      setActiveChannel: (channel) =>
        set({ activeChannel: channel, selectedConversationId: null }),

      setSelectedConversation: (id) =>
        set({ selectedConversationId: id, detailPaneOpen: id !== null }),

      toggleSidebar: () =>
        set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),

      toggleDetailPane: () =>
        set((state) => ({ detailPaneOpen: !state.detailPaneOpen })),

      setSearchQuery: (query) => set({ searchQuery: query }),

      // -- Conversation actions --

      markAsRead: (conversationId) =>
        set((state) => ({
          conversations: state.conversations.map((c) =>
            c.id === conversationId
              ? {
                  ...c,
                  unreadCount: 0,
                  messages: c.messages.map((m) => ({ ...m, isRead: true })),
                }
              : c,
          ),
        })),

      updateStatus: (conversationId, status) =>
        set((state) => ({
          conversations: state.conversations.map((c) =>
            c.id === conversationId ? { ...c, status } : c,
          ),
        })),

      assignTo: (conversationId, userId) =>
        set((state) => ({
          conversations: state.conversations.map((c) =>
            c.id === conversationId ? { ...c, assignedTo: userId } : c,
          ),
        })),

      addTag: (conversationId, tag) =>
        set((state) => ({
          conversations: state.conversations.map((c) =>
            c.id === conversationId && !c.tags.includes(tag)
              ? { ...c, tags: [...c.tags, tag] }
              : c,
          ),
        })),

      removeTag: (conversationId, tag) =>
        set((state) => ({
          conversations: state.conversations.map((c) =>
            c.id === conversationId
              ? { ...c, tags: c.tags.filter((t) => t !== tag) }
              : c,
          ),
        })),

      addMessage: (conversationId, message) => {
        const id = `msg${nextMsgId++}`
        const timestamp = new Date().toISOString()
        set((state) => ({
          conversations: state.conversations.map((c) =>
            c.id === conversationId
              ? {
                  ...c,
                  messages: [...c.messages, { ...message, id, timestamp, isRead: true }],
                  lastMessage: message.content,
                  lastMessageAt: timestamp,
                  updatedAt: timestamp,
                }
              : c,
          ),
        }))
      },

      // -- Canned response actions --

      addCannedResponse: (response) => {
        const id = `cr${nextCrId++}`
        set((state) => ({
          cannedResponses: [...state.cannedResponses, { ...response, id, usageCount: 0 }],
        }))
      },

      deleteCannedResponse: (id) =>
        set((state) => ({
          cannedResponses: state.cannedResponses.filter((r) => r.id !== id),
        })),

      incrementCannedResponseUsage: (id) =>
        set((state) => ({
          cannedResponses: state.cannedResponses.map((r) =>
            r.id === id ? { ...r, usageCount: r.usageCount + 1 } : r,
          ),
        })),
    }),
    {
      name: 'kmuhub-kommunikation',
      partialize: (state) => ({
        activeView: state.activeView,
        activeChannel: state.activeChannel,
        sidebarCollapsed: state.sidebarCollapsed,
      }),
    },
  ),
)
