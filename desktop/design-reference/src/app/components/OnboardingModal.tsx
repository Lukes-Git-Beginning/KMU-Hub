import { useState } from 'react';
import { useProfile } from '../contexts/ProfileContext';

const PROFILE_TEMPLATES = [
  {
    id: 'project-management',
    name: 'Projektmanagement',
    icon: '📊',
    description: 'Perfekt für Projekt-Leiter',
    color: '#6366f1',
    modules: ['projects', 'tasks', 'team', 'documents'],
    widgets: ['projects', 'tasks', 'team', 'activity', 'stats'],
  },
  {
    id: 'accounting',
    name: 'Buchhaltung & Finanzen',
    icon: '💰',
    description: 'Für Finanz- und Controlling-Aufgaben',
    color: '#f59e0b',
    modules: ['accounting', 'documents', 'projects'],
    widgets: ['documents', 'stats', 'activity'],
  },
  {
    id: 'team-lead',
    name: 'Team-Lead',
    icon: '👥',
    description: 'Für Teamleiter und Koordinatoren',
    color: '#06b6d4',
    modules: ['team', 'communication', 'tasks', 'projects'],
    widgets: ['team', 'messages', 'tasks', 'activity', 'calendar'],
  },
];

export function OnboardingModal() {
  const { isFirstTime, completeOnboarding, createProfile } = useProfile();
  const [selectedTemplate, setSelectedTemplate] = useState<string | null>(null);

  if (!isFirstTime) return null;

  const handleSelectTemplate = (templateId: string) => {
    const template = PROFILE_TEMPLATES.find(t => t.id === templateId);
    if (!template) return;

    createProfile({
      name: template.name,
      description: template.description,
      icon: template.icon,
      color: template.color,
      isDefault: true,
      dashboardWidgets: template.widgets.map((type, index) => ({
        id: `${type}-${index}`,
        type,
        position: { x: index % 4, y: Math.floor(index / 4) },
        size: { w: 2, h: 1 },
        enabled: true,
      })),
      modules: template.modules.map((id, order) => ({
        id,
        enabled: true,
        order,
      })),
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
    });

    completeOnboarding();
  };

  return (
    <div className="fixed inset-0 bg-gradient-to-br from-emerald-50 to-blue-50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-2xl shadow-2xl max-w-4xl w-full p-8 md:p-12">
        {/* Header */}
        <div className="text-center mb-10">
          <h1 className="text-3xl md:text-4xl font-bold text-slate-900 mb-3">
            Willkommen bei KMU Digital Hub! 👋
          </h1>
          <p className="text-lg text-slate-600 max-w-2xl mx-auto">
            Erstelle dein erstes Arbeitsprofil, um die Plattform auf deine Bedürfnisse anzupassen.
          </p>
        </div>

        {/* Template Selection */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
          {PROFILE_TEMPLATES.map((template) => (
            <button
              key={template.id}
              onClick={() => handleSelectTemplate(template.id)}
              className="group p-6 bg-white border-2 border-slate-200 rounded-xl hover:border-emerald-500 hover:shadow-lg transition-all text-left relative overflow-hidden"
              style={{
                borderColor: selectedTemplate === template.id ? template.color : undefined,
              }}
            >
              {/* Background Gradient */}
              <div
                className="absolute inset-0 opacity-0 group-hover:opacity-10 transition-opacity"
                style={{ background: `linear-gradient(135deg, ${template.color}, transparent)` }}
              />

              {/* Content */}
              <div className="relative z-10">
                <div className="text-5xl mb-4">{template.icon}</div>
                <h3 className="text-xl font-semibold text-slate-900 mb-2">{template.name}</h3>
                <p className="text-sm text-slate-600 mb-4">{template.description}</p>

                <div className="space-y-2">
                  <p className="text-xs font-medium text-slate-500">Enthält:</p>
                  <div className="flex flex-wrap gap-1">
                    {template.modules.slice(0, 3).map((module) => (
                      <span
                        key={module}
                        className="px-2 py-1 bg-slate-100 text-slate-700 rounded text-xs"
                      >
                        {module}
                      </span>
                    ))}
                    {template.modules.length > 3 && (
                      <span className="px-2 py-1 text-slate-500 text-xs">
                        +{template.modules.length - 3}
                      </span>
                    )}
                  </div>
                </div>

                <div className="mt-6">
                  <span
                    className="inline-flex items-center justify-center w-full px-4 py-2 rounded-lg font-medium text-white transition-colors"
                    style={{ backgroundColor: template.color }}
                  >
                    Verwenden
                  </span>
                </div>
              </div>
            </button>
          ))}
        </div>

        {/* Custom Option */}
        <div className="border-t border-slate-200 pt-6">
          <button
            onClick={() => {
              completeOnboarding();
            }}
            className="w-full p-4 border-2 border-dashed border-slate-300 rounded-xl hover:border-emerald-500 hover:bg-emerald-50 transition-all text-center group"
          >
            <div className="text-3xl mb-2">⚙️</div>
            <h3 className="font-semibold text-slate-900 mb-1">Individuell konfigurieren</h3>
            <p className="text-sm text-slate-600">
              Erstelle dein eigenes Profil von Grund auf
            </p>
          </button>
        </div>

        {/* Footer Note */}
        <p className="text-center text-sm text-slate-500 mt-8">
          Du kannst später jederzeit weitere Profile erstellen und anpassen.
        </p>
      </div>
    </div>
  );
}
