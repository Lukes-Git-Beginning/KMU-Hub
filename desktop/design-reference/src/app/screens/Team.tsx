import { Plus, Search, MoreVertical, Mail, Phone, Calendar, Clock, ChevronDown, ChevronUp, X, MessageCircle, Check, XCircle, Bell } from 'lucide-react';
import { Avatar, AvatarFallback, AvatarImage } from '../components/ui/avatar';
import { Badge } from '../components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../components/ui/tabs';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '../components/ui/dialog';
import { Label } from '../components/ui/label';
import { Textarea } from '../components/ui/textarea';
import { RoomsTab } from '../components/RoomsTab';
import { useState } from 'react';

const teamMembers = [
  {
    id: 1,
    name: 'Anna Müller',
    role: 'Project Manager',
    department: 'Entwicklung',
    email: 'anna.mueller@swissbiz.ch',
    phone: '+41 79 123 45 67',
    status: 'online',
    avatar: 'AM',
    projects: 12,
    tasks: 24,
    joinDate: '2020-03-15',
    currentTask: 'Projekt Dokumentation',
    currentTaskDuration: 7320,
    todayTasks: [
      { name: 'E-Mail Bearbeitung', duration: 3600 },
      { name: 'Team Meeting', duration: 1800 },
      { name: 'Projekt Dokumentation', duration: 7320 },
    ],
  },
  {
    id: 2,
    name: 'Michael Berg',
    role: 'Senior Developer',
    department: 'Entwicklung',
    email: 'michael.berg@swissbiz.ch',
    phone: '+41 79 234 56 78',
    status: 'online',
    avatar: 'MB',
    projects: 8,
    tasks: 32,
    joinDate: '2019-07-22',
    currentTask: 'Code Review',
    currentTaskDuration: 2580,
    todayTasks: [
      { name: 'Bugfixing', duration: 5400 },
      { name: 'Code Review', duration: 2580 },
    ],
  },
  {
    id: 3,
    name: 'Sarah Klein',
    role: 'UI/UX Designer',
    department: 'Design',
    email: 'sarah.klein@swissbiz.ch',
    phone: '+41 79 345 67 89',
    status: 'away',
    avatar: 'SK',
    projects: 15,
    tasks: 18,
    joinDate: '2021-01-10',
    currentTask: null,
    currentTaskDuration: 0,
    todayTasks: [
      { name: 'Design Review', duration: 3600 },
      { name: 'Wireframes', duration: 5400 },
    ],
  },
  {
    id: 4,
    name: 'Tom Weber',
    role: 'Backend Developer',
    department: 'Entwicklung',
    email: 'tom.weber@swissbiz.ch',
    phone: '+41 79 456 78 90',
    status: 'offline',
    avatar: 'TW',
    projects: 6,
    tasks: 28,
    joinDate: '2022-05-18',
    currentTask: null,
    currentTaskDuration: 0,
    todayTasks: [
      { name: 'API Development', duration: 7200 },
      { name: 'Database Optimization', duration: 3600 },
    ],
  },
  {
    id: 5,
    name: 'Lisa Fischer',
    role: 'Marketing Manager',
    department: 'Marketing',
    email: 'lisa.fischer@swissbiz.ch',
    phone: '+41 79 567 89 01',
    status: 'online',
    avatar: 'LF',
    projects: 10,
    tasks: 14,
    joinDate: '2020-11-03',
    currentTask: 'Kundengespräch',
    currentTaskDuration: 1920,
    todayTasks: [
      { name: 'Social Media', duration: 2700 },
      { name: 'Kundengespräch', duration: 1920 },
    ],
  },
  {
    id: 6,
    name: 'David Zimmermann',
    role: 'DevOps Engineer',
    department: 'Entwicklung',
    email: 'david.zimmermann@swissbiz.ch',
    phone: '+41 79 678 90 12',
    status: 'away',
    avatar: 'DZ',
    projects: 4,
    tasks: 16,
    joinDate: '2021-08-25',
    currentTask: null,
    currentTaskDuration: 0,
    todayTasks: [
      { name: 'Server Maintenance', duration: 4500 },
      { name: 'Deployment', duration: 1800 },
    ],
  },
];

const departments = [
  { name: 'Entwicklung', count: 12, color: 'bg-blue-500' },
  { name: 'Design', count: 8, color: 'bg-purple-500' },
  { name: 'Marketing', count: 6, color: 'bg-pink-500' },
  { name: 'Verwaltung', count: 4, color: 'bg-emerald-500' },
];

// Mock Requests Data
const mockRequests = [
  {
    id: 1,
    type: 'vacation',
    icon: '🌴',
    title: 'URLAUBSANTRAG',
    memberId: 1,
    memberName: 'Anna Müller',
    memberAvatar: 'AM',
    memberRole: 'Project Manager',
    memberDepartment: 'Entwicklung',
    memberStatus: 'online',
    status: 'pending',
    dateFrom: '08.01.2025',
    dateTo: '12.01.2025',
    duration: '5 Tage',
    reason: 'Familienurlaub in den Bergen',
    vacationDaysAvailable: 12,
    vacationDaysRemaining: 7,
    submittedAt: 'vor 2 Stunden',
  },
  {
    id: 2,
    type: 'sick',
    icon: '🏥',
    title: 'KRANKMELDUNG',
    memberId: 2,
    memberName: 'Michael Berg',
    memberAvatar: 'MB',
    memberRole: 'Senior Developer',
    memberDepartment: 'Entwicklung',
    memberStatus: 'online',
    status: 'pending',
    dateFrom: '06.01.2025',
    dateTo: 'Offen',
    duration: null,
    reason: null,
    description: 'Grippale Infektion',
    hasAttest: true,
    attestFile: 'attest_berg.pdf',
    attestSize: '120KB',
    submittedAt: 'vor 5 Stunden',
  },
  {
    id: 3,
    type: 'doctor',
    icon: '🏥',
    title: 'ARZTTERMIN',
    memberId: 3,
    memberName: 'Sarah Klein',
    memberAvatar: 'SK',
    memberRole: 'UI/UX Designer',
    memberDepartment: 'Design',
    memberStatus: 'away',
    status: 'pending',
    date: '09.01.2025',
    timeFrom: '09:00',
    timeTo: '11:00',
    duration: '2.0h',
    countAsWorkTime: true,
    note: 'Routine-Untersuchung',
    submittedAt: 'vor 1 Tag',
  },
  {
    id: 4,
    type: 'overtime',
    icon: '⏰',
    title: 'ÜBERSTUNDEN ABBAUEN',
    memberId: 4,
    memberName: 'Tom Weber',
    memberAvatar: 'TW',
    memberRole: 'Backend Developer',
    memberDepartment: 'Entwicklung',
    memberStatus: 'offline',
    status: 'approved',
    date: '10.01.2025',
    duration: '8.0h',
    overtimeAvailable: 24.5,
    overtimeRemaining: 16.5,
    reason: 'Ausgleich für Wochenendarbeit',
    submittedAt: 'vor 3 Tagen',
    approvedBy: 'Anna Müller',
    approvedAt: 'vor 2 Tagen',
  },
  {
    id: 5,
    type: 'vacation',
    icon: '🌴',
    title: 'URLAUBSANTRAG',
    memberId: 5,
    memberName: 'Lisa Fischer',
    memberAvatar: 'LF',
    memberRole: 'Marketing Manager',
    memberDepartment: 'Marketing',
    memberStatus: 'online',
    status: 'rejected',
    dateFrom: '15.01.2025',
    dateTo: '19.01.2025',
    duration: '5 Tage',
    reason: 'Kurzurlaub',
    submittedAt: 'vor 1 Woche',
    rejectedBy: 'Anna Müller',
    rejectedAt: 'vor 5 Tagen',
    rejectionReason: 'Zu viele Projekte in dieser Woche aktiv',
  },
];

type RequestType = 'all' | 'vacation' | 'sick' | 'doctor' | 'overtime';

export function Team() {
  const [expandedMembers, setExpandedMembers] = useState<number[]>([]);
  const [activeTab, setActiveTab] = useState<'members' | 'requests' | 'rooms'>('members');
  const [activeFilter, setActiveFilter] = useState<RequestType>('all');
  const [selectedRequest, setSelectedRequest] = useState<any>(null);
  const [feedbackModal, setFeedbackModal] = useState<{ type: 'feedback' | 'reject' | null; request: any }>({ type: null, request: null });
  const [feedbackMessage, setFeedbackMessage] = useState('');
  const [rejectReason, setRejectReason] = useState('');
  const [toast, setToast] = useState<{ show: boolean; message: string; type: 'success' | 'error' | 'info' }>({ show: false, message: '', type: 'success' });

  const toggleExpanded = (memberId: number) => {
    setExpandedMembers(prev =>
      prev.includes(memberId)
        ? prev.filter(id => id !== memberId)
        : [...prev, memberId]
    );
  };

  const formatTime = (seconds: number) => {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    return `${hours}h ${minutes}m`;
  };

  const getTotalDayTime = (tasks: { name: string; duration: number }[]) => {
    return tasks.reduce((sum, task) => sum + task.duration, 0);
  };

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

  const getRequestStatusBadge = (status: string) => {
    switch (status) {
      case 'pending':
        return { text: '⏳ Ausstehend', bg: 'bg-yellow-100 dark:bg-yellow-900/30', color: 'text-yellow-700 dark:text-yellow-300' };
      case 'approved':
        return { text: '✅ Genehmigt', bg: 'bg-green-100 dark:bg-green-900/30', color: 'text-green-700 dark:text-green-300' };
      case 'rejected':
        return { text: '❌ Abgelehnt', bg: 'bg-red-100 dark:bg-red-900/30', color: 'text-red-700 dark:text-red-300' };
      case 'feedback':
        return { text: '💬 Rückfrage', bg: 'bg-orange-100 dark:bg-orange-900/30', color: 'text-orange-700 dark:text-orange-300' };
      default:
        return { text: status, bg: 'bg-slate-100 dark:bg-slate-800', color: 'text-slate-700 dark:text-slate-300' };
    }
  };

  const filteredRequests = mockRequests.filter(req => {
    if (activeFilter === 'all') return true;
    return req.type === activeFilter;
  });

  const pendingCount = mockRequests.filter(r => r.status === 'pending').length;

  const filterCounts = {
    all: mockRequests.length,
    vacation: mockRequests.filter(r => r.type === 'vacation').length,
    sick: mockRequests.filter(r => r.type === 'sick').length,
    doctor: mockRequests.filter(r => r.type === 'doctor').length,
    overtime: mockRequests.filter(r => r.type === 'overtime').length,
  };

  const showToast = (message: string, type: 'success' | 'error' | 'info' = 'success') => {
    setToast({ show: true, message, type });
    setTimeout(() => setToast({ show: false, message: '', type: 'success' }), 4000);
  };

  const handleApprove = (request: any) => {
    showToast(`✅ ${request.title.toLowerCase()} genehmigt`, 'success');
  };

  const handleReject = () => {
    if (feedbackModal.request) {
      showToast(`❌ ${feedbackModal.request.title.toLowerCase()} abgelehnt`, 'error');
      setFeedbackModal({ type: null, request: null });
      setRejectReason('');
    }
  };

  const handleFeedback = () => {
    if (feedbackModal.request && feedbackMessage.length >= 10) {
      showToast(`💬 Rückfrage an ${feedbackModal.request.memberName} gesendet`, 'info');
      setFeedbackModal({ type: null, request: null });
      setFeedbackMessage('');
    }
  };

  return (
    <div className="flex-1 overflow-y-auto p-4 md:p-8 bg-slate-50 dark:bg-slate-900">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between mb-6 md:mb-8 gap-4">
        <div>
          <h1 className="text-slate-900 dark:text-white mb-1">Team & CRM</h1>
          <p className="text-sm text-slate-500 dark:text-slate-400">Verwalte Mitarbeiter und Kontakte</p>
        </div>
        <button className="flex items-center justify-center gap-2 px-4 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors">
          <Plus className="w-4 h-4" />
          <span className="hidden sm:inline">Mitglied hinzufügen</span>
          <span className="sm:hidden">Hinzufügen</span>
        </button>
      </div>

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 md:gap-6 mb-6 md:mb-8">
        {departments.map((dept) => (
          <div
            key={dept.name}
            className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-4 md:p-6"
          >
            <div className="flex items-center gap-3 mb-3">
              <div className={`w-10 h-10 rounded-lg ${dept.color} flex items-center justify-center`}>
                <span className="text-white text-lg">{dept.count}</span>
              </div>
            </div>
            <h3 className="text-sm md:text-base text-slate-900 dark:text-white mb-1">{dept.name}</h3>
            <p className="text-xs text-slate-500 dark:text-slate-400">{dept.count} Mitglieder</p>
          </div>
        ))}
      </div>

      <div className="mb-6">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400 dark:text-slate-500" />
          <input
            type="text"
            placeholder="Team durchsuchen..."
            className="w-full pl-10 pr-4 py-2 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent text-sm text-slate-900 dark:text-white placeholder:text-slate-400 dark:placeholder:text-slate-500"
          />
        </div>
      </div>

      {/* Main Tabs */}
      <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as any)} className="w-full">
        <div className="flex items-center justify-between mb-6">
          <TabsList>
            <TabsTrigger value="members" className="gap-2">
              Mitglieder
            </TabsTrigger>
            <TabsTrigger value="requests" className="gap-2">
              Anfragen
              {pendingCount > 0 && (
                <span className="ml-1 w-5 h-5 flex items-center justify-center bg-red-500 text-white text-xs rounded-full">
                  {pendingCount}
                </span>
              )}
            </TabsTrigger>
            <TabsTrigger value="rooms" className="gap-2">
              Räume
            </TabsTrigger>
          </TabsList>

          {activeTab === 'members' && (
            <div className="flex gap-2">
              <Tabs defaultValue="grid">
                <TabsList>
                  <TabsTrigger value="grid">Grid</TabsTrigger>
                  <TabsTrigger value="list">Liste</TabsTrigger>
                </TabsList>
              </Tabs>
            </div>
          )}
        </div>

        {/* Members Tab */}
        <TabsContent value="members">
          <Tabs defaultValue="grid" className="w-full">
            <TabsContent value="grid">
              <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4 md:gap-6">
                {teamMembers.map((member) => (
                  <div
                    key={member.id}
                    className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-4 md:p-6 hover:shadow-lg dark:hover:shadow-emerald-900/10 transition-shadow"
                  >
                    <div className="flex items-start justify-between mb-4">
                      <div className="flex items-center gap-3">
                        <div className="relative">
                          <Avatar className="w-12 h-12 md:w-14 md:h-14">
                            <AvatarFallback className="bg-emerald-100 dark:bg-emerald-900 text-emerald-700 dark:text-emerald-300 text-sm md:text-base">
                              {member.avatar}
                            </AvatarFallback>
                          </Avatar>
                          <span
                            className={`absolute bottom-0 right-0 w-3 h-3 md:w-4 md:h-4 rounded-full border-2 border-white dark:border-slate-800 ${getStatusColor(
                              member.status
                            )}`}
                          />
                        </div>
                        <div className="min-w-0">
                          <h3 className="text-sm md:text-base text-slate-900 dark:text-white truncate">{member.name}</h3>
                          <p className="text-xs md:text-sm text-slate-500 dark:text-slate-400 truncate">{member.role}</p>
                        </div>
                      </div>
                      <button className="p-1 hover:bg-slate-100 dark:hover:bg-slate-700 rounded flex-shrink-0">
                        <MoreVertical className="w-4 h-4 text-slate-400 dark:text-slate-500" />
                      </button>
                    </div>

                    <div className="space-y-2 mb-4">
                      <div className="flex items-center gap-2 text-xs md:text-sm text-slate-600 dark:text-slate-300">
                        <Mail className="w-4 h-4 text-slate-400 dark:text-slate-500 flex-shrink-0" />
                        <span className="truncate">{member.email}</span>
                      </div>
                      <div className="flex items-center gap-2 text-xs md:text-sm text-slate-600 dark:text-slate-300">
                        <Phone className="w-4 h-4 text-slate-400 dark:text-slate-500 flex-shrink-0" />
                        <span className="truncate">{member.phone}</span>
                      </div>
                      <div className="flex items-center gap-2 text-xs md:text-sm text-slate-600 dark:text-slate-300">
                        <Calendar className="w-4 h-4 text-slate-400 dark:text-slate-500 flex-shrink-0" />
                        <span className="truncate">Seit {new Date(member.joinDate).toLocaleDateString('de-DE')}</span>
                      </div>
                    </div>

                    <div className="grid grid-cols-2 gap-3 pt-4 border-t border-slate-200 dark:border-slate-700">
                      <div>
                        <p className="text-xs text-slate-500 dark:text-slate-400 mb-1">Projekte</p>
                        <p className="text-lg md:text-xl text-slate-900 dark:text-white">{member.projects}</p>
                      </div>
                      <div>
                        <p className="text-xs text-slate-500 dark:text-slate-400 mb-1">Aufgaben</p>
                        <p className="text-lg md:text-xl text-slate-900 dark:text-white">{member.tasks}</p>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </TabsContent>

            <TabsContent value="list">
              <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg divide-y divide-slate-200 dark:divide-slate-700 overflow-hidden">
                {teamMembers.map((member) => {
                  const isExpanded = expandedMembers.includes(member.id);
                  const totalDayTime = getTotalDayTime(member.todayTasks);
                  
                  return (
                    <div key={member.id} className="transition-colors">
                      <div className="flex flex-col sm:flex-row sm:items-center gap-3 sm:gap-4 p-4 hover:bg-slate-50 dark:hover:bg-slate-700">
                        <div className="flex items-center gap-3 flex-1 min-w-0">
                          <div className="relative flex-shrink-0">
                            <Avatar className="w-10 h-10 md:w-12 md:h-12">
                              <AvatarFallback className="bg-emerald-100 dark:bg-emerald-900 text-emerald-700 dark:text-emerald-300 text-sm">
                                {member.avatar}
                              </AvatarFallback>
                            </Avatar>
                            <span
                              className={`absolute bottom-0 right-0 w-3 h-3 rounded-full border-2 border-white dark:border-slate-800 ${getStatusColor(
                                member.status
                              )}`}
                            />
                          </div>
                          <div className="flex-1 min-w-0">
                            <h3 className="text-sm md:text-base text-slate-900 dark:text-white truncate">{member.name}</h3>
                            <p className="text-xs md:text-sm text-slate-500 dark:text-slate-400 truncate">{member.role}</p>
                          </div>
                        </div>

                        <div className="flex items-center justify-between sm:justify-start gap-4 sm:gap-6 text-xs md:text-sm text-slate-600 dark:text-slate-300">
                          <div className="hidden lg:flex items-center gap-2 min-w-0">
                            <Mail className="w-4 h-4 text-slate-400 dark:text-slate-500 flex-shrink-0" />
                            <span className="truncate">{member.email}</span>
                          </div>
                          <div className="flex items-center gap-1">
                            <span className="text-slate-500 dark:text-slate-400">Projekte:</span>
                            <span className="text-slate-900 dark:text-white">{member.projects}</span>
                          </div>
                          <div className="flex items-center gap-1">
                            <span className="text-slate-500 dark:text-slate-400">Aufgaben:</span>
                            <span className="text-slate-900 dark:text-white">{member.tasks}</span>
                          </div>
                          <button className="p-1 hover:bg-slate-200 dark:hover:bg-slate-600 rounded flex-shrink-0">
                            <MoreVertical className="w-4 h-4 text-slate-400 dark:text-slate-500" />
                          </button>
                        </div>
                      </div>

                      <div className="px-4 pb-3 bg-slate-50 dark:bg-slate-800/50 border-t border-slate-200 dark:border-slate-700">
                        <div className="flex items-center justify-between py-3">
                          <div className="flex items-center gap-4 flex-1 min-w-0">
                            <Clock className="w-4 h-4 text-emerald-600 dark:text-emerald-400 flex-shrink-0" />
                            <div className="flex-1 min-w-0">
                              {member.currentTask ? (
                                <div className="flex items-center gap-2">
                                  <span className="text-sm text-slate-900 dark:text-white truncate">
                                    {member.currentTask}
                                  </span>
                                  <Badge className="bg-emerald-100 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-300 text-xs">
                                    {formatTime(member.currentTaskDuration)}
                                  </Badge>
                                  <span className="text-xs text-emerald-600 dark:text-emerald-400 animate-pulse">● Aktiv</span>
                                </div>
                              ) : (
                                <span className="text-sm text-slate-500 dark:text-slate-400">Keine aktive Aufgabe</span>
                              )}
                              <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                                Heute gesamt: <span className="font-medium text-slate-900 dark:text-white">{formatTime(totalDayTime)}</span>
                              </p>
                            </div>
                          </div>
                          
                          <button
                            onClick={() => toggleExpanded(member.id)}
                            className="p-1.5 hover:bg-slate-200 dark:hover:bg-slate-700 rounded transition-colors flex-shrink-0"
                          >
                            {isExpanded ? (
                              <ChevronUp className="w-4 h-4 text-slate-500 dark:text-slate-400" />
                            ) : (
                              <ChevronDown className="w-4 h-4 text-slate-500 dark:text-slate-400" />
                            )}
                          </button>
                        </div>

                        {isExpanded && member.todayTasks.length > 0 && (
                          <div className="space-y-2 pb-3">
                            {member.todayTasks.map((task, index) => (
                              <div key={index} className="flex items-center justify-between text-xs">
                                <span className="text-slate-700 dark:text-slate-300">{task.name}</span>
                                <span className="text-slate-500 dark:text-slate-400">{formatTime(task.duration)}</span>
                              </div>
                            ))}
                          </div>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            </TabsContent>
          </Tabs>
        </TabsContent>

        {/* Requests Tab */}
        <TabsContent value="requests" className="space-y-6">
          <div className="flex items-center justify-between">
            <h2 className="text-xl font-semibold text-slate-900 dark:text-white">Anfragen</h2>
            <div className="flex items-center gap-2">
              <Bell className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
              <span className="text-sm font-medium text-slate-900 dark:text-white">{pendingCount} Offen</span>
            </div>
          </div>

          {/* Filter Chips */}
          <div className="flex gap-2 overflow-x-auto pb-2">
            <button
              onClick={() => setActiveFilter('all')}
              className={`px-4 py-2 rounded-full text-sm font-medium whitespace-nowrap transition-colors ${
                activeFilter === 'all'
                  ? 'bg-emerald-600 text-white'
                  : 'bg-slate-100 dark:bg-slate-700 text-slate-700 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-600'
              }`}
            >
              Alle {filterCounts.all}
            </button>
            <button
              onClick={() => setActiveFilter('vacation')}
              className={`px-4 py-2 rounded-full text-sm font-medium whitespace-nowrap transition-colors ${
                activeFilter === 'vacation'
                  ? 'bg-emerald-600 text-white'
                  : 'bg-slate-100 dark:bg-slate-700 text-slate-700 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-600'
              }`}
            >
              🌴 Urlaub {filterCounts.vacation}
            </button>
            <button
              onClick={() => setActiveFilter('sick')}
              className={`px-4 py-2 rounded-full text-sm font-medium whitespace-nowrap transition-colors ${
                activeFilter === 'sick'
                  ? 'bg-emerald-600 text-white'
                  : 'bg-slate-100 dark:bg-slate-700 text-slate-700 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-600'
              }`}
            >
              🏥 Krank {filterCounts.sick}
            </button>
            <button
              onClick={() => setActiveFilter('doctor')}
              className={`px-4 py-2 rounded-full text-sm font-medium whitespace-nowrap transition-colors ${
                activeFilter === 'doctor'
                  ? 'bg-emerald-600 text-white'
                  : 'bg-slate-100 dark:bg-slate-700 text-slate-700 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-600'
              }`}
            >
              🏥 Arzt {filterCounts.doctor}
            </button>
            <button
              onClick={() => setActiveFilter('overtime')}
              className={`px-4 py-2 rounded-full text-sm font-medium whitespace-nowrap transition-colors ${
                activeFilter === 'overtime'
                  ? 'bg-emerald-600 text-white'
                  : 'bg-slate-100 dark:bg-slate-700 text-slate-700 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-600'
              }`}
            >
              ⏰ Überstunden {filterCounts.overtime}
            </button>
          </div>

          {/* Requests List */}
          <div className="space-y-4">
            {filteredRequests.length === 0 ? (
              <div className="text-center py-20">
                <div className="text-6xl mb-4">📭</div>
                <h3 className="text-lg font-medium text-slate-900 dark:text-white mb-2">Keine offenen Anfragen</h3>
                <p className="text-slate-500 dark:text-slate-400">Alle Anfragen wurden bearbeitet! 🎉</p>
              </div>
            ) : (
              filteredRequests.map((request) => {
                const statusBadge = getRequestStatusBadge(request.status);
                
                return (
                  <div
                    key={request.id}
                    className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl p-6 hover:shadow-lg transition-shadow"
                  >
                    {/* Header */}
                    <div className="flex items-start justify-between mb-4 pb-4 border-b border-slate-200 dark:border-slate-700">
                      <div className="flex items-center gap-2">
                        <span className="text-xl">{request.icon}</span>
                        <h3 className="font-semibold text-slate-900 dark:text-white">{request.title}</h3>
                      </div>
                      <Badge className={`${statusBadge.bg} ${statusBadge.color}`}>
                        {statusBadge.text}
                      </Badge>
                    </div>

                    {/* Member Info */}
                    <div className="flex items-center gap-3 mb-4">
                      <div className="relative">
                        <Avatar className="w-10 h-10">
                          <AvatarFallback className="bg-emerald-100 dark:bg-emerald-900 text-emerald-700 dark:text-emerald-300">
                            {request.memberAvatar}
                          </AvatarFallback>
                        </Avatar>
                        <span className={`absolute bottom-0 right-0 w-3 h-3 rounded-full border-2 border-white dark:border-slate-800 ${getStatusColor(request.memberStatus)}`} />
                      </div>
                      <div>
                        <p className="font-medium text-slate-900 dark:text-white">{request.memberName}</p>
                        <p className="text-sm text-slate-500 dark:text-slate-400">
                          {request.memberRole} • {request.memberDepartment}
                        </p>
                      </div>
                    </div>

                    {/* Request Details */}
                    <div className="space-y-2 mb-4">
                      {request.type === 'vacation' && (
                        <>
                          <div className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
                            <Calendar className="w-4 h-4 text-slate-400" />
                            <span>{request.dateFrom} - {request.dateTo} ({request.duration})</span>
                          </div>
                          {request.reason && (
                            <div className="flex items-start gap-2 text-sm text-slate-600 dark:text-slate-400 italic">
                              <MessageCircle className="w-4 h-4 text-slate-400 flex-shrink-0 mt-0.5" />
                              <span>Grund: "{request.reason}"</span>
                            </div>
                          )}
                          {request.vacationDaysAvailable !== undefined && (
                            <div className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
                              <span>📊 Urlaubstage: {request.vacationDaysAvailable} verfügbar → {request.vacationDaysRemaining} verbleibend</span>
                            </div>
                          )}
                        </>
                      )}

                      {request.type === 'sick' && (
                        <>
                          <div className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
                            <Calendar className="w-4 h-4 text-slate-400" />
                            <span>{request.dateFrom} - {request.dateTo}</span>
                          </div>
                          {request.hasAttest && (
                            <div className="flex items-center gap-2 text-sm text-green-600 dark:text-green-400">
                              <Check className="w-4 h-4" />
                              <span>Attest: ✅ {request.attestFile} ({request.attestSize})</span>
                            </div>
                          )}
                          {request.description && (
                            <div className="flex items-start gap-2 text-sm text-slate-600 dark:text-slate-400 italic">
                              <MessageCircle className="w-4 h-4 text-slate-400 flex-shrink-0 mt-0.5" />
                              <span>Beschreibung: "{request.description}"</span>
                            </div>
                          )}
                        </>
                      )}

                      {request.type === 'doctor' && (
                        <>
                          <div className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
                            <Calendar className="w-4 h-4 text-slate-400" />
                            <span>{request.date}, {request.timeFrom} - {request.timeTo} ({request.duration})</span>
                          </div>
                          {request.countAsWorkTime && (
                            <div className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
                              <Check className="w-4 h-4 text-emerald-600" />
                              <span>Als Arbeitszeit anrechnen</span>
                            </div>
                          )}
                          {request.note && (
                            <div className="flex items-start gap-2 text-sm text-slate-600 dark:text-slate-400 italic">
                              <MessageCircle className="w-4 h-4 text-slate-400 flex-shrink-0 mt-0.5" />
                              <span>Notiz: "{request.note}"</span>
                            </div>
                          )}
                        </>
                      )}

                      {request.type === 'overtime' && (
                        <>
                          <div className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
                            <Calendar className="w-4 h-4 text-slate-400" />
                            <span>{request.date} ({request.duration})</span>
                          </div>
                          {request.overtimeAvailable !== undefined && (
                            <div className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
                              <span>📊 Verfügbare Überstunden: {request.overtimeAvailable}h → {request.overtimeRemaining}h</span>
                            </div>
                          )}
                          {request.reason && (
                            <div className="flex items-start gap-2 text-sm text-slate-600 dark:text-slate-400 italic">
                              <MessageCircle className="w-4 h-4 text-slate-400 flex-shrink-0 mt-0.5" />
                              <span>Begründung: "{request.reason}"</span>
                            </div>
                          )}
                        </>
                      )}

                      <div className="flex items-center gap-2 text-xs text-slate-500 dark:text-slate-400 pt-2">
                        <Clock className="w-3 h-3" />
                        <span>Eingereicht: {request.submittedAt}</span>
                      </div>

                      {request.status === 'approved' && request.approvedBy && (
                        <div className="flex items-center gap-2 text-sm text-green-600 dark:text-green-400 pt-2">
                          <Check className="w-4 h-4" />
                          <span>Genehmigt von {request.approvedBy} • {request.approvedAt}</span>
                        </div>
                      )}

                      {request.status === 'rejected' && request.rejectedBy && (
                        <div className="space-y-2 pt-2">
                          <div className="flex items-center gap-2 text-sm text-red-600 dark:text-red-400">
                            <XCircle className="w-4 h-4" />
                            <span>Abgelehnt von {request.rejectedBy} • {request.rejectedAt}</span>
                          </div>
                          {request.rejectionReason && (
                            <div className="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg text-sm text-red-700 dark:text-red-300">
                              💬 "{request.rejectionReason}"
                            </div>
                          )}
                        </div>
                      )}
                    </div>

                    {/* Action Buttons */}
                    {request.status === 'pending' && (
                      <div className="grid grid-cols-3 gap-3 pt-4 border-t border-slate-200 dark:border-slate-700">
                        <button
                          onClick={() => handleApprove(request)}
                          className="flex items-center justify-center gap-2 px-4 py-3 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors"
                        >
                          <Check className="w-4 h-4" />
                          <span className="text-sm font-medium">{request.type === 'sick' ? 'Bestätigen' : 'Genehmigen'}</span>
                        </button>
                        <button
                          onClick={() => setFeedbackModal({ type: 'reject', request })}
                          className="flex items-center justify-center gap-2 px-4 py-3 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors"
                        >
                          <XCircle className="w-4 h-4" />
                          <span className="text-sm font-medium">Ablehnen</span>
                        </button>
                        <button
                          onClick={() => setFeedbackModal({ type: 'feedback', request })}
                          className="flex items-center justify-center gap-2 px-4 py-3 border-2 border-orange-500 text-orange-600 dark:text-orange-400 rounded-lg hover:bg-orange-50 dark:hover:bg-orange-900/20 transition-colors"
                        >
                          <MessageCircle className="w-4 h-4" />
                          <span className="text-sm font-medium">Rückfrage</span>
                        </button>
                      </div>
                    )}

                    {request.status !== 'pending' && (
                      <div className="grid grid-cols-3 gap-3 pt-4 border-t border-slate-200 dark:border-slate-700 opacity-50">
                        <button
                          disabled
                          className="flex items-center justify-center gap-2 px-4 py-3 bg-slate-300 dark:bg-slate-600 text-slate-500 dark:text-slate-400 rounded-lg cursor-not-allowed"
                        >
                          <Check className="w-4 h-4" />
                          <span className="text-sm font-medium">Genehmigen</span>
                        </button>
                        <button
                          disabled
                          className="flex items-center justify-center gap-2 px-4 py-3 bg-slate-300 dark:bg-slate-600 text-slate-500 dark:text-slate-400 rounded-lg cursor-not-allowed"
                        >
                          <XCircle className="w-4 h-4" />
                          <span className="text-sm font-medium">Ablehnen</span>
                        </button>
                        <button
                          disabled
                          className="flex items-center justify-center gap-2 px-4 py-3 border-2 border-slate-300 dark:border-slate-600 text-slate-500 dark:text-slate-400 rounded-lg cursor-not-allowed"
                        >
                          <MessageCircle className="w-4 h-4" />
                          <span className="text-sm font-medium">Rückfrage</span>
                        </button>
                      </div>
                    )}
                  </div>
                );
              })
            )}
          </div>
        </TabsContent>

        {/* Rooms Tab (Placeholder) */}
        <TabsContent value="rooms">
          <RoomsTab />
        </TabsContent>
      </Tabs>

      {/* Feedback Modal */}
      <Dialog open={feedbackModal.type === 'feedback'} onOpenChange={() => setFeedbackModal({ type: null, request: null })}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <MessageCircle className="w-5 h-5 text-orange-600" />
              Rückfrage stellen
            </DialogTitle>
            <DialogDescription>
              An: {feedbackModal.request?.memberName}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>Betreff</Label>
              <input
                type="text"
                value={feedbackModal.request?.title || ''}
                readOnly
                className="w-full px-3 py-2 bg-slate-100 dark:bg-slate-700 border border-slate-200 dark:border-slate-600 rounded-lg text-slate-900 dark:text-white"
              />
            </div>

            <div className="space-y-2">
              <Label>Nachricht *</Label>
              <Textarea
                value={feedbackMessage}
                onChange={(e) => setFeedbackMessage(e.target.value)}
                placeholder="Kannst du deinen Urlaub auf die Folgewoche verschieben? Wir haben in dieser Woche wichtige Meetings mit Kunden geplant."
                rows={5}
                className="resize-none"
              />
              <p className="text-xs text-slate-500 dark:text-slate-400">
                {feedbackMessage.length} / 250 Zeichen (min. 10)
              </p>
            </div>

            <div className="space-y-2">
              <label className="flex items-center gap-2">
                <input type="checkbox" defaultChecked className="rounded" />
                <span className="text-sm text-slate-700 dark:text-slate-300">Antrag in Status "Rückfrage" setzen</span>
              </label>
              <label className="flex items-center gap-2">
                <input type="checkbox" defaultChecked className="rounded" />
                <span className="text-sm text-slate-700 dark:text-slate-300">Mitarbeiter per E-Mail benachrichtigen</span>
              </label>
            </div>

            <div className="flex gap-3 pt-4">
              <button
                onClick={() => {
                  setFeedbackModal({ type: null, request: null });
                  setFeedbackMessage('');
                }}
                className="flex-1 px-4 py-2 bg-slate-200 dark:bg-slate-700 text-slate-700 dark:text-slate-300 rounded-lg hover:bg-slate-300 dark:hover:bg-slate-600 transition-colors"
              >
                Abbrechen
              </button>
              <button
                onClick={handleFeedback}
                disabled={feedbackMessage.length < 10}
                className="flex-1 px-4 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Nachricht senden
              </button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* Reject Modal */}
      <Dialog open={feedbackModal.type === 'reject'} onOpenChange={() => setFeedbackModal({ type: null, request: null })}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <XCircle className="w-5 h-5 text-red-600" />
              Anfrage ablehnen
            </DialogTitle>
            <DialogDescription className="sr-only">
              Bestätigung für die Ablehnung der Anfrage
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="p-4 bg-slate-50 dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700">
              <p className="text-sm text-slate-700 dark:text-slate-300 mb-2">
                Möchtest du diese Anfrage wirklich ablehnen?
              </p>
              <div className="space-y-1">
                <p className="text-sm font-medium text-slate-900 dark:text-white">
                  {feedbackModal.request?.icon} {feedbackModal.request?.title} von {feedbackModal.request?.memberName}
                </p>
                <p className="text-sm text-slate-600 dark:text-slate-400">
                  {feedbackModal.request?.dateFrom} - {feedbackModal.request?.dateTo || feedbackModal.request?.date}
                </p>
              </div>
            </div>

            <div className="space-y-2">
              <Label>Begründung (optional)</Label>
              <Textarea
                value={rejectReason}
                onChange={(e) => setRejectReason(e.target.value)}
                placeholder='z.B. "Zu viele Projekte aktiv"'
                rows={3}
                className="resize-none"
              />
            </div>

            <div>
              <label className="flex items-center gap-2">
                <input type="checkbox" defaultChecked className="rounded" />
                <span className="text-sm text-slate-700 dark:text-slate-300">Mitarbeiter per E-Mail benachrichtigen</span>
              </label>
            </div>

            <div className="flex gap-3 pt-4">
              <button
                onClick={() => {
                  setFeedbackModal({ type: null, request: null });
                  setRejectReason('');
                }}
                className="flex-1 px-4 py-2 bg-slate-200 dark:bg-slate-700 text-slate-700 dark:text-slate-300 rounded-lg hover:bg-slate-300 dark:hover:bg-slate-600 transition-colors"
              >
                Zurück
              </button>
              <button
                onClick={handleReject}
                className="flex-1 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors"
              >
                Ablehnen
              </button>
            </div>
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
    </div>
  );
}