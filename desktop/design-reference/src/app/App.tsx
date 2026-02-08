import { useState, useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { Sidebar } from './components/Sidebar';
import { Header } from './components/Header';
import { OnboardingWizard } from './components/OnboardingWizard';
import { OnboardingModal } from './components/OnboardingModal';
import { ProfileModal } from './components/ProfileModal';
import { ProfileSwitchTransition } from './components/ProfileSwitchTransition';
import { ProfileSwitchToast } from './components/ProfileSwitchToast';
import { HelpWidget } from './components/HelpWidget';
import { TimeTrackerWidget } from './components/TimeTrackerWidget';
import { ProfileProvider } from './contexts/ProfileContext';
import { ThemeProvider } from './contexts/ThemeContext';
import { Dashboard } from './screens/Dashboard';
import { Projekte } from './screens/Projekte';
import { ProjektDetail } from './screens/ProjektDetail';
import { Aufgaben } from './screens/Aufgaben';
import { Dokumente } from './screens/Dokumente';
import { Buchhaltung } from './screens/Buchhaltung';
import { Nachrichten } from './screens/Nachrichten';
import { Mails } from './screens/Mails';
import { Kontakte } from './screens/Kontakte';
import { Meetings } from './screens/Meetings';
import { Team } from './screens/Team';
import { Einstellungen } from './screens/Einstellungen';
import { ModuleVerwalten } from './screens/ModuleVerwalten';
import { Arbeitsprofile } from './screens/Arbeitsprofile';
import { Profil } from './screens/Profil';

export default function App() {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [showOnboarding, setShowOnboarding] = useState(false);
  const [showProfileModal, setShowProfileModal] = useState(false);
  const [helpWidgetOpen, setHelpWidgetOpen] = useState(false);
  const [timeTrackingPresets, setTimeTrackingPresets] = useState<string[]>([
    'Projekt Dokumentation',
    'Team Meeting',
    'Code Review',
    'Kundengespräch',
    'E-Mail Bearbeitung',
  ]);

  useEffect(() => {
    // Check if user has completed onboarding
    const hasCompletedOnboarding = localStorage.getItem('onboarding_completed');
    if (!hasCompletedOnboarding) {
      setShowOnboarding(true);
    }
    
    // Load time tracking presets from localStorage
    const savedPresets = localStorage.getItem('time_tracking_presets');
    if (savedPresets) {
      try {
        setTimeTrackingPresets(JSON.parse(savedPresets));
      } catch (e) {
        console.error('Failed to parse time tracking presets:', e);
      }
    }
  }, []);

  const handleOnboardingComplete = () => {
    localStorage.setItem('onboarding_completed', 'true');
    setShowOnboarding(false);
  };

  const handleOnboardingSkip = () => {
    localStorage.setItem('onboarding_completed', 'true');
    setShowOnboarding(false);
  };

  return (
    <BrowserRouter>
      <ThemeProvider>
        <ProfileProvider>
          <div className="flex h-screen bg-slate-50 dark:bg-slate-900">
            <Sidebar isOpen={sidebarOpen} onClose={() => setSidebarOpen(false)} />
            
            <div className="flex-1 flex flex-col overflow-hidden">
              <Header 
                onMenuClick={() => setSidebarOpen(true)} 
                onCreateProfile={() => setShowProfileModal(true)}
                onSupportClick={() => setHelpWidgetOpen(true)}
                timeTrackingPresets={timeTrackingPresets}
              />
              
              <Routes>
                <Route path="/" element={<Dashboard />} />
                <Route path="/notifications" element={<Navigate to="/" replace />} />
                <Route path="/projekte" element={<Projekte />} />
                <Route path="/projekte/:id" element={<ProjektDetail />} />
                <Route path="/aufgaben" element={<Aufgaben />} />
                <Route path="/dokumente" element={<Dokumente />} />
                <Route path="/buchhaltung" element={<Buchhaltung />} />
                <Route path="/nachrichten" element={<Nachrichten />} />
                <Route path="/mails" element={<Mails />} />
                <Route path="/kontakte" element={<Kontakte />} />
                <Route path="/meetings" element={<Meetings />} />
                <Route path="/team" element={<Team />} />
                <Route path="/profil" element={<Profil />} />
                <Route path="/einstellungen" element={<Einstellungen />} />
                <Route path="/module-verwalten" element={<ModuleVerwalten />} />
                <Route path="/arbeitsprofile" element={<Arbeitsprofile />} />
              </Routes>
            </div>

            {/* Time Tracker Widget - always visible */}
            <TimeTrackerWidget />

            {/* Help Widget - controlled by Profile Menu */}
            <HelpWidget isOpen={helpWidgetOpen} onToggle={setHelpWidgetOpen} />

            {/* Profile System Modals */}
            <OnboardingModal />
            <ProfileSwitchTransition />
            <ProfileSwitchToast />
            
            <ProfileModal 
              isOpen={showProfileModal} 
              onClose={() => setShowProfileModal(false)} 
            />

            {showOnboarding && (
              <OnboardingWizard
                onComplete={handleOnboardingComplete}
                onSkip={handleOnboardingSkip}
              />
            )}
          </div>
        </ProfileProvider>
      </ThemeProvider>
    </BrowserRouter>
  );
}