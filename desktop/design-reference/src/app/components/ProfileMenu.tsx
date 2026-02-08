import { useState, useRef, useEffect } from 'react';
import { Settings, HelpCircle, ChevronDown } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { Avatar, AvatarFallback, AvatarImage } from './ui/avatar';
import { TimeTracker } from './TimeTracker';

interface ProfileMenuProps {
  onSupportClick: () => void;
  timeTrackingPresets: string[];
}

export function ProfileMenu({ onSupportClick, timeTrackingPresets }: ProfileMenuProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [isTimeTrackerExpanded, setIsTimeTrackerExpanded] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();

  // Close on click outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [isOpen]);

  const handleSettingsClick = () => {
    navigate('/einstellungen');
    setIsOpen(false);
  };

  const handleSupportClickInternal = () => {
    onSupportClick();
    setIsOpen(false);
  };

  const handleProfileClick = () => {
    navigate('/profil');
    setIsOpen(false);
  };

  return (
    <>
      <div ref={menuRef} className="relative">
        {/* Profile Button */}
        <button
          onClick={() => setIsOpen(!isOpen)}
          className="flex items-center gap-2 md:gap-3 pl-2 md:pl-4 border-l border-slate-200 dark:border-slate-700 hover:opacity-80 transition-opacity"
        >
          <div className="text-right hidden md:block">
            <p className="text-sm text-slate-900 dark:text-white">Anna Müller</p>
            <p className="text-xs text-slate-500 dark:text-slate-400">Administrator</p>
          </div>
          <Avatar className="ring-2 ring-transparent hover:ring-emerald-500 dark:hover:ring-emerald-400 transition-all">
            <AvatarImage src="https://images.unsplash.com/photo-1494790108377-be9c29b29330?w=100&h=100&fit=crop" />
            <AvatarFallback>AM</AvatarFallback>
          </Avatar>
          <ChevronDown 
            className={`w-4 h-4 md:w-5 md:h-5 text-slate-500 dark:text-slate-400 transition-transform duration-300 ${
              isOpen ? 'rotate-180' : ''
            }`} 
          />
        </button>

        {/* Dropdown Menu */}
        {isOpen && (
          <>
            {/* Backdrop */}
            <div 
              className="fixed inset-0 z-40" 
              onClick={() => setIsOpen(false)}
            />
            
            {/* Menu Content */}
            <div className="absolute top-full right-0 mt-2 w-80 md:w-96 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg shadow-xl z-50 overflow-hidden">
              {/* User Info */}
              <button
                onClick={handleProfileClick}
                className="p-4 border-b border-slate-200 dark:border-slate-700 bg-gradient-to-br from-emerald-50 to-white dark:from-slate-700 dark:to-slate-800 hover:from-emerald-100 dark:hover:from-slate-600 transition-all w-full text-left"
              >
                <div className="flex items-center gap-3 mb-3">
                  <Avatar className="w-12 h-12 ring-2 ring-emerald-500 dark:ring-emerald-400">
                    <AvatarImage src="https://images.unsplash.com/photo-1494790108377-be9c29b29330?w=100&h=100&fit=crop" />
                    <AvatarFallback>AM</AvatarFallback>
                  </Avatar>
                  <div>
                    <p className="font-medium text-slate-900 dark:text-white">Anna Müller</p>
                    <p className="text-sm text-slate-500 dark:text-slate-400">anna.mueller@firma.ch</p>
                  </div>
                </div>
              </button>

              {/* Action Buttons */}
              <div className="p-3 space-y-2 border-b border-slate-200 dark:border-slate-700">
                <button
                  onClick={handleSettingsClick}
                  className="w-full flex items-center gap-3 px-3 py-2 text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg transition-colors"
                >
                  <div className="w-8 h-8 flex items-center justify-center bg-emerald-100 dark:bg-emerald-900/30 rounded-lg">
                    <Settings className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
                  </div>
                  <span className="text-sm">Einstellungen</span>
                </button>

                <button
                  onClick={handleSupportClickInternal}
                  className="w-full flex items-center gap-3 px-3 py-2 text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg transition-colors"
                >
                  <div className="w-8 h-8 flex items-center justify-center bg-blue-100 dark:bg-blue-900/30 rounded-lg">
                    <HelpCircle className="w-4 h-4 text-blue-600 dark:text-blue-400" />
                  </div>
                  <span className="text-sm">Hilfe & Support</span>
                </button>
              </div>

              {/* Time Tracker */}
              <TimeTracker 
                isExpanded={isTimeTrackerExpanded}
                onToggleExpanded={() => setIsTimeTrackerExpanded(!isTimeTrackerExpanded)}
                presets={timeTrackingPresets}
              />
            </div>
          </>
        )}
      </div>

      {/* Expanded Time Tracker Modal */}
      {isTimeTrackerExpanded && (
        <TimeTracker 
          isExpanded={true}
          onToggleExpanded={() => setIsTimeTrackerExpanded(false)}
          presets={timeTrackingPresets}
        />
      )}
    </>
  );
}