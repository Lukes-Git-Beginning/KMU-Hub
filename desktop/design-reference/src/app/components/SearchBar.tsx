import { useState, useRef, useEffect } from 'react';
import { Search, FolderKanban, CheckSquare, FileText, MessageSquare, Users, Calculator, ArrowRight } from 'lucide-react';
import { useNavigate } from 'react-router-dom';

interface SearchResult {
  id: string;
  title: string;
  category: string;
  categoryIcon: any;
  path: string;
  description?: string;
  iconColor: string;
  bgColor: string;
}

// Mock-Daten für die Suche
const allSearchData: SearchResult[] = [
  // Projekte
  { id: 'p1', title: 'Website Relaunch', category: 'Projekte', categoryIcon: FolderKanban, path: '/projekte/1', description: 'In Progress', iconColor: 'text-blue-600', bgColor: 'bg-blue-500/10' },
  { id: 'p2', title: 'Mobile App Development', category: 'Projekte', categoryIcon: FolderKanban, path: '/projekte/2', description: 'Planning', iconColor: 'text-blue-600', bgColor: 'bg-blue-500/10' },
  { id: 'p3', title: 'Brand Refresh', category: 'Projekte', categoryIcon: FolderKanban, path: '/projekte/3', description: 'Review', iconColor: 'text-blue-600', bgColor: 'bg-blue-500/10' },
  { id: 'p4', title: 'API Integration', category: 'Projekte', categoryIcon: FolderKanban, path: '/projekte/4', description: 'Testing', iconColor: 'text-blue-600', bgColor: 'bg-blue-500/10' },
  
  // Aufgaben
  { id: 't1', title: 'API Dokumentation schreiben', category: 'Aufgaben', categoryIcon: CheckSquare, path: '/aufgaben', description: 'Fällig morgen', iconColor: 'text-emerald-600', bgColor: 'bg-emerald-500/10' },
  { id: 't2', title: 'Design Review vorbereiten', category: 'Aufgaben', categoryIcon: CheckSquare, path: '/aufgaben', description: 'Hohe Priorität', iconColor: 'text-emerald-600', bgColor: 'bg-emerald-500/10' },
  { id: 't3', title: 'Logo Redesign', category: 'Aufgaben', categoryIcon: CheckSquare, path: '/aufgaben', description: 'In Bearbeitung', iconColor: 'text-emerald-600', bgColor: 'bg-emerald-500/10' },
  { id: 't4', title: 'Testing Sprint vorbereiten', category: 'Aufgaben', categoryIcon: CheckSquare, path: '/aufgaben', description: 'Diese Woche', iconColor: 'text-emerald-600', bgColor: 'bg-emerald-500/10' },
  
  // Dokumente
  { id: 'd1', title: 'Q1 Budget.xlsx', category: 'Dokumente', categoryIcon: FileText, path: '/dokumente', description: 'Finanzen', iconColor: 'text-purple-600', bgColor: 'bg-purple-500/10' },
  { id: 'd2', title: 'Vertrag_Kunde_A.pdf', category: 'Dokumente', categoryIcon: FileText, path: '/dokumente', description: 'Legal', iconColor: 'text-purple-600', bgColor: 'bg-purple-500/10' },
  { id: 'd3', title: 'Projektplan_2025.docx', category: 'Dokumente', categoryIcon: FileText, path: '/dokumente', description: 'Planung', iconColor: 'text-purple-600', bgColor: 'bg-purple-500/10' },
  { id: 'd4', title: 'Team_Handbook.pdf', category: 'Dokumente', categoryIcon: FileText, path: '/dokumente', description: 'HR', iconColor: 'text-purple-600', bgColor: 'bg-purple-500/10' },
  
  // Buchhaltung
  { id: 'a1', title: 'Rechnung #2024-156', category: 'Buchhaltung', categoryIcon: Calculator, path: '/buchhaltung', description: 'CHF 12,500', iconColor: 'text-orange-600', bgColor: 'bg-orange-500/10' },
  { id: 'a2', title: 'Bexio Integration', category: 'Buchhaltung', categoryIcon: Calculator, path: '/buchhaltung', description: 'Verbunden', iconColor: 'text-orange-600', bgColor: 'bg-orange-500/10' },
  { id: 'a3', title: 'Ausgaben Dezember', category: 'Buchhaltung', categoryIcon: Calculator, path: '/buchhaltung', description: 'CHF 45,200', iconColor: 'text-orange-600', bgColor: 'bg-orange-500/10' },
  { id: 'a4', title: 'Abacus Sync', category: 'Buchhaltung', categoryIcon: Calculator, path: '/buchhaltung', description: 'Konfigurieren', iconColor: 'text-orange-600', bgColor: 'bg-orange-500/10' },
  
  // Kontakte/Nachrichten
  { id: 'c1', title: 'Michael Berg', category: 'Kontakte', categoryIcon: MessageSquare, path: '/nachrichten', description: 'Senior Designer', iconColor: 'text-pink-600', bgColor: 'bg-pink-500/10' },
  { id: 'c2', title: 'Sarah Klein', category: 'Kontakte', categoryIcon: MessageSquare, path: '/nachrichten', description: 'Lead Developer', iconColor: 'text-pink-600', bgColor: 'bg-pink-500/10' },
  { id: 'c3', title: 'Anna Müller', category: 'Kontakte', categoryIcon: MessageSquare, path: '/nachrichten', description: 'Project Manager', iconColor: 'text-pink-600', bgColor: 'bg-pink-500/10' },
  { id: 'c4', title: 'Thomas Klein', category: 'Kontakte', categoryIcon: MessageSquare, path: '/nachrichten', description: 'CEO', iconColor: 'text-pink-600', bgColor: 'bg-pink-500/10' },
  
  // Team
  { id: 'tm1', title: 'Team-Mitglieder verwalten', category: 'Team', categoryIcon: Users, path: '/team', description: '12 Mitglieder', iconColor: 'text-cyan-600', bgColor: 'bg-cyan-500/10' },
  { id: 'tm2', title: 'Entwickler Team', category: 'Team', categoryIcon: Users, path: '/team', description: '5 Mitglieder', iconColor: 'text-cyan-600', bgColor: 'bg-cyan-500/10' },
  { id: 'tm3', title: 'Design Team', category: 'Team', categoryIcon: Users, path: '/team', description: '3 Mitglieder', iconColor: 'text-cyan-600', bgColor: 'bg-cyan-500/10' },
];

export function SearchBar() {
  const [query, setQuery] = useState('');
  const [isOpen, setIsOpen] = useState(false);
  const [results, setResults] = useState<SearchResult[]>([]);
  const searchRef = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();

  // Search function
  useEffect(() => {
    if (query.trim().length === 0) {
      setResults([]);
      setIsOpen(false);
      return;
    }

    const searchQuery = query.toLowerCase();
    const filtered = allSearchData.filter((item) => {
      return (
        item.title.toLowerCase().includes(searchQuery) ||
        item.category.toLowerCase().includes(searchQuery) ||
        item.description?.toLowerCase().includes(searchQuery)
      );
    });

    // Limit to 3 results
    setResults(filtered.slice(0, 3));
    setIsOpen(filtered.length > 0);
  }, [query]);

  // Click outside to close
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (searchRef.current && !searchRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleResultClick = (path: string) => {
    navigate(path);
    setQuery('');
    setIsOpen(false);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && results.length > 0) {
      handleResultClick(results[0].path);
    }
    if (e.key === 'Escape') {
      setIsOpen(false);
    }
  };

  return (
    <div ref={searchRef} className="relative flex-1 max-w-2xl">
      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400 dark:text-slate-500 pointer-events-none" />
        <input
          type="text"
          placeholder="Suchen... (Projekte, Aufgaben, Dokumente, Kontakte...)"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={handleKeyDown}
          onFocus={() => query.trim() && results.length > 0 && setIsOpen(true)}
          className="w-full pl-10 pr-4 py-2 bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent text-sm text-slate-900 dark:text-white placeholder:text-slate-400 dark:placeholder:text-slate-500"
        />
      </div>

      {/* Search Results Dropdown */}
      {isOpen && results.length > 0 && (
        <div className="absolute top-full left-0 right-0 mt-2 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg shadow-lg overflow-hidden z-50">
          <div className="p-2">
            <p className="text-xs text-slate-500 dark:text-slate-400 px-3 py-2">
              Top {results.length} Ergebnis{results.length > 1 ? 'se' : ''}
            </p>
            <div className="space-y-1">
              {results.map((result) => {
                const Icon = result.categoryIcon;
                return (
                  <button
                    key={result.id}
                    onClick={() => handleResultClick(result.path)}
                    className="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors text-left group"
                  >
                    {/* Icon */}
                    <div className={`w-10 h-10 rounded-lg ${result.bgColor} dark:${result.bgColor} flex items-center justify-center flex-shrink-0`}>
                      <Icon className={`w-5 h-5 ${result.iconColor} dark:${result.iconColor}`} />
                    </div>

                    {/* Content */}
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-slate-900 dark:text-white truncate">
                        {result.title}
                      </p>
                      <div className="flex items-center gap-2 mt-0.5">
                        <span className="text-xs text-slate-500 dark:text-slate-400">
                          {result.category}
                        </span>
                        {result.description && (
                          <>
                            <span className="text-xs text-slate-300 dark:text-slate-600">•</span>
                            <span className="text-xs text-slate-500 dark:text-slate-400">
                              {result.description}
                            </span>
                          </>
                        )}
                      </div>
                    </div>

                    {/* Arrow */}
                    <ArrowRight className="w-4 h-4 text-slate-400 dark:text-slate-500 group-hover:text-emerald-600 dark:group-hover:text-emerald-400 group-hover:translate-x-1 transition-all flex-shrink-0" />
                  </button>
                );
              })}
            </div>
          </div>

          {/* Footer */}
          <div className="px-3 py-2 bg-slate-50 dark:bg-slate-900/50 border-t border-slate-200 dark:border-slate-700">
            <p className="text-xs text-slate-500 dark:text-slate-400">
              <kbd className="px-1.5 py-0.5 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded text-xs">Enter</kbd>
              {' '}zum Öffnen • 
              <kbd className="ml-1 px-1.5 py-0.5 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded text-xs">Esc</kbd>
              {' '}zum Schließen
            </p>
          </div>
        </div>
      )}
    </div>
  );
}
