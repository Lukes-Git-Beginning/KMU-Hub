import { useState, useEffect } from 'react';
import { 
  Search, Plus, Folder, FolderOpen, File, FileText, Image, Video, Archive, 
  MoreVertical, ChevronDown, Star, Grid3x3, List, Filter, Download, Share2,
  Trash2, Upload, X, Check, HelpCircle, Settings, Users, ChevronRight,
  Copy, Clock, AlertTriangle, Tag, Move, Eye, ChevronLeft, Lock, Shield,
  BarChart3, User, EyeOff
} from 'lucide-react';
import { Avatar, AvatarFallback } from '../components/ui/avatar';
import { Badge } from '../components/ui/badge';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '../components/ui/dialog';
import { Label } from '../components/ui/label';
import { Input } from '../components/ui/input';
import { Textarea } from '../components/ui/textarea';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../components/ui/tabs';
import { Progress } from '../components/ui/progress';
import { VaultSettings } from '../components/VaultSettings';

// Mock Data - Extended with 20+ documents
const documents = [
  {
    id: 1,
    name: 'Projektplan 2024.pdf',
    type: 'pdf',
    size: '2.4 MB',
    sizeBytes: 2400000,
    date: '2024-01-20',
    folder: 'Projekte/Website Relaunch',
    tags: ['Projekt', '2024'],
    createdBy: 'Achim Weber',
    isFavorite: true,
    isShared: true,
    sharedWith: ['Lisa Meier', 'Max Müller'],
    version: 3,
    lastModified: new Date('2024-12-18'),
    isVault: false,
  },
  {
    id: 2,
    name: 'Team Foto.jpg',
    type: 'image',
    size: '1.8 MB',
    sizeBytes: 1800000,
    date: '2024-01-19',
    folder: 'Team Dokumente/Entwicklung',
    tags: ['Team'],
    createdBy: 'Sarah Schmidt',
    isFavorite: false,
    isShared: false,
    sharedWith: [],
    version: 1,
    lastModified: new Date('2024-12-19'),
    thumbnail: 'https://images.unsplash.com/photo-1522071820081-009f0129c71c?w=400&h=300&fit=crop',
    isVault: false,
  },
  {
    id: 3,
    name: 'Präsentation Q1.pptx',
    type: 'powerpoint',
    size: '5.2 MB',
    sizeBytes: 5200000,
    date: '2024-01-18',
    folder: 'Vorlagen',
    tags: ['Präsentation', 'Q1', 'Wichtig'],
    createdBy: 'Michael Berg',
    isFavorite: true,
    isShared: true,
    sharedWith: ['Achim Weber', 'Lisa Meier', 'Sarah Schmidt'],
    version: 2,
    lastModified: new Date('2024-12-17'),
    isVault: false,
  },
  {
    id: 4,
    name: 'Budget Übersicht.xlsx',
    type: 'excel',
    size: '892 KB',
    sizeBytes: 892000,
    date: '2024-01-17',
    folder: 'Projekte/Website Relaunch',
    tags: ['Budget', 'Finanzen'],
    createdBy: 'Lisa Meier',
    isFavorite: true,
    isShared: true,
    sharedWith: ['Max Müller'],
    version: 4,
    lastModified: new Date('2024-12-16'),
    isVault: false,
  },
  {
    id: 5,
    name: 'Strategie Dokument.docx',
    type: 'word',
    size: '1.2 MB',
    sizeBytes: 1200000,
    date: '2024-01-16',
    folder: 'Team Dokumente/Marketing',
    tags: ['Strategie', '2025'],
    createdBy: 'Achim Weber',
    isFavorite: false,
    isShared: false,
    sharedWith: [],
    version: 1,
    lastModified: new Date('2024-12-15'),
    isVault: false,
  },
  {
    id: 6,
    name: 'Logo SwissBiz.svg',
    type: 'image',
    size: '45 KB',
    sizeBytes: 45000,
    date: '2024-01-15',
    folder: 'Team Dokumente/Marketing',
    tags: ['Brand', 'Logo'],
    createdBy: 'Sarah Schmidt',
    isFavorite: false,
    isShared: true,
    sharedWith: ['Team Marketing'],
    version: 2,
    lastModified: new Date('2024-12-14'),
    isVault: false,
  },
  {
    id: 7,
    name: 'Vertragsvorlage.docx',
    type: 'word',
    size: '680 KB',
    sizeBytes: 680000,
    date: '2024-01-14',
    folder: 'Vorlagen',
    tags: ['Vertrag', 'Vorlage'],
    createdBy: 'Max Müller',
    isFavorite: true,
    isShared: true,
    sharedWith: ['Alle'],
    version: 1,
    lastModified: new Date('2024-12-13'),
    isVault: false,
  },
  {
    id: 8,
    name: 'Meeting Recording Q1.mp4',
    type: 'video',
    size: '245 MB',
    sizeBytes: 245000000,
    date: '2024-01-13',
    folder: 'Team Dokumente/Entwicklung',
    tags: ['Meeting', 'Q1', 'Video'],
    createdBy: 'Michael Berg',
    isFavorite: false,
    isShared: true,
    sharedWith: ['Team Entwicklung'],
    version: 1,
    lastModified: new Date('2024-12-12'),
    isVault: false,
  },
  {
    id: 9,
    name: 'Anforderungen API.pdf',
    type: 'pdf',
    size: '3.1 MB',
    sizeBytes: 3100000,
    date: '2024-01-12',
    folder: 'Projekte/API Integration',
    tags: ['API', 'Spezifikation'],
    createdBy: 'Thomas Klein',
    isFavorite: false,
    isShared: true,
    sharedWith: ['Achim Weber', 'Julia Meier'],
    version: 2,
    lastModified: new Date('2024-12-11'),
    isVault: false,
  },
  {
    id: 10,
    name: 'Backup Archive.zip',
    type: 'archive',
    size: '125 MB',
    sizeBytes: 125000000,
    date: '2024-01-11',
    folder: 'Archiviert',
    tags: ['Backup', 'Archiv'],
    createdBy: 'System',
    isFavorite: false,
    isShared: false,
    sharedWith: [],
    version: 1,
    lastModified: new Date('2024-12-10'),
    isVault: false,
  },
  // VAULT DOCUMENTS (Verschlüsselt)
  {
    id: 11,
    name: 'Arbeitsvertrag_Weber.pdf',
    type: 'pdf',
    size: '1.5 MB',
    sizeBytes: 1500000,
    date: '2024-01-09',
    folder: 'Datentresor/Verträge/Arbeitsverträge',
    tags: ['Vertraulich', 'Personal'],
    createdBy: 'HR System',
    isFavorite: false,
    isShared: false,
    sharedWith: [],
    version: 1,
    lastModified: new Date('2024-12-09'),
    isVault: true,
    encryption: 'AES-256-GCM',
  },
  {
    id: 12,
    name: 'Kundenvertrag_MüllerAG.pdf',
    type: 'pdf',
    size: '2.1 MB',
    sizeBytes: 2100000,
    date: '2024-01-08',
    folder: 'Datentresor/Verträge/Kundenverträge',
    tags: ['Vertrag', 'Kunde', 'Geheim'],
    createdBy: 'Max Müller',
    isFavorite: true,
    isShared: false,
    sharedWith: [],
    version: 2,
    lastModified: new Date('2024-12-08'),
    isVault: true,
    encryption: 'AES-256-GCM',
  },
  {
    id: 13,
    name: 'Steuererklärung_2024.pdf',
    type: 'pdf',
    size: '3.4 MB',
    sizeBytes: 3400000,
    date: '2024-01-07',
    folder: 'Datentresor/Finanzen/Steuererklärungen',
    tags: ['Steuer', 'Finanzen', 'Vertraulich'],
    createdBy: 'Buchhaltung',
    isFavorite: false,
    isShared: false,
    sharedWith: [],
    version: 1,
    lastModified: new Date('2024-12-07'),
    isVault: true,
    encryption: 'AES-256-GCM',
  },
  {
    id: 14,
    name: 'Gehaltsliste_Q1_2024.xlsx',
    type: 'excel',
    size: '456 KB',
    sizeBytes: 456000,
    date: '2024-01-06',
    folder: 'Datentresor/Persönliche Daten/Mitarbeiterdaten',
    tags: ['Personal', 'Gehalt', 'Geheim'],
    createdBy: 'HR System',
    isFavorite: false,
    isShared: false,
    sharedWith: [],
    version: 3,
    lastModified: new Date('2024-12-06'),
    isVault: true,
    encryption: 'AES-256-GCM',
  },
  {
    id: 15,
    name: 'Produktentwicklung_Geheim.docx',
    type: 'word',
    size: '2.8 MB',
    sizeBytes: 2800000,
    date: '2024-01-05',
    folder: 'Datentresor/Geheime Projekte/Produktentwicklung',
    tags: ['Geheim', 'Produkt', 'Strategie'],
    createdBy: 'Achim Weber',
    isFavorite: true,
    isShared: false,
    sharedWith: [],
    version: 5,
    lastModified: new Date('2024-12-05'),
    isVault: true,
    encryption: 'AES-256-GCM',
  },
  // Additional regular documents
  {
    id: 16,
    name: 'Wireframes Mobile App.fig',
    type: 'other',
    size: '12.3 MB',
    sizeBytes: 12300000,
    date: '2024-01-04',
    folder: 'Projekte/Mobile App Dev',
    tags: ['Design', 'Mobile', 'Wireframe'],
    createdBy: 'Sarah Schmidt',
    isFavorite: false,
    isShared: true,
    sharedWith: ['Design Team'],
    version: 8,
    lastModified: new Date('2024-12-04'),
    isVault: false,
  },
  {
    id: 17,
    name: 'API Dokumentation.pdf',
    type: 'pdf',
    size: '4.2 MB',
    sizeBytes: 4200000,
    date: '2024-01-03',
    folder: 'Projekte/API Integration',
    tags: ['API', 'Dokumentation', 'Tech'],
    createdBy: 'Thomas Klein',
    isFavorite: false,
    isShared: true,
    sharedWith: ['Entwicklung'],
    version: 1,
    lastModified: new Date('2024-12-03'),
    isVault: false,
  },
  {
    id: 18,
    name: 'Marketing Strategie 2025.pptx',
    type: 'powerpoint',
    size: '8.9 MB',
    sizeBytes: 8900000,
    date: '2024-01-02',
    folder: 'Team Dokumente/Marketing',
    tags: ['Marketing', 'Strategie', '2025'],
    createdBy: 'Lisa Meier',
    isFavorite: true,
    isShared: true,
    sharedWith: ['Marketing Team', 'Geschäftsführung'],
    version: 3,
    lastModified: new Date('2024-12-02'),
    isVault: false,
  },
  {
    id: 19,
    name: 'Code Repository Backup.zip',
    type: 'archive',
    size: '450 MB',
    sizeBytes: 450000000,
    date: '2024-01-01',
    folder: 'Archiviert',
    tags: ['Backup', 'Code', 'Archiv'],
    createdBy: 'System',
    isFavorite: false,
    isShared: false,
    sharedWith: [],
    version: 1,
    lastModified: new Date('2024-12-01'),
    isVault: false,
  },
  {
    id: 20,
    name: 'Onboarding Checkliste.docx',
    type: 'word',
    size: '234 KB',
    sizeBytes: 234000,
    date: '2023-12-31',
    folder: 'Vorlagen',
    tags: ['HR', 'Onboarding', 'Vorlage'],
    createdBy: 'HR System',
    isFavorite: false,
    isShared: true,
    sharedWith: ['Alle'],
    version: 2,
    lastModified: new Date('2023-12-31'),
    isVault: false,
  },
];

// Folder Structure - Extended with Vault
const folderStructure = [
  {
    id: 'all',
    name: 'Alle Dokumente',
    icon: FolderOpen,
    count: 156,
    isExpanded: true,
    children: [],
    isVault: false,
  },
  {
    id: 'vault',
    name: 'Datentresor',
    icon: Shield,
    count: 41,
    isExpanded: false,
    isVault: true,
    children: [
      { id: 'vault-contracts', name: 'Verträge & Vereinbarungen', icon: FileText, count: 12, isVault: true,
        children: [
          { id: 'vault-contracts-work', name: 'Arbeitsverträge', count: 5, isVault: true },
          { id: 'vault-contracts-customer', name: 'Kundenverträge', count: 5, isVault: true },
          { id: 'vault-contracts-supplier', name: 'Lieferantenverträge', count: 2, isVault: true },
        ]
      },
      { id: 'vault-finance', name: 'Finanzielle Dokumente', icon: BarChart3, count: 8, isVault: true,
        children: [
          { id: 'vault-finance-invoices', name: 'Rechnungen', count: 3, isVault: true },
          { id: 'vault-finance-tax', name: 'Steuererklärungen', count: 3, isVault: true },
          { id: 'vault-finance-bank', name: 'Bankdokumente', count: 2, isVault: true },
        ]
      },
      { id: 'vault-personal', name: 'Persönliche Daten', icon: User, count: 15, isVault: true,
        children: [
          { id: 'vault-personal-employee', name: 'Mitarbeiterdaten', count: 8, isVault: true },
          { id: 'vault-personal-customer', name: 'Kundendaten', count: 5, isVault: true },
          { id: 'vault-personal-reviews', name: 'Kundenbewertungen', count: 2, isVault: true },
        ]
      },
      { id: 'vault-secret', name: 'Geheime Projekte', icon: EyeOff, count: 6, isVault: true, isHighSecurity: true,
        children: [
          { id: 'vault-secret-product', name: 'Produktentwicklung (Geheim)', count: 3, isVault: true },
          { id: 'vault-secret-strategy', name: 'Strategische Pläne', count: 2, isVault: true },
          { id: 'vault-secret-ma', name: 'M&A Dokumente', count: 1, isVault: true },
        ]
      },
    ],
  },
  {
    id: 'projekte',
    name: 'Projekte',
    icon: Folder,
    count: 24,
    isExpanded: false,
    isVault: false,
    children: [
      { id: 'projekte-website', name: 'Website Relaunch', count: 8, isVault: false },
      { id: 'projekte-mobile', name: 'Mobile App Dev', count: 12, isVault: false },
      { id: 'projekte-api', name: 'API Integration', count: 4, isVault: false },
    ],
  },
  {
    id: 'team',
    name: 'Team Dokumente',
    icon: Users,
    count: 12,
    isExpanded: false,
    isVault: false,
    children: [
      { id: 'team-dev', name: 'Entwicklung', count: 5, isVault: false },
      { id: 'team-marketing', name: 'Marketing', count: 4, isVault: false },
      { id: 'team-sales', name: 'Vertrieb', count: 3, isVault: false },
    ],
  },
  {
    id: 'vorlagen',
    name: 'Vorlagen',
    icon: FileText,
    count: 8,
    isExpanded: false,
    isVault: false,
    children: [],
  },
  {
    id: 'archiviert',
    name: 'Archiviert',
    icon: Archive,
    count: 23,
    isExpanded: false,
    isVault: false,
    children: [],
  },
];

type ViewType = 'grid' | 'list';

export function Dokumente() {
  const [view, setView] = useState<ViewType>('grid');
  const [selectedFolder, setSelectedFolder] = useState('all');
  const [selectedDocuments, setSelectedDocuments] = useState<number[]>([]);
  const [selectedDocument, setSelectedDocument] = useState<any>(null);
  const [showDetailSidebar, setShowDetailSidebar] = useState(false);
  const [showUploadModal, setShowUploadModal] = useState(false);
  const [showShareModal, setShowShareModal] = useState(false);
  const [showFilters, setShowFilters] = useState(false);
  const [showFavorites, setShowFavorites] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [expandedFolders, setExpandedFolders] = useState<string[]>(['all']);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(() => {
    // Initialize from localStorage or default based on screen size
    const saved = localStorage.getItem('dokumente-sidebar-collapsed');
    if (saved !== null) return saved === 'true';
    return window.innerWidth < 768; // Auto-collapse on mobile
  });
  const [showVaultSettings, setShowVaultSettings] = useState(false);
  const [toast, setToast] = useState<{ show: boolean; message: string; type: 'success' | 'error' | 'info' }>({ 
    show: false, 
    message: '', 
    type: 'success' 
  });

  // Persist sidebar state to localStorage
  useEffect(() => {
    localStorage.setItem('dokumente-sidebar-collapsed', String(sidebarCollapsed));
  }, [sidebarCollapsed]);

  // Handle responsive behavior
  useEffect(() => {
    const handleResize = () => {
      if (window.innerWidth < 768) {
        setSidebarCollapsed(true); // Auto-collapse on mobile
      }
    };
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  const showToast = (message: string, type: 'success' | 'error' | 'info' = 'success') => {
    setToast({ show: true, message, type });
    setTimeout(() => setToast({ show: false, message: '', type: 'success' }), 3000);
  };

  const toggleFolder = (folderId: string) => {
    setExpandedFolders(prev =>
      prev.includes(folderId) 
        ? prev.filter(id => id !== folderId)
        : [...prev, folderId]
    );
  };

  const toggleDocumentSelection = (id: number) => {
    setSelectedDocuments(prev =>
      prev.includes(id) ? prev.filter(did => did !== id) : [...prev, id]
    );
  };

  const handleDocumentClick = (doc: any) => {
    setSelectedDocument(doc);
    setShowDetailSidebar(true);
  };

  const getFileTypeIcon = (type: string, size: number = 48) => {
    const iconClass = `w-${size === 48 ? '12' : size === 20 ? '5' : '4'} h-${size === 48 ? '12' : size === 20 ? '5' : '4'}`;
    
    switch (type) {
      case 'pdf':
        return <FileText className={`${iconClass} text-white`} />;
      case 'word':
        return <FileText className={`${iconClass} text-white`} />;
      case 'excel':
        return <FileText className={`${iconClass} text-white`} />;
      case 'powerpoint':
        return <FileText className={`${iconClass} text-white`} />;
      case 'image':
        return <Image className={`${iconClass} text-white`} />;
      case 'video':
        return <Video className={`${iconClass} text-white`} />;
      case 'archive':
        return <Archive className={`${iconClass} text-white`} />;
      default:
        return <File className={`${iconClass} text-white`} />;
    }
  };

  const getFileTypeColor = (type: string) => {
    // Using warm, sophisticated colors from Design System 2.0
    switch (type) {
      case 'pdf': return 'bg-[#8b6f47] dark:bg-[#8b6f47]'; // Warm brown
      case 'word': return 'bg-[#4a6b8a] dark:bg-[#4a6b8a]'; // Warm slate blue
      case 'excel': return 'bg-[#5a7a4d] dark:bg-[#5a7a4d]'; // Warm moss green
      case 'powerpoint': return 'bg-[#a67c52] dark:bg-[#a67c52]'; // Warm terracotta
      case 'image': return 'bg-[#7a6b5a] dark:bg-[#7a6b5a]'; // Warm taupe
      case 'video': return 'bg-[#5a6b7a] dark:bg-[#5a6b7a]'; // Warm slate
      case 'archive': return 'bg-[#6a5a7a] dark:bg-[#6a5a7a]'; // Warm mauve
      default: return 'bg-[#7a7a7a] dark:bg-[#7a7a7a]'; // Warm gray
    }
  };

  const filteredDocuments = documents.filter(doc => {
    // Vault filter
    if (selectedFolder === 'vault' || selectedFolder.startsWith('vault-')) {
      if (!doc.isVault) return false;
      if (selectedFolder !== 'vault') {
        const folderPath = doc.folder.toLowerCase();
        const selectedPath = selectedFolder.replace('vault-', '').replace('-', ' ');
        if (!folderPath.includes(selectedPath)) return false;
      }
    } else if (selectedFolder !== 'all') {
      // Regular folder filter
      if (doc.isVault) return false;
    }

    // Search filter
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      return (
        doc.name.toLowerCase().includes(query) ||
        doc.tags.some(tag => tag.toLowerCase().includes(query)) ||
        doc.createdBy.toLowerCase().includes(query)
      );
    }
    
    return true;
  });

  const favoriteDocuments = documents.filter(doc => doc.isFavorite).slice(0, 3);

  const renderFolderItem = (folder: any, level: number = 0) => {
    const Icon = folder.icon || Folder;
    const isExpanded = expandedFolders.includes(folder.id);
    const isActive = selectedFolder === folder.id;
    const hasChildren = folder.children && folder.children.length > 0;

    return (
      <div key={folder.id}>
        <button
          onClick={() => {
            setSelectedFolder(folder.id);
            if (hasChildren) {
              toggleFolder(folder.id);
            }
          }}
          title={sidebarCollapsed ? folder.name : undefined}
          className={`w-full flex items-center justify-between px-2 py-2 rounded-lg transition-colors text-sm ${
            isActive
              ? 'bg-emerald-50 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-300 border-l-3 border-emerald-600'
              : folder.isVault
              ? 'text-green-600 dark:text-green-400 hover:bg-green-50 dark:hover:bg-green-900/20'
              : 'text-slate-600 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700'
          }`}
          style={{ paddingLeft: sidebarCollapsed ? '8px' : `${8 + level * 16}px` }}
        >
          <div className="flex items-center gap-2">
            <Icon className={`w-4 h-4 ${folder.isHighSecurity ? 'text-red-500' : ''}`} />
            {!sidebarCollapsed && <span>{folder.name}</span>}
            {folder.isVault && !sidebarCollapsed && (
              <Lock className="w-3 h-3 text-green-600 dark:text-green-400" />
            )}
          </div>
          {!sidebarCollapsed && (
            <div className="flex items-center gap-2">
              <span className="text-xs text-slate-400">({folder.count})</span>
              {hasChildren && (
                <ChevronDown className={`w-3 h-3 transition-transform ${isExpanded ? '' : '-rotate-90'}`} />
              )}
            </div>
          )}
        </button>
        
        {/* Subfolders */}
        {!sidebarCollapsed && isExpanded && hasChildren && (
          <div className="mt-0.5 space-y-0.5">
            {folder.children.map((child: any) => renderFolderItem(child, level + 1))}
          </div>
        )}
      </div>
    );
  };

  return (
    <div className="flex h-full bg-slate-50 dark:bg-slate-900">
      {/* LEFT SIDEBAR - NAVIGATION & FOLDERS */}
      <div className={`border-r border-slate-200 dark:border-slate-700 flex flex-col bg-white dark:bg-slate-800 transition-all duration-300 ${
        sidebarCollapsed ? 'w-16' : 'w-60'
      }`}>
        {/* Header */}
        <div className="p-4 border-b border-slate-200 dark:border-slate-700 relative">
          {!sidebarCollapsed && (
            <>
              <h2 className="text-slate-900 dark:text-white mb-1">Dokumente</h2>
              <p className="text-xs text-slate-500 dark:text-slate-400 mb-4">DMS & Dateiverwaltung</p>
            </>
          )}
          
          {/* Collapse/Expand Toggle */}
          <button
            onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
            className={`absolute ${sidebarCollapsed ? 'top-4 left-1/2 -translate-x-1/2' : 'top-4 right-4'} p-2 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg transition-colors`}
            title={sidebarCollapsed ? 'Sidebar erweitern' : 'Sidebar verkleinern'}
          >
            {sidebarCollapsed ? (
              <ChevronRight className="w-4 h-4 text-emerald-600" />
            ) : (
              <ChevronLeft className="w-4 h-4 text-emerald-600" />
            )}
          </button>

          {!sidebarCollapsed && (
            <button
              onClick={() => setShowUploadModal(true)}
              className="w-full px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg transition-colors flex items-center justify-center gap-2 text-sm"
            >
              <Plus className="w-4 h-4" />
              Neues Dokument
            </button>
          )}
        </div>

        {/* Floating Add Button when collapsed */}
        {sidebarCollapsed && (
          <button
            onClick={() => setShowUploadModal(true)}
            className="mx-auto mt-4 p-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg transition-colors"
            title="Neues Dokument"
          >
            <Plus className="w-4 h-4" />
          </button>
        )}

        {/* Search */}
        {!sidebarCollapsed && (
          <div className="p-3 border-b border-slate-200 dark:border-slate-700">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
              <input
                type="text"
                placeholder="Dateien durchsuchen..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full pl-10 pr-4 py-2 bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg text-sm text-slate-900 dark:text-white placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-emerald-500"
              />
            </div>
          </div>
        )}

        {/* Favorites */}
        <div className="p-3 border-b border-slate-200 dark:border-slate-700">
          {!sidebarCollapsed ? (
            <>
              <button
                onClick={() => setShowFavorites(!showFavorites)}
                className="w-full flex items-center justify-between text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase mb-2"
              >
                <span>Favoriten</span>
                <ChevronDown className={`w-3 h-3 transition-transform ${showFavorites ? '' : '-rotate-90'}`} />
              </button>
              {showFavorites && (
                <div className="space-y-1">
                  {favoriteDocuments.map((doc) => (
                    <button
                      key={doc.id}
                      onClick={() => handleDocumentClick(doc)}
                      className="w-full flex items-center gap-2 px-2 py-1.5 text-xs text-slate-600 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 rounded transition-colors"
                    >
                      <Star className="w-3 h-3 fill-yellow-400 text-yellow-400" />
                      <span className="truncate">{doc.name}</span>
                    </button>
                  ))}
                </div>
              )}
            </>
          ) : (
            <button
              onClick={() => setShowFavorites(!showFavorites)}
              className="w-full flex justify-center p-2 hover:bg-slate-50 dark:hover:bg-slate-700 rounded-lg transition-colors"
              title={`Favoriten (${favoriteDocuments.length})`}
            >
              <Star className="w-4 h-4 fill-yellow-400 text-yellow-400" />
            </button>
          )}
        </div>

        {/* Vault Security Badge (when vault section visible) */}
        {!sidebarCollapsed && expandedFolders.includes('vault') && (
          <div className="mx-3 mt-2 p-2 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg">
            <div className="flex items-center gap-2">
              <Shield className="w-4 h-4 text-green-600 dark:text-green-400" />
              <div>
                <p className="text-xs font-semibold text-green-900 dark:text-green-100">🔐 Aktiviert</p>
                <p className="text-xs text-green-700 dark:text-green-300">AES-256 Verschlüsselung</p>
              </div>
            </div>
          </div>
        )}

        {/* Folder Navigation */}
        <div className="flex-1 overflow-y-auto p-3">
          {!sidebarCollapsed && (
            <h3 className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase mb-2">Ordner</h3>
          )}
          <div className="space-y-0.5">
            {folderStructure.map((folder) => renderFolderItem(folder))}
          </div>

          {/* Vault Settings Link */}
          {!sidebarCollapsed && expandedFolders.includes('vault') && (
            <button
              onClick={() => setShowVaultSettings(true)}
              className="w-full mt-2 px-2 py-2 text-xs text-emerald-600 dark:text-emerald-400 hover:bg-green-50 dark:hover:bg-green-900/20 rounded-lg flex items-center gap-2 transition-colors"
            >
              <Settings className="w-3 h-3" />
              🔑 Tresor-Einstellungen
            </button>
          )}
        </div>

        {/* Storage Widget */}
        <div className="p-3 border-t border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900">
          {!sidebarCollapsed ? (
            <>
              <div className="mb-2">
                <div className="flex items-center justify-between mb-1">
                  <span className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase">Speicher</span>
                  <span className="text-xs text-slate-400">75%</span>
                </div>
                <Progress value={75} className="h-1.5" />
              </div>
              <p className="text-xs text-slate-500 dark:text-slate-400 mb-2">
                523 GB von 1 TB verwendet
              </p>
              <button className="text-xs text-emerald-600 dark:text-emerald-400 hover:underline">
                Speicher erweitern
              </button>
            </>
          ) : (
            <div className="flex flex-col items-center" title="Speicher: 75% verwendet">
              <BarChart3 className="w-4 h-4 text-slate-500 mb-1" />
              <span className="text-xs text-slate-500">75%</span>
            </div>
          )}
        </div>

        {/* Help & Settings */}
        <div className={`p-3 border-t border-slate-200 dark:border-slate-700 flex items-center ${sidebarCollapsed ? 'flex-col gap-2' : 'gap-3 justify-center'}`}>
          <button 
            className="p-2 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg transition-colors"
            title="Hilfe"
          >
            <HelpCircle className="w-4 h-4 text-slate-500" />
          </button>
          <button 
            className="p-2 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg transition-colors"
            title="Einstellungen"
          >
            <Settings className="w-4 h-4 text-slate-500" />
          </button>
        </div>
      </div>

      {/* MIDDLE - DOCUMENT GRID/LIST */}
      <div className="flex-1 flex flex-col bg-white dark:bg-slate-800">
        {/* Header */}
        <div className="h-12 bg-slate-50 dark:bg-slate-900 border-b border-slate-200 dark:border-slate-700 px-4 flex items-center justify-between">
          <div className="flex items-center gap-4">
            <input type="checkbox" className="rounded border-slate-300" />
            <span className="text-xs text-slate-500 dark:text-slate-400">
              Alle Dokumente / {folderStructure.find(f => f.id === selectedFolder)?.name || 'Unbekannt'}
            </span>
          </div>
          <div className="flex items-center gap-3">
            <div className="flex items-center gap-1 bg-white dark:bg-slate-800 rounded-lg p-1 border border-slate-200 dark:border-slate-700">
              <button
                onClick={() => setView('grid')}
                className={`p-1.5 rounded ${view === 'grid' ? 'bg-slate-100 dark:bg-slate-700' : ''}`}
              >
                <Grid3x3 className={`w-4 h-4 ${view === 'grid' ? 'text-emerald-600' : 'text-slate-400'}`} />
              </button>
              <button
                onClick={() => setView('list')}
                className={`p-1.5 rounded ${view === 'list' ? 'bg-slate-100 dark:bg-slate-700' : ''}`}
              >
                <List className={`w-4 h-4 ${view === 'list' ? 'text-emerald-600' : 'text-slate-400'}`} />
              </button>
            </div>
            <button className="text-xs text-slate-600 dark:text-slate-300 flex items-center gap-1 hover:bg-slate-100 dark:hover:bg-slate-700 px-2 py-1.5 rounded">
              Neueste zuerst
              <ChevronDown className="w-3 h-3" />
            </button>
            <button
              onClick={() => setShowFilters(!showFilters)}
              className={`p-1.5 rounded ${showFilters ? 'bg-emerald-50 dark:bg-emerald-900/30 text-emerald-600' : 'hover:bg-slate-100 dark:hover:bg-slate-700'}`}
            >
              <Filter className="w-4 h-4" />
            </button>
          </div>
        </div>

        {/* Toolbar */}
        {selectedDocuments.length > 0 && (
          <div className="h-10 bg-white dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700 px-4 flex items-center gap-2">
            <span className="text-sm font-medium text-slate-900 dark:text-white mr-2">
              {selectedDocuments.length} ausgewählt
            </span>
            <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
              <Download className="w-4 h-4 text-slate-500" />
            </button>
            <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
              <Share2 className="w-4 h-4 text-slate-500" />
            </button>
            <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
              <Move className="w-4 h-4 text-slate-500" />
            </button>
            <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
              <Archive className="w-4 h-4 text-slate-500" />
            </button>
            <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
              <Trash2 className="w-4 h-4 text-red-500" />
            </button>
          </div>
        )}

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-4">
          {view === 'grid' ? (
            <div className="grid grid-cols-4 gap-4">
              {filteredDocuments.map((doc) => (
                <div
                  key={doc.id}
                  onClick={() => handleDocumentClick(doc)}
                  className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl overflow-hidden hover:border-emerald-600 hover:shadow-lg transition-all cursor-pointer group relative"
                >
                  {/* Thumbnail */}
                  <div className={`h-36 ${getFileTypeColor(doc.type)} flex items-center justify-center relative`}>
                    {doc.thumbnail ? (
                      <img src={doc.thumbnail} alt={doc.name} className="w-full h-full object-cover" />
                    ) : (
                      getFileTypeIcon(doc.type, 48)
                    )}
                    
                    {/* Vault Badge */}
                    {doc.isVault && (
                      <div className="absolute top-2 left-2 bg-green-900/90 text-white text-xs px-2 py-1 rounded flex items-center gap-1">
                        <Lock className="w-3 h-3" />
                        Encrypted
                      </div>
                    )}

                    {/* Hover Actions */}
                    <div className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity flex gap-1">
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          showToast('Zu Favoriten hinzugefügt', 'success');
                        }}
                        className="p-1.5 bg-white/90 dark:bg-slate-800/90 rounded-lg hover:bg-white"
                      >
                        <Star className={`w-4 h-4 ${doc.isFavorite ? 'fill-yellow-400 text-yellow-400' : 'text-slate-600'}`} />
                      </button>
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                        }}
                        className="p-1.5 bg-white/90 dark:bg-slate-800/90 rounded-lg hover:bg-white"
                      >
                        <MoreVertical className="w-4 h-4 text-slate-600" />
                      </button>
                    </div>
                  </div>

                  {/* Body */}
                  <div className="p-3">
                    <h4 className="text-sm font-semibold text-slate-900 dark:text-white truncate mb-1">
                      {doc.name}
                    </h4>
                    <p className="text-xs text-slate-500 dark:text-slate-400 mb-2">
                      {doc.size} • {doc.type.toUpperCase()} • {doc.date}
                    </p>
                    <div className="flex flex-wrap gap-1">
                      {doc.tags.slice(0, 2).map((tag, idx) => (
                        <Badge key={idx} className="bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 text-xs px-2 py-0">
                          {tag}
                        </Badge>
                      ))}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="space-y-1">
              {filteredDocuments.map((doc) => (
                <div
                  key={doc.id}
                  onClick={() => handleDocumentClick(doc)}
                  className="flex items-center gap-3 p-3 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-700 cursor-pointer transition-colors"
                >
                  <input
                    type="checkbox"
                    checked={selectedDocuments.includes(doc.id)}
                    onChange={(e) => {
                      e.stopPropagation();
                      toggleDocumentSelection(doc.id);
                    }}
                    className="rounded border-slate-300"
                  />
                  <div className={`w-10 h-10 ${getFileTypeColor(doc.type)} rounded-lg flex items-center justify-center relative`}>
                    {getFileTypeIcon(doc.type, 20)}
                    {doc.isVault && (
                      <Lock className="w-3 h-3 text-white absolute -bottom-1 -right-1 bg-green-600 rounded-full p-0.5" />
                    )}
                  </div>
                  <div className="flex-1 min-w-0">
                    <h4 className="text-sm font-semibold text-slate-900 dark:text-white truncate">
                      {doc.name}
                    </h4>
                    <p className="text-xs text-slate-500 dark:text-slate-400">
                      {doc.size} • {doc.type.toUpperCase()} • {doc.date} • {doc.createdBy}
                    </p>
                  </div>
                  <div className="flex gap-1">
                    {doc.tags.slice(0, 2).map((tag, idx) => (
                      <Badge key={idx} className="bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 text-xs px-2 py-0">
                        {tag}
                      </Badge>
                    ))}
                  </div>
                  <span className="text-xs text-slate-400">vor 3 Tagen</span>
                  <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-600 rounded">
                    <MoreVertical className="w-4 h-4 text-slate-400" />
                  </button>
                </div>
              ))}
            </div>
          )}

          {filteredDocuments.length === 0 && (
            <div className="flex flex-col items-center justify-center h-full p-8">
              <FolderOpen className="w-12 h-12 text-slate-300 dark:text-slate-600 mb-4" />
              <p className="text-slate-500 dark:text-slate-400 text-center mb-2">
                {searchQuery ? 'Keine Dokumente gefunden' : 'Keine Dokumente in diesem Ordner'}
              </p>
              <button
                onClick={() => setShowUploadModal(true)}
                className="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg transition-colors flex items-center gap-2 text-sm"
              >
                <Plus className="w-4 h-4" />
                Dokument hochladen
              </button>
            </div>
          )}
        </div>

        {/* Pagination */}
        <div className="h-10 border-t border-slate-200 dark:border-slate-700 px-4 flex items-center justify-between bg-slate-50 dark:bg-slate-900">
          <span className="text-xs text-slate-500 dark:text-slate-400">
            Zeige 1-{Math.min(20, filteredDocuments.length)} von {filteredDocuments.length} Dokumenten
          </span>
          <div className="flex gap-2">
            <button className="text-xs text-slate-600 dark:text-slate-300 hover:text-emerald-600 px-2 py-1">
              &lt; Vorherige
            </button>
            <button className="text-xs text-slate-600 dark:text-slate-300 hover:text-emerald-600 px-2 py-1">
              Nächste &gt;
            </button>
          </div>
        </div>
      </div>

      {/* RIGHT SIDEBAR - FILTERS (Collapsible) */}
      {showFilters && (
        <div className="w-52 border-l border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-4 overflow-y-auto">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-semibold text-slate-900 dark:text-white">Filter & Suche</h3>
            <button onClick={() => setShowFilters(false)}>
              <X className="w-4 h-4 text-slate-400" />
            </button>
          </div>

          {/* File Type Filter */}
          <div className="mb-4">
            <h4 className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase mb-2">Dateityp</h4>
            <div className="space-y-2">
              {['PDF', 'Word/DOCX', 'Excel/XLSX', 'PowerPoint', 'Images', 'Videos'].map((type) => (
                <label key={type} className="flex items-center gap-2 text-xs text-slate-600 dark:text-slate-300">
                  <input type="checkbox" className="rounded border-slate-300" />
                  {type}
                </label>
              ))}
            </div>
          </div>

          {/* Date Filter */}
          <div className="mb-4">
            <h4 className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase mb-2">Änderungsdatum</h4>
            <div className="flex flex-wrap gap-2">
              {['Diese Woche', 'Dieser Monat', 'Dieses Jahr'].map((period) => (
                <button key={period} className="px-3 py-1.5 bg-slate-100 dark:bg-slate-700 hover:bg-emerald-50 dark:hover:bg-emerald-900/30 text-slate-700 dark:text-slate-300 rounded-full text-xs transition-colors">
                  {period}
                </button>
              ))}
            </div>
          </div>

          {/* Size Filter */}
          <div className="mb-4">
            <h4 className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase mb-2">Dateigrösse</h4>
            <div className="space-y-2">
              {['< 1 MB', '1 - 10 MB', '10 - 100 MB', '> 100 MB'].map((size) => (
                <label key={size} className="flex items-center gap-2 text-xs text-slate-600 dark:text-slate-300">
                  <input type="radio" name="size" className="rounded-full border-slate-300" />
                  {size}
                </label>
              ))}
            </div>
          </div>

          {/* Tags */}
          <div className="mb-4">
            <h4 className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase mb-2">Tags</h4>
            <div className="flex flex-wrap gap-1">
              {['#Projekt', '#Wichtig', '#2024', '#Budget', '#Vertraulich', '#Geheim'].map((tag) => (
                <button key={tag} className="px-2 py-1 bg-white dark:bg-slate-700 border border-slate-200 dark:border-slate-600 hover:bg-emerald-50 dark:hover:bg-emerald-900/30 rounded-full text-xs transition-colors">
                  {tag}
                </button>
              ))}
            </div>
          </div>

          {/* Reset */}
          <button className="text-xs text-emerald-600 dark:text-emerald-400 hover:underline">
            Alle Filter zurücksetzen
          </button>
        </div>
      )}

      {/* DETAIL SIDEBAR */}
      {showDetailSidebar && selectedDocument && (
        <div className="w-80 border-l border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 flex flex-col">
          {/* Header */}
          <div className="p-4 border-b border-slate-200 dark:border-slate-700 flex items-center justify-between">
            <h3 className="text-sm font-semibold text-slate-900 dark:text-white truncate">{selectedDocument.name}</h3>
            <div className="flex gap-1">
              <button className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
                <MoreVertical className="w-4 h-4 text-slate-500" />
              </button>
              <button onClick={() => setShowDetailSidebar(false)} className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
                <X className="w-4 h-4 text-slate-500" />
              </button>
            </div>
          </div>

          {/* Preview */}
          <div className={`h-48 ${getFileTypeColor(selectedDocument.type)} flex items-center justify-center relative`}>
            {selectedDocument.thumbnail ? (
              <img src={selectedDocument.thumbnail} alt={selectedDocument.name} className="w-full h-full object-cover" />
            ) : (
              getFileTypeIcon(selectedDocument.type, 48)
            )}
            {selectedDocument.isVault && (
              <div className="absolute top-2 left-2 bg-green-900/90 text-white text-xs px-2 py-1 rounded flex items-center gap-1">
                <Lock className="w-3 h-3" />
                🔐 Encrypted
              </div>
            )}
          </div>

          {/* Content */}
          <div className="flex-1 overflow-y-auto p-4 space-y-4">
            {/* File Info */}
            <div>
              <h4 className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase mb-2">Dateiinformationen</h4>
              <div className="space-y-2 text-xs">
                <div className="flex justify-between">
                  <span className="text-slate-500">Typ:</span>
                  <span className="text-slate-900 dark:text-white font-medium">{selectedDocument.type.toUpperCase()}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-500">Größe:</span>
                  <span className="text-slate-900 dark:text-white font-medium">{selectedDocument.size}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-500">Erstellt:</span>
                  <span className="text-slate-900 dark:text-white font-medium">{selectedDocument.date}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-500">Erstellt von:</span>
                  <span className="text-slate-900 dark:text-white font-medium">{selectedDocument.createdBy}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-500">Geändert:</span>
                  <span className="text-slate-900 dark:text-white font-medium">vor 3 Tagen</span>
                </div>
              </div>
            </div>

            {/* Vault Encryption Details (if applicable) */}
            {selectedDocument.isVault && (
              <div>
                <h4 className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase mb-2">Verschlüsselung</h4>
                <div className="p-3 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg">
                  <div className="flex items-center gap-2 mb-2">
                    <Shield className="w-4 h-4 text-green-600 dark:text-green-400" />
                    <span className="text-xs font-semibold text-green-900 dark:text-green-100">
                      {selectedDocument.encryption || 'AES-256-GCM'} Verschlüsselung
                    </span>
                  </div>
                  <div className="space-y-1 text-xs text-green-700 dark:text-green-300">
                    <p>Schlüssel-Größe: 256 Bit</p>
                    <p>Verschlüsselt am: {selectedDocument.date}</p>
                    <p>Verschlüsselt von: {selectedDocument.createdBy}</p>
                  </div>
                </div>
              </div>
            )}

            {/* Sharing */}
            <div>
              <h4 className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase mb-2">Zugriff & Freigabe</h4>
              <div className="mb-2">
                {selectedDocument.isShared ? (
                  <Badge className="bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300 text-xs">
                    👥 Freigegeben für {selectedDocument.sharedWith.length} Personen
                  </Badge>
                ) : (
                  <Badge className="bg-gray-100 dark:bg-gray-900/30 text-gray-700 dark:text-gray-300 text-xs">
                    🔒 Nur für mich
                  </Badge>
                )}
              </div>
              <button
                onClick={() => setShowShareModal(true)}
                className="w-full px-3 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg text-sm transition-colors"
              >
                Zugriff verwalten
              </button>
            </div>

            {/* Version History */}
            <div>
              <h4 className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase mb-2">Versionshistorie</h4>
              <div className="space-y-2">
                <div className="p-2 bg-slate-50 dark:bg-slate-900 rounded-lg">
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-xs font-medium text-slate-900 dark:text-white">Version {selectedDocument.version} (aktuell)</span>
                    <span className="text-xs text-slate-400">vor 3 Tagen</span>
                  </div>
                  <p className="text-xs text-slate-500">{selectedDocument.createdBy}</p>
                </div>
              </div>
              <button className="text-xs text-emerald-600 dark:text-emerald-400 hover:underline mt-2">
                Alle Versionen anzeigen
              </button>
            </div>

            {/* Tags */}
            <div>
              <h4 className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase mb-2">Tags & Labels</h4>
              <div className="flex flex-wrap gap-1">
                {selectedDocument.tags.map((tag: string, idx: number) => (
                  <Badge key={idx} className="bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 text-xs px-2 py-1">
                    {tag}
                    <button className="ml-1">
                      <X className="w-3 h-3" />
                    </button>
                  </Badge>
                ))}
                <button className="px-2 py-1 border border-dashed border-slate-300 dark:border-slate-600 rounded text-xs text-slate-500 hover:bg-slate-50 dark:hover:bg-slate-700">
                  + Tag hinzufügen
                </button>
              </div>
            </div>

            {/* Activity (Vault Access Log if applicable) */}
            <div>
              <h4 className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase mb-2">
                {selectedDocument.isVault ? 'Zugriffs-Protokoll' : 'Aktivitätsverlauf'}
              </h4>
              <div className="space-y-2">
                <div className="flex gap-2 text-xs">
                  <Clock className="w-3 h-3 text-slate-400 mt-0.5" />
                  <div className="flex-1">
                    <p className="text-slate-900 dark:text-white">{selectedDocument.createdBy} hat die Datei {selectedDocument.isVault ? 'geöffnet' : 'bearbeitet'}</p>
                    <p className="text-slate-400">vor 3 Tagen</p>
                  </div>
                </div>
                {selectedDocument.isVault && (
                  <div className="flex gap-2 text-xs">
                    <Eye className="w-3 h-3 text-slate-400 mt-0.5" />
                    <div className="flex-1">
                      <p className="text-slate-900 dark:text-white">Lisa Meier hat die Datei angesehen</p>
                      <p className="text-slate-400">vor 5 Tagen</p>
                    </div>
                  </div>
                )}
              </div>
              {selectedDocument.isVault && (
                <button className="text-xs text-emerald-600 dark:text-emerald-400 hover:underline mt-2">
                  Vollständiges Zugriffs-Protokoll
                </button>
              )}
            </div>
          </div>

          {/* Actions */}
          <div className="p-4 border-t border-slate-200 dark:border-slate-700 space-y-2">
            <button
              onClick={() => showToast('Download gestartet...', 'success')}
              className="w-full px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg transition-colors text-sm"
            >
              Herunterladen
            </button>
            <button
              onClick={() => setShowShareModal(true)}
              className="w-full px-4 py-2 bg-slate-100 dark:bg-slate-700 hover:bg-slate-200 dark:hover:bg-slate-600 text-slate-700 dark:text-slate-300 rounded-lg transition-colors text-sm"
            >
              Freigeben
            </button>
          </div>
        </div>
      )}

      {/* UPLOAD MODAL */}
      <Dialog open={showUploadModal} onOpenChange={setShowUploadModal}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>{selectedFolder.startsWith('vault') ? 'In den Tresor hochladen' : 'Neues Dokument hochladen'}</DialogTitle>
            <DialogDescription>
              {selectedFolder.startsWith('vault') 
                ? '⚠️ Diese Dokumente werden mit AES-256 verschlüsselt und in einem separaten sicheren Bereich gespeichert'
                : 'Wähle eine oder mehrere Dateien aus'}
            </DialogDescription>
          </DialogHeader>

          <div className="py-4">
            {/* Drag & Drop Area */}
            <div className="border-2 border-dashed border-slate-200 dark:border-slate-700 rounded-xl p-12 bg-slate-50 dark:bg-slate-900 hover:bg-slate-100 dark:hover:bg-slate-800 cursor-pointer transition-colors">
              <div className="flex flex-col items-center text-center">
                <Upload className="w-12 h-12 text-emerald-600 dark:text-emerald-400 mb-3" />
                <p className="text-sm text-slate-700 dark:text-slate-300 mb-1">
                  Dateien hier ablegen oder klicken zum Durchsuchen
                </p>
                <p className="text-xs text-slate-500 dark:text-slate-400">
                  Unterstützte Formate: PDF, DOCX, XLSX, PPTX, JPG, PNG, SVG
                </p>
              </div>
            </div>

            {/* Folder Selection */}
            <div className="mt-4">
              <Label>Zielordner</Label>
              <select className="w-full mt-1 px-3 py-2 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg text-sm">
                <option>Alle Dokumente</option>
                <option>Projekte</option>
                <option>Team Dokumente</option>
                <option>Vorlagen</option>
                <option>🔐 Datentresor</option>
              </select>
            </div>

            {/* Vault Options */}
            {selectedFolder.startsWith('vault') && (
              <div className="mt-4 p-3 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg space-y-3">
                <div>
                  <Label>Zugriffsablauf (Optional)</Label>
                  <select className="w-full mt-1 px-3 py-2 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg text-sm">
                    <option>Nie</option>
                    <option>30 Tage</option>
                    <option>90 Tage</option>
                    <option>1 Jahr</option>
                  </select>
                </div>
                <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
                  <input type="checkbox" className="rounded border-slate-300" />
                  Zwei-Faktor-Authentifizierung beim Zugriff erforderlich
                </label>
              </div>
            )}
          </div>

          <div className="flex justify-end gap-3">
            <button
              onClick={() => setShowUploadModal(false)}
              className="px-4 py-2 bg-slate-100 dark:bg-slate-700 hover:bg-slate-200 dark:hover:bg-slate-600 text-slate-700 dark:text-slate-300 rounded-lg transition-colors"
            >
              Abbrechen
            </button>
            <button
              onClick={() => {
                showToast(selectedFolder.startsWith('vault') ? '✅ Dokument verschlüsselt & gespeichert' : '✅ Dokument hochgeladen', 'success');
                setShowUploadModal(false);
              }}
              className="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg transition-colors"
            >
              Hochladen
            </button>
          </div>
        </DialogContent>
      </Dialog>

      {/* SHARE MODAL */}
      <Dialog open={showShareModal} onOpenChange={setShowShareModal}>
        <DialogContent className="max-w-xl">
          <DialogHeader>
            <DialogTitle>Zugriff für "{selectedDocument?.name}" verwalten</DialogTitle>
          </DialogHeader>

          <div className="py-4 space-y-4">
            {/* Public Link */}
            <div>
              <div className="flex items-center justify-between mb-2">
                <Label>Öffentlicher Link</Label>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input type="checkbox" className="sr-only peer" />
                  <div className="w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-emerald-300 dark:peer-focus:ring-emerald-800 rounded-full peer dark:bg-slate-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-slate-600 peer-checked:bg-emerald-600"></div>
                </label>
              </div>
              <div className="flex gap-2">
                <Input 
                  value="https://kmu-hub.ch/share/abc123xyz" 
                  readOnly 
                  className="flex-1"
                />
                <button
                  onClick={() => showToast('Link kopiert', 'success')}
                  className="px-3 py-2 bg-slate-100 dark:bg-slate-700 hover:bg-slate-200 dark:hover:bg-slate-600 rounded-lg"
                >
                  <Copy className="w-4 h-4" />
                </button>
              </div>
            </div>

            {/* Add Users */}
            <div>
              <Label>Zugriff für Benutzer</Label>
              <div className="flex gap-2 mt-1">
                <Input placeholder="Email oder Benutzername eingeben..." className="flex-1" />
                <select className="px-3 py-2 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg text-sm">
                  <option>Viewer</option>
                  <option>Editor</option>
                  <option>Commenter</option>
                </select>
                <button className="px-3 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg">
                  <Plus className="w-4 h-4" />
                </button>
              </div>
            </div>

            {/* Access List */}
            {selectedDocument?.isShared && (
              <div>
                <h4 className="text-sm font-medium text-slate-900 dark:text-white mb-2">Zugriff gewährt für:</h4>
                <div className="space-y-2">
                  {selectedDocument.sharedWith.map((user: string, idx: number) => (
                    <div key={idx} className="flex items-center justify-between p-2 bg-slate-50 dark:bg-slate-900 rounded-lg">
                      <div className="flex items-center gap-2">
                        <Avatar className="w-8 h-8">
                          <AvatarFallback className="bg-emerald-100 dark:bg-emerald-900 text-emerald-700 dark:text-emerald-300 text-xs">
                            {user.split(' ').map(n => n[0]).join('')}
                          </AvatarFallback>
                        </Avatar>
                        <div>
                          <p className="text-xs font-medium text-slate-900 dark:text-white">{user}</p>
                          <p className="text-xs text-slate-500">Editor</p>
                        </div>
                      </div>
                      <button className="p-1 hover:bg-slate-200 dark:hover:bg-slate-700 rounded">
                        <X className="w-4 h-4 text-slate-400" />
                      </button>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>

          <div className="flex justify-end gap-3">
            <button
              onClick={() => setShowShareModal(false)}
              className="px-4 py-2 bg-slate-100 dark:bg-slate-700 hover:bg-slate-200 dark:hover:bg-slate-600 text-slate-700 dark:text-slate-300 rounded-lg transition-colors"
            >
              Abbrechen
            </button>
            <button
              onClick={() => {
                showToast('✅ Zugriff gespeichert', 'success');
                setShowShareModal(false);
              }}
              className="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg transition-colors"
            >
              Speichern
            </button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Vault Settings Modal */}
      <VaultSettings 
        isOpen={showVaultSettings} 
        onClose={() => setShowVaultSettings(false)} 
      />

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
