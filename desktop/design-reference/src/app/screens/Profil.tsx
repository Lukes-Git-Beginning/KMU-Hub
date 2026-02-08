import { useState } from 'react';
import { 
  User, 
  FileText, 
  Clock, 
  Download, 
  Upload, 
  Briefcase,
  GraduationCap,
  FileCheck,
  TrendingUp,
  Eye,
  Trash2,
  Plus,
  Calendar as CalendarIcon,
  BarChart3,
  CalendarDays,
  TrendingDown,
  ChevronLeft,
  ChevronRight,
  X,
  Paperclip,
  Info
} from 'lucide-react';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../components/ui/tabs';
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/card';
import { Avatar, AvatarFallback, AvatarImage } from '../components/ui/avatar';
import { Badge } from '../components/ui/badge';
import { Label } from '../components/ui/label';
import { Input } from '../components/ui/input';
import { Textarea } from '../components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../components/ui/select';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '../components/ui/dialog';
import { PieChart, Pie, Cell, ResponsiveContainer, LineChart, Line, XAxis, YAxis, Tooltip as RechartsTooltip } from 'recharts';

// Mock data
const projectDistribution = [
  { name: 'Website Redesign', value: 18.5, color: '#4ECCA3', percentage: 45 },
  { name: 'Mobile App', value: 12.5, color: '#5B9FED', percentage: 30 },
  { name: 'Backend API', value: 8.2, color: '#9B7EDE', percentage: 20 },
  { name: 'Meetings', value: 2.0, color: '#FFB347', percentage: 5 },
];

const weeklyOverview = [
  { day: 'Mo', hours: 8.2, color: '#4ECCA3' },
  { day: 'Di', hours: 7.8, color: '#5B9FED' },
  { day: 'Mi', hours: 8.5, color: '#4ECCA3' },
  { day: 'Do', hours: 7.9, color: '#5B9FED' },
  { day: 'Fr', hours: 6.6, color: '#FFB347' },
  { day: 'Sa', hours: 0, color: '#E5E7EB' },
  { day: 'So', hours: 0, color: '#E5E7EB' },
];

const productivityTrend = [
  { week: 'KW 48', avg: 7.8 },
  { week: 'KW 49', avg: 8.1 },
  { week: 'KW 50', avg: 7.9 },
  { week: 'KW 51', avg: 8.2 },
];

const calendarDays = [
  { date: 1, hours: 8.2, status: null },
  { date: 2, hours: 7.5, status: null },
  { date: 3, hours: 8.0, status: null },
  { date: 4, hours: 0, status: 'sick', icon: '🏥', statusText: 'Krank' },
  { date: 5, hours: 0, status: 'vacation', icon: '🌴', statusText: 'Urlaub' },
  { date: 6, hours: 0, status: null },
  { date: 7, hours: 0, status: null },
  { date: 8, hours: 8.1, status: null },
  { date: 9, hours: 7.8, status: null },
  { date: 10, hours: 0, status: 'vacation', icon: '🌴', statusText: 'Urlaub' },
  { date: 11, hours: 0, status: 'vacation', icon: '🌴', statusText: 'Urlaub' },
  { date: 12, hours: 0, status: 'vacation', icon: '🌴', statusText: 'Urlaub' },
  { date: 13, hours: 0, status: null },
  { date: 14, hours: 0, status: null },
];

const documents = {
  payroll: [
    { id: 1, name: 'Lohnabrechnung_Dezember_2024.pdf', date: '2024-12-01', size: '245 KB' },
    { id: 2, name: 'Lohnabrechnung_November_2024.pdf', date: '2024-11-01', size: '243 KB' },
    { id: 3, name: 'Lohnabrechnung_Oktober_2024.pdf', date: '2024-10-01', size: '241 KB' },
  ],
  contracts: [
    { id: 1, name: 'Arbeitsvertrag_2020.pdf', date: '2020-03-15', size: '892 KB' },
    { id: 2, name: 'Nachtrag_Gehaltsanpassung_2023.pdf', date: '2023-01-01', size: '156 KB' },
  ],
  certificates: [
    { id: 1, name: 'Zertifikat_Scrum_Master.pdf', date: '2022-06-15', size: '1.2 MB' },
    { id: 2, name: 'Weiterbildung_Leadership.pdf', date: '2023-03-20', size: '890 KB' },
  ],
  other: [
    { id: 1, name: 'Arbeitsbescheinigung_2023.pdf', date: '2023-12-31', size: '345 KB' },
  ],
};

export function Profil() {
  const [selectedDay, setSelectedDay] = useState<number | null>(null);
  const [entryType, setEntryType] = useState<string>('');
  const [calendarView, setCalendarView] = useState<'month' | 'week' | 'day'>('month');
  const [currentMonth, setCurrentMonth] = useState('Januar 2025');

  const formatTime = (hours: number) => {
    const h = Math.floor(hours);
    const m = Math.round((hours - h) * 60);
    return `${h}h ${m}m`;
  };

  const getHoursColor = (hours: number) => {
    if (hours === 0) return '#E5E7EB';
    if (hours > 8) return '#4ECCA3';
    if (hours >= 7) return '#5B9FED';
    return '#FFB347';
  };

  const totalHours = projectDistribution.reduce((sum, p) => sum + p.value, 0);

  return (
    <div className="flex-1 overflow-y-auto p-4 md:p-8 bg-slate-50 dark:bg-slate-900">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <div className="flex items-start gap-6 mb-6">
            <Avatar className="w-24 h-24 ring-4 ring-emerald-500 dark:ring-emerald-400">
              <AvatarImage src="https://images.unsplash.com/photo-1494790108377-be9c29b29330?w=200&h=200&fit=crop" />
              <AvatarFallback className="text-2xl">AM</AvatarFallback>
            </Avatar>
            <div className="flex-1">
              <h1 className="text-slate-900 dark:text-white mb-1">Anna Müller</h1>
              <p className="text-slate-500 dark:text-slate-400 mb-3">Project Manager • Abteilung: Entwicklung</p>
              <div className="flex flex-wrap gap-2">
                <Badge className="bg-emerald-100 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-300">
                  Vollzeit
                </Badge>
                <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300">
                  Seit März 2020
                </Badge>
                <Badge className="bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-300">
                  12 Aktive Projekte
                </Badge>
              </div>
            </div>
          </div>
        </div>

        <Tabs defaultValue="time" className="w-full">
          <TabsList className="mb-8">
            <TabsTrigger value="time" className="gap-2">
              <Clock className="w-4 h-4" />
              Zeiterfassung
            </TabsTrigger>
            <TabsTrigger value="calendar" className="gap-2">
              <CalendarIcon className="w-4 h-4" />
              Arbeitskalender
            </TabsTrigger>
            <TabsTrigger value="documents" className="gap-2">
              <FileText className="w-4 h-4" />
              Dokumente
            </TabsTrigger>
            <TabsTrigger value="personal" className="gap-2">
              <User className="w-4 h-4" />
              Mitarbeiterakte
            </TabsTrigger>
          </TabsList>

          {/* Time Tracking Tab */}
          <TabsContent value="time" className="space-y-6">
            {/* Enhanced Stat Cards */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
              <Card className="border-0 shadow-lg overflow-hidden hover:-translate-y-1 transition-transform duration-200">
                <div className="bg-gradient-to-br from-[#4ECCA3] to-[#45B393] p-6">
                  <div className="flex items-start justify-between mb-4">
                    <BarChart3 className="w-8 h-8 text-white" />
                  </div>
                  <p className="text-3xl font-bold text-white mb-1">41h 00m</p>
                  <p className="text-sm text-white/90 mb-2">Diese Woche</p>
                  <div className="flex items-center gap-1 text-white/90 text-xs">
                    <TrendingUp className="w-3 h-3" />
                    <span>2.5h vs. letzte Woche</span>
                  </div>
                </div>
              </Card>

              <Card className="border-0 shadow-lg overflow-hidden hover:-translate-y-1 transition-transform duration-200">
                <div className="bg-gradient-to-br from-[#5B9FED] to-[#4A8FDD] p-6">
                  <div className="flex items-start justify-between mb-4">
                    <CalendarDays className="w-8 h-8 text-white" />
                  </div>
                  <p className="text-3xl font-bold text-white mb-1">164h 30m</p>
                  <p className="text-sm text-white/90 mb-2">Dieser Monat</p>
                  <p className="text-white/90 text-xs">20.5 Arbeitstage</p>
                </div>
              </Card>

              <Card className="border-0 shadow-lg overflow-hidden hover:-translate-y-1 transition-transform duration-200">
                <div className="bg-gradient-to-br from-[#9B7EDE] to-[#8B6ECE] p-6">
                  <div className="flex items-start justify-between mb-4">
                    <Clock className="w-8 h-8 text-white" />
                  </div>
                  <p className="text-3xl font-bold text-white mb-1">8h 12m</p>
                  <p className="text-sm text-white/90 mb-2">Ø pro Tag</p>
                  <p className="text-white/90 text-xs">Gut im Ziel</p>
                </div>
              </Card>

              <Card className="border-0 shadow-lg overflow-hidden hover:-translate-y-1 transition-transform duration-200">
                <div className="bg-gradient-to-br from-[#FFB347] to-[#FFA337] p-6">
                  <div className="flex items-start justify-between mb-4">
                    <TrendingUp className="w-8 h-8 text-white" />
                  </div>
                  <p className="text-3xl font-bold text-white mb-1">+12h 30m</p>
                  <p className="text-sm text-white/90 mb-2">Überstunden</p>
                  <p className="text-white/90 text-xs">Diesen Monat</p>
                </div>
              </Card>
            </div>

            {/* New Visualization */}
            <div className="grid grid-cols-1 lg:grid-cols-5 gap-6">
              {/* Project Distribution - Donut Chart */}
              <Card className="border-slate-200 dark:border-slate-700 lg:col-span-3">
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <TrendingUp className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
                    Projekt-Zeitverteilung
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="flex flex-col md:flex-row items-center gap-8">
                    <div className="relative w-64 h-64">
                      <ResponsiveContainer width="100%" height="100%">
                        <PieChart>
                          <Pie
                            data={projectDistribution}
                            cx="50%"
                            cy="50%"
                            innerRadius={70}
                            outerRadius={100}
                            paddingAngle={2}
                            dataKey="value"
                          >
                            {projectDistribution.map((entry, index) => (
                              <Cell key={`cell-${index}`} fill={entry.color} />
                            ))}
                          </Pie>
                        </PieChart>
                      </ResponsiveContainer>
                      <div className="absolute inset-0 flex items-center justify-center">
                        <div className="text-center">
                          <p className="text-3xl font-bold text-slate-900 dark:text-white">{formatTime(totalHours)}</p>
                          <p className="text-sm text-slate-500 dark:text-slate-400">Gesamt</p>
                        </div>
                      </div>
                    </div>

                    <div className="flex-1 space-y-3">
                      {projectDistribution.map((project) => (
                        <div key={project.name} className="flex items-center justify-between">
                          <div className="flex items-center gap-3">
                            <div
                              className="w-4 h-4 rounded"
                              style={{ backgroundColor: project.color }}
                            />
                            <span className="text-sm text-slate-900 dark:text-white">{project.name}</span>
                          </div>
                          <div className="flex items-center gap-3">
                            <span className="text-sm font-semibold text-slate-900 dark:text-white">
                              {project.percentage}%
                            </span>
                            <span className="text-sm text-slate-500 dark:text-slate-400 min-w-[60px] text-right">
                              {formatTime(project.value)}
                            </span>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                </CardContent>
              </Card>

              {/* Weekly Overview */}
              <Card className="border-slate-200 dark:border-slate-700 lg:col-span-2">
                <CardHeader>
                  <CardTitle className="text-base">Wochenübersicht</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-3">
                    {weeklyOverview.map((day, index) => (
                      <div key={day.day} className="space-y-1">
                        <div className="flex items-center justify-between text-sm">
                          <span className={`font-medium ${index === new Date().getDay() - 1 ? 'text-emerald-600 dark:text-emerald-400' : 'text-slate-700 dark:text-slate-300'}`}>
                            {day.day}
                          </span>
                          <span className="text-slate-900 dark:text-white font-semibold">
                            {formatTime(day.hours)}
                          </span>
                        </div>
                        <div className="w-full bg-slate-200 dark:bg-slate-700 rounded-full h-2 overflow-hidden">
                          <div
                            className={`h-2 rounded-full transition-all duration-500 ${index === new Date().getDay() - 1 ? 'border-l-4 border-emerald-600' : ''}`}
                            style={{
                              width: `${(day.hours / 10) * 100}%`,
                              backgroundColor: day.color,
                            }}
                          />
                        </div>
                      </div>
                    ))}

                    <div className="pt-3 border-t border-slate-200 dark:border-slate-700">
                      <div className="flex items-center justify-between">
                        <span className="text-sm text-slate-600 dark:text-slate-400">Durchschnitt:</span>
                        <span className="text-sm font-semibold text-slate-900 dark:text-white">8.1h/Tag</span>
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>

            {/* Productivity Trend */}
            <Card className="border-slate-200 dark:border-slate-700">
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-base">
                  📈 Produktivitäts-Trend
                </CardTitle>
              </CardHeader>
              <CardContent>
                <ResponsiveContainer width="100%" height={80}>
                  <LineChart data={productivityTrend}>
                    <defs>
                      <linearGradient id="colorAvg" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="#4ECCA3" stopOpacity={0.3}/>
                        <stop offset="95%" stopColor="#4ECCA3" stopOpacity={0}/>
                      </linearGradient>
                    </defs>
                    <XAxis dataKey="week" tick={{ fontSize: 12 }} stroke="#94a3b8" />
                    <YAxis hide domain={[7, 9]} />
                    <RechartsTooltip 
                      contentStyle={{ 
                        backgroundColor: 'rgba(30, 41, 59, 0.95)', 
                        border: 'none',
                        borderRadius: '8px',
                        color: 'white'
                      }}
                      formatter={(value: number) => [`${value}h Ø`, 'Durchschnitt']}
                    />
                    <Line 
                      type="monotone" 
                      dataKey="avg" 
                      stroke="#4ECCA3" 
                      strokeWidth={3}
                      dot={{ fill: '#4ECCA3', r: 4 }}
                      fill="url(#colorAvg)"
                    />
                  </LineChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>
          </TabsContent>

          {/* Calendar Tab */}
          <TabsContent value="calendar" className="space-y-6">
            {/* Calendar Header */}
            <Card className="border-slate-200 dark:border-slate-700">
              <CardContent className="pt-6">
                <div className="flex flex-col md:flex-row items-center justify-between gap-4 mb-6">
                  <button className="flex items-center gap-2 px-4 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors">
                    <Plus className="w-4 h-4" />
                    Neuer Eintrag
                  </button>

                  <div className="flex items-center gap-4">
                    <button className="p-2 hover:bg-slate-100 dark:hover:bg-slate-700 rounded transition-colors">
                      <ChevronLeft className="w-5 h-5 text-slate-600 dark:text-slate-400" />
                    </button>
                    <h3 className="text-lg font-semibold text-slate-900 dark:text-white min-w-[150px] text-center">
                      {currentMonth}
                    </h3>
                    <button className="p-2 hover:bg-slate-100 dark:hover:bg-slate-700 rounded transition-colors">
                      <ChevronRight className="w-5 h-5 text-slate-600 dark:text-slate-400" />
                    </button>
                  </div>

                  <div className="flex gap-2">
                    <button
                      onClick={() => setCalendarView('month')}
                      className={`px-3 py-1.5 rounded-lg text-sm transition-colors ${
                        calendarView === 'month'
                          ? 'bg-emerald-600 text-white'
                          : 'bg-slate-100 dark:bg-slate-700 text-slate-700 dark:text-slate-300'
                      }`}
                    >
                      Monat
                    </button>
                    <button
                      onClick={() => setCalendarView('week')}
                      className={`px-3 py-1.5 rounded-lg text-sm transition-colors ${
                        calendarView === 'week'
                          ? 'bg-emerald-600 text-white'
                          : 'bg-slate-100 dark:bg-slate-700 text-slate-700 dark:text-slate-300'
                      }`}
                    >
                      Woche
                    </button>
                    <button
                      onClick={() => setCalendarView('day')}
                      className={`px-3 py-1.5 rounded-lg text-sm transition-colors ${
                        calendarView === 'day'
                          ? 'bg-emerald-600 text-white'
                          : 'bg-slate-100 dark:bg-slate-700 text-slate-700 dark:text-slate-300'
                      }`}
                    >
                      Tag
                    </button>
                  </div>
                </div>

                {/* Calendar Grid */}
                <div className="grid grid-cols-7 gap-2">
                  {/* Header */}
                  {['MO', 'DI', 'MI', 'DO', 'FR', 'SA', 'SO'].map((day) => (
                    <div key={day} className="text-center text-sm font-semibold text-slate-600 dark:text-slate-400 py-2">
                      {day}
                    </div>
                  ))}

                  {/* Days */}
                  {calendarDays.map((day) => {
                    const isWeekend = day.date % 7 === 6 || day.date % 7 === 0;
                    const isToday = day.date === 8;
                    
                    return (
                      <button
                        key={day.date}
                        onClick={() => setSelectedDay(day.date)}
                        className={`
                          relative p-3 rounded-lg border-2 transition-all hover:shadow-md min-h-[80px]
                          ${isToday ? 'border-emerald-500' : 'border-slate-200 dark:border-slate-700'}
                          ${isWeekend ? 'bg-slate-50 dark:bg-slate-800/50' : 'bg-white dark:bg-slate-800'}
                          ${day.status === 'vacation' ? 'bg-green-50 dark:bg-green-900/20' : ''}
                          ${day.status === 'sick' ? 'bg-red-50 dark:bg-red-900/20' : ''}
                        `}
                      >
                        <div className="text-left">
                          <span className="text-xs text-slate-500 dark:text-slate-400">{day.date}</span>
                        </div>
                        {day.status && (
                          <div className="flex flex-col items-center justify-center mt-2">
                            <span className="text-2xl mb-1">{day.icon}</span>
                            <span className="text-xs text-slate-600 dark:text-slate-400">{day.statusText}</span>
                          </div>
                        )}
                        {!day.status && day.hours > 0 && (
                          <div className="flex items-center justify-center mt-4">
                            <span className="text-base font-semibold" style={{ color: getHoursColor(day.hours) }}>
                              {formatTime(day.hours)}
                            </span>
                          </div>
                        )}
                      </button>
                    );
                  })}
                </div>
              </CardContent>
            </Card>

            {/* Calendar Stats Sidebar */}
            <Card className="border-slate-200 dark:border-slate-700">
              <CardHeader>
                <CardTitle className="text-base">Jahresübersicht 2025</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-slate-600 dark:text-slate-400">Urlaubstage genommen</span>
                    <span className="text-sm font-semibold text-slate-900 dark:text-white">20 / 25</span>
                  </div>
                  <div className="w-full bg-slate-200 dark:bg-slate-700 rounded-full h-2">
                    <div className="h-2 bg-green-500 rounded-full" style={{ width: '80%' }} />
                  </div>

                  <div className="flex items-center justify-between pt-2">
                    <span className="text-sm text-slate-600 dark:text-slate-400">Krankheitstage</span>
                    <span className="text-sm font-semibold text-slate-900 dark:text-white">3</span>
                  </div>

                  <div className="flex items-center justify-between">
                    <span className="text-sm text-slate-600 dark:text-slate-400">Überstunden kumuliert</span>
                    <span className="text-sm font-semibold text-emerald-600 dark:text-emerald-400">+12.5h</span>
                  </div>

                  <div className="flex items-center justify-between">
                    <span className="text-sm text-slate-600 dark:text-slate-400">Ø Wochenarbeitszeit</span>
                    <span className="text-sm font-semibold text-slate-900 dark:text-white">41.2h</span>
                  </div>
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          {/* Documents Tab */}
          <TabsContent value="documents" className="space-y-6">
            {/* Lohnabrechnungen */}
            <Card className="border-slate-200 dark:border-slate-700">
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <FileText className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
                  Lohnabrechnungen
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-2">
                  {documents.payroll.map((doc) => (
                    <div
                      key={doc.id}
                      className="flex items-center justify-between p-3 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
                    >
                      <div className="flex items-center gap-3 flex-1 min-w-0">
                        <FileText className="w-5 h-5 text-red-500 flex-shrink-0" />
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-medium text-slate-900 dark:text-white truncate">{doc.name}</p>
                          <p className="text-xs text-slate-500 dark:text-slate-400">
                            {new Date(doc.date).toLocaleDateString('de-CH')} • {doc.size}
                          </p>
                        </div>
                      </div>
                      <div className="flex gap-2">
                        <button className="p-2 hover:bg-slate-200 dark:hover:bg-slate-600 rounded transition-colors">
                          <Eye className="w-4 h-4 text-slate-600 dark:text-slate-400" />
                        </button>
                        <button className="p-2 hover:bg-slate-200 dark:hover:bg-slate-600 rounded transition-colors">
                          <Download className="w-4 h-4 text-slate-600 dark:text-slate-400" />
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>

            {/* Verträge */}
            <Card className="border-slate-200 dark:border-slate-700">
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <FileCheck className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
                  Arbeitsverträge & Nachträge
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-2">
                  {documents.contracts.map((doc) => (
                    <div
                      key={doc.id}
                      className="flex items-center justify-between p-3 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
                    >
                      <div className="flex items-center gap-3 flex-1 min-w-0">
                        <FileText className="w-5 h-5 text-blue-500 flex-shrink-0" />
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-medium text-slate-900 dark:text-white truncate">{doc.name}</p>
                          <p className="text-xs text-slate-500 dark:text-slate-400">
                            {new Date(doc.date).toLocaleDateString('de-CH')} • {doc.size}
                          </p>
                        </div>
                      </div>
                      <div className="flex gap-2">
                        <button className="p-2 hover:bg-slate-200 dark:hover:bg-slate-600 rounded transition-colors">
                          <Eye className="w-4 h-4 text-slate-600 dark:text-slate-400" />
                        </button>
                        <button className="p-2 hover:bg-slate-200 dark:hover:bg-slate-600 rounded transition-colors">
                          <Download className="w-4 h-4 text-slate-600 dark:text-slate-400" />
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>

            {/* Zertifikate & Weiterbildungen */}
            <Card className="border-slate-200 dark:border-slate-700">
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <GraduationCap className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
                  Zertifikate & Weiterbildungen
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-2">
                  {documents.certificates.map((doc) => (
                    <div
                      key={doc.id}
                      className="flex items-center justify-between p-3 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
                    >
                      <div className="flex items-center gap-3 flex-1 min-w-0">
                        <FileText className="w-5 h-5 text-emerald-500 flex-shrink-0" />
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-medium text-slate-900 dark:text-white truncate">{doc.name}</p>
                          <p className="text-xs text-slate-500 dark:text-slate-400">
                            {new Date(doc.date).toLocaleDateString('de-CH')} • {doc.size}
                          </p>
                        </div>
                      </div>
                      <div className="flex gap-2">
                        <button className="p-2 hover:bg-slate-200 dark:hover:bg-slate-600 rounded transition-colors">
                          <Eye className="w-4 h-4 text-slate-600 dark:text-slate-400" />
                        </button>
                        <button className="p-2 hover:bg-slate-200 dark:hover:bg-slate-600 rounded transition-colors">
                          <Download className="w-4 h-4 text-slate-600 dark:text-slate-400" />
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>

            {/* Sonstige Dokumente */}
            <Card className="border-slate-200 dark:border-slate-700">
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle className="flex items-center gap-2">
                    <FileText className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
                    Sonstige Dokumente
                  </CardTitle>
                  <button className="flex items-center gap-2 px-3 py-1.5 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors text-sm">
                    <Upload className="w-4 h-4" />
                    Hochladen
                  </button>
                </div>
              </CardHeader>
              <CardContent>
                <div className="space-y-2">
                  {documents.other.map((doc) => (
                    <div
                      key={doc.id}
                      className="flex items-center justify-between p-3 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
                    >
                      <div className="flex items-center gap-3 flex-1 min-w-0">
                        <FileText className="w-5 h-5 text-slate-500 flex-shrink-0" />
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-medium text-slate-900 dark:text-white truncate">{doc.name}</p>
                          <p className="text-xs text-slate-500 dark:text-slate-400">
                            {new Date(doc.date).toLocaleDateString('de-CH')} • {doc.size}
                          </p>
                        </div>
                      </div>
                      <div className="flex gap-2">
                        <button className="p-2 hover:bg-slate-200 dark:hover:bg-slate-600 rounded transition-colors">
                          <Eye className="w-4 h-4 text-slate-600 dark:text-slate-400" />
                        </button>
                        <button className="p-2 hover:bg-slate-200 dark:hover:bg-slate-600 rounded transition-colors">
                          <Download className="w-4 h-4 text-slate-600 dark:text-slate-400" />
                        </button>
                        <button className="p-2 hover:bg-red-100 dark:hover:bg-red-900/20 rounded transition-colors">
                          <Trash2 className="w-4 h-4 text-red-600 dark:text-red-400" />
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          {/* Personal Data / Employee File Tab */}
          <TabsContent value="personal" className="space-y-6">
            {/* Basic Information */}
            <Card className="border-slate-200 dark:border-slate-700">
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <User className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
                  Persönliche Daten
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label>Vorname</Label>
                    <Input defaultValue="Anna" />
                  </div>
                  <div className="space-y-2">
                    <Label>Nachname</Label>
                    <Input defaultValue="Müller" />
                  </div>
                  <div className="space-y-2">
                    <Label>Geburtsdatum</Label>
                    <Input type="date" defaultValue="1990-05-15" />
                  </div>
                  <div className="space-y-2">
                    <Label>Nationalität</Label>
                    <Input defaultValue="Schweiz" />
                  </div>
                  <div className="space-y-2">
                    <Label>AHV-Nummer</Label>
                    <Input defaultValue="756.1234.5678.90" />
                  </div>
                  <div className="space-y-2">
                    <Label>Zivilstand</Label>
                    <Select defaultValue="ledig">
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="ledig">Ledig</SelectItem>
                        <SelectItem value="verheiratet">Verheiratet</SelectItem>
                        <SelectItem value="geschieden">Geschieden</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* Contact Information */}
            <Card className="border-slate-200 dark:border-slate-700">
              <CardHeader>
                <CardTitle>Kontaktinformationen</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label>E-Mail (Geschäftlich)</Label>
                    <Input type="email" defaultValue="anna.mueller@swissbiz.ch" />
                  </div>
                  <div className="space-y-2">
                    <Label>E-Mail (Privat)</Label>
                    <Input type="email" defaultValue="anna.mueller@gmail.com" />
                  </div>
                  <div className="space-y-2">
                    <Label>Telefon (Geschäftlich)</Label>
                    <Input defaultValue="+41 44 123 45 67" />
                  </div>
                  <div className="space-y-2">
                    <Label>Telefon (Mobil)</Label>
                    <Input defaultValue="+41 79 123 45 67" />
                  </div>
                  <div className="space-y-2 md:col-span-2">
                    <Label>Adresse</Label>
                    <Input defaultValue="Bahnhofstrasse 123" />
                  </div>
                  <div className="space-y-2">
                    <Label>PLZ</Label>
                    <Input defaultValue="8001" />
                  </div>
                  <div className="space-y-2">
                    <Label>Stadt</Label>
                    <Input defaultValue="Zürich" />
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* Employment Details */}
            <Card className="border-slate-200 dark:border-slate-700">
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Briefcase className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
                  Anstellungsdetails
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label>Position</Label>
                    <Input defaultValue="Project Manager" />
                  </div>
                  <div className="space-y-2">
                    <Label>Abteilung</Label>
                    <Input defaultValue="Entwicklung" />
                  </div>
                  <div className="space-y-2">
                    <Label>Eintrittsdatum</Label>
                    <Input type="date" defaultValue="2020-03-15" />
                  </div>
                  <div className="space-y-2">
                    <Label>Beschäftigungsgrad</Label>
                    <Input defaultValue="100%" />
                  </div>
                  <div className="space-y-2">
                    <Label>Wochenarbeitszeit</Label>
                    <Input defaultValue="40 Stunden" />
                  </div>
                  <div className="space-y-2">
                    <Label>Vertragslaufzeit</Label>
                    <Select defaultValue="unbefristet">
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="unbefristet">Unbefristet</SelectItem>
                        <SelectItem value="befristet">Befristet</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label>Vorgesetzter</Label>
                    <Input defaultValue="Dr. Thomas Schmidt" />
                  </div>
                  <div className="space-y-2">
                    <Label>Urlaubsanspruch</Label>
                    <Input defaultValue="25 Tage / Jahr" />
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* Education & CV */}
            <Card className="border-slate-200 dark:border-slate-700">
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle className="flex items-center gap-2">
                    <GraduationCap className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
                    Ausbildung & Lebenslauf
                  </CardTitle>
                  <button className="flex items-center gap-2 px-3 py-1.5 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors text-sm">
                    <Plus className="w-4 h-4" />
                    Hinzufügen
                  </button>
                </div>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  {/* Education Entry */}
                  <div className="p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700">
                    <div className="flex items-start justify-between mb-2">
                      <div>
                        <h4 className="font-medium text-slate-900 dark:text-white">Master of Science in Business Administration</h4>
                        <p className="text-sm text-slate-600 dark:text-slate-400">Universität Zürich</p>
                      </div>
                      <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300">
                        2012 - 2015
                      </Badge>
                    </div>
                    <p className="text-sm text-slate-500 dark:text-slate-400">
                      Schwerpunkt: Projektmanagement & Digitale Transformation
                    </p>
                  </div>

                  <div className="p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700">
                    <div className="flex items-start justify-between mb-2">
                      <div>
                        <h4 className="font-medium text-slate-900 dark:text-white">Bachelor of Arts in Wirtschaftsinformatik</h4>
                        <p className="text-sm text-slate-600 dark:text-slate-400">ETH Zürich</p>
                      </div>
                      <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300">
                        2009 - 2012
                      </Badge>
                    </div>
                  </div>

                  {/* Work Experience */}
                  <div className="border-t border-slate-200 dark:border-slate-700 pt-4 mt-4">
                    <h4 className="font-medium text-slate-900 dark:text-white mb-3">Berufserfahrung</h4>
                    
                    <div className="space-y-3">
                      <div className="p-4 bg-emerald-50 dark:bg-emerald-900/20 rounded-lg border border-emerald-200 dark:border-emerald-800">
                        <div className="flex items-start justify-between mb-2">
                          <div>
                            <h5 className="font-medium text-slate-900 dark:text-white">Project Manager</h5>
                            <p className="text-sm text-slate-600 dark:text-slate-400">Swiss Business Solutions AG</p>
                          </div>
                          <Badge className="bg-emerald-600 text-white">
                            2020 - Heute
                          </Badge>
                        </div>
                        <p className="text-sm text-slate-500 dark:text-slate-400">
                          Leitung von 12+ digitalen Transformationsprojekten
                        </p>
                      </div>

                      <div className="p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700">
                        <div className="flex items-start justify-between mb-2">
                          <div>
                            <h5 className="font-medium text-slate-900 dark:text-white">Junior Project Manager</h5>
                            <p className="text-sm text-slate-600 dark:text-slate-400">Digital Innovations GmbH</p>
                          </div>
                          <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300">
                            2015 - 2020
                          </Badge>
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* CV Upload */}
                  <div className="border-t border-slate-200 dark:border-slate-700 pt-4">
                    <div className="flex items-center justify-between p-3 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700">
                      <div className="flex items-center gap-3">
                        <FileText className="w-5 h-5 text-blue-500" />
                        <div>
                          <p className="text-sm font-medium text-slate-900 dark:text-white">Lebenslauf_Anna_Mueller.pdf</p>
                          <p className="text-xs text-slate-500 dark:text-slate-400">Aktualisiert am 15.12.2024 • 456 KB</p>
                        </div>
                      </div>
                      <div className="flex gap-2">
                        <button className="p-2 hover:bg-slate-200 dark:hover:bg-slate-600 rounded transition-colors">
                          <Download className="w-4 h-4 text-slate-600 dark:text-slate-400" />
                        </button>
                        <button className="p-2 hover:bg-slate-200 dark:hover:bg-slate-600 rounded transition-colors">
                          <Upload className="w-4 h-4 text-slate-600 dark:text-slate-400" />
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* Emergency Contact */}
            <Card className="border-slate-200 dark:border-slate-700">
              <CardHeader>
                <CardTitle>Notfallkontakt</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label>Name</Label>
                    <Input defaultValue="Max Müller" />
                  </div>
                  <div className="space-y-2">
                    <Label>Beziehung</Label>
                    <Input defaultValue="Ehepartner" />
                  </div>
                  <div className="space-y-2">
                    <Label>Telefon</Label>
                    <Input defaultValue="+41 79 987 65 43" />
                  </div>
                  <div className="space-y-2">
                    <Label>E-Mail</Label>
                    <Input type="email" defaultValue="max.mueller@gmail.com" />
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* Save Button */}
            <div className="flex justify-end">
              <button className="px-6 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors">
                Änderungen speichern
              </button>
            </div>
          </TabsContent>
        </Tabs>

        {/* Day Detail Modal */}
        <Dialog open={selectedDay !== null} onOpenChange={() => setSelectedDay(null)}>
          <DialogContent className="max-w-md">
            <DialogHeader>
              <DialogTitle className="flex items-center justify-between">
                <span>Montag, {selectedDay}. Januar 2025</span>
                <button onClick={() => setSelectedDay(null)} className="p-1 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
                  <X className="w-5 h-5" />
                </button>
              </DialogTitle>
              <DialogDescription className="sr-only">
                Tagesdetails und Eintragsverwaltung für den ausgewählten Tag
              </DialogDescription>
            </DialogHeader>

            <div className="space-y-4 py-4">
              <div>
                <p className="text-sm font-semibold text-slate-900 dark:text-white mb-2">📊 ARBEITSZEIT: 8h 12m</p>
                <div className="space-y-2">
                  {projectDistribution.map((project) => (
                    <div key={project.name} className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <div className="w-3 h-3 rounded" style={{ backgroundColor: project.color }} />
                        <span className="text-sm text-slate-700 dark:text-slate-300">{project.name}</span>
                      </div>
                      <span className="text-sm font-medium text-slate-900 dark:text-white">
                        {formatTime(project.value / 5)}
                      </span>
                    </div>
                  ))}
                </div>
              </div>

              <div className="border-t border-slate-200 dark:border-slate-700 pt-4">
                <Label className="mb-2">➕ Eintrag hinzufügen</Label>
                <Select value={entryType} onValueChange={setEntryType}>
                  <SelectTrigger>
                    <SelectValue placeholder="Typ auswählen..." />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="vacation">🌴 Urlaub</SelectItem>
                    <SelectItem value="sick">🏥 Krankmeldung</SelectItem>
                    <SelectItem value="doctor">🏥 Arzttermin</SelectItem>
                    <SelectItem value="overtime">⏰ Überstunden abbauen</SelectItem>
                    <SelectItem value="note">✏️ Notiz hinzufügen</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              {entryType === 'vacation' && (
                <div className="space-y-4 p-4 bg-green-50 dark:bg-green-900/20 rounded-lg border border-green-200 dark:border-green-800">
                  <h4 className="font-semibold text-slate-900 dark:text-white">🌴 Urlaub beantragen</h4>
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label>Von</Label>
                      <Input type="date" />
                    </div>
                    <div className="space-y-2">
                      <Label>Bis</Label>
                      <Input type="date" />
                    </div>
                  </div>
                  <div>
                    <p className="text-sm text-slate-700 dark:text-slate-300 mb-2">Dauer: 5 Tage</p>
                    <div className="space-y-2">
                      <label className="flex items-center gap-2">
                        <input type="radio" name="vacation-type" defaultChecked />
                        <span className="text-sm text-slate-700 dark:text-slate-300">Von Urlaubstagen (12 verfügbar)</span>
                      </label>
                      <label className="flex items-center gap-2">
                        <input type="radio" name="vacation-type" />
                        <span className="text-sm text-slate-700 dark:text-slate-300">Von Überstunden (12.5h verfügbar)</span>
                      </label>
                    </div>
                  </div>
                  <div className="space-y-2">
                    <Label>Grund (optional)</Label>
                    <Textarea placeholder="z.B. Familienurlaub" />
                  </div>
                  <div className="flex items-center gap-2 text-sm text-blue-600 dark:text-blue-400">
                    <Info className="w-4 h-4" />
                    <span>Wird an Teamleiter gesendet</span>
                  </div>
                  <div className="flex gap-2">
                    <button 
                      onClick={() => setEntryType('')}
                      className="flex-1 px-4 py-2 bg-slate-200 dark:bg-slate-700 text-slate-700 dark:text-slate-300 rounded-lg hover:bg-slate-300 dark:hover:bg-slate-600 transition-colors"
                    >
                      Abbrechen
                    </button>
                    <button className="flex-1 px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors">
                      Antrag einreichen
                    </button>
                  </div>
                </div>
              )}

              {entryType === 'sick' && (
                <div className="space-y-4 p-4 bg-red-50 dark:bg-red-900/20 rounded-lg border border-red-200 dark:border-red-800">
                  <h4 className="font-semibold text-slate-900 dark:text-white">🏥 Krankmeldung</h4>
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label>Von</Label>
                      <Input type="date" />
                    </div>
                    <div className="space-y-2">
                      <Label>Bis</Label>
                      <Input type="date" placeholder="Offen" />
                    </div>
                  </div>
                  <div className="space-y-2">
                    <label className="flex items-center gap-2">
                      <input type="checkbox" />
                      <span className="text-sm text-slate-700 dark:text-slate-300">Ärztliches Attest liegt vor</span>
                    </label>
                    <button className="flex items-center gap-2 text-sm text-blue-600 dark:text-blue-400">
                      <Paperclip className="w-4 h-4" />
                      Datei hochladen
                    </button>
                  </div>
                  <div className="space-y-2">
                    <Label>Beschreibung (optional)</Label>
                    <Textarea />
                  </div>
                  <div className="flex items-center gap-2 text-sm text-blue-600 dark:text-blue-400">
                    <Info className="w-4 h-4" />
                    <span>Wird an Teamleiter gesendet</span>
                  </div>
                  <div className="flex gap-2">
                    <button 
                      onClick={() => setEntryType('')}
                      className="flex-1 px-4 py-2 bg-slate-200 dark:bg-slate-700 text-slate-700 dark:text-slate-300 rounded-lg hover:bg-slate-300 dark:hover:bg-slate-600 transition-colors"
                    >
                      Abbrechen
                    </button>
                    <button className="flex-1 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors">
                      Meldung einreichen
                    </button>
                  </div>
                </div>
              )}
            </div>
          </DialogContent>
        </Dialog>
      </div>
    </div>
  );
}