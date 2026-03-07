import { useState } from 'react';
import { X, Clock, Users, Edit, Trash2, ExternalLink, Copy, QrCode, FileText, Download, Upload, Check, ChevronLeft, Bell } from 'lucide-react';
import { Avatar, AvatarFallback } from './ui/avatar';
import { Badge } from './ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from './ui/tabs';

interface MeetingDetailViewProps {
  meeting: any;
  onClose: () => void;
}

const teamMembers = [
  { id: 1, name: 'Anna Müller', avatar: 'AM', role: 'Project Manager', status: 'online', rsvp: 'accepted' },
  { id: 2, name: 'Sarah Klein', avatar: 'SK', role: 'UI/UX Designer', status: 'away', rsvp: 'pending' },
  { id: 3, name: 'Michael Berg', avatar: 'MB', role: 'Senior Developer', status: 'online', rsvp: 'accepted' },
];

const documents = [
  { 
    id: 'd1', 
    name: 'Design_Mockups_v3.fig', 
    size: '2.4 MB', 
    uploadedBy: 'Sarah Klein', 
    uploadedAt: 'vor 3 Stunden',
    type: 'figma'
  },
  { 
    id: 'd2', 
    name: 'Accessibility_Checklist.pdf', 
    size: '156 KB', 
    uploadedBy: 'Anna Müller', 
    uploadedAt: 'gestern',
    type: 'pdf'
  },
];

export function MeetingDetailView({ meeting, onClose }: MeetingDetailViewProps) {
  const [activeTab, setActiveTab] = useState('overview');
  const [showToast, setShowToast] = useState(false);

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'online':
        return 'bg-green-500';
      case 'away':
        return 'bg-yellow-500';
      default:
        return 'bg-slate-300 dark:bg-slate-600';
    }
  };

  const getRsvpIcon = (rsvp: string) => {
    switch (rsvp) {
      case 'accepted':
        return <Check className="w-4 h-4 text-green-500" />;
      case 'pending':
        return <Clock className="w-4 h-4 text-yellow-500" />;
      case 'declined':
        return <X className="w-4 h-4 text-red-500" />;
      default:
        return null;
    }
  };

  const getRsvpText = (rsvp: string) => {
    switch (rsvp) {
      case 'accepted':
        return '✅ Zugesagt';
      case 'pending':
        return '⏳ Ausstehend';
      case 'declined':
        return '❌ Abgesagt';
      default:
        return '';
    }
  };

  const handleCopyLink = () => {
    setShowToast(true);
    setTimeout(() => setShowToast(false), 3000);
  };

  const acceptedCount = teamMembers.filter(m => m.rsvp === 'accepted').length;
  const pendingCount = teamMembers.filter(m => m.rsvp === 'pending').length;
  const declinedCount = teamMembers.filter(m => m.rsvp === 'declined').length;

  return (
    <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
      <div className="bg-[#0f172a] w-full max-w-4xl max-h-[90vh] rounded-xl shadow-2xl flex flex-col">
        {/* Header */}
        <div className="p-6 border-b border-slate-700">
          <div className="flex items-start justify-between mb-2">
            <div className="flex items-center gap-3">
              <button
                onClick={onClose}
                className="p-2 hover:bg-slate-800 rounded-lg transition-colors"
              >
                <ChevronLeft className="w-5 h-5 text-slate-400" />
              </button>
              <div>
                <div className="flex items-center gap-3">
                  <h2 className="text-xl font-semibold text-white">📅 {meeting.name}</h2>
                  <Badge className={`${
                    meeting.status === 'live' 
                      ? 'bg-red-500/20 text-red-400 border-red-500/30' 
                      : 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30'
                  } border`}>
                    {meeting.status === 'live' ? (
                      <div className="flex items-center gap-2">
                        <div className="w-2 h-2 bg-red-500 rounded-full animate-pulse" />
                        Live
                      </div>
                    ) : (
                      <div className="flex items-center gap-2">
                        <Clock className="w-3 h-3" />
                        Geplant
                      </div>
                    )}
                  </Badge>
                </div>
                <p className="text-sm text-slate-400 mt-1">
                  {meeting.project} • {meeting.date}, {meeting.time}
                </p>
              </div>
            </div>
            <button
              onClick={onClose}
              className="p-2 hover:bg-slate-800 rounded-lg transition-colors"
            >
              <X className="w-5 h-5 text-slate-400" />
            </button>
          </div>
        </div>

        {/* Tabs */}
        <Tabs value={activeTab} onValueChange={setActiveTab} className="flex-1 flex flex-col overflow-hidden">
          <TabsList className="px-6 pt-4 bg-transparent border-b border-slate-700 justify-start">
            <TabsTrigger 
              value="overview" 
              className="data-[state=active]:border-b-2 data-[state=active]:border-emerald-500 data-[state=active]:text-emerald-400 rounded-none"
            >
              Übersicht
            </TabsTrigger>
            <TabsTrigger 
              value="participants" 
              className="data-[state=active]:border-b-2 data-[state=active]:border-emerald-500 data-[state=active]:text-emerald-400 rounded-none"
            >
              Teilnehmer
            </TabsTrigger>
            <TabsTrigger 
              value="documents" 
              className="data-[state=active]:border-b-2 data-[state=active]:border-emerald-500 data-[state=active]:text-emerald-400 rounded-none"
            >
              Dokumente
            </TabsTrigger>
            <TabsTrigger 
              value="workroom" 
              className="data-[state=active]:border-b-2 data-[state=active]:border-emerald-500 data-[state=active]:text-emerald-400 rounded-none"
            >
              Workroom
            </TabsTrigger>
          </TabsList>

          <div className="flex-1 overflow-y-auto">
            {/* Overview Tab */}
            <TabsContent value="overview" className="p-6 space-y-6 mt-0">
              {/* Status */}
              <div>
                <h3 className="text-sm font-semibold text-slate-400 uppercase tracking-wide mb-3">Status</h3>
                <div className="space-y-2">
                  <div className="flex items-center gap-2 text-slate-200">
                    <Clock className="w-4 h-4 text-yellow-500" />
                    <span className="text-sm">Geplant für {meeting.date}, {meeting.time.split('-')[0]} Uhr</span>
                  </div>
                  <div className="flex items-center gap-2 text-slate-200">
                    <Bell className="w-4 h-4 text-yellow-500" />
                    <span className="text-sm">Erinnerung 15 Min. vorher</span>
                  </div>
                  <div className="flex items-center gap-2 text-slate-200">
                    <span className="text-sm">⏱️ Startet in 2 Tagen, 3 Stunden</span>
                  </div>
                </div>
              </div>

              {/* Description */}
              <div>
                <h3 className="text-sm font-semibold text-slate-400 uppercase tracking-wide mb-3">Beschreibung</h3>
                <p className="text-sm text-slate-300 leading-relaxed">
                  {meeting.description}
                </p>
              </div>

              {/* Assignment */}
              <div>
                <h3 className="text-sm font-semibold text-slate-400 uppercase tracking-wide mb-3">Zuordnung</h3>
                <div className="space-y-2">
                  <div className="flex items-center gap-2 text-slate-200">
                    <span className="text-sm">📁 Projekt: {meeting.project}</span>
                  </div>
                  <div className="flex items-center gap-2 text-slate-200">
                    <span className="text-sm">👥 Team: {meeting.team}</span>
                  </div>
                </div>
              </div>

              {/* Quick Actions */}
              <div>
                <h3 className="text-sm font-semibold text-slate-400 uppercase tracking-wide mb-3">Quick-Actions</h3>
                <div className="grid grid-cols-3 gap-3">
                  <button className="p-4 bg-[#1e293b] border border-slate-700 rounded-xl hover:border-emerald-500 transition-colors text-center">
                    <div className="text-3xl mb-2">📝</div>
                    <div className="text-xs text-slate-300 font-medium">Notizen</div>
                    <div className="text-xs text-slate-500 mt-1">0 Einträge</div>
                  </button>
                  <button className="p-4 bg-[#1e293b] border border-slate-700 rounded-xl hover:border-emerald-500 transition-colors text-center">
                    <div className="text-3xl mb-2">📄</div>
                    <div className="text-xs text-slate-300 font-medium">Dokumente</div>
                    <div className="text-xs text-slate-500 mt-1">{documents.length} Dateien</div>
                  </button>
                  <button className="p-4 bg-[#1e293b] border border-slate-700 rounded-xl hover:border-emerald-500 transition-colors text-center">
                    <div className="text-3xl mb-2">🎨</div>
                    <div className="text-xs text-slate-300 font-medium">Whiteboard</div>
                    <div className="text-xs text-slate-500 mt-1">Vorbereiten</div>
                  </button>
                </div>
              </div>

              {/* Meeting Link */}
              <div>
                <h3 className="text-sm font-semibold text-slate-400 uppercase tracking-wide mb-3">Meeting-Link</h3>
                <div className="flex items-center gap-2 p-3 bg-[#1e293b] border border-slate-700 rounded-lg">
                  <code className="flex-1 text-sm text-emerald-400 font-mono">
                    https://swissbiz.ch/meet/xyz789abc
                  </code>
                  <button 
                    onClick={handleCopyLink}
                    className="p-2 hover:bg-slate-700 rounded transition-colors"
                  >
                    <Copy className="w-4 h-4 text-slate-400" />
                  </button>
                  <button className="p-2 hover:bg-slate-700 rounded transition-colors">
                    <QrCode className="w-4 h-4 text-slate-400" />
                  </button>
                </div>
              </div>

              {/* Actions */}
              <div className="flex flex-wrap gap-3 pt-4 border-t border-slate-700">
                <button className="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg transition-colors text-sm flex items-center gap-2">
                  <Edit className="w-4 h-4" />
                  Meeting bearbeiten
                </button>
                <button className="px-4 py-2 bg-red-600/20 hover:bg-red-600/30 text-red-400 rounded-lg transition-colors text-sm flex items-center gap-2">
                  <Trash2 className="w-4 h-4" />
                  Meeting löschen
                </button>
                <button className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-slate-300 rounded-lg transition-colors text-sm flex items-center gap-2">
                  <ExternalLink className="w-4 h-4" />
                  Zum Kalender-Eintrag
                </button>
                <button className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-slate-300 rounded-lg transition-colors text-sm flex items-center gap-2">
                  <Bell className="w-4 h-4" />
                  Erinnerung ändern
                </button>
              </div>
            </TabsContent>

            {/* Participants Tab */}
            <TabsContent value="participants" className="p-6 space-y-6 mt-0">
              <div className="space-y-3">
                {teamMembers.map((member) => (
                  <div key={member.id} className="p-4 bg-[#1e293b] border border-slate-700 rounded-lg">
                    <div className="flex items-center gap-3 mb-2">
                      <div className="relative">
                        <Avatar className="w-10 h-10">
                          <AvatarFallback className="bg-emerald-900 text-emerald-300 text-sm">
                            {member.avatar}
                          </AvatarFallback>
                        </Avatar>
                        <div className={`absolute -bottom-0.5 -right-0.5 w-3 h-3 rounded-full border-2 border-[#1e293b] ${getStatusColor(member.status)}`} />
                      </div>
                      <div className="flex-1">
                        <p className="text-sm font-medium text-white">{member.name}</p>
                        <p className="text-xs text-slate-400">{member.role}</p>
                      </div>
                      <div className="text-sm text-slate-300">
                        {getRsvpText(member.rsvp)}
                      </div>
                    </div>
                  </div>
                ))}
              </div>

              {/* RSVP Stats */}
              <div className="p-4 bg-[#1e293b] border border-slate-700 rounded-lg">
                <h4 className="text-sm font-semibold text-slate-400 uppercase tracking-wide mb-3">Teilnehmer-Status</h4>
                <div className="space-y-2 text-sm">
                  <div className="flex items-center justify-between">
                    <span className="text-slate-300">• ✅ Zugesagt:</span>
                    <span className="text-white font-medium">{acceptedCount} Personen</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-slate-300">• ⏳ Ausstehend:</span>
                    <span className="text-white font-medium">{pendingCount} Person</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-slate-300">• ❌ Abgesagt:</span>
                    <span className="text-white font-medium">{declinedCount} Personen</span>
                  </div>
                </div>
              </div>

              <button className="w-full px-4 py-3 bg-slate-700 hover:bg-slate-600 text-slate-300 rounded-lg transition-colors text-sm flex items-center justify-center gap-2">
                <Users className="w-4 h-4" />
                Weitere Teilnehmer hinzufügen
              </button>
            </TabsContent>

            {/* Documents Tab */}
            <TabsContent value="documents" className="p-6 space-y-6 mt-0">
              <div className="flex gap-3">
                <button className="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg transition-colors text-sm flex items-center gap-2">
                  <Upload className="w-4 h-4" />
                  Dateien hochladen
                </button>
                <button className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-slate-300 rounded-lg transition-colors text-sm flex items-center gap-2">
                  <FileText className="w-4 h-4" />
                  Aus Projekt verlinken
                </button>
              </div>

              <div>
                <h3 className="text-sm font-semibold text-slate-400 uppercase tracking-wide mb-3">Meeting-Dokumente</h3>
                <div className="space-y-3">
                  {documents.map((doc) => (
                    <div key={doc.id} className="p-4 bg-[#1e293b] border border-slate-700 rounded-lg">
                      <div className="flex items-center gap-3 mb-3">
                        <FileText className={`w-6 h-6 ${doc.type === 'pdf' ? 'text-red-500' : 'text-purple-500'}`} />
                        <div className="flex-1">
                          <p className="text-sm font-medium text-white">{doc.name}</p>
                          <p className="text-xs text-slate-400">
                            {doc.size} • Hochgeladen von {doc.uploadedBy} • {doc.uploadedAt}
                          </p>
                        </div>
                      </div>
                      <div className="flex gap-2">
                        <button className="px-3 py-1.5 bg-slate-700 hover:bg-slate-600 text-slate-200 rounded text-xs transition-colors">
                          Vorschau
                        </button>
                        <button className="px-3 py-1.5 bg-slate-700 hover:bg-slate-600 text-slate-200 rounded text-xs transition-colors flex items-center gap-1">
                          <Download className="w-3 h-3" />
                          Download
                        </button>
                        <button className="px-3 py-1.5 bg-red-600/20 hover:bg-red-600/30 text-red-400 rounded text-xs transition-colors">
                          Entfernen
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {/* Agenda */}
              <div>
                <h3 className="text-sm font-semibold text-slate-400 uppercase tracking-wide mb-3">Agenda & Notizen (Vorbereitung)</h3>
                <div className="p-5 bg-[#1e293b] border border-slate-700 rounded-lg">
                  <div className="font-mono text-sm text-slate-300 space-y-2 whitespace-pre-line">
{`Agenda:
1. Begrüßung & Ziele (5 min)
2. Design-Walkthrough (20 min)
3. Accessibility Review (15 min)
4. Feedback-Runde (15 min)
5. Nächste Schritte (5 min)

Offene Fragen:
- Farb-Kontraste für WCAG 2.1 AA prüfen
- Touch-Target Größen verifizieren`}
                  </div>
                </div>
                <div className="flex gap-2 mt-3">
                  <button className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-slate-300 rounded-lg transition-colors text-sm">
                    Bearbeiten
                  </button>
                  <button className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-slate-300 rounded-lg transition-colors text-sm flex items-center gap-2">
                    <Download className="w-4 h-4" />
                    Als PDF exportieren
                  </button>
                </div>
              </div>
            </TabsContent>

            {/* Workroom Tab */}
            <TabsContent value="workroom" className="p-6 mt-0">
              <div className="text-center py-8">
                <div className="text-6xl mb-4">🎨</div>
                <h3 className="text-lg font-semibold text-white mb-2">WHITEBOARD & COLLABORATION</h3>
                
                <div className="max-w-md mx-auto mb-6">
                  <div className="p-3 bg-slate-700/50 rounded-lg mb-4">
                    <p className="text-sm text-slate-300">
                      <span className="font-medium">Status:</span> ⚪ Workroom ist noch nicht aktiv
                    </p>
                    <p className="text-xs text-slate-400 mt-1">(Meeting startet in 2 Tagen)</p>
                  </div>
                  <p className="text-sm text-slate-400">
                    Du kannst das Whiteboard bereits jetzt vorbereiten und Materialien für die Präsentation hochladen.
                  </p>
                </div>

                <div className="max-w-md mx-auto space-y-4">
                  <div className="p-6 bg-[#1e293b] border border-slate-700 rounded-lg text-left">
                    <div className="text-4xl mb-3">🎨</div>
                    <h4 className="text-sm font-semibold text-white mb-2">Whiteboard vorbereiten</h4>
                    <p className="text-xs text-slate-400 mb-4 leading-relaxed">
                      Erstelle Diagramme, skizziere Ideen oder strukturiere die Agenda visuell.
                    </p>
                    <button className="w-full px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg transition-colors text-sm">
                      Whiteboard öffnen
                    </button>
                    <p className="text-xs text-blue-400 mt-3">
                      ℹ️ Änderungen werden automatisch gespeichert
                    </p>
                  </div>

                  <div className="p-6 bg-[#1e293b] border border-slate-700 rounded-lg text-left">
                    <div className="text-4xl mb-3">📊</div>
                    <h4 className="text-sm font-semibold text-white mb-2">Präsentationen & Screens</h4>
                    <div className="space-y-2 mb-4">
                      <div className="text-xs text-slate-300 p-2 bg-slate-800 rounded">
                        📄 Design_Mockups_v3.fig (verlinkt)
                      </div>
                      <div className="text-xs text-slate-300 p-2 bg-slate-800 rounded">
                        🖼️ Accessibility_Examples.png
                      </div>
                    </div>
                    <button className="w-full px-4 py-2 bg-slate-700 hover:bg-slate-600 text-slate-200 rounded-lg transition-colors text-sm">
                      + Weitere Präsentation hinzufügen
                    </button>
                  </div>

                  <div className="p-6 bg-[#1e293b] border border-slate-700 rounded-lg text-left">
                    <div className="text-4xl mb-3">📝</div>
                    <h4 className="text-sm font-semibold text-white mb-2">Gemeinsame Notizen</h4>
                    <p className="text-xs text-slate-400 mb-4 leading-relaxed">
                      Kollaboratives Live-Dokument für alle Teilnehmer während des Meetings.
                    </p>
                    <button className="w-full px-4 py-2 bg-slate-700 hover:bg-slate-600 text-slate-200 rounded-lg transition-colors text-sm">
                      Notizen öffnen
                    </button>
                  </div>
                </div>

                <div className="mt-8 p-4 bg-blue-900/20 border border-blue-700 rounded-lg max-w-md mx-auto">
                  <p className="text-xs text-blue-300 leading-relaxed text-left">
                    <strong>Während des Meetings verfügbar:</strong><br />
                    • 🎨 Live-Whiteboard mit allen Teilnehmern<br />
                    • 🖥️ Screen-Sharing<br />
                    • 📄 Gemeinsames Dokument-Editing<br />
                    • 🎥 Video & Audio (optional)<br />
                    • 💬 Meeting-Chat
                  </p>
                </div>
              </div>
            </TabsContent>
          </div>
        </Tabs>
      </div>

      {/* Toast */}
      {showToast && (
        <div className="fixed top-4 right-4 z-50 animate-in slide-in-from-right duration-300">
          <div className="px-4 py-3 rounded-xl shadow-lg border bg-green-900/90 border-green-700 text-green-100">
            <p className="text-sm font-medium">✅ Link kopiert!</p>
          </div>
        </div>
      )}
    </div>
  );
}
