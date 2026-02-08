import { useState } from 'react';
import { 
  Search, Inbox, FileText, Send, Archive, AlertCircle, Trash2, 
  Users, Settings, ChevronDown, Star, MoreVertical, ChevronLeft, 
  Paperclip, Download, Plus, X, Minimize, Upload, Bold, Italic, 
  Underline, Link as LinkIcon, List, Clock, Filter, Grid3x3, 
  Menu, Edit, Check
} from 'lucide-react';
import { Avatar, AvatarFallback } from '../components/ui/avatar';
import { Badge } from '../components/ui/badge';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '../components/ui/dialog';
import { Label } from '../components/ui/label';
import { Input } from '../components/ui/input';
import { Textarea } from '../components/ui/textarea';

// Mock Data
const emails = [
  {
    id: 1,
    from: { name: 'Achim Weber', email: 'achim.weber@company.ch', avatar: 'AW' },
    subject: 'Neue Funktion für Dashboard',
    preview: 'Bitte Review für die neue Zeiterfassung-Funktion...',
    body: 'Hallo Team,\n\nich habe die neue Zeiterfassung-Funktion fertiggestellt. Könnt ihr bitte ein Review machen?\n\nDie wichtigsten Features:\n- Automatische Pausenerkennung\n- Export nach Bexio\n- Mobile-optimiert\n\nBitte gebt mir bis morgen Feedback.\n\nViele Grüße,\nAchim',
    date: 'Vor 2h',
    timestamp: new Date('2024-12-21T12:00'),
    folder: 'inbox',
    isRead: false,
    isStarred: false,
    attachments: [
      { name: 'Zeiterfassung_Spec.pdf', size: '2.5 MB', type: 'pdf' },
      { name: 'Screenshots.zip', size: '5.2 MB', type: 'zip' },
    ],
  },
  {
    id: 2,
    from: { name: 'Lisa Meier', email: 'lisa.meier@company.ch', avatar: 'LM' },
    subject: 'Meeting Notes vom Sprint-Planning',
    preview: 'Anbei die Meeting Notes vom letzten Sprint-Planning...',
    body: 'Hallo,\n\nim Anhang findet ihr die Notes vom Sprint-Planning.\n\nKEY DECISIONS:\n- Sprint dauert 2 Wochen\n- Daily um 9:00 Uhr\n- Demo am Freitag\n\nBitte durchlesen!\n\nLG Lisa',
    date: 'Gestern',
    timestamp: new Date('2024-12-20T14:30'),
    folder: 'inbox',
    isRead: true,
    isStarred: true,
    attachments: [
      { name: 'Meeting_Notes.pdf', size: '1.2 MB', type: 'pdf' },
    ],
  },
  {
    id: 3,
    from: { name: 'Thomas Klein', email: 'thomas.klein@company.ch', avatar: 'TK' },
    subject: 'Projektupdate: Website Relaunch - Status Gelb',
    preview: 'Kurzes Update zum Website-Relaunch...',
    body: 'Team,\n\nkurzes Update:\n\nSTATUS:\n- Frontend: 90% done\n- Backend: 75% done\n- Testing: Pending\n\nRISIKOS:\n- API-Integration verzögert sich\n- Mehr Testing-Zeit nötig\n\nNÄCHSTE SCHRITTE:\n1. Code Review bis Donnerstag\n2. Staging setup bis Freitag\n3. UAT nächste Woche\n\nViele Grüße,\nThomas',
    date: '15 Dez',
    timestamp: new Date('2024-12-15T10:00'),
    folder: 'inbox',
    isRead: true,
    isStarred: false,
    attachments: [],
  },
  {
    id: 4,
    from: { name: 'Sarah Schmidt', email: 'sarah.schmidt@company.ch', avatar: 'SS' },
    subject: 'Design System Updates',
    preview: 'Die neuen Design System Komponenten sind fertig...',
    body: 'Hi Team,\n\ndie Updates am Design System sind ready:\n\n- Neue Button-Varianten\n- Erweiterte Farb-Palette\n- Responsive Grid\n\nDokumentation ist im Wiki.\n\nLG Sarah',
    date: '14 Dez',
    timestamp: new Date('2024-12-14T16:00'),
    folder: 'inbox',
    isRead: false,
    isStarred: false,
    attachments: [],
  },
  {
    id: 5,
    from: { name: 'Max Müller', email: 'max.mueller@company.ch', avatar: 'MM' },
    subject: 'Budget-Freigabe Q1 2025',
    preview: 'Anbei das Budget für Q1 2025 zur Freigabe...',
    body: 'Guten Tag,\n\nim Anhang findet ihr das Budget für Q1 2025.\n\nBitte bis Ende Woche freigeben.\n\nDanke!\nMax',
    date: '13 Dez',
    timestamp: new Date('2024-12-13T09:00'),
    folder: 'inbox',
    isRead: true,
    isStarred: true,
    attachments: [
      { name: 'Budget_Q1_2025.xlsx', size: '1.8 MB', type: 'xlsx' },
    ],
  },
];

const folders = [
  { id: 'inbox', name: 'Posteingang', icon: Inbox, unread: 12, total: 45 },
  { id: 'drafts', name: 'Entwürfe', icon: FileText, unread: 3, total: 8 },
  { id: 'sent', name: 'Gesendet', icon: Send, unread: 0, total: 156 },
  { id: 'archive', name: 'Archiv', icon: Archive, unread: 0, total: 234 },
  { id: 'spam', name: 'Spam', icon: AlertCircle, unread: 5, total: 12 },
  { id: 'trash', name: 'Gelöscht', icon: Trash2, unread: 0, total: 23 },
];

const groups = [
  { id: 1, name: 'Team Verkauf', members: ['AW', 'LM', 'TK', 'SS', 'NW'], count: 5 },
  { id: 2, name: 'Projekt Website', members: ['AW', 'SM', 'JS'], count: 3 },
  { id: 3, name: 'Geschäftsführung', members: ['GM', 'AM'], count: 2 },
];

const teamMembers = [
  { id: 1, name: 'Anna Müller', email: 'anna.mueller@company.ch', avatar: 'AM' },
  { id: 2, name: 'Michael Berg', email: 'michael.berg@company.ch', avatar: 'MB' },
  { id: 3, name: 'Sarah Klein', email: 'sarah.klein@company.ch', avatar: 'SK' },
  { id: 4, name: 'Tom Weber', email: 'tom.weber@company.ch', avatar: 'TW' },
  { id: 5, name: 'Lisa Fischer', email: 'lisa.fischer@company.ch', avatar: 'LF' },
];

type ViewType = 'list' | 'detail' | 'compose';

export function Mails() {
  const [activeFolder, setActiveFolder] = useState('inbox');
  const [currentView, setCurrentView] = useState<ViewType>('list');
  const [selectedEmail, setSelectedEmail] = useState<any>(null);
  const [selectedEmails, setSelectedEmails] = useState<number[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [showGroupsModal, setShowGroupsModal] = useState(false);
  const [showExportMenu, setShowExportMenu] = useState(false);
  const [showGroupsSection, setShowGroupsSection] = useState(true);

  // Compose form state
  const [composeTo, setComposeTo] = useState<string[]>([]);
  const [composeCC, setComposeCC] = useState<string[]>([]);
  const [composeBCC, setComposeBCC] = useState<string[]>([]);
  const [composeSubject, setComposeSubject] = useState('');
  const [composeBody, setComposeBody] = useState('');
  const [composeAttachments, setComposeAttachments] = useState<any[]>([]);
  const [showCC, setShowCC] = useState(false);
  const [showBCC, setShowBCC] = useState(false);
  const [toast, setToast] = useState<{ show: boolean; message: string; type: 'success' | 'error' | 'info' }>({ 
    show: false, 
    message: '', 
    type: 'success' 
  });

  const showToast = (message: string, type: 'success' | 'error' | 'info' = 'success') => {
    setToast({ show: true, message, type });
    setTimeout(() => setToast({ show: false, message: '', type: 'success' }), 3000);
  };

  const handleEmailClick = (email: any) => {
    setSelectedEmail(email);
    setCurrentView('detail');
  };

  const handleBackToList = () => {
    setCurrentView('list');
    setSelectedEmail(null);
  };

  const handleNewEmail = () => {
    setCurrentView('compose');
    // Reset compose form
    setComposeTo([]);
    setComposeCC([]);
    setComposeBCC([]);
    setComposeSubject('');
    setComposeBody('');
    setComposeAttachments([]);
    setShowCC(false);
    setShowBCC(false);
  };

  const handleSendEmail = () => {
    if (composeTo.length === 0 || !composeSubject || !composeBody) {
      showToast('Bitte fülle alle Pflichtfelder aus', 'error');
      return;
    }
    showToast('✅ Email erfolgreich gesendet', 'success');
    setCurrentView('list');
  };

  const toggleEmailSelection = (id: number) => {
    setSelectedEmails(prev =>
      prev.includes(id) ? prev.filter(eid => eid !== id) : [...prev, id]
    );
  };

  const filteredEmails = emails.filter(email => 
    email.folder === activeFolder &&
    (email.subject.toLowerCase().includes(searchQuery.toLowerCase()) ||
     email.from.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
     email.preview.toLowerCase().includes(searchQuery.toLowerCase()))
  );

  const activefolderData = folders.find(f => f.id === activeFolder);

  return (
    <div className="flex h-full bg-slate-50 dark:bg-slate-900">
      {/* LEFT SIDEBAR - NAVIGATION */}
      <div className="w-72 border-r border-slate-200 dark:border-slate-700 flex flex-col bg-white dark:bg-slate-800">
        {/* Header */}
        <div className="p-4 border-b border-slate-200 dark:border-slate-700">
          <h2 className="text-slate-900 dark:text-white mb-1">Mails</h2>
          <p className="text-xs text-slate-500 dark:text-slate-400 mb-4">Email Management</p>
          <button
            onClick={handleNewEmail}
            className="w-full px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg transition-colors flex items-center justify-center gap-2"
          >
            <Plus className="w-4 h-4" />
            Neue Email
          </button>
        </div>

        {/* Search */}
        <div className="p-4 border-b border-slate-200 dark:border-slate-700">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
            <input
              type="text"
              placeholder="Emails durchsuchen..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-10 pr-4 py-2 bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg text-sm text-slate-900 dark:text-white placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-emerald-500"
            />
          </div>
        </div>

        {/* Folders */}
        <div className="flex-1 overflow-y-auto p-2">
          <div className="space-y-1">
            {folders.map((folder) => {
              const Icon = folder.icon;
              const isActive = activeFolder === folder.id;
              return (
                <button
                  key={folder.id}
                  onClick={() => {
                    setActiveFolder(folder.id);
                    setCurrentView('list');
                  }}
                  className={`w-full flex items-center justify-between px-3 py-2.5 rounded-lg transition-colors ${
                    isActive
                      ? 'bg-emerald-50 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-300 border-l-3 border-emerald-600'
                      : 'text-slate-600 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700'
                  }`}
                >
                  <div className="flex items-center gap-3">
                    <Icon className="w-4 h-4" />
                    <span className="text-sm">{folder.name}</span>
                  </div>
                  {folder.unread > 0 && (
                    <Badge className="bg-red-500 text-white text-xs px-1.5 py-0 h-5 min-w-[20px] flex items-center justify-center rounded-full">
                      {folder.unread}
                    </Badge>
                  )}
                </button>
              );
            })}
          </div>

          {/* Separator */}
          <div className="h-px bg-slate-200 dark:bg-slate-700 my-3" />

          {/* Groups */}
          <div>
            <button
              onClick={() => setShowGroupsSection(!showGroupsSection)}
              className="w-full flex items-center justify-between px-3 py-2 text-xs text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 rounded-lg"
            >
              <div className="flex items-center gap-2">
                <ChevronDown className={`w-3 h-3 transition-transform ${showGroupsSection ? '' : '-rotate-90'}`} />
                <span className="font-medium uppercase">Gruppen</span>
              </div>
            </button>
            
            {showGroupsSection && (
              <div className="space-y-1 mt-1">
                {groups.map((group) => (
                  <button
                    key={group.id}
                    className="w-full flex items-center justify-between px-3 py-2 rounded-lg text-slate-600 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700"
                  >
                    <div className="flex items-center gap-2">
                      <Users className="w-3.5 h-3.5" />
                      <span className="text-xs">{group.name}</span>
                    </div>
                    <span className="text-xs text-slate-400">({group.count})</span>
                  </button>
                ))}
                <button
                  onClick={() => setShowGroupsModal(true)}
                  className="w-full flex items-center gap-2 px-3 py-2 rounded-lg text-emerald-600 dark:text-emerald-400 hover:bg-emerald-50 dark:hover:bg-emerald-900/30 text-xs"
                >
                  <Plus className="w-3 h-3" />
                  Neue Gruppe
                </button>
              </div>
            )}
          </div>
        </div>

        {/* Settings */}
        <div className="p-4 border-t border-slate-200 dark:border-slate-700">
          <button className="w-full flex items-center gap-2 px-3 py-2 text-slate-600 dark:text-slate-300 hover:text-emerald-600 dark:hover:text-emerald-400 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors text-sm">
            <Settings className="w-4 h-4" />
            Mail-Einstellungen
          </button>
        </div>
      </div>

      {/* RIGHT SIDE - EMAIL LIST OR DETAIL */}
      <div className="flex-1 flex flex-col bg-white dark:bg-slate-800">
        {currentView === 'list' && (
          <>
            {/* List Header */}
            <div className="h-12 bg-slate-50 dark:bg-slate-900 border-b border-slate-200 dark:border-slate-700 px-4 flex items-center justify-between">
              <div className="flex items-center gap-4">
                <input type="checkbox" className="rounded border-slate-300" />
                <span className="text-sm text-slate-900 dark:text-white font-medium">
                  {activefolderData?.name}
                </span>
                <span className="text-xs text-slate-500 dark:text-slate-400">
                  {activefolderData?.unread} ungelesen von {activefolderData?.total}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
                  <Grid3x3 className="w-4 h-4 text-slate-500" />
                </button>
                <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded flex items-center gap-1 text-xs text-slate-600 dark:text-slate-400">
                  Neueste zuerst
                  <ChevronDown className="w-3 h-3" />
                </button>
                <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
                  <Filter className="w-4 h-4 text-slate-500" />
                </button>
              </div>
            </div>

            {/* Toolbar */}
            {selectedEmails.length > 0 && (
              <div className="h-10 bg-white dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700 px-4 flex items-center gap-2">
                <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
                  <Archive className="w-4 h-4 text-slate-500" />
                </button>
                <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
                  <Trash2 className="w-4 h-4 text-slate-500" />
                </button>
                <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
                  <AlertCircle className="w-4 h-4 text-slate-500" />
                </button>
              </div>
            )}

            {/* Email List */}
            <div className="flex-1 overflow-y-auto">
              {filteredEmails.map((email) => (
                <div
                  key={email.id}
                  onClick={() => handleEmailClick(email)}
                  className={`flex items-start gap-3 p-4 border-b border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-700 cursor-pointer transition-colors ${
                    selectedEmails.includes(email.id) ? 'bg-emerald-50 dark:bg-emerald-900/20 border-l-3 border-emerald-600' : ''
                  } ${!email.isRead ? 'border-l-2 border-emerald-600' : ''}`}
                >
                  <input
                    type="checkbox"
                    checked={selectedEmails.includes(email.id)}
                    onChange={(e) => {
                      e.stopPropagation();
                      toggleEmailSelection(email.id);
                    }}
                    className="mt-1 rounded border-slate-300"
                  />
                  <Avatar className="w-10 h-10 flex-shrink-0">
                    <AvatarFallback className="bg-emerald-100 dark:bg-emerald-900 text-emerald-700 dark:text-emerald-300">
                      {email.from.avatar}
                    </AvatarFallback>
                  </Avatar>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-start justify-between gap-2 mb-1">
                      <div className="flex-1 min-w-0">
                        <p className={`text-sm truncate ${!email.isRead ? 'font-semibold' : 'font-medium'} text-slate-900 dark:text-white`}>
                          {email.from.name}
                        </p>
                        <p className="text-xs text-slate-500 dark:text-slate-400 truncate">
                          {email.from.email}
                        </p>
                      </div>
                      <div className="flex items-center gap-2 flex-shrink-0">
                        <span className="text-xs text-slate-400">{email.date}</span>
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                          }}
                          className="p-1 hover:bg-slate-100 dark:hover:bg-slate-600 rounded"
                        >
                          <Star className={`w-4 h-4 ${email.isStarred ? 'fill-yellow-400 text-yellow-400' : 'text-slate-400'}`} />
                        </button>
                      </div>
                    </div>
                    <p className={`text-sm mb-1 ${!email.isRead ? 'font-medium' : ''} text-slate-900 dark:text-white`}>
                      {email.subject}
                    </p>
                    <p className="text-xs text-slate-500 dark:text-slate-400 truncate">
                      {email.preview}
                    </p>
                  </div>
                </div>
              ))}

              {filteredEmails.length === 0 && (
                <div className="flex flex-col items-center justify-center h-full p-8">
                  <Inbox className="w-12 h-12 text-slate-300 dark:text-slate-600 mb-4" />
                  <p className="text-slate-500 dark:text-slate-400 text-center">
                    {searchQuery ? 'Keine Emails gefunden' : 'Keine Emails in diesem Ordner'}
                  </p>
                </div>
              )}
            </div>
          </>
        )}

        {currentView === 'detail' && selectedEmail && (
          <>
            {/* Detail Header */}
            <div className="h-14 bg-slate-50 dark:bg-slate-900 border-b border-slate-200 dark:border-slate-700 px-5 flex items-center justify-between">
              <div className="flex items-center gap-3">
                <button
                  onClick={handleBackToList}
                  className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded"
                >
                  <ChevronLeft className="w-5 h-5 text-slate-600 dark:text-slate-300" />
                </button>
                <span className="text-sm text-slate-600 dark:text-slate-400">
                  {activefolderData?.name}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
                  <Star className={`w-5 h-5 ${selectedEmail.isStarred ? 'fill-yellow-400 text-yellow-400' : 'text-slate-400'}`} />
                </button>
                <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
                  <Archive className="w-5 h-5 text-slate-500" />
                </button>
                <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
                  <Trash2 className="w-5 h-5 text-slate-500" />
                </button>
                <div className="relative">
                  <button
                    onClick={() => setShowExportMenu(!showExportMenu)}
                    className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded"
                  >
                    <MoreVertical className="w-5 h-5 text-slate-500" />
                  </button>
                  
                  {showExportMenu && (
                    <div className="absolute right-0 mt-2 w-64 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg shadow-lg p-1 z-50">
                      <button className="w-full flex items-center gap-3 px-3 py-2 text-sm text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 rounded">
                        <FileText className="w-4 h-4" />
                        Als PDF exportieren
                      </button>
                      <button className="w-full flex items-center gap-3 px-3 py-2 text-sm text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 rounded">
                        <Send className="w-4 h-4" />
                        An Outlook exportieren
                      </button>
                      <button className="w-full flex items-center gap-3 px-3 py-2 text-sm text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 rounded">
                        <Send className="w-4 h-4" />
                        An Gmail exportieren
                      </button>
                      <button className="w-full flex items-center gap-3 px-3 py-2 text-sm text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 rounded">
                        <Send className="w-4 h-4" />
                        An Apple Mail exportieren
                      </button>
                      <div className="h-px bg-slate-200 dark:bg-slate-700 my-1" />
                      <button className="w-full flex items-center gap-3 px-3 py-2 text-sm text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 rounded">
                        <Archive className="w-4 h-4" />
                        In Dateimanager speichern
                      </button>
                    </div>
                  )}
                </div>
              </div>
            </div>

            {/* Email Header */}
            <div className="p-5 border-b border-slate-200 dark:border-slate-700">
              <div className="flex items-start gap-4 mb-4">
                <Avatar className="w-12 h-12">
                  <AvatarFallback className="bg-emerald-100 dark:bg-emerald-900 text-emerald-700 dark:text-emerald-300">
                    {selectedEmail.from.avatar}
                  </AvatarFallback>
                </Avatar>
                <div className="flex-1">
                  <p className="text-slate-900 dark:text-white font-semibold">{selectedEmail.from.name}</p>
                  <p className="text-sm text-slate-500 dark:text-slate-400">{selectedEmail.from.email}</p>
                  <p className="text-xs text-slate-400 mt-1">
                    {selectedEmail.timestamp.toLocaleDateString('de-CH', { 
                      weekday: 'long', 
                      year: 'numeric', 
                      month: 'long', 
                      day: 'numeric',
                      hour: '2-digit',
                      minute: '2-digit'
                    })}
                  </p>
                </div>
              </div>
              <h1 className="text-lg font-semibold text-slate-900 dark:text-white">
                {selectedEmail.subject}
              </h1>
            </div>

            {/* Email Body */}
            <div className="flex-1 overflow-y-auto p-5">
              <div className="prose dark:prose-invert max-w-none">
                <p className="text-slate-900 dark:text-white whitespace-pre-wrap leading-relaxed">
                  {selectedEmail.body}
                </p>
              </div>

              {/* Attachments */}
              {selectedEmail.attachments.length > 0 && (
                <div className="mt-6 p-4 bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg">
                  <p className="text-xs font-semibold text-slate-600 dark:text-slate-400 uppercase mb-3">
                    Anhänge ({selectedEmail.attachments.length})
                  </p>
                  <div className="space-y-2">
                    {selectedEmail.attachments.map((file: any, idx: number) => (
                      <div
                        key={idx}
                        className="flex items-center justify-between p-2 bg-white dark:bg-slate-800 rounded border border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-700 cursor-pointer"
                      >
                        <div className="flex items-center gap-3">
                          <FileText className="w-5 h-5 text-slate-400" />
                          <div>
                            <p className="text-sm font-medium text-slate-900 dark:text-white">{file.name}</p>
                            <p className="text-xs text-slate-500">({file.size})</p>
                          </div>
                        </div>
                        <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-600 rounded">
                          <Download className="w-4 h-4 text-slate-500" />
                        </button>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>

            {/* Actions Footer */}
            <div className="p-4 border-t border-slate-200 dark:border-slate-700 flex gap-3">
              <button className="flex-1 px-4 py-2 bg-slate-100 dark:bg-slate-700 hover:bg-slate-200 dark:hover:bg-slate-600 text-slate-700 dark:text-slate-300 rounded-lg transition-colors">
                Antworten
              </button>
              <button className="flex-1 px-4 py-2 bg-slate-100 dark:bg-slate-700 hover:bg-slate-200 dark:hover:bg-slate-600 text-slate-700 dark:text-slate-300 rounded-lg transition-colors">
                Allen antworten
              </button>
              <button className="flex-1 px-4 py-2 bg-slate-100 dark:bg-slate-700 hover:bg-slate-200 dark:hover:bg-slate-600 text-slate-700 dark:text-slate-300 rounded-lg transition-colors">
                Weiterleiten
              </button>
            </div>
          </>
        )}

        {currentView === 'compose' && (
          <>
            {/* Compose Header */}
            <div className="h-14 bg-slate-50 dark:bg-slate-900 border-b border-slate-200 dark:border-slate-700 px-5 flex items-center justify-between">
              <h3 className="text-sm font-semibold text-slate-900 dark:text-white">Neue Email</h3>
              <div className="flex items-center gap-2">
                <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
                  <Minimize className="w-4 h-4 text-slate-500" />
                </button>
                <button
                  onClick={handleBackToList}
                  className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded"
                >
                  <X className="w-4 h-4 text-slate-500" />
                </button>
              </div>
            </div>

            {/* Compose Form */}
            <div className="flex-1 overflow-y-auto">
              {/* To Field */}
              <div className="p-4 border-b border-slate-200 dark:border-slate-700">
                <div className="flex items-start gap-3">
                  <label className="text-sm font-medium text-slate-600 dark:text-slate-400 mt-2 w-16">An:</label>
                  <div className="flex-1">
                    <input
                      type="text"
                      placeholder="Empfänger hinzufügen oder wählen..."
                      className="w-full px-3 py-2 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg text-sm text-slate-900 dark:text-white placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                    />
                  </div>
                </div>
                
                {/* CC/BCC Toggle */}
                {!showCC && !showBCC && (
                  <div className="flex gap-3 mt-2 ml-[76px]">
                    <button
                      onClick={() => setShowCC(true)}
                      className="text-xs text-emerald-600 dark:text-emerald-400 hover:underline"
                    >
                      CC hinzufügen
                    </button>
                    <button
                      onClick={() => setShowBCC(true)}
                      className="text-xs text-emerald-600 dark:text-emerald-400 hover:underline"
                    >
                      BCC hinzufügen
                    </button>
                  </div>
                )}
              </div>

              {/* CC Field */}
              {showCC && (
                <div className="p-4 border-b border-slate-200 dark:border-slate-700">
                  <div className="flex items-start gap-3">
                    <label className="text-sm font-medium text-slate-600 dark:text-slate-400 mt-2 w-16">CC:</label>
                    <input
                      type="text"
                      placeholder="CC-Empfänger..."
                      className="flex-1 px-3 py-2 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg text-sm text-slate-900 dark:text-white placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                    />
                  </div>
                </div>
              )}

              {/* BCC Field */}
              {showBCC && (
                <div className="p-4 border-b border-slate-200 dark:border-slate-700">
                  <div className="flex items-start gap-3">
                    <label className="text-sm font-medium text-slate-600 dark:text-slate-400 mt-2 w-16">BCC:</label>
                    <input
                      type="text"
                      placeholder="BCC-Empfänger..."
                      className="flex-1 px-3 py-2 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg text-sm text-slate-900 dark:text-white placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                    />
                  </div>
                </div>
              )}

              {/* Subject */}
              <div className="p-4 border-b border-slate-200 dark:border-slate-700">
                <div className="flex items-start gap-3">
                  <label className="text-sm font-medium text-slate-600 dark:text-slate-400 mt-2 w-16">Betreff:</label>
                  <input
                    type="text"
                    placeholder="Email-Betreff eingeben..."
                    value={composeSubject}
                    onChange={(e) => setComposeSubject(e.target.value)}
                    className="flex-1 px-3 py-2 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg text-sm text-slate-900 dark:text-white placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                  />
                </div>
              </div>

              {/* Body */}
              <div className="p-4 border-b border-slate-200 dark:border-slate-700">
                {/* Formatting Toolbar */}
                <div className="flex items-center gap-1 mb-3 pb-3 border-b border-slate-200 dark:border-slate-700">
                  <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
                    <Bold className="w-4 h-4 text-slate-500" />
                  </button>
                  <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
                    <Italic className="w-4 h-4 text-slate-500" />
                  </button>
                  <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
                    <Underline className="w-4 h-4 text-slate-500" />
                  </button>
                  <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
                    <LinkIcon className="w-4 h-4 text-slate-500" />
                  </button>
                  <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
                    <List className="w-4 h-4 text-slate-500" />
                  </button>
                </div>

                <textarea
                  placeholder="Schreibe deine Email hier..."
                  value={composeBody}
                  onChange={(e) => setComposeBody(e.target.value)}
                  rows={12}
                  className="w-full px-3 py-2 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg text-sm text-slate-900 dark:text-white placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-emerald-500 resize-none"
                />
              </div>

              {/* Attachments */}
              <div className="p-4">
                <div className="border-2 border-dashed border-slate-200 dark:border-slate-700 rounded-lg p-8 bg-slate-50 dark:bg-slate-900 hover:bg-slate-100 dark:hover:bg-slate-800 cursor-pointer transition-colors">
                  <div className="flex flex-col items-center text-center">
                    <Upload className="w-8 h-8 text-emerald-600 dark:text-emerald-400 mb-2" />
                    <p className="text-sm text-slate-600 dark:text-slate-400">
                      Dateien hier ablegen oder klicken zum Durchsuchen
                    </p>
                  </div>
                </div>

                {composeAttachments.length > 0 && (
                  <div className="mt-4 space-y-2">
                    {composeAttachments.map((file, idx) => (
                      <div
                        key={idx}
                        className="flex items-center justify-between p-2 bg-white dark:bg-slate-800 rounded border border-slate-200 dark:border-slate-700"
                      >
                        <div className="flex items-center gap-2">
                          <FileText className="w-4 h-4 text-slate-400" />
                          <span className="text-sm text-slate-900 dark:text-white">{file.name}</span>
                          <span className="text-xs text-slate-500">({file.size})</span>
                        </div>
                        <button className="p-1 hover:bg-slate-100 dark:hover:bg-slate-600 rounded">
                          <X className="w-4 h-4 text-slate-500" />
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>

            {/* Compose Footer */}
            <div className="p-4 bg-slate-50 dark:bg-slate-900 border-t border-slate-200 dark:border-slate-700 flex items-center justify-between">
              <div className="flex items-center gap-2 text-xs text-slate-500 dark:text-slate-400">
                <Clock className="w-3 h-3" />
                Alle Änderungen gespeichert
              </div>
              <div className="flex items-center gap-3">
                <button
                  onClick={handleBackToList}
                  className="px-4 py-2 bg-slate-100 dark:bg-slate-700 hover:bg-slate-200 dark:hover:bg-slate-600 text-slate-700 dark:text-slate-300 rounded-lg transition-colors text-sm"
                >
                  Entwurf speichern
                </button>
                <button
                  onClick={handleSendEmail}
                  className="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg transition-colors text-sm flex items-center gap-2"
                >
                  <Send className="w-4 h-4" />
                  Senden
                </button>
              </div>
            </div>
          </>
        )}
      </div>

      {/* Groups Modal */}
      <Dialog open={showGroupsModal} onOpenChange={setShowGroupsModal}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Email-Gruppen verwalten</DialogTitle>
            <DialogDescription>
              Erstelle Gruppen um an mehrere Personen gleichzeitig Emails zu versenden
            </DialogDescription>
          </DialogHeader>

          <div className="py-4">
            <button className="w-full px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg transition-colors mb-4 flex items-center justify-center gap-2">
              <Plus className="w-4 h-4" />
              Neue Gruppe erstellen
            </button>

            <div className="space-y-3">
              {groups.map((group) => (
                <div
                  key={group.id}
                  className="p-4 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg"
                >
                  <div className="flex items-start justify-between">
                    <div>
                      <h4 className="text-sm font-semibold text-slate-900 dark:text-white mb-1">
                        {group.name}
                      </h4>
                      <p className="text-xs text-slate-500 dark:text-slate-400 mb-3">
                        ({group.count} Personen)
                      </p>
                      <div className="flex -space-x-2">
                        {group.members.map((avatar, idx) => (
                          <Avatar key={idx} className="w-6 h-6 border-2 border-white dark:border-slate-800">
                            <AvatarFallback className="bg-emerald-100 dark:bg-emerald-900 text-emerald-700 dark:text-emerald-300 text-xs">
                              {avatar}
                            </AvatarFallback>
                          </Avatar>
                        ))}
                      </div>
                    </div>
                    <div className="flex gap-2">
                      <button className="p-2 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
                        <Edit className="w-4 h-4 text-slate-500" />
                      </button>
                      <button className="p-2 hover:bg-red-100 dark:hover:bg-red-900/30 rounded">
                        <Trash2 className="w-4 h-4 text-red-500" />
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* Toast */}
      {toast.show && (
        <div className="fixed bottom-4 right-4 z-50 animate-in slide-in-from-bottom duration-300">
          <div
            className={`px-4 py-3 rounded-xl shadow-lg border flex items-center gap-2 ${
              toast.type === 'success'
                ? 'bg-green-50 dark:bg-green-900/30 border-green-200 dark:border-green-800 text-green-700 dark:text-green-300'
                : toast.type === 'error'
                ? 'bg-red-50 dark:bg-red-900/30 border-red-200 dark:border-red-800 text-red-700 dark:text-red-300'
                : 'bg-blue-50 dark:bg-blue-900/30 border-blue-200 dark:border-blue-800 text-blue-700 dark:text-blue-300'
            }`}
          >
            {toast.type === 'success' && <Check className="w-4 h-4" />}
            <p className="text-sm font-medium">{toast.message}</p>
          </div>
        </div>
      )}
    </div>
  );
}
