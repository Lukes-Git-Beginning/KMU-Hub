import { createContext, useContext, useState, useEffect, ReactNode } from 'react';

export interface WorkProfile {
  id: string;
  name: string;
  description: string;
  icon: string;
  color: string;
  isDefault: boolean;
  createdAt: Date;
  lastUsed: Date;
  
  // Dashboard configuration
  dashboardWidgets: {
    id: string;
    type: string;
    position: { x: number; y: number };
    size: { w: number; h: number };
    enabled: boolean;
  }[];
  
  // Module priorities
  modules: {
    id: string;
    enabled: boolean;
    order: number;
  }[];
  
  // Quick actions
  quickActions: {
    id: string;
    type: 'project' | 'task' | 'document' | 'channel' | 'url';
    label: string;
    target: string;
  }[];
  
  // Saved filters
  filters: {
    projects?: string[];
    tasks?: string[];
    documents?: string[];
  };
  
  // Notifications
  notifications: {
    desktop: boolean;
    sound: boolean;
    doNotDisturb: boolean;
    projectDeadlines: boolean;
    taskAssignments: boolean;
    mentions: boolean;
    documentShares: boolean;
    dailyDigestTime: string;
    weeklyReportDay: string;
  };
}

interface ProfileContextType {
  profiles: WorkProfile[];
  activeProfile: WorkProfile | null;
  switchProfile: (profileId: string) => void;
  createProfile: (profile: Omit<WorkProfile, 'id' | 'createdAt' | 'lastUsed'>) => void;
  updateProfile: (profileId: string, updates: Partial<WorkProfile>) => void;
  deleteProfile: (profileId: string) => void;
  duplicateProfile: (profileId: string) => void;
  setDefaultProfile: (profileId: string) => void;
  isFirstTime: boolean;
  completeOnboarding: () => void;
  isSwitching: boolean;
}

const ProfileContext = createContext<ProfileContextType | undefined>(undefined);

const DEFAULT_PROFILES: WorkProfile[] = [
  {
    id: 'default',
    name: 'Standard',
    description: 'Standardansicht mit allen Modulen',
    icon: '⚙️',
    color: '#059669',
    isDefault: true,
    createdAt: new Date(),
    lastUsed: new Date(),
    dashboardWidgets: [],
    modules: [
      { id: 'projects', enabled: true, order: 0 },
      { id: 'tasks', enabled: true, order: 1 },
      { id: 'documents', enabled: true, order: 2 },
      { id: 'communication', enabled: true, order: 3 },
      { id: 'team', enabled: true, order: 4 },
      { id: 'accounting', enabled: true, order: 5 },
    ],
    quickActions: [],
    filters: {},
    notifications: {
      desktop: true,
      sound: true,
      doNotDisturb: false,
      projectDeadlines: true,
      taskAssignments: true,
      mentions: true,
      documentShares: true,
      dailyDigestTime: '08:00',
      weeklyReportDay: 'Montag',
    },
  },
];

export function ProfileProvider({ children }: { children: ReactNode }) {
  const [profiles, setProfiles] = useState<WorkProfile[]>(() => {
    const saved = localStorage.getItem('workProfiles');
    return saved ? JSON.parse(saved) : DEFAULT_PROFILES;
  });

  const [activeProfileId, setActiveProfileId] = useState<string>(() => {
    const saved = localStorage.getItem('activeProfileId');
    return saved || 'default';
  });

  const [isFirstTime, setIsFirstTime] = useState<boolean>(() => {
    const saved = localStorage.getItem('profileOnboardingCompleted');
    return saved !== 'true';
  });

  const [isSwitching, setIsSwitching] = useState(false);

  const activeProfile = profiles.find(p => p.id === activeProfileId) || profiles[0];

  // Save to localStorage whenever profiles change
  useEffect(() => {
    localStorage.setItem('workProfiles', JSON.stringify(profiles));
  }, [profiles]);

  useEffect(() => {
    localStorage.setItem('activeProfileId', activeProfileId);
  }, [activeProfileId]);

  const switchProfile = (profileId: string) => {
    setIsSwitching(true);
    
    // Update last used timestamp
    setProfiles(prev => prev.map(p => 
      p.id === profileId 
        ? { ...p, lastUsed: new Date() }
        : p
    ));

    // Smooth transition
    setTimeout(() => {
      setActiveProfileId(profileId);
      setTimeout(() => setIsSwitching(false), 300);
    }, 200);
  };

  const createProfile = (profile: Omit<WorkProfile, 'id' | 'createdAt' | 'lastUsed'>) => {
    const newProfile: WorkProfile = {
      ...profile,
      id: `profile-${Date.now()}`,
      createdAt: new Date(),
      lastUsed: new Date(),
    };
    setProfiles(prev => [...prev, newProfile]);
  };

  const updateProfile = (profileId: string, updates: Partial<WorkProfile>) => {
    setProfiles(prev => prev.map(p => 
      p.id === profileId ? { ...p, ...updates } : p
    ));
  };

  const deleteProfile = (profileId: string) => {
    if (profileId === 'default') return; // Can't delete default
    setProfiles(prev => prev.filter(p => p.id !== profileId));
    if (activeProfileId === profileId) {
      setActiveProfileId('default');
    }
  };

  const duplicateProfile = (profileId: string) => {
    const profile = profiles.find(p => p.id === profileId);
    if (!profile) return;

    const newProfile: WorkProfile = {
      ...profile,
      id: `profile-${Date.now()}`,
      name: `${profile.name} (Kopie)`,
      isDefault: false,
      createdAt: new Date(),
      lastUsed: new Date(),
    };
    setProfiles(prev => [...prev, newProfile]);
  };

  const setDefaultProfile = (profileId: string) => {
    setProfiles(prev => prev.map(p => ({
      ...p,
      isDefault: p.id === profileId,
    })));
  };

  const completeOnboarding = () => {
    setIsFirstTime(false);
    localStorage.setItem('profileOnboardingCompleted', 'true');
  };

  return (
    <ProfileContext.Provider
      value={{
        profiles,
        activeProfile,
        switchProfile,
        createProfile,
        updateProfile,
        deleteProfile,
        duplicateProfile,
        setDefaultProfile,
        isFirstTime,
        completeOnboarding,
        isSwitching,
      }}
    >
      {children}
    </ProfileContext.Provider>
  );
}

export function useProfile() {
  const context = useContext(ProfileContext);
  if (!context) {
    throw new Error('useProfile must be used within ProfileProvider');
  }
  return context;
}