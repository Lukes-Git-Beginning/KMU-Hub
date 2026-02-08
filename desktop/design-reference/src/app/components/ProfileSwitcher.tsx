import { useState, useRef, useEffect } from 'react';
import { Check, ChevronDown, Edit2, Plus, Settings } from 'lucide-react';
import { useProfile } from '../contexts/ProfileContext';
import { Link } from 'react-router-dom';

export function ProfileSwitcher({ onCreateProfile }: { onCreateProfile: () => void }) {
  const { profiles, activeProfile, switchProfile } = useProfile();
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [isOpen]);

  const handleSwitchProfile = (profileId: string) => {
    switchProfile(profileId);
    setIsOpen(false);
  };

  return (
    <div className="relative" ref={dropdownRef}>
      {/* Trigger Button */}
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center gap-2 px-3 py-2 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
      >
        <span className="text-lg">{activeProfile?.icon || '⚙️'}</span>
        <div className="hidden md:flex flex-col items-start">
          <span className="text-xs text-slate-500 dark:text-slate-400">Profil</span>
          <span className="text-sm font-medium text-slate-900 dark:text-white">{activeProfile?.name}</span>
        </div>
        <ChevronDown className={`w-4 h-4 text-slate-400 dark:text-slate-500 transition-transform ${isOpen ? 'rotate-180' : ''}`} />
      </button>

      {/* Dropdown Menu */}
      {isOpen && (
        <div className="absolute right-0 mt-2 w-80 bg-white dark:bg-slate-800 rounded-lg shadow-xl border border-slate-200 dark:border-slate-700 z-50 overflow-hidden">
          {/* Header */}
          <div className="px-4 py-3 border-b border-slate-100 dark:border-slate-700">
            <h3 className="font-semibold text-slate-900 dark:text-white">Arbeitsprofile wechseln</h3>
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
              ⌘/Ctrl + Shift + [1-9] für Schnellwechsel
            </p>
          </div>

          {/* Profile List */}
          <div className="max-h-96 overflow-y-auto">
            {profiles.map((profile, index) => (
              <button
                key={profile.id}
                onClick={() => handleSwitchProfile(profile.id)}
                className={`w-full px-4 py-3 flex items-start gap-3 hover:bg-emerald-50 dark:hover:bg-emerald-900/30 transition-colors border-l-2 ${
                  profile.id === activeProfile?.id
                    ? 'border-emerald-500 bg-emerald-50/50 dark:bg-emerald-900/30'
                    : 'border-transparent'
                }`}
              >
                {/* Icon */}
                <span className="text-2xl flex-shrink-0">{profile.icon}</span>

                {/* Content */}
                <div className="flex-1 text-left">
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-slate-900 dark:text-white">{profile.name}</span>
                    {profile.isDefault && (
                      <span className="text-xs">⭐</span>
                    )}
                    {index < 9 && (
                      <span className="text-xs text-slate-400 dark:text-slate-500 ml-auto">⌘{index + 1}</span>
                    )}
                  </div>
                  <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5 line-clamp-1">
                    {profile.description}
                  </p>
                </div>

                {/* Active Check */}
                {profile.id === activeProfile?.id && (
                  <Check className="w-4 h-4 text-emerald-600 dark:text-emerald-400 flex-shrink-0" />
                )}
              </button>
            ))}
          </div>

          {/* Actions */}
          <div className="border-t border-slate-100 dark:border-slate-700 p-3 space-y-2">
            <button
              onClick={() => {
                onCreateProfile();
                setIsOpen(false);
              }}
              className="w-full px-4 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors flex items-center justify-center gap-2 text-sm font-medium"
            >
              <Plus className="w-4 h-4" />
              Neues Profil erstellen
            </button>
            <Link
              to="/arbeitsprofile"
              onClick={() => setIsOpen(false)}
              className="block w-full text-center text-sm text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white transition-colors py-1"
            >
              Profile verwalten
            </Link>
          </div>
        </div>
      )}
    </div>
  );
}