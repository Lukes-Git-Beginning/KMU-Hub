import { useState, useEffect } from 'react';
import { LayoutDashboard, FolderKanban, CheckSquare, FileText, MessageSquare, Settings, Users, X, Calculator, Calendar, Mail, Contact, ChevronLeft, ChevronRight } from 'lucide-react';
import { Link, useLocation } from 'react-router-dom';

const navigationItems = [
  { name: 'Übersicht', icon: LayoutDashboard, path: '/' },
  { name: 'Projekte', icon: FolderKanban, path: '/projekte' },
  { name: 'Aufgaben', icon: CheckSquare, path: '/aufgaben' },
  { name: 'Dokumente', icon: FileText, path: '/dokumente' },
  { name: 'Buchhaltung', icon: Calculator, path: '/buchhaltung', badge: 'Neu' },
  { name: 'Nachrichten', icon: MessageSquare, path: '/nachrichten' },
  { name: 'Team', icon: Users, path: '/team' },
  { name: 'Mails', icon: Mail, path: '/mails', badge: '12' },
  { name: 'Kontakte', icon: Contact, path: '/kontakte', badge: '45' },
  { name: 'Meetings', icon: Calendar, path: '/meetings', liveBadge: true },
];

interface SidebarProps {
  isOpen: boolean;
  onClose: () => void;
}

export function Sidebar({ isOpen, onClose }: SidebarProps) {
  const location = useLocation();
  
  // Collapsed state with localStorage persistence
  const [collapsed, setCollapsed] = useState(() => {
    const saved = localStorage.getItem('main-sidebar-collapsed');
    if (saved !== null) return saved === 'true';
    return window.innerWidth < 1200; // Auto-collapse on smaller screens
  });

  // Persist collapsed state
  useEffect(() => {
    localStorage.setItem('main-sidebar-collapsed', String(collapsed));
  }, [collapsed]);

  // Handle responsive behavior
  useEffect(() => {
    const handleResize = () => {
      if (window.innerWidth < 768) {
        // Mobile: keep as drawer
      } else if (window.innerWidth < 1200) {
        setCollapsed(true); // Auto-collapse on tablet
      }
    };
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  const toggleCollapsed = () => {
    setCollapsed(!collapsed);
  };
  
  return (
    <>
      {/* Mobile Overlay */}
      {isOpen && (
        <div
          className="fixed inset-0 bg-black/50 z-40 lg:hidden"
          onClick={onClose}
        />
      )}
      
      {/* Sidebar */}
      <aside
        className={`
          fixed lg:static inset-y-0 left-0 z-50
          h-screen bg-sidebar dark:bg-sidebar border-r border-sidebar-border dark:border-sidebar-border flex flex-col
          transform transition-all duration-300 ease-in-out
          ${isOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'}
          ${collapsed ? 'lg:w-16' : 'lg:w-60'}
          w-60
        `}
      >
        <div className="border-b border-slate-200 dark:border-slate-700">
          <div className="p-6 flex items-center justify-between">
            {!collapsed && (
              <div className="pr-8">
                <h1 className="text-slate-900 dark:text-white">KMU Digital Hub</h1>
                <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">Schweizer Lösung</p>
              </div>
            )}
            
            {/* Close Button (Mobile) */}
            <button
              onClick={onClose}
              className="lg:hidden p-1 hover:bg-slate-100 dark:hover:bg-slate-700 rounded ml-auto"
            >
              <X className="w-5 h-5 text-slate-600 dark:text-slate-300" />
            </button>
          </div>

          {/* Toggle Button (Desktop) - Below Header */}
          <div className="px-4 pb-3">
            <button
              onClick={toggleCollapsed}
              className={`hidden lg:flex items-center gap-2 w-full px-3 py-2 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg transition-colors text-sm text-slate-600 dark:text-slate-300 ${
                collapsed ? 'justify-center' : ''
              }`}
              title={collapsed ? 'Sidebar erweitern' : 'Sidebar minimieren'}
            >
              {collapsed ? (
                <ChevronRight className="w-4 h-4 text-primary dark:text-primary" />
              ) : (
                <>
                  <ChevronLeft className="w-4 h-4 text-primary dark:text-primary" />
                  <span>Minimieren</span>
                </>
              )}
            </button>
          </div>
        </div>
        
        <nav className="flex-1 p-4 overflow-y-auto">
          <ul className="space-y-1">
            {navigationItems.map((item) => {
              const Icon = item.icon;
              const isActive = location.pathname === item.path;
              return (
                <li key={item.name}>
                  <Link
                    to={item.path}
                    onClick={onClose}
                    title={collapsed ? item.name : undefined}
                    className={`flex items-center ${collapsed ? 'justify-center' : 'justify-between'} gap-3 px-3 py-2 rounded-lg transition-colors ${
                      isActive
                        ? 'bg-primary-light dark:bg-primary-subtle text-primary dark:text-primary-light'
                        : 'text-slate-600 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 hover:text-slate-900 dark:hover:text-white'
                    }`}
                  >
                    <div className={`flex items-center gap-3 ${collapsed ? 'relative' : ''}`}>
                      <Icon className="w-5 h-5" />
                      {!collapsed && <span className="text-sm">{item.name}</span>}
                      
                      {/* Badge in collapsed mode (as dot) */}
                      {collapsed && (item.badge || item.liveBadge) && (
                        <span className={`absolute -top-1 -right-1 w-2 h-2 rounded-full ${
                          item.liveBadge ? 'bg-red-500' : 'bg-primary'
                        }`} />
                      )}
                    </div>
                    
                    {/* Badge in expanded mode */}
                    {!collapsed && item.badge && (
                      <span className="text-xs px-2 py-0.5 bg-primary-light dark:bg-primary-subtle text-primary dark:text-primary-light rounded-full">
                        {item.badge}
                      </span>
                    )}
                    {!collapsed && item.liveBadge && (
                      <span className="text-xs px-2 py-0.5 bg-error-light dark:bg-error-light text-error dark:text-error rounded-full">
                        Live
                      </span>
                    )}
                  </Link>
                </li>
              );
            })}
          </ul>
        </nav>

        {/* Settings at bottom */}
        <div className="p-4 border-t border-slate-200 dark:border-slate-700">
          <Link
            to="/einstellungen"
            onClick={onClose}
            title={collapsed ? 'Einstellungen' : undefined}
            className={`flex items-center ${collapsed ? 'justify-center' : ''} gap-3 px-3 py-2 rounded-lg transition-colors text-slate-600 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 hover:text-slate-900 dark:hover:text-white`}
          >
            <Settings className="w-5 h-5" />
            {!collapsed && <span className="text-sm">Einstellungen</span>}
          </Link>
        </div>
      </aside>
    </>
  );
}