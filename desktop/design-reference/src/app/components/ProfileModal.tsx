import { useState } from 'react';
import { X, ArrowRight, ArrowLeft, Check } from 'lucide-react';
import { useProfile, WorkProfile } from '../contexts/ProfileContext';

interface ProfileModalProps {
  isOpen: boolean;
  onClose: () => void;
  editingProfile?: WorkProfile | null;
}

const AVAILABLE_ICONS = ['📊', '💰', '👥', '✅', '📞', '📈', '💼', '🎯', '📝', '⚙️', '🗂️', '📑', '💬', '🔍', '📅', '🏆'];
const AVAILABLE_COLORS = [
  { name: 'Mint', value: '#059669' },
  { name: 'Indigo', value: '#6366f1' },
  { name: 'Orange', value: '#f59e0b' },
  { name: 'Rot', value: '#ef4444' },
  { name: 'Grün', value: '#10b981' },
  { name: 'Violett', value: '#8b5cf6' },
  { name: 'Cyan', value: '#06b6d4' },
  { name: 'Pink', value: '#ec4899' },
];

const AVAILABLE_WIDGETS = [
  { id: 'projects', label: 'Projekte-Übersicht', defaultSize: { w: 2, h: 1 } },
  { id: 'tasks', label: 'Meine Aufgaben', defaultSize: { w: 2, h: 2 } },
  { id: 'team', label: 'Team-Status', defaultSize: { w: 1, h: 1 } },
  { id: 'activity', label: 'Letzte Aktivitäten', defaultSize: { w: 2, h: 1 } },
  { id: 'documents', label: 'Dokumente (Kürzlich)', defaultSize: { w: 1, h: 1 } },
  { id: 'notifications', label: 'Benachrichtigungen', defaultSize: { w: 1, h: 1 } },
  { id: 'calendar', label: 'Kalender (Heute)', defaultSize: { w: 1, h: 1 } },
  { id: 'quickActions', label: 'Quick-Actions', defaultSize: { w: 1, h: 1 } },
  { id: 'stats', label: 'Statistiken', defaultSize: { w: 2, h: 1 } },
  { id: 'messages', label: 'Nachrichten-Feed', defaultSize: { w: 1, h: 2 } },
];

const DEFAULT_MODULES = [
  { id: 'projects', name: 'Projekte', icon: '📊', enabled: true },
  { id: 'tasks', name: 'Aufgaben', icon: '✅', enabled: true },
  { id: 'documents', name: 'Dokumente', icon: '📄', enabled: true },
  { id: 'communication', name: 'Nachrichten', icon: '💬', enabled: true },
  { id: 'team', name: 'Team', icon: '👥', enabled: true },
  { id: 'accounting', name: 'Buchhaltung', icon: '💰', enabled: false },
];

export function ProfileModal({ isOpen, onClose, editingProfile }: ProfileModalProps) {
  const { createProfile, updateProfile } = useProfile();
  const [step, setStep] = useState(1);
  
  // Step 1: Basic settings
  const [name, setName] = useState(editingProfile?.name || '');
  const [description, setDescription] = useState(editingProfile?.description || '');
  const [selectedIcon, setSelectedIcon] = useState(editingProfile?.icon || '📊');
  const [selectedColor, setSelectedColor] = useState(editingProfile?.color || '#059669');
  
  // Step 2: Dashboard widgets
  const [selectedWidgets, setSelectedWidgets] = useState<string[]>(
    editingProfile?.dashboardWidgets.map(w => w.type) || []
  );
  
  // Step 3: Module priorities
  const [modules, setModules] = useState(
    editingProfile?.modules.map((m, index) => {
      const defaultModule = DEFAULT_MODULES.find(dm => dm.id === m.id);
      return {
        ...m,
        name: defaultModule?.name || '',
        icon: defaultModule?.icon || '',
      };
    }) || DEFAULT_MODULES.map((m, index) => ({ ...m, order: index }))
  );
  
  // Step 4: Quick actions
  const [quickActions, setQuickActions] = useState(editingProfile?.quickActions || []);
  
  // Step 5: Notifications
  const [notifications, setNotifications] = useState(
    editingProfile?.notifications || {
      desktop: true,
      sound: true,
      doNotDisturb: false,
      projectDeadlines: true,
      taskAssignments: true,
      mentions: true,
      documentShares: true,
      dailyDigestTime: '08:00',
      weeklyReportDay: 'Montag',
    }
  );

  const handleSave = () => {
    const profileData = {
      name,
      description,
      icon: selectedIcon,
      color: selectedColor,
      isDefault: false,
      dashboardWidgets: selectedWidgets.map((type, index) => {
        const widget = AVAILABLE_WIDGETS.find(w => w.id === type);
        return {
          id: `${type}-${index}`,
          type,
          position: { x: index % 4, y: Math.floor(index / 4) },
          size: widget?.defaultSize || { w: 1, h: 1 },
          enabled: true,
        };
      }),
      modules: modules.map((m, index) => ({
        id: m.id,
        enabled: m.enabled,
        order: index,
      })),
      quickActions,
      filters: {},
      notifications,
    };

    if (editingProfile) {
      updateProfile(editingProfile.id, profileData);
    } else {
      createProfile(profileData);
    }

    onClose();
  };

  const toggleWidget = (widgetId: string) => {
    setSelectedWidgets(prev =>
      prev.includes(widgetId)
        ? prev.filter(id => id !== widgetId)
        : [...prev, widgetId]
    );
  };

  const toggleModule = (moduleId: string) => {
    setModules(prev =>
      prev.map(m => (m.id === moduleId ? { ...m, enabled: !m.enabled } : m))
    );
  };

  const moveModule = (index: number, direction: 'up' | 'down') => {
    const newModules = [...modules];
    const newIndex = direction === 'up' ? index - 1 : index + 1;
    if (newIndex < 0 || newIndex >= modules.length) return;
    [newModules[index], newModules[newIndex]] = [newModules[newIndex], newModules[index]];
    setModules(newModules);
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-lg shadow-2xl max-w-2xl w-full max-h-[90vh] overflow-hidden flex flex-col">
        {/* Header */}
        <div className="px-6 py-4 border-b border-slate-200 flex items-center justify-between">
          <div>
            <h2 className="text-xl font-semibold text-slate-900">
              {editingProfile ? 'Profil bearbeiten' : 'Neues Arbeitsprofil erstellen'}
            </h2>
            <p className="text-sm text-slate-500 mt-1">Schritt {step} von 5</p>
          </div>
          <button
            onClick={onClose}
            className="p-2 hover:bg-slate-100 rounded-lg transition-colors"
          >
            <X className="w-5 h-5 text-slate-400" />
          </button>
        </div>

        {/* Progress Bar */}
        <div className="h-1 bg-slate-100">
          <div
            className="h-full bg-emerald-500 transition-all duration-300"
            style={{ width: `${(step / 5) * 100}%` }}
          />
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6">
          {/* Step 1: Basic Settings */}
          {step === 1 && (
            <div className="space-y-6">
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-2">
                  Profilname *
                </label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="z.B. Projektmanagement, Buchhaltung, Team-Lead"
                  className="w-full px-4 py-2 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-emerald-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-700 mb-2">
                  Icon auswählen
                </label>
                <div className="grid grid-cols-8 gap-2">
                  {AVAILABLE_ICONS.map((icon) => (
                    <button
                      key={icon}
                      onClick={() => setSelectedIcon(icon)}
                      className={`p-3 text-2xl rounded-lg border-2 transition-colors ${
                        selectedIcon === icon
                          ? 'border-emerald-500 bg-emerald-50'
                          : 'border-slate-200 hover:border-slate-300'
                      }`}
                    >
                      {icon}
                    </button>
                  ))}
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-700 mb-2">
                  Beschreibung (Optional)
                </label>
                <textarea
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Kurze Beschreibung wofür dieses Profil verwendet wird"
                  rows={3}
                  className="w-full px-4 py-2 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-emerald-500 resize-none"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-700 mb-2">
                  Farbe
                </label>
                <div className="grid grid-cols-8 gap-2">
                  {AVAILABLE_COLORS.map((color) => (
                    <button
                      key={color.value}
                      onClick={() => setSelectedColor(color.value)}
                      className={`h-10 rounded-lg border-2 transition-all ${
                        selectedColor === color.value
                          ? 'border-slate-900 scale-110'
                          : 'border-transparent hover:scale-105'
                      }`}
                      style={{ backgroundColor: color.value }}
                      title={color.name}
                    />
                  ))}
                </div>
              </div>
            </div>
          )}

          {/* Step 2: Dashboard Widgets */}
          {step === 2 && (
            <div className="space-y-6">
              <div>
                <h3 className="font-medium text-slate-900 mb-4">Dashboard-Layout konfigurieren</h3>
                <p className="text-sm text-slate-500 mb-6">
                  Wähle die Widgets aus, die in deinem Dashboard angezeigt werden sollen.
                </p>
              </div>

              <div className="space-y-2">
                {AVAILABLE_WIDGETS.map((widget) => (
                  <label
                    key={widget.id}
                    className="flex items-center gap-3 p-3 border border-slate-200 rounded-lg hover:bg-slate-50 cursor-pointer transition-colors"
                  >
                    <input
                      type="checkbox"
                      checked={selectedWidgets.includes(widget.id)}
                      onChange={() => toggleWidget(widget.id)}
                      className="w-4 h-4 text-emerald-600 rounded focus:ring-emerald-500"
                    />
                    <span className="text-sm text-slate-700">{widget.label}</span>
                  </label>
                ))}
              </div>

              <div className="bg-emerald-50 border border-emerald-200 rounded-lg p-4">
                <p className="text-sm text-emerald-800">
                  <strong>{selectedWidgets.length}</strong> Widgets ausgewählt
                </p>
              </div>
            </div>
          )}

          {/* Step 3: Module Priorities */}
          {step === 3 && (
            <div className="space-y-6">
              <div>
                <h3 className="font-medium text-slate-900 mb-4">
                  Welche Module sind in diesem Profil wichtig?
                </h3>
                <p className="text-sm text-slate-500 mb-6">
                  Die Reihenfolge bestimmt die Sidebar-Anordnung. Deaktivierte Module werden ausgeblendet.
                </p>
              </div>

              <div className="space-y-2">
                {modules.map((module, index) => (
                  <div
                    key={module.id}
                    className="flex items-center gap-3 p-3 border border-slate-200 rounded-lg bg-white"
                  >
                    <div className="flex flex-col gap-1">
                      <button
                        onClick={() => moveModule(index, 'up')}
                        disabled={index === 0}
                        className="p-1 hover:bg-slate-100 rounded disabled:opacity-30"
                      >
                        ▲
                      </button>
                      <button
                        onClick={() => moveModule(index, 'down')}
                        disabled={index === modules.length - 1}
                        className="p-1 hover:bg-slate-100 rounded disabled:opacity-30"
                      >
                        ▼
                      </button>
                    </div>

                    <span className="text-xl">{module.icon}</span>
                    <span className="flex-1 text-sm text-slate-700">{module.name}</span>

                    <label className="relative inline-flex items-center cursor-pointer">
                      <input
                        type="checkbox"
                        checked={module.enabled}
                        onChange={() => toggleModule(module.id)}
                        className="sr-only peer"
                      />
                      <div className="w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-emerald-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-emerald-600"></div>
                    </label>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Step 4: Quick Actions */}
          {step === 4 && (
            <div className="space-y-6">
              <div>
                <h3 className="font-medium text-slate-900 mb-4">Schnellzugriffe definieren</h3>
                <p className="text-sm text-slate-500 mb-6">
                  Füge bis zu 6 Schnellzugriffe hinzu (optional).
                </p>
              </div>

              <div className="flex items-center justify-center py-12 border-2 border-dashed border-slate-200 rounded-lg">
                <div className="text-center">
                  <p className="text-sm text-slate-500 mb-3">Noch keine Schnellzugriffe</p>
                  <button className="px-4 py-2 bg-slate-100 text-slate-700 rounded-lg hover:bg-slate-200 transition-colors text-sm">
                    + Schnellzugriff hinzufügen
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* Step 5: Notifications */}
          {step === 5 && (
            <div className="space-y-6">
              <div>
                <h3 className="font-medium text-slate-900 mb-4">Benachrichtigungen für dieses Profil</h3>
              </div>

              <div className="space-y-4">
                <div className="flex items-center justify-between p-3 border border-slate-200 rounded-lg">
                  <span className="text-sm text-slate-700">Desktop-Benachrichtigungen</span>
                  <input
                    type="checkbox"
                    checked={notifications.desktop}
                    onChange={(e) => setNotifications({ ...notifications, desktop: e.target.checked })}
                    className="w-4 h-4 text-emerald-600 rounded focus:ring-emerald-500"
                  />
                </div>

                <div className="flex items-center justify-between p-3 border border-slate-200 rounded-lg">
                  <span className="text-sm text-slate-700">Sound bei neuen Nachrichten</span>
                  <input
                    type="checkbox"
                    checked={notifications.sound}
                    onChange={(e) => setNotifications({ ...notifications, sound: e.target.checked })}
                    className="w-4 h-4 text-emerald-600 rounded focus:ring-emerald-500"
                  />
                </div>

                <div className="flex items-center justify-between p-3 border border-slate-200 rounded-lg">
                  <span className="text-sm text-slate-700">Nicht stören aktivieren</span>
                  <input
                    type="checkbox"
                    checked={notifications.doNotDisturb}
                    onChange={(e) => setNotifications({ ...notifications, doNotDisturb: e.target.checked })}
                    className="w-4 h-4 text-emerald-600 rounded focus:ring-emerald-500"
                  />
                </div>
              </div>

              <div>
                <h4 className="text-sm font-medium text-slate-900 mb-3">Wichtige Updates</h4>
                <div className="space-y-2">
                  {[
                    { key: 'projectDeadlines', label: 'Projekt-Deadline Reminder' },
                    { key: 'taskAssignments', label: 'Neue Aufgaben-Zuweisung' },
                    { key: 'mentions', label: 'Team-Erwähnungen' },
                    { key: 'documentShares', label: 'Dokument-Freigaben' },
                  ].map((item) => (
                    <div key={item.key} className="flex items-center justify-between p-2">
                      <span className="text-sm text-slate-700">{item.label}</span>
                      <input
                        type="checkbox"
                        checked={notifications[item.key as keyof typeof notifications] as boolean}
                        onChange={(e) =>
                          setNotifications({ ...notifications, [item.key]: e.target.checked })
                        }
                        className="w-4 h-4 text-emerald-600 rounded focus:ring-emerald-500"
                      />
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t border-slate-200 flex items-center justify-between">
          <button
            onClick={() => step > 1 ? setStep(step - 1) : onClose()}
            className="px-4 py-2 text-slate-600 hover:text-slate-900 transition-colors flex items-center gap-2"
          >
            <ArrowLeft className="w-4 h-4" />
            {step > 1 ? 'Zurück' : 'Abbrechen'}
          </button>

          {step < 5 ? (
            <button
              onClick={() => setStep(step + 1)}
              disabled={step === 1 && !name}
              className="px-6 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Weiter
              <ArrowRight className="w-4 h-4" />
            </button>
          ) : (
            <button
              onClick={handleSave}
              disabled={!name}
              className="px-6 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <Check className="w-4 h-4" />
              Profil speichern
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
