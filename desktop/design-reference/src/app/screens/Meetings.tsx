import { useState } from 'react';
import { Plus, Calendar as CalendarIcon, Clock, Users, Video, FileText, Edit, Trash2, ExternalLink, Download, Upload, Copy, QrCode, MessageCircle, Check, ChevronDown } from 'lucide-react';
import { Avatar, AvatarFallback } from '../components/ui/avatar';
import { Badge } from '../components/ui/badge';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '../components/ui/dialog';
import { Label } from '../components/ui/label';
import { Input } from '../components/ui/input';
import { Textarea } from '../components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../components/ui/select';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../components/ui/tabs';
import { MeetingDetailView } from '../components/MeetingDetailView';

const meetings = [
  { 
    id: 'm1', 
    name: 'Sprint Planning', 
    status: 'live', 
    project: 'Website Redesign', 
    date: '20.12.2024', 
    time: '14:00-15:30',
    dateObj: new Date('2024-12-20T14:00'),
    participants: 4,
    description: 'Planung des nächsten Sprints mit Review der User Stories und Schätzung der Aufwände.',
    team: 'Entwicklung',
  },
  { 
    id: 'm2', 
    name: 'Design Review', 
    status: 'scheduled', 
    project: 'Mobile App', 
    date: '20.12.2024', 
    time: '16:00-17:00',
    dateObj: new Date('2024-12-20T16:00'),
    participants: 3,
    description: 'Review der neuen Design-Mockups für die Mobile App.',
    team: 'Design',
  },
  { 
    id: 'm3', 
    name: 'Kundenpräsentation', 
    status: 'scheduled', 
    project: 'Website Redesign',
    date: '22.12.2024', 
    time: '10:00-11:30',
    dateObj: new Date('2024-12-22T10:00'),
    participants: 5,
    external: 2,
    description: 'Präsentation des aktuellen Projektstands für den Kunden.',
    team: 'Entwicklung',
  },
  { 
    id: 'm4', 
    name: 'Daily Standup', 
    status: 'live', 
    project: 'Mobile App', 
    date: '20.12.2024', 
    time: '09:00-09:15',
    dateObj: new Date('2024-12-20T09:00'),
    participants: 6,
    description: 'Tägliches Sync-Meeting des Entwicklungsteams.',
    team: 'Entwicklung',
  },
  { 
    id: 'm5', 
    name: 'Marketing Strategiemeeting', 
    status: 'scheduled', 
    project: 'Q1 Kampagne', 
    date: '20.12.2024', 
    time: '15:00-16:30',
    dateObj: new Date('2024-12-20T15:00'),
    participants: 5,
    description: 'Planung der Marketing-Aktivitäten für Q1 2025.',
    team: 'Marketing',
  },
  { 
    id: 'm6', 
    name: 'Tech Talk - Cloud Migration', 
    status: 'scheduled', 
    project: 'Infrastructure', 
    date: '21.12.2024', 
    time: '11:00-12:00',
    dateObj: new Date('2024-12-21T11:00'),
    participants: 8,
    description: 'Präsentation und Diskussion über die geplante Cloud-Migration.',
    team: 'Entwicklung',
  },
  { 
    id: 'm7', 
    name: 'UX Research Präsentation', 
    status: 'scheduled', 
    project: 'Mobile App', 
    date: '21.12.2024', 
    time: '14:00-15:00',
    dateObj: new Date('2024-12-21T14:00'),
    participants: 7,
    description: 'Vorstellung der Ergebnisse aus der UX Research Phase.',
    team: 'Design',
  },
  { 
    id: 'm8', 
    name: 'Budget Review Q4', 
    status: 'scheduled', 
    project: 'Finance', 
    date: '23.12.2024', 
    time: '09:00-10:30',
    dateObj: new Date('2024-12-23T09:00'),
    participants: 4,
    external: 1,
    description: 'Quartalsabschluss und Budget-Review für Q4 2024.',
    team: 'Management',
  },
  { 
    id: 'm9', 
    name: 'Team Retro', 
    status: 'scheduled', 
    project: 'Website Redesign', 
    date: '23.12.2024', 
    time: '13:00-14:30',
    dateObj: new Date('2024-12-23T13:00'),
    participants: 6,
    description: 'Sprint Retrospektive - Was lief gut, was können wir verbessern?',
    team: 'Entwicklung',
  },
  { 
    id: 'm10', 
    name: 'Client Onboarding', 
    status: 'scheduled', 
    project: 'New Client Project', 
    date: '24.12.2024', 
    time: '10:00-11:00',
    dateObj: new Date('2024-12-24T10:00'),
    participants: 3,
    external: 4,
    description: 'Kickoff-Meeting mit neuem Kunden und Projektvorstellung.',
    team: 'Management',
  },
  { 
    id: 'm11', 
    name: 'Security Audit Meeting', 
    status: 'scheduled', 
    project: 'Infrastructure', 
    date: '26.12.2024', 
    time: '14:00-16:00',
    dateObj: new Date('2024-12-26T14:00'),
    participants: 5,
    external: 2,
    description: 'Security Review mit externen Auditoren.',
    team: 'Entwicklung',
  },
  { 
    id: 'm12', 
    name: 'All Hands Meeting', 
    status: 'scheduled', 
    project: 'Company Update', 
    date: '27.12.2024', 
    time: '15:00-16:00',
    dateObj: new Date('2024-12-27T15:00'),
    participants: 20,
    description: 'Firmenweites Update-Meeting mit dem gesamten Team.',
    team: 'Management',
  },
  { 
    id: 'm13', 
    name: 'Code Review Session', 
    status: 'past', 
    project: 'Mobile App', 
    date: '18.12.2024', 
    time: '10:00-11:00',
    dateObj: new Date('2024-12-18T10:00'),
    participants: 4,
    description: 'Review des neuen Authentication-Moduls.',
    team: 'Entwicklung',
  },
  { 
    id: 'm14', 
    name: 'Design System Workshop', 
    status: 'past', 
    project: 'Design System', 
    date: '17.12.2024', 
    time: '13:00-15:00',
    dateObj: new Date('2024-12-17T13:00'),
    participants: 6,
    description: 'Workshop zur Einführung des neuen Design Systems.',
    team: 'Design',
  },
];

const teamMembers = [
  { id: 1, name: 'Anna Müller', avatar: 'AM', role: 'Project Manager', status: 'online', selected: true, organizer: true, rsvp: 'accepted' },
  { id: 2, name: 'Michael Berg', avatar: 'MB', role: 'Senior Developer', status: 'online', selected: true, rsvp: 'accepted' },
  { id: 3, name: 'Sarah Klein', avatar: 'SK', role: 'UI/UX Designer', status: 'away', selected: true, rsvp: 'pending' },
  { id: 4, name: 'Tom Weber', avatar: 'TW', role: 'Backend Developer', status: 'offline', selected: false, rsvp: null },
  { id: 5, name: 'Lisa Fischer', avatar: 'LF', role: 'Marketing Manager', status: 'online', selected: false, rsvp: null },
];

const projects = [
  { id: 1, name: 'Website Redesign', members: 5 },
  { id: 2, name: 'Mobile App', members: 4 },
  { id: 3, name: 'Backend API', members: 3 },
];

const teams = [
  { id: 1, name: 'Entwicklung', members: 12 },
  { id: 2, name: 'Design', members: 8 },
  { id: 3, name: 'Marketing', members: 6 },
];

type FilterType = 'all' | 'live' | 'scheduled' | 'past';

export function Meetings() {
  const [activeFilter, setActiveFilter] = useState<FilterType>('all');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [selectedMeeting, setSelectedMeeting] = useState<any>(null);
  const [meetingType, setMeetingType] = useState<'project' | 'team' | 'individual'>('project');
  const [meetingTitle, setMeetingTitle] = useState('');
  const [meetingDescription, setMeetingDescription] = useState('');
  const [selectedProject, setSelectedProject] = useState('');
  const [selectedTeam, setSelectedTeam] = useState('');
  const [meetingDate, setMeetingDate] = useState('');
  const [meetingTimeFrom, setMeetingTimeFrom] = useState('14:00');
  const [meetingTimeTo, setMeetingTimeTo] = useState('15:30');
  const [selectedParticipants, setSelectedParticipants] = useState<number[]>([1, 2, 3]);
  const [toast, setToast] = useState<{ show: boolean; message: string; type: 'success' | 'error' | 'info' }>({ show: false, message: '', type: 'success' });

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

  const showToast = (message: string, type: 'success' | 'error' | 'info' = 'success') => {
    setToast({ show: true, message, type });
    setTimeout(() => setToast({ show: false, message: '', type: 'success' }), 4000);
  };

  const handleCreateMeeting = () => {
    if (!meetingTitle || !meetingDate) {
      showToast('Bitte fülle alle Pflichtfelder aus', 'error');
      return;
    }
    showToast('✅ Meeting erstellt', 'success');
    setShowCreateModal(false);
    setMeetingTitle('');
    setMeetingDescription('');
    setMeetingDate('');
  };

  const toggleParticipant = (id: number) => {
    setSelectedParticipants(prev =>
      prev.includes(id) ? prev.filter(p => p !== id) : [...prev, id]
    );
  };

  const filteredMeetings = meetings.filter(m => {
    if (activeFilter === 'all') return true;
    return m.status === activeFilter;
  });

  const liveMeetings = filteredMeetings.filter(m => m.status === 'live');
  const todayMeetings = filteredMeetings.filter(m => {
    const today = new Date();
    return m.dateObj.toDateString() === today.toDateString() && m.status !== 'live';
  });
  const thisWeekMeetings = filteredMeetings.filter(m => {
    const today = new Date();
    const weekFromNow = new Date(today.getTime() + 7 * 24 * 60 * 60 * 1000);
    return m.dateObj > today && m.dateObj < weekFromNow && m.status !== 'live';
  });
  const scheduledMeetings = filteredMeetings.filter(m => m.status === 'scheduled');
  const pastMeetings = filteredMeetings.filter(m => m.status === 'past');

  const liveCount = meetings.filter(m => m.status === 'live').length;

  return (
    <div className="flex-1 overflow-y-auto p-4 md:p-8 bg-slate-50 dark:bg-slate-900">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between mb-6 md:mb-8">
          <div>
            <h1 className="text-slate-900 dark:text-white mb-1">Meetings</h1>
            <p className="text-sm text-slate-500 dark:text-slate-400">
              Verwalte deine Meetings und Termine
            </p>
          </div>
          <button
            onClick={() => setShowCreateModal(true)}
            className="flex items-center gap-2 px-4 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors"
          >
            <Plus className="w-4 h-4" />
            <span className="hidden sm:inline">Neues Meeting</span>
          </button>
        </div>

        {/* Filter Chips */}
        <div className="flex gap-2 mb-6 overflow-x-auto pb-2">
          <button
            onClick={() => setActiveFilter('all')}
            className={`px-4 py-2 rounded-full text-sm font-medium whitespace-nowrap transition-colors ${
              activeFilter === 'all'
                ? 'bg-emerald-600 text-white'
                : 'bg-white dark:bg-slate-800 text-slate-700 dark:text-slate-300 border border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-700'
            }`}
          >
            Alle
          </button>
          <button
            onClick={() => setActiveFilter('live')}
            className={`px-4 py-2 rounded-full text-sm font-medium whitespace-nowrap transition-colors flex items-center gap-2 ${
              activeFilter === 'live'
                ? 'bg-red-600 text-white'
                : 'bg-white dark:bg-slate-800 text-slate-700 dark:text-slate-300 border border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-700'
            }`}
          >
            {liveCount > 0 && <div className="w-2 h-2 bg-red-500 rounded-full animate-pulse" />}
            🔴 Live {liveCount > 0 && `(${liveCount})`}
          </button>
          <button
            onClick={() => setActiveFilter('scheduled')}
            className={`px-4 py-2 rounded-full text-sm font-medium whitespace-nowrap transition-colors ${
              activeFilter === 'scheduled'
                ? 'bg-emerald-600 text-white'
                : 'bg-white dark:bg-slate-800 text-slate-700 dark:text-slate-300 border border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-700'
            }`}
          >
            ⏰ Geplant
          </button>
          <button
            onClick={() => setActiveFilter('past')}
            className={`px-4 py-2 rounded-full text-sm font-medium whitespace-nowrap transition-colors ${
              activeFilter === 'past'
                ? 'bg-emerald-600 text-white'
                : 'bg-white dark:bg-slate-800 text-slate-700 dark:text-slate-300 border border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-700'
            }`}
          >
            ✅ Vergangene
          </button>
        </div>

        {/* Meetings List */}
        <div className="space-y-8">
          {/* Live Now */}
          {liveMeetings.length > 0 && (
            <div>
              <h2 className="text-lg font-semibold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
                <div className="w-3 h-3 bg-red-500 rounded-full animate-pulse" />
                LIVE JETZT ({liveMeetings.length})
              </h2>
              <div className="space-y-4">
                {liveMeetings.map((meeting) => (
                  <div
                    key={meeting.id}
                    className="bg-white dark:bg-slate-800 border-2 border-red-500 rounded-xl p-6 shadow-lg hover:shadow-xl transition-shadow cursor-pointer"
                    onClick={() => setSelectedMeeting(meeting)}
                  >
                    <div className="flex items-start justify-between mb-4">
                      <div className="flex items-center gap-3">
                        <CalendarIcon className="w-6 h-6 text-red-500" />
                        <div>
                          <h3 className="font-semibold text-slate-900 dark:text-white">{meeting.name}</h3>
                          <p className="text-sm text-slate-600 dark:text-slate-400">
                            {meeting.project} • Jetzt bis {meeting.time.split('-')[1]}
                          </p>
                        </div>
                      </div>
                      <Badge className="bg-red-500 text-white px-3 py-1 flex items-center gap-2">
                        <div className="w-2 h-2 bg-white rounded-full animate-pulse" />
                        Live
                      </Badge>
                    </div>
                    <div className="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-400 mb-4">
                      <Users className="w-4 h-4" />
                      <span>{meeting.participants} Teilnehmer aktiv</span>
                    </div>
                    <button className="w-full px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg transition-colors font-medium">
                      Meeting beitreten
                    </button>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Today */}
          {todayMeetings.length > 0 && (
            <div>
              <h2 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">HEUTE</h2>
              <div className="space-y-4">
                {todayMeetings.map((meeting) => (
                  <div
                    key={meeting.id}
                    className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl p-4 md:p-6 hover:shadow-lg transition-shadow cursor-pointer"
                    onClick={() => setSelectedMeeting(meeting)}
                  >
                    <div className="flex flex-col sm:flex-row items-start justify-between mb-3 gap-2">
                      <div className="flex items-start gap-3 flex-1 min-w-0">
                        <CalendarIcon className="w-5 h-5 text-emerald-600 dark:text-emerald-400 flex-shrink-0 mt-0.5" />
                        <div className="min-w-0 flex-1">
                          <h3 className="font-semibold text-slate-900 dark:text-white text-sm md:text-base truncate">{meeting.name}</h3>
                          <p className="text-xs md:text-sm text-slate-600 dark:text-slate-400 truncate">
                            {meeting.project} • Heute, {meeting.time}
                          </p>
                        </div>
                      </div>
                      <div className="flex items-center gap-2 text-xs md:text-sm text-yellow-600 dark:text-yellow-400 whitespace-nowrap">
                        <Clock className="w-3 h-3 md:w-4 md:h-4" />
                        <span>{meeting.time.split('-')[0]}</span>
                      </div>
                    </div>
                    <div className="flex items-center gap-2 text-xs md:text-sm text-slate-600 dark:text-slate-400 mb-4">
                      <Users className="w-3 h-3 md:w-4 md:h-4" />
                      <span>{meeting.participants} Teilnehmer{meeting.external ? ` + ${meeting.external} Externe` : ''}</span>
                    </div>
                    <div className="flex flex-col sm:flex-row gap-2">
                      <button className="flex-1 px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg transition-colors text-sm">
                        Details
                      </button>
                      <button className="flex-1 sm:flex-initial px-4 py-2 bg-slate-100 dark:bg-slate-700 hover:bg-slate-200 dark:hover:bg-slate-600 text-slate-700 dark:text-slate-300 rounded-lg transition-colors text-sm">
                        Zum Kalender
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* This Week */}
          {thisWeekMeetings.length > 0 && (
            <div>
              <h2 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">DIESE WOCHE</h2>
              <div className="space-y-4">
                {thisWeekMeetings.map((meeting) => (
                  <div
                    key={meeting.id}
                    className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl p-6 hover:shadow-lg transition-shadow cursor-pointer"
                    onClick={() => setSelectedMeeting(meeting)}
                  >
                    <div className="flex items-start justify-between mb-3">
                      <div className="flex items-center gap-3">
                        <CalendarIcon className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
                        <div>
                          <h3 className="font-semibold text-slate-900 dark:text-white">{meeting.name}</h3>
                          <p className="text-sm text-slate-600 dark:text-slate-400">
                            {meeting.project} • {meeting.date}, {meeting.time}
                          </p>
                        </div>
                      </div>
                      <div className="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-400">
                        <Clock className="w-4 h-4" />
                        <span>{meeting.date.split('.')[0]}.{meeting.date.split('.')[1]}.</span>
                      </div>
                    </div>
                    <div className="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-400 mb-4">
                      <Users className="w-4 h-4" />
                      <span>{meeting.participants} Teilnehmer{meeting.external ? ` + ${meeting.external} Externe` : ''}</span>
                    </div>
                    <div className="flex gap-2">
                      <button className="flex-1 px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg transition-colors text-sm">
                        Details
                      </button>
                      <button className="px-4 py-2 bg-slate-100 dark:bg-slate-700 hover:bg-slate-200 dark:hover:bg-slate-600 text-slate-700 dark:text-slate-300 rounded-lg transition-colors text-sm">
                        Vorbereiten
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* All Scheduled - shown when filter is 'scheduled' */}
          {activeFilter === 'scheduled' && scheduledMeetings.length > 0 && (
            <div>
              <h2 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">GEPLANTE MEETINGS ({scheduledMeetings.length})</h2>
              <div className="space-y-4">
                {scheduledMeetings.map((meeting) => (
                  <div
                    key={meeting.id}
                    className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl p-4 md:p-6 hover:shadow-lg transition-shadow cursor-pointer"
                    onClick={() => setSelectedMeeting(meeting)}
                  >
                    <div className="flex flex-col sm:flex-row items-start justify-between mb-3 gap-2">
                      <div className="flex items-start gap-3 flex-1 min-w-0">
                        <CalendarIcon className="w-5 h-5 text-emerald-600 dark:text-emerald-400 flex-shrink-0 mt-0.5" />
                        <div className="min-w-0 flex-1">
                          <h3 className="font-semibold text-slate-900 dark:text-white text-sm md:text-base truncate">{meeting.name}</h3>
                          <p className="text-xs md:text-sm text-slate-600 dark:text-slate-400 truncate">
                            {meeting.project} • {meeting.date}, {meeting.time}
                          </p>
                        </div>
                      </div>
                      <div className="flex items-center gap-2 text-xs md:text-sm text-slate-600 dark:text-slate-400 whitespace-nowrap">
                        <Clock className="w-3 h-3 md:w-4 md:h-4" />
                        <span>{meeting.date}</span>
                      </div>
                    </div>
                    <div className="flex items-center gap-2 text-xs md:text-sm text-slate-600 dark:text-slate-400 mb-4">
                      <Users className="w-3 h-3 md:w-4 md:h-4" />
                      <span>{meeting.participants} Teilnehmer{meeting.external ? ` + ${meeting.external} Externe` : ''}</span>
                    </div>
                    <div className="flex flex-col sm:flex-row gap-2">
                      <button className="flex-1 px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg transition-colors text-sm">
                        Details
                      </button>
                      <button className="flex-1 sm:flex-initial px-4 py-2 bg-slate-100 dark:bg-slate-700 hover:bg-slate-200 dark:hover:bg-slate-600 text-slate-700 dark:text-slate-300 rounded-lg transition-colors text-sm">
                        Vorbereiten
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Past Meetings - shown when filter is 'past' */}
          {activeFilter === 'past' && pastMeetings.length > 0 && (
            <div>
              <h2 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">VERGANGENE MEETINGS ({pastMeetings.length})</h2>
              <div className="space-y-4">
                {pastMeetings.map((meeting) => (
                  <div
                    key={meeting.id}
                    className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl p-4 md:p-6 hover:shadow-lg transition-shadow cursor-pointer opacity-75"
                    onClick={() => setSelectedMeeting(meeting)}
                  >
                    <div className="flex flex-col sm:flex-row items-start justify-between mb-3 gap-2">
                      <div className="flex items-start gap-3 flex-1 min-w-0">
                        <CalendarIcon className="w-5 h-5 text-slate-400 flex-shrink-0 mt-0.5" />
                        <div className="min-w-0 flex-1">
                          <h3 className="font-semibold text-slate-900 dark:text-white text-sm md:text-base truncate">{meeting.name}</h3>
                          <p className="text-xs md:text-sm text-slate-600 dark:text-slate-400 truncate">
                            {meeting.project} • {meeting.date}, {meeting.time}
                          </p>
                        </div>
                      </div>
                      <Badge variant="outline" className="text-xs whitespace-nowrap">
                        Abgeschlossen
                      </Badge>
                    </div>
                    <div className="flex items-center gap-2 text-xs md:text-sm text-slate-600 dark:text-slate-400 mb-4">
                      <Users className="w-3 h-3 md:w-4 md:h-4" />
                      <span>{meeting.participants} Teilnehmer</span>
                    </div>
                    <div className="flex flex-col sm:flex-row gap-2">
                      <button className="flex-1 px-4 py-2 bg-slate-100 dark:bg-slate-700 hover:bg-slate-200 dark:hover:bg-slate-600 text-slate-700 dark:text-slate-300 rounded-lg transition-colors text-sm">
                        Notizen ansehen
                      </button>
                      <button className="flex-1 sm:flex-initial px-4 py-2 bg-slate-100 dark:bg-slate-700 hover:bg-slate-200 dark:hover:bg-slate-600 text-slate-700 dark:text-slate-300 rounded-lg transition-colors text-sm">
                        Aufzeichnung
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {filteredMeetings.length === 0 && (
            <div className="text-center py-20">
              <div className="text-6xl mb-4">📅</div>
              <h3 className="text-lg font-medium text-slate-900 dark:text-white mb-2">Keine Meetings</h3>
              <p className="text-slate-500 dark:text-slate-400 mb-6">
                {activeFilter === 'all' ? 'Erstelle dein erstes Meeting' : 'Keine Meetings in diesem Filter'}
              </p>
              <button
                onClick={() => setShowCreateModal(true)}
                className="inline-flex items-center gap-2 px-6 py-3 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors"
              >
                <Plus className="w-4 h-4" />
                Neues Meeting erstellen
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Create Meeting Modal - Same as in Nachrichten */}
      <Dialog open={showCreateModal} onOpenChange={setShowCreateModal}>
        <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <CalendarIcon className="w-5 h-5 text-emerald-600" />
              Neues Meeting erstellen
            </DialogTitle>
            <DialogDescription>
              Erstelle ein neues Meeting und lade Teilnehmer ein
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-6 py-4">
            {/* Step 1: Basic Info */}
            <div>
              <h4 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-4 uppercase">
                Schritt 1: Grundinformationen
              </h4>
              <div className="space-y-4">
                <div>
                  <Label>Meeting-Titel *</Label>
                  <Input
                    value={meetingTitle}
                    onChange={(e) => setMeetingTitle(e.target.value)}
                    placeholder="z.B. Sprint Planning, Design Review, ..."
                  />
                </div>
                <div>
                  <Label>Beschreibung</Label>
                  <Textarea
                    value={meetingDescription}
                    onChange={(e) => setMeetingDescription(e.target.value)}
                    placeholder="Beschreibe das Meeting..."
                    rows={3}
                    className="resize-none"
                  />
                </div>
              </div>
            </div>

            {/* Step 2: Assignment */}
            <div className="pt-4 border-t border-slate-200 dark:border-slate-700">
              <h4 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-4 uppercase">
                Schritt 2: Zuordnung
              </h4>
              <div className="space-y-4">
                <div>
                  <Label className="mb-3 block">Typ auswählen:</Label>
                  <div className="space-y-2">
                    <label className="flex items-center gap-3 p-3 bg-slate-50 dark:bg-slate-800 rounded-lg cursor-pointer hover:bg-slate-100 dark:hover:bg-slate-700">
                      <input
                        type="radio"
                        name="meetingType"
                        checked={meetingType === 'project'}
                        onChange={() => setMeetingType('project')}
                        className="text-emerald-600"
                      />
                      <span className="text-sm">Projekt-Meeting</span>
                    </label>
                    <label className="flex items-center gap-3 p-3 bg-slate-50 dark:bg-slate-800 rounded-lg cursor-pointer hover:bg-slate-100 dark:hover:bg-slate-700">
                      <input
                        type="radio"
                        name="meetingType"
                        checked={meetingType === 'team'}
                        onChange={() => setMeetingType('team')}
                        className="text-emerald-600"
                      />
                      <span className="text-sm">Team-Meeting</span>
                    </label>
                    <label className="flex items-center gap-3 p-3 bg-slate-50 dark:bg-slate-800 rounded-lg cursor-pointer hover:bg-slate-100 dark:hover:bg-slate-700">
                      <input
                        type="radio"
                        name="meetingType"
                        checked={meetingType === 'individual'}
                        onChange={() => setMeetingType('individual')}
                        className="text-emerald-600"
                      />
                      <span className="text-sm">Individuell (eigener Grund)</span>
                    </label>
                  </div>
                </div>

                {meetingType === 'project' && (
                  <div>
                    <Label>Projekt *</Label>
                    <Select value={selectedProject} onValueChange={setSelectedProject}>
                      <SelectTrigger>
                        <SelectValue placeholder="Projekt auswählen..." />
                      </SelectTrigger>
                      <SelectContent>
                        {projects.map((project) => (
                          <SelectItem key={project.id} value={project.name}>
                            {project.name} ({project.members} Mitglieder)
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                )}

                {meetingType === 'team' && (
                  <div>
                    <Label>Team *</Label>
                    <Select value={selectedTeam} onValueChange={setSelectedTeam}>
                      <SelectTrigger>
                        <SelectValue placeholder="Team auswählen..." />
                      </SelectTrigger>
                      <SelectContent>
                        {teams.map((team) => (
                          <SelectItem key={team.id} value={team.name}>
                            {team.name} ({team.members} Mitglieder)
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                )}
              </div>
            </div>

            {/* Step 3: Date & Time */}
            <div className="pt-4 border-t border-slate-200 dark:border-slate-700">
              <h4 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-4 uppercase">
                Schritt 3: Datum & Zeit
              </h4>
              <div className="space-y-4">
                <div>
                  <Label>Datum *</Label>
                  <Input
                    type="date"
                    value={meetingDate}
                    onChange={(e) => setMeetingDate(e.target.value)}
                  />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <Label>Von</Label>
                    <Select value={meetingTimeFrom} onValueChange={setMeetingTimeFrom}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="09:00">09:00</SelectItem>
                        <SelectItem value="10:00">10:00</SelectItem>
                        <SelectItem value="11:00">11:00</SelectItem>
                        <SelectItem value="14:00">14:00</SelectItem>
                        <SelectItem value="15:00">15:00</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div>
                    <Label>Bis</Label>
                    <Select value={meetingTimeTo} onValueChange={setMeetingTimeTo}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="10:00">10:00</SelectItem>
                        <SelectItem value="11:00">11:00</SelectItem>
                        <SelectItem value="12:00">12:00</SelectItem>
                        <SelectItem value="15:30">15:30</SelectItem>
                        <SelectItem value="16:00">16:00</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>
              </div>
            </div>

            {/* Step 4: Participants */}
            <div className="pt-4 border-t border-slate-200 dark:border-slate-700">
              <h4 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-4 uppercase">
                Schritt 4: Teilnehmer
              </h4>
              <div className="space-y-3 max-h-64 overflow-y-auto">
                {teamMembers.map((member) => (
                  <label
                    key={member.id}
                    className="flex items-center gap-3 p-3 bg-slate-50 dark:bg-slate-800 rounded-lg cursor-pointer hover:bg-slate-100 dark:hover:bg-slate-700"
                  >
                    <input
                      type="checkbox"
                      checked={selectedParticipants.includes(member.id)}
                      onChange={() => toggleParticipant(member.id)}
                      disabled={member.organizer}
                      className="rounded text-emerald-600"
                    />
                    <Avatar className="w-8 h-8">
                      <AvatarFallback className="bg-emerald-100 dark:bg-emerald-900 text-emerald-700 dark:text-emerald-300 text-xs">
                        {member.avatar}
                      </AvatarFallback>
                    </Avatar>
                    <div className="flex-1">
                      <p className="text-sm font-medium text-slate-900 dark:text-white">
                        {member.name} {member.organizer && '(Organisator)'}
                      </p>
                      <p className="text-xs text-slate-500 dark:text-slate-400">{member.role}</p>
                    </div>
                  </label>
                ))}
              </div>
              <button className="w-full mt-3 px-4 py-2 bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-300 rounded-lg transition-colors text-sm flex items-center justify-center gap-2">
                <Plus className="w-4 h-4" />
                Externe Person hinzufügen
              </button>
            </div>

            {/* Options */}
            <div className="pt-4 border-t border-slate-200 dark:border-slate-700">
              <h4 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-4 uppercase">
                Optionen
              </h4>
              <div className="space-y-3">
                <label className="flex items-center gap-3 cursor-pointer">
                  <input type="checkbox" defaultChecked className="rounded text-emerald-600" />
                  <span className="text-sm text-slate-700 dark:text-slate-300">
                    Kalender-Eintrag für alle Teilnehmer
                  </span>
                </label>
                <label className="flex items-center gap-3 cursor-pointer">
                  <input type="checkbox" defaultChecked className="rounded text-emerald-600" />
                  <span className="text-sm text-slate-700 dark:text-slate-300">
                    E-Mail-Benachrichtigung senden
                  </span>
                </label>
                <label className="flex items-center gap-3 cursor-pointer">
                  <input type="checkbox" defaultChecked className="rounded text-emerald-600" />
                  <span className="text-sm text-slate-700 dark:text-slate-300">15 Min. vorher erinnern</span>
                </label>
                <label className="flex items-center gap-3 cursor-pointer">
                  <input type="checkbox" className="rounded text-emerald-600" />
                  <span className="text-sm text-slate-700 dark:text-slate-300">
                    Meeting-Raum mit Passwort schützen
                  </span>
                </label>
              </div>
            </div>
          </div>

          <div className="flex gap-3 pt-4 border-t border-slate-200 dark:border-slate-700">
            <button
              onClick={() => setShowCreateModal(false)}
              className="flex-1 px-4 py-2 bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-300 rounded-lg transition-colors"
            >
              Abbrechen
            </button>
            <button
              onClick={handleCreateMeeting}
              className="flex-1 px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg transition-colors"
            >
              Meeting erstellen
            </button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Toast Notification */}
      {toast.show && (
        <div className="fixed top-4 right-4 z-50 animate-in slide-in-from-right duration-300">
          <div
            className={`px-4 py-3 rounded-xl shadow-lg border ${
              toast.type === 'success'
                ? 'bg-green-50 dark:bg-green-900/30 border-green-200 dark:border-green-800 text-green-700 dark:text-green-300'
                : toast.type === 'error'
                ? 'bg-red-50 dark:bg-red-900/30 border-red-200 dark:border-red-800 text-red-700 dark:text-red-300'
                : 'bg-blue-50 dark:bg-blue-900/30 border-blue-200 dark:border-blue-800 text-blue-700 dark:text-blue-300'
            }`}
          >
            <p className="text-sm font-medium">{toast.message}</p>
          </div>
        </div>
      )}

      {/* Meeting Detail View */}
      {selectedMeeting && (
        <MeetingDetailView
          meeting={selectedMeeting}
          onClose={() => setSelectedMeeting(null)}
        />
      )}
    </div>
  );
}