import { Search, Bell, Menu, Globe } from 'lucide-react';
import { Avatar, AvatarFallback, AvatarImage } from './ui/avatar';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from './ui/dropdown-menu';
import { ProfileSwitcher } from './ProfileSwitcher';
import { NotificationCenter } from './NotificationCenter';
import { SearchBar } from './SearchBar';
import { ProfileMenu } from './ProfileMenu';
import { DailyPlannerWidget } from './DailyPlannerWidget';

interface HeaderProps {
  onMenuClick: () => void;
  onCreateProfile: () => void;
  onSupportClick: () => void;
  timeTrackingPresets: string[];
}

const languages = [
  { code: 'de', name: 'Deutsch', flag: '🇩🇪' },
  { code: 'fr', name: 'Français', flag: '🇫🇷', disabled: true },
  { code: 'it', name: 'Italiano', flag: '🇮🇹' },
  { code: 'en', name: 'English', flag: '🇬🇧', disabled: true },
];

export function Header({ onMenuClick, onCreateProfile, onSupportClick, timeTrackingPresets }: HeaderProps) {
  const currentLanguage = languages[0]; // Default: Deutsch

  return (
    <header className="h-16 bg-header-background dark:bg-header-background border-b border-header-border dark:border-header-border flex items-center justify-between px-4 md:px-8">
      <div className="flex items-center gap-2 md:gap-4 flex-1 min-w-0">
        <button
          onClick={onMenuClick}
          className="lg:hidden p-2 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg flex-shrink-0"
        >
          <Menu className="w-5 h-5 text-slate-600 dark:text-slate-300" />
        </button>
        
        <div className="flex-1 min-w-0">
          <SearchBar />
        </div>
      </div>
      
      <div className="flex items-center gap-1 md:gap-2 flex-shrink-0">
        {/* Daily Planner Widget */}
        <div className="hidden sm:block">
          <DailyPlannerWidget />
        </div>

        {/* Language Switcher - Hidden on mobile */}
        <div className="hidden md:block">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button className="flex items-center gap-2 px-3 py-2 hover:bg-slate-50 dark:hover:bg-slate-700 rounded-lg transition-colors">
                <Globe className="w-4 h-4 text-slate-600 dark:text-slate-300" />
                <span className="text-sm text-slate-700 dark:text-slate-300 hidden lg:inline">{currentLanguage.flag}</span>
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-48">
              {languages.map((lang) => (
                <DropdownMenuItem
                  key={lang.code}
                  disabled={lang.disabled}
                  className={lang.disabled ? 'opacity-50' : ''}
                >
                  <span className="mr-2">{lang.flag}</span>
                  {lang.name}
                  {lang.disabled && (
                    <span className="ml-auto text-xs text-slate-400 dark:text-slate-500">(Bald)</span>
                  )}
                </DropdownMenuItem>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        {/* Notification Center */}
        <NotificationCenter />

        {/* Profile Switcher - Hidden on mobile */}
        <div className="hidden sm:block">
          <ProfileSwitcher onCreateProfile={onCreateProfile} />
        </div>
        
        {/* Profile Menu with Settings & Support */}
        <ProfileMenu onSupportClick={onSupportClick} timeTrackingPresets={timeTrackingPresets} />
      </div>
    </header>
  );
}