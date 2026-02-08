import { useEffect, useState } from 'react';
import { useProfile } from '../contexts/ProfileContext';

export function ProfileSwitchToast() {
  const { activeProfile } = useProfile();
  const [show, setShow] = useState(false);
  const [previousProfileId, setPreviousProfileId] = useState<string | null>(null);

  useEffect(() => {
    if (activeProfile && previousProfileId && activeProfile.id !== previousProfileId) {
      setShow(true);
      const timer = setTimeout(() => {
        setShow(false);
      }, 2000);

      return () => clearTimeout(timer);
    }

    if (activeProfile) {
      setPreviousProfileId(activeProfile.id);
    }
  }, [activeProfile, previousProfileId]);

  if (!show || !activeProfile) return null;

  return (
    <div className="fixed top-20 right-8 z-50 animate-slide-in-right">
      <div className="bg-white border border-slate-200 rounded-lg shadow-lg px-4 py-3 flex items-center gap-3 max-w-sm">
        <span className="text-2xl">{activeProfile.icon}</span>
        <div className="flex-1">
          <p className="text-sm font-medium text-slate-900">Gewechselt zu:</p>
          <p className="text-sm text-slate-600">{activeProfile.name}</p>
        </div>
      </div>
    </div>
  );
}
