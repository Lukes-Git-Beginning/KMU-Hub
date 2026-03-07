import { useEffect, useState } from 'react';
import { useProfile } from '../contexts/ProfileContext';

export function ProfileSwitchTransition() {
  const { isSwitching, activeProfile } = useProfile();
  const [showTransition, setShowTransition] = useState(false);

  useEffect(() => {
    if (isSwitching) {
      setShowTransition(true);
      const timer = setTimeout(() => {
        setShowTransition(false);
      }, 500);
      return () => clearTimeout(timer);
    }
  }, [isSwitching]);

  if (!showTransition || !activeProfile) return null;

  return (
    <div className="fixed inset-0 bg-white z-50 flex items-center justify-center animate-fade-in-out">
      <div className="text-center">
        <div className="text-7xl mb-4 animate-bounce">{activeProfile.icon}</div>
        <h2 className="text-2xl font-semibold text-slate-900">{activeProfile.name}</h2>
        <div className="mt-4 flex items-center justify-center gap-2">
          <div className="w-2 h-2 bg-emerald-500 rounded-full animate-pulse"></div>
          <div className="w-2 h-2 bg-emerald-500 rounded-full animate-pulse delay-75"></div>
          <div className="w-2 h-2 bg-emerald-500 rounded-full animate-pulse delay-150"></div>
        </div>
      </div>
    </div>
  );
}
