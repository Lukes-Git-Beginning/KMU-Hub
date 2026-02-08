import { useState } from 'react';
import { 
  Search, Plus, Users, Building, Handshake, Archive, ChevronDown, Phone, 
  Mail, MessageCircle, ChevronLeft, MoreVertical, Edit, Trash2, X,
  List, Grid3x3, FolderPlus, Download, Check, Clock, MapPin, Briefcase,
  FileText, Link as LinkIcon
} from 'lucide-react';
import { Avatar, AvatarFallback } from '../components/ui/avatar';
import { Badge } from '../components/ui/badge';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '../components/ui/dialog';
import { Label } from '../components/ui/label';
import { Input } from '../components/ui/input';
import { Textarea } from '../components/ui/textarea';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../components/ui/tabs';

// Mock Data
const contacts = [
  {
    id: 1,
    firstName: 'Achim',
    lastName: 'Weber',
    email: 'achim.weber@kmu-hub.ch',
    phone: '+41 76 123 45 67',
    avatar: 'AW',
    company: 'KMU Digital Hub',
    jobTitle: 'Senior Developer',
    department: 'Entwicklung',
    address: 'Bahnhofstrasse 42',
    zipCode: '8000',
    city: 'Zürich',
    country: 'Schweiz',
    category: 'employee',
    status: 'active',
    createdDate: '15. November 2024',
    projects: ['Website Relaunch', 'Mobile App Dev', 'API Integration'],
    teams: ['Development', 'Project Management'],
    lastContact: new Date('2024-12-19'),
  },
  {
    id: 2,
    firstName: 'Hans',
    lastName: 'Müller',
    email: 'hans.mueller@mueller-ag.ch',
    phone: '+41 44 555 67 89',
    avatar: 'HM',
    company: 'Müller AG',
    jobTitle: 'Geschäftsführer',
    department: 'Management',
    address: 'Industriestrasse 15',
    zipCode: '8050',
    city: 'Zürich',
    country: 'Schweiz',
    category: 'customer',
    status: 'active',
    createdDate: '3. Oktober 2024',
    projects: ['Website Relaunch'],
    teams: [],
    lastContact: new Date('2024-12-18'),
  },
  {
    id: 3,
    firstName: 'Lisa',
    lastName: 'Partner',
    email: 'lisa.partner@techsol.ch',
    phone: '+41 31 888 22 33',
    avatar: 'LP',
    company: 'Tech Solutions GmbH',
    jobTitle: 'Account Manager',
    department: 'Verkauf',
    address: 'Bundesplatz 8',
    zipCode: '3011',
    city: 'Bern',
    country: 'Schweiz',
    category: 'partner',
    status: 'prospect',
    createdDate: '1. Dezember 2024',
    projects: [],
    teams: [],
    lastContact: new Date('2024-12-15'),
  },
  {
    id: 4,
    firstName: 'Sarah',
    lastName: 'Schmidt',
    email: 'sarah.schmidt@kmu-hub.ch',
    phone: '+41 76 234 56 78',
    avatar: 'SS',
    company: 'KMU Digital Hub',
    jobTitle: 'UI/UX Designer',
    department: 'Design',
    address: 'Bahnhofstrasse 42',
    zipCode: '8000',
    city: 'Zürich',
    country: 'Schweiz',
    category: 'employee',
    status: 'active',
    createdDate: '20. September 2024',
    projects: ['Website Relaunch', 'Mobile App Dev'],
    teams: ['Design'],
    lastContact: new Date('2024-12-20'),
  },
  {
    id: 5,
    firstName: 'Michael',
    lastName: 'Berg',
    email: 'michael.berg@kmu-hub.ch',
    phone: '+41 76 345 67 89',
    avatar: 'MB',
    company: 'KMU Digital Hub',
    jobTitle: 'Project Manager',
    department: 'Management',
    address: 'Bahnhofstrasse 42',
    zipCode: '8000',
    city: 'Zürich',
    country: 'Schweiz',
    category: 'employee',
    status: 'active',
    createdDate: '5. August 2024',
    projects: ['Website Relaunch', 'Mobile App Dev', 'API Integration'],
    teams: ['Project Management', 'Development'],
    lastContact: new Date('2024-12-21'),
  },
  {
    id: 6,
    firstName: 'Anna',
    lastName: 'Keller',
    email: 'anna.keller@designco.ch',
    phone: '+41 44 777 88 99',
    avatar: 'AK',
    company: 'Design Co.',
    jobTitle: 'Creative Director',
    department: 'Design',
    address: 'Kreativstrasse 5',
    zipCode: '8001',
    city: 'Zürich',
    country: 'Schweiz',
    category: 'customer',
    status: 'active',
    createdDate: '12. Juli 2024',
    projects: ['Mobile App Dev'],
    teams: [],
    lastContact: new Date('2024-12-17'),
  },
  {
    id: 7,
    firstName: 'Thomas',
    lastName: 'Klein',
    email: 'thomas.klein@kmu-hub.ch',
    phone: '+41 76 456 78 90',
    avatar: 'TK',
    company: 'KMU Digital Hub',
    jobTitle: 'Backend Developer',
    department: 'Entwicklung',
    address: 'Bahnhofstrasse 42',
    zipCode: '8000',
    city: 'Zürich',
    country: 'Schweiz',
    category: 'employee',
    status: 'active',
    createdDate: '28. Juni 2024',
    projects: ['API Integration', 'Website Relaunch'],
    teams: ['Development'],
    lastContact: new Date('2024-12-20'),
  },
  {
    id: 8,
    firstName: 'Sophie',
    lastName: 'Wagner',
    email: 'sophie.wagner@marketing-plus.ch',
    phone: '+41 21 333 44 55',
    avatar: 'SW',
    company: 'Marketing Plus AG',
    jobTitle: 'Marketing Manager',
    department: 'Marketing',
    address: 'Avenue de la Gare 12',
    zipCode: '1003',
    city: 'Lausanne',
    country: 'Schweiz',
    category: 'partner',
    status: 'active',
    createdDate: '15. Mai 2024',
    projects: [],
    teams: [],
    lastContact: new Date('2024-12-10'),
  },
  {
    id: 9,
    firstName: 'Max',
    lastName: 'Fischer',
    email: 'max.fischer@finance-corp.ch',
    phone: '+41 61 666 77 88',
    avatar: 'MF',
    company: 'Finance Corp',
    jobTitle: 'CFO',
    department: 'Finance',
    address: 'Freie Strasse 25',
    zipCode: '4051',
    city: 'Basel',
    country: 'Schweiz',
    category: 'customer',
    status: 'prospect',
    createdDate: '8. November 2024',
    projects: [],
    teams: [],
    lastContact: new Date('2024-12-12'),
  },
  {
    id: 10,
    firstName: 'Julia',
    lastName: 'Meier',
    email: 'julia.meier@kmu-hub.ch',
    phone: '+41 76 567 89 01',
    avatar: 'JM',
    company: 'KMU Digital Hub',
    jobTitle: 'Frontend Developer',
    department: 'Entwicklung',
    address: 'Bahnhofstrasse 42',
    zipCode: '8000',
    city: 'Zürich',
    country: 'Schweiz',
    category: 'employee',
    status: 'active',
    createdDate: '22. März 2024',
    projects: ['Website Relaunch', 'Mobile App Dev'],
    teams: ['Development'],
    lastContact: new Date('2024-12-21'),
  },
];

const categories = [
  { id: 'all', name: 'Alle Kontakte', icon: Users, count: contacts.length },
  { id: 'employee', name: 'Mitarbeiter', icon: Users, count: contacts.filter(c => c.category === 'employee').length },
  { id: 'customer', name: 'Kunden', icon: Building, count: contacts.filter(c => c.category === 'customer').length },
  { id: 'partner', name: 'Partner & Lieferanten', icon: Handshake, count: contacts.filter(c => c.category === 'partner').length },
  { id: 'archived', name: 'Archiviert', icon: Archive, count: 0 },
];

const projectsList = [
  { id: 1, name: 'Website Relaunch', status: 'In Progress', members: 5, role: 'Developer' },
  { id: 2, name: 'Mobile App Dev', status: 'Planning', members: 8, role: 'Technical Lead' },
  { id: 3, name: 'API Integration', status: 'In Progress', members: 3, role: 'Developer' },
];

const teamsList = [
  { id: 1, name: 'Development', description: 'Entwicklung & Architektur', members: 8 },
  { id: 2, name: 'Project Management', description: 'Projektmanagement & Koordination', members: 5 },
  { id: 3, name: 'Design', description: 'UI/UX Design & Branding', members: 4 },
];

type ViewType = 'list' | 'detail';
type CategoryType = 'all' | 'employee' | 'customer' | 'partner' | 'archived';

export function Kontakte() {
  const [currentView, setCurrentView] = useState<ViewType>('list');
  const [selectedContact, setSelectedContact] = useState<any>(null);
  const [activeCategory, setActiveCategory] = useState<CategoryType>('all');
  const [selectedContacts, setSelectedContacts] = useState<number[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [showNewContactModal, setShowNewContactModal] = useState(false);
  const [showFilters, setShowFilters] = useState(true);
  const [activeTab, setActiveTab] = useState('overview');
  const [toast, setToast] = useState<{ show: boolean; message: string; type: 'success' | 'error' | 'info' }>({ 
    show: false, 
    message: '', 
    type: 'success' 
  });

  const showToast = (message: string, type: 'success' | 'error' | 'info' = 'success') => {
    setToast({ show: true, message, type });
    setTimeout(() => setToast({ show: false, message: '', type: 'success' }), 3000);
  };

  const handleContactClick = (contact: any) => {
    setSelectedContact(contact);
    setCurrentView('detail');
    setActiveTab('overview');
  };

  const handleBackToList = () => {
    setCurrentView('list');
    setSelectedContact(null);
  };

  const toggleContactSelection = (id: number) => {
    setSelectedContacts(prev =>
      prev.includes(id) ? prev.filter(cid => cid !== id) : [...prev, id]
    );
  };

  const filteredContacts = contacts.filter(contact => {
    // Category filter
    if (activeCategory !== 'all' && contact.category !== activeCategory) return false;
    
    // Search filter
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      return (
        contact.firstName.toLowerCase().includes(query) ||
        contact.lastName.toLowerCase().includes(query) ||
        contact.email.toLowerCase().includes(query) ||
        contact.company.toLowerCase().includes(query) ||
        contact.jobTitle.toLowerCase().includes(query)
      );
    }
    
    return true;
  });

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active':
        return 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300';
      case 'prospect':
        return 'bg-orange-100 dark:bg-orange-900/30 text-orange-700 dark:text-orange-300';
      case 'inactive':
        return 'bg-gray-100 dark:bg-gray-900/30 text-gray-700 dark:text-gray-300';
      default:
        return 'bg-gray-100 dark:bg-gray-900/30 text-gray-700 dark:text-gray-300';
    }
  };

  const getStatusLabel = (status: string) => {
    switch (status) {
      case 'active': return '✓ Aktiv';
      case 'prospect': return 'Prospect';
      case 'inactive': return 'Inaktiv';
      default: return status;
    }
  };

  return (
    <div className="flex h-full bg-slate-50 dark:bg-slate-900">
      {/* LEFT SIDEBAR - NAVIGATION & FILTERS */}
      <div className="w-72 border-r border-slate-200 dark:border-slate-700 flex flex-col bg-white dark:bg-slate-800">
        {/* Header */}
        <div className="p-4 border-b border-slate-200 dark:border-slate-700">
          <h2 className="text-slate-900 dark:text-white mb-1">Kontakte</h2>
          <p className="text-xs text-slate-500 dark:text-slate-400 mb-4">CRM & Verwaltung</p>
          <button
            onClick={() => setShowNewContactModal(true)}
            className="w-full px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg transition-colors flex items-center justify-center gap-2"
          >
            <Plus className="w-4 h-4" />
            Neuer Kontakt
          </button>
        </div>

        {/* Search */}
        <div className="p-4 border-b border-slate-200 dark:border-slate-700">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
            <input
              type="text"
              placeholder="Kontakte durchsuchen..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-10 pr-4 py-2 bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg text-sm text-slate-900 dark:text-white placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-emerald-500"
            />
          </div>
        </div>

        {/* Categories */}
        <div className="flex-1 overflow-y-auto p-4">
          <div className="mb-4">
            <h3 className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase mb-2">
              Kategorien
            </h3>
            <div className="space-y-1">
              {categories.map((category) => {
                const Icon = category.icon;
                const isActive = activeCategory === category.id;
                return (
                  <button
                    key={category.id}
                    onClick={() => {
                      setActiveCategory(category.id as CategoryType);
                      setCurrentView('list');
                    }}
                    className={`w-full flex items-center justify-between px-3 py-2 rounded-lg transition-colors text-sm ${
                      isActive
                        ? 'bg-emerald-50 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-300'
                        : 'text-slate-600 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700'
                    }`}
                  >
                    <div className="flex items-center gap-2">
                      <Icon className="w-4 h-4" />
                      <span>{category.name}</span>
                    </div>
                    <span className="text-xs text-slate-400">({category.count})</span>
                  </button>
                );
              })}
            </div>
          </div>

          {/* Filters */}
          <div className="mb-4">
            <button
              onClick={() => setShowFilters(!showFilters)}
              className="w-full flex items-center justify-between px-3 py-2 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase hover:bg-slate-50 dark:hover:bg-slate-700 rounded-lg"
            >
              <span>Filter</span>
              <ChevronDown className={`w-3 h-3 transition-transform ${showFilters ? '' : '-rotate-90'}`} />
            </button>
            
            {showFilters && (
              <div className="mt-2 space-y-3 px-3">
                {/* Status Filter */}
                <div>
                  <p className="text-xs font-medium text-slate-600 dark:text-slate-400 mb-2">Nach Status</p>
                  <div className="space-y-1.5">
                    <label className="flex items-center gap-2 text-xs text-slate-600 dark:text-slate-300">
                      <input type="checkbox" defaultChecked className="rounded border-slate-300" />
                      Aktiv
                    </label>
                    <label className="flex items-center gap-2 text-xs text-slate-600 dark:text-slate-300">
                      <input type="checkbox" className="rounded border-slate-300" />
                      Prospect
                    </label>
                    <label className="flex items-center gap-2 text-xs text-slate-600 dark:text-slate-300">
                      <input type="checkbox" className="rounded border-slate-300" />
                      Inaktiv
                    </label>
                  </div>
                </div>

                {/* City Filter */}
                <div>
                  <p className="text-xs font-medium text-slate-600 dark:text-slate-400 mb-2">Nach Stadt</p>
                  <div className="space-y-1.5">
                    <label className="flex items-center gap-2 text-xs text-slate-600 dark:text-slate-300">
                      <input type="checkbox" className="rounded border-slate-300" />
                      Zürich
                    </label>
                    <label className="flex items-center gap-2 text-xs text-slate-600 dark:text-slate-300">
                      <input type="checkbox" className="rounded border-slate-300" />
                      Bern
                    </label>
                    <label className="flex items-center gap-2 text-xs text-slate-600 dark:text-slate-300">
                      <input type="checkbox" className="rounded border-slate-300" />
                      Basel
                    </label>
                  </div>
                </div>
              </div>
            )}
          </div>

          {/* Sorting */}
          <div>
            <h3 className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase mb-2 px-3">
              Sortierung
            </h3>
            <button className="w-full flex items-center justify-between px-3 py-2 text-xs text-slate-600 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 rounded-lg">
              <span>Neueste zuerst</span>
              <ChevronDown className="w-3 h-3" />
            </button>
          </div>
        </div>
      </div>

      {/* RIGHT SIDE - CONTACT LIST OR DETAIL */}
      <div className="flex-1 flex flex-col bg-white dark:bg-slate-800">
        {currentView === 'list' && (
          <>
            {/* List Header */}
            <div className="h-12 bg-slate-50 dark:bg-slate-900 border-b border-slate-200 dark:border-slate-700 px-4 flex items-center justify-between">
              <div className="flex items-center gap-4">
                <input type="checkbox" className="rounded border-slate-300" />
                <span className="text-sm text-slate-900 dark:text-white font-medium">
                  {categories.find(c => c.id === activeCategory)?.name}
                </span>
                <span className="text-xs text-slate-500 dark:text-slate-400">
                  {filteredContacts.length} Kontakte total
                </span>
              </div>
              <div className="flex items-center gap-2">
                <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
                  <List className="w-5 h-5 text-emerald-600" />
                </button>
                <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
                  <Grid3x3 className="w-5 h-5 text-slate-400" />
                </button>
              </div>
            </div>

            {/* Toolbar */}
            {selectedContacts.length > 0 && (
              <div className="h-10 bg-white dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700 px-4 flex items-center gap-2">
                <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
                  <Edit className="w-4 h-4 text-slate-500" />
                </button>
                <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
                  <Trash2 className="w-4 h-4 text-slate-500" />
                </button>
                <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded flex items-center gap-1">
                  <Download className="w-4 h-4 text-slate-500" />
                  <ChevronDown className="w-3 h-3 text-slate-500" />
                </button>
                <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded flex items-center gap-1">
                  <FolderPlus className="w-4 h-4 text-slate-500" />
                  <ChevronDown className="w-3 h-3 text-slate-500" />
                </button>
              </div>
            )}

            {/* Contact List */}
            <div className="flex-1 overflow-y-auto">
              {filteredContacts.map((contact) => (
                <div
                  key={contact.id}
                  onClick={() => handleContactClick(contact)}
                  className={`flex items-center gap-3 p-4 border-b border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-700 cursor-pointer transition-colors ${
                    selectedContacts.includes(contact.id) ? 'bg-emerald-50 dark:bg-emerald-900/20 border-l-3 border-emerald-600' : ''
                  }`}
                >
                  <input
                    type="checkbox"
                    checked={selectedContacts.includes(contact.id)}
                    onChange={(e) => {
                      e.stopPropagation();
                      toggleContactSelection(contact.id);
                    }}
                    className="rounded border-slate-300"
                  />
                  <Avatar className="w-11 h-11 flex-shrink-0">
                    <AvatarFallback className="bg-emerald-100 dark:bg-emerald-900 text-emerald-700 dark:text-emerald-300">
                      {contact.avatar}
                    </AvatarFallback>
                  </Avatar>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-semibold text-slate-900 dark:text-white">
                      {contact.firstName} {contact.lastName}
                    </p>
                    <p className="text-xs text-slate-600 dark:text-slate-400 truncate">
                      {contact.jobTitle} bei {contact.company}
                    </p>
                    <p className="text-xs text-slate-400 truncate">
                      {contact.email}
                    </p>
                  </div>
                  <div className="flex items-center gap-2 mr-3">
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        showToast('Anruf wird gestartet...', 'info');
                      }}
                      className="p-2 hover:bg-emerald-50 dark:hover:bg-emerald-900/30 rounded-lg transition-colors"
                    >
                      <Phone className="w-4 h-4 text-slate-400 hover:text-emerald-600" />
                    </button>
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        showToast('Email-Editor wird geöffnet...', 'info');
                      }}
                      className="p-2 hover:bg-emerald-50 dark:hover:bg-emerald-900/30 rounded-lg transition-colors"
                    >
                      <Mail className="w-4 h-4 text-slate-400 hover:text-emerald-600" />
                    </button>
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        showToast('Chat wird geöffnet...', 'info');
                      }}
                      className="p-2 hover:bg-emerald-50 dark:hover:bg-emerald-900/30 rounded-lg transition-colors"
                    >
                      <MessageCircle className="w-4 h-4 text-slate-400 hover:text-emerald-600" />
                    </button>
                  </div>
                  <Badge className={`${getStatusColor(contact.status)} text-xs px-2 py-1`}>
                    {getStatusLabel(contact.status)}
                  </Badge>
                </div>
              ))}

              {filteredContacts.length === 0 && (
                <div className="flex flex-col items-center justify-center h-full p-8">
                  <Users className="w-12 h-12 text-slate-300 dark:text-slate-600 mb-4" />
                  <p className="text-slate-500 dark:text-slate-400 text-center">
                    {searchQuery ? 'Keine Kontakte gefunden' : 'Keine Kontakte in dieser Kategorie'}
                  </p>
                </div>
              )}
            </div>
          </>
        )}

        {currentView === 'detail' && selectedContact && (
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
                <span className="text-sm text-slate-600 dark:text-slate-400">Kontakte</span>
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => setShowNewContactModal(true)}
                  className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded"
                >
                  <Edit className="w-5 h-5 text-slate-500" />
                </button>
                <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
                  <Trash2 className="w-5 h-5 text-slate-500" />
                </button>
                <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
                  <MoreVertical className="w-5 h-5 text-slate-500" />
                </button>
              </div>
            </div>

            {/* Hero Section */}
            <div className="m-5 p-5 bg-gradient-to-br from-emerald-600 to-emerald-700 rounded-xl">
              <div className="flex items-center gap-5">
                <Avatar className="w-20 h-20 border-3 border-white">
                  <AvatarFallback className="bg-white text-emerald-600 text-2xl font-semibold">
                    {selectedContact.avatar}
                  </AvatarFallback>
                </Avatar>
                <div className="flex-1">
                  <h1 className="text-2xl font-semibold text-white mb-1">
                    {selectedContact.firstName} {selectedContact.lastName}
                  </h1>
                  <p className="text-white/90 text-sm mb-1">{selectedContact.jobTitle}</p>
                  <p className="text-white/85 text-sm">{selectedContact.company}</p>
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={() => showToast('Anruf wird gestartet...', 'info')}
                    className="px-4 py-2 bg-white/20 hover:bg-white/30 backdrop-blur text-white rounded-lg transition-colors flex items-center gap-2 text-sm"
                  >
                    <Phone className="w-4 h-4" />
                    Anrufen
                  </button>
                  <button
                    onClick={() => showToast('Email-Editor wird geöffnet...', 'info')}
                    className="px-4 py-2 bg-white/20 hover:bg-white/30 backdrop-blur text-white rounded-lg transition-colors flex items-center gap-2 text-sm"
                  >
                    <Mail className="w-4 h-4" />
                    Email
                  </button>
                  <button
                    onClick={() => showToast('Chat wird geöffnet...', 'info')}
                    className="px-4 py-2 bg-white/20 hover:bg-white/30 backdrop-blur text-white rounded-lg transition-colors flex items-center gap-2 text-sm"
                  >
                    <MessageCircle className="w-4 h-4" />
                    Nachricht
                  </button>
                </div>
              </div>
            </div>

            {/* Tabs */}
            <Tabs value={activeTab} onValueChange={setActiveTab} className="flex-1 flex flex-col">
              <TabsList className="w-full justify-start border-b border-slate-200 dark:border-slate-700 rounded-none h-auto p-0 bg-transparent">
                <TabsTrigger 
                  value="overview" 
                  className="rounded-none border-b-2 border-transparent data-[state=active]:border-emerald-600 data-[state=active]:text-emerald-600 px-5"
                >
                  Übersicht
                </TabsTrigger>
                <TabsTrigger 
                  value="projects"
                  className="rounded-none border-b-2 border-transparent data-[state=active]:border-emerald-600 data-[state=active]:text-emerald-600 px-5"
                >
                  Projekte
                </TabsTrigger>
                <TabsTrigger 
                  value="teams"
                  className="rounded-none border-b-2 border-transparent data-[state=active]:border-emerald-600 data-[state=active]:text-emerald-600 px-5"
                >
                  Teams
                </TabsTrigger>
                <TabsTrigger 
                  value="activity"
                  className="rounded-none border-b-2 border-transparent data-[state=active]:border-emerald-600 data-[state=active]:text-emerald-600 px-5"
                >
                  Aktivität
                </TabsTrigger>
                <TabsTrigger 
                  value="notes"
                  className="rounded-none border-b-2 border-transparent data-[state=active]:border-emerald-600 data-[state=active]:text-emerald-600 px-5"
                >
                  Notizen
                </TabsTrigger>
              </TabsList>

              <div className="flex-1 overflow-y-auto">
                <TabsContent value="overview" className="p-5 space-y-5 m-0">
                  {/* Contact Information */}
                  <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl p-5">
                    <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4">Kontaktinformationen</h3>
                    <div className="grid grid-cols-2 gap-6">
                      <div className="space-y-4">
                        <div>
                          <p className="text-xs font-semibold text-slate-500 dark:text-slate-400 mb-1">Name</p>
                          <p className="text-sm text-slate-900 dark:text-white">
                            {selectedContact.firstName} {selectedContact.lastName}
                          </p>
                        </div>
                        <div>
                          <p className="text-xs font-semibold text-slate-500 dark:text-slate-400 mb-1">Email</p>
                          <a href={`mailto:${selectedContact.email}`} className="text-sm text-emerald-600 hover:underline">
                            {selectedContact.email}
                          </a>
                        </div>
                        <div>
                          <p className="text-xs font-semibold text-slate-500 dark:text-slate-400 mb-1">Telefon</p>
                          <a href={`tel:${selectedContact.phone}`} className="text-sm text-emerald-600 hover:underline flex items-center gap-2">
                            <Phone className="w-3 h-3" />
                            {selectedContact.phone}
                          </a>
                        </div>
                        <div>
                          <p className="text-xs font-semibold text-slate-500 dark:text-slate-400 mb-1">Adresse</p>
                          <p className="text-sm text-slate-900 dark:text-white">
                            {selectedContact.address}<br />
                            {selectedContact.zipCode} {selectedContact.city}<br />
                            {selectedContact.country}
                          </p>
                        </div>
                      </div>
                      <div className="space-y-4">
                        <div>
                          <p className="text-xs font-semibold text-slate-500 dark:text-slate-400 mb-1">Firma</p>
                          <p className="text-sm text-emerald-600 cursor-pointer hover:underline">
                            {selectedContact.company}
                          </p>
                        </div>
                        <div>
                          <p className="text-xs font-semibold text-slate-500 dark:text-slate-400 mb-1">Position</p>
                          <p className="text-sm text-slate-900 dark:text-white">{selectedContact.jobTitle}</p>
                        </div>
                        <div>
                          <p className="text-xs font-semibold text-slate-500 dark:text-slate-400 mb-1">Status</p>
                          <Badge className={`${getStatusColor(selectedContact.status)} text-xs px-2 py-1`}>
                            {getStatusLabel(selectedContact.status)}
                          </Badge>
                        </div>
                        <div>
                          <p className="text-xs font-semibold text-slate-500 dark:text-slate-400 mb-1">Erstellt am</p>
                          <p className="text-sm text-slate-900 dark:text-white">{selectedContact.createdDate}</p>
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* Communication History */}
                  <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl p-5">
                    <h3 className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase mb-3">
                      Letzte Kommunikation
                    </h3>
                    <div className="space-y-2">
                      <div className="flex items-center justify-between text-xs">
                        <span className="text-slate-600 dark:text-slate-300">Email gesendet: Website Relaunch Update</span>
                        <span className="text-slate-400">Vor 2 Tagen</span>
                      </div>
                      <div className="flex items-center justify-between text-xs">
                        <span className="text-slate-600 dark:text-slate-300">Anruf: Sprint Planning Discussion</span>
                        <span className="text-slate-400">Vor 5 Tagen</span>
                      </div>
                      <div className="flex items-center justify-between text-xs">
                        <span className="text-slate-600 dark:text-slate-300">Nachricht: Code Review Request</span>
                        <span className="text-slate-400">Vor 1 Woche</span>
                      </div>
                    </div>
                    <button className="text-xs text-emerald-600 hover:underline mt-3">
                      Vollständige Aktivität ansehen
                    </button>
                  </div>
                </TabsContent>

                <TabsContent value="projects" className="p-5 m-0">
                  <div className="mb-4">
                    <h3 className="text-sm font-semibold text-slate-900 dark:text-white">
                      Zugeordnete Projekte <span className="text-slate-400">({selectedContact.projects.length})</span>
                    </h3>
                  </div>
                  <div className="grid gap-4">
                    {projectsList.filter(p => selectedContact.projects.includes(p.name)).map((project) => (
                      <div
                        key={project.id}
                        className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl p-4"
                      >
                        <div className="flex items-start justify-between mb-3">
                          <div>
                            <h4 className="text-sm font-semibold text-slate-900 dark:text-white mb-1">
                              {project.name}
                            </h4>
                            <div className="flex items-center gap-2 mb-2">
                              <div className={`w-2 h-2 rounded-full ${
                                project.status === 'In Progress' ? 'bg-blue-500' : 
                                project.status === 'Planning' ? 'bg-orange-500' : 'bg-green-500'
                              }`} />
                              <span className="text-xs text-slate-600 dark:text-slate-400">{project.status}</span>
                            </div>
                            <p className="text-xs text-slate-500">Rolle: {project.role}</p>
                          </div>
                          <button className="px-3 py-1.5 bg-slate-100 dark:bg-slate-700 hover:bg-slate-200 dark:hover:bg-slate-600 text-slate-700 dark:text-slate-300 rounded-lg text-xs transition-colors">
                            Zum Projekt
                          </button>
                        </div>
                        <div className="flex items-center gap-2 text-xs text-slate-500">
                          <Users className="w-3 h-3" />
                          <span>Team: {project.members} Personen</span>
                        </div>
                      </div>
                    ))}
                  </div>
                </TabsContent>

                <TabsContent value="teams" className="p-5 m-0">
                  <div className="mb-4">
                    <h3 className="text-sm font-semibold text-slate-900 dark:text-white">
                      Mitgliedschaften in Teams <span className="text-slate-400">({selectedContact.teams.length})</span>
                    </h3>
                  </div>
                  <div className="space-y-3">
                    {teamsList.filter(t => selectedContact.teams.includes(t.name)).map((team) => (
                      <div
                        key={team.id}
                        className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl p-4 flex items-center justify-between"
                      >
                        <div className="flex items-center gap-3">
                          <div className="w-10 h-10 bg-emerald-100 dark:bg-emerald-900 rounded-lg flex items-center justify-center">
                            <Users className="w-5 h-5 text-emerald-600" />
                          </div>
                          <div>
                            <h4 className="text-sm font-semibold text-slate-900 dark:text-white">{team.name}</h4>
                            <p className="text-xs text-slate-500">{team.description}</p>
                            <p className="text-xs text-slate-400 mt-0.5">{team.members} Mitglieder</p>
                          </div>
                        </div>
                        <div className="flex items-center gap-3">
                          <Badge className="bg-emerald-50 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-300 text-xs px-2 py-1">
                            {selectedContact.jobTitle.includes('Manager') ? 'Manager' : 'Member'}
                          </Badge>
                          <button className="px-3 py-1.5 bg-slate-100 dark:bg-slate-700 hover:bg-slate-200 dark:hover:bg-slate-600 text-slate-700 dark:text-slate-300 rounded-lg text-xs transition-colors">
                            Details
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                </TabsContent>

                <TabsContent value="activity" className="p-5 m-0">
                  <div className="mb-4">
                    <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-3">Aktivitätsverlauf</h3>
                    <div className="flex gap-2">
                      <button className="px-3 py-1.5 bg-emerald-50 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-300 rounded-lg text-xs">
                        Alle
                      </button>
                      <button className="px-3 py-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-600 dark:text-slate-300 rounded-lg text-xs">
                        Emails
                      </button>
                      <button className="px-3 py-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-600 dark:text-slate-300 rounded-lg text-xs">
                        Anrufe
                      </button>
                      <button className="px-3 py-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-600 dark:text-slate-300 rounded-lg text-xs">
                        Nachrichten
                      </button>
                    </div>
                  </div>
                  <div className="space-y-4 border-l-2 border-slate-200 dark:border-slate-700 pl-4">
                    <div className="relative">
                      <div className="absolute -left-[21px] top-1 w-3 h-3 bg-emerald-600 rounded-full border-2 border-white dark:border-slate-800" />
                      <div className="flex items-start gap-2">
                        <Mail className="w-4 h-4 text-slate-400 mt-0.5" />
                        <div className="flex-1">
                          <p className="text-xs font-medium text-slate-900 dark:text-white">Email gesendet</p>
                          <p className="text-xs text-slate-500">Website Relaunch Update</p>
                          <p className="text-xs text-slate-400 mt-1">2 Dezember 2024, 14:30</p>
                        </div>
                        <button className="text-xs text-emerald-600 hover:underline">Ansehen</button>
                      </div>
                    </div>
                    <div className="relative">
                      <div className="absolute -left-[21px] top-1 w-3 h-3 bg-emerald-600 rounded-full border-2 border-white dark:border-slate-800" />
                      <div className="flex items-start gap-2">
                        <Phone className="w-4 h-4 text-slate-400 mt-0.5" />
                        <div className="flex-1">
                          <p className="text-xs font-medium text-slate-900 dark:text-white">Anruf getätigt</p>
                          <p className="text-xs text-slate-500">Sprint Planning Diskussion</p>
                          <p className="text-xs text-slate-400 mt-1">29 November 2024, 09:15 • 45 min</p>
                        </div>
                        <button className="text-xs text-emerald-600 hover:underline">Notizen</button>
                      </div>
                    </div>
                    <div className="relative">
                      <div className="absolute -left-[21px] top-1 w-3 h-3 bg-emerald-600 rounded-full border-2 border-white dark:border-slate-800" />
                      <div className="flex items-start gap-2">
                        <MessageCircle className="w-4 h-4 text-slate-400 mt-0.5" />
                        <div className="flex-1">
                          <p className="text-xs font-medium text-slate-900 dark:text-white">Nachricht gesendet</p>
                          <p className="text-xs text-slate-500">Code Review Request</p>
                          <p className="text-xs text-slate-400 mt-1">25 November 2024, 16:45</p>
                        </div>
                        <button className="text-xs text-emerald-600 hover:underline">Ansehen</button>
                      </div>
                    </div>
                  </div>
                </TabsContent>

                <TabsContent value="notes" className="p-5 m-0">
                  <div className="mb-4 flex items-center justify-between">
                    <h3 className="text-sm font-semibold text-slate-900 dark:text-white">Notizen zu diesem Kontakt</h3>
                    <button className="px-3 py-1.5 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg text-xs flex items-center gap-1 transition-colors">
                      <Plus className="w-3 h-3" />
                      Neue Notiz
                    </button>
                  </div>
                  <div className="space-y-3">
                    <div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-xl p-3">
                      <div className="flex items-center justify-between mb-2">
                        <span className="text-xs font-semibold text-slate-900 dark:text-white">Max Müller</span>
                        <span className="text-xs text-slate-400">vor 3 Tagen</span>
                      </div>
                      <p className="text-xs text-slate-700 dark:text-slate-300 leading-relaxed">
                        Möchte gerne über die Budget-Freigabe für Q1 2025 sprechen. Bitte Termin vereinbaren.
                      </p>
                    </div>
                    <div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-xl p-3">
                      <div className="flex items-center justify-between mb-2">
                        <span className="text-xs font-semibold text-slate-900 dark:text-white">Anna Müller</span>
                        <span className="text-xs text-slate-400">vor 1 Woche</span>
                      </div>
                      <p className="text-xs text-slate-700 dark:text-slate-300 leading-relaxed">
                        Sehr kooperativ und technisch versiert. Empfehlung für Lead-Position im nächsten Projekt.
                      </p>
                    </div>
                  </div>
                </TabsContent>
              </div>
            </Tabs>
          </>
        )}
      </div>

      {/* New/Edit Contact Modal */}
      <Dialog open={showNewContactModal} onOpenChange={setShowNewContactModal}>
        <DialogContent className="max-w-3xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Neuer Kontakt</DialogTitle>
            <DialogDescription>
              Füge einen neuen Kontakt zu deinem CRM hinzu
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-6 py-4">
            {/* Basic Info */}
            <div>
              <h4 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-3 uppercase">
                Grundinformationen
              </h4>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label>Vorname *</Label>
                  <Input placeholder="z.B. Achim" />
                </div>
                <div>
                  <Label>Nachname *</Label>
                  <Input placeholder="z.B. Weber" />
                </div>
                <div>
                  <Label>Email</Label>
                  <Input type="email" placeholder="z.B. achim.weber@company.ch" />
                </div>
                <div>
                  <Label>Telefon</Label>
                  <Input placeholder="z.B. +41 76 123 45 67" />
                </div>
              </div>
            </div>

            {/* Company Info */}
            <div>
              <h4 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-3 uppercase">
                Firmeninformationen
              </h4>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label>Firma</Label>
                  <Input placeholder="Wähle oder gib neue Firma ein" />
                </div>
                <div>
                  <Label>Position</Label>
                  <Input placeholder="z.B. Senior Developer" />
                </div>
                <div className="col-span-2">
                  <Label>Abteilung</Label>
                  <Input placeholder="z.B. Entwicklung" />
                </div>
              </div>
            </div>

            {/* Address */}
            <div>
              <h4 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-3 uppercase">
                Adresse
              </h4>
              <div className="space-y-4">
                <div>
                  <Label>Strasse & Nummer</Label>
                  <Input placeholder="z.B. Bahnhofstrasse 42" />
                </div>
                <div className="grid grid-cols-3 gap-4">
                  <div>
                    <Label>PLZ</Label>
                    <Input placeholder="8000" />
                  </div>
                  <div className="col-span-2">
                    <Label>Ort</Label>
                    <Input placeholder="Zürich" />
                  </div>
                </div>
              </div>
            </div>

            {/* Categorization */}
            <div>
              <h4 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-3 uppercase">
                Kategorisierung
              </h4>
              <div className="space-y-4">
                <div>
                  <Label>Kontakt-Typ</Label>
                  <div className="space-y-2 mt-2">
                    <label className="flex items-center gap-2">
                      <input type="radio" name="category" value="employee" className="rounded-full" />
                      <span className="text-sm text-slate-700 dark:text-slate-300">Mitarbeiter</span>
                    </label>
                    <label className="flex items-center gap-2">
                      <input type="radio" name="category" value="customer" className="rounded-full" />
                      <span className="text-sm text-slate-700 dark:text-slate-300">Kunde</span>
                    </label>
                    <label className="flex items-center gap-2">
                      <input type="radio" name="category" value="partner" className="rounded-full" />
                      <span className="text-sm text-slate-700 dark:text-slate-300">Partner & Lieferant</span>
                    </label>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div className="flex justify-end gap-3 pt-4 border-t border-slate-200 dark:border-slate-700">
            <button
              onClick={() => setShowNewContactModal(false)}
              className="px-4 py-2 bg-slate-100 dark:bg-slate-700 hover:bg-slate-200 dark:hover:bg-slate-600 text-slate-700 dark:text-slate-300 rounded-lg transition-colors"
            >
              Abbrechen
            </button>
            <button
              onClick={() => {
                showToast('✅ Kontakt gespeichert', 'success');
                setShowNewContactModal(false);
              }}
              className="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg transition-colors"
            >
              Kontakt speichern
            </button>
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
