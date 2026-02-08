import { useState } from 'react';
import { Plus, Edit2, Copy, Trash2, Star } from 'lucide-react';
import { useProfile } from '../contexts/ProfileContext';
import { ProfileModal } from '../components/ProfileModal';
import { WorkProfile } from '../contexts/ProfileContext';

export function Arbeitsprofile() {
  const {
    profiles,
    activeProfile,
    switchProfile,
    deleteProfile,
    duplicateProfile,
    setDefaultProfile,
  } = useProfile();

  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingProfile, setEditingProfile] = useState<WorkProfile | null>(null);

  const handleEdit = (profile: WorkProfile) => {
    setEditingProfile(profile);
    setIsModalOpen(true);
  };

  const handleCreate = () => {
    setEditingProfile(null);
    setIsModalOpen(true);
  };

  const handleCloseModal = () => {
    setIsModalOpen(false);
    setEditingProfile(null);
  };

  const handleDelete = (profileId: string) => {
    if (confirm('Möchten Sie dieses Profil wirklich löschen?')) {
      deleteProfile(profileId);
    }
  };

  const formatLastUsed = (date: Date) => {
    const now = new Date();
    const diffMs = now.getTime() - new Date(date).getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 60) return `vor ${diffMins} Minuten`;
    if (diffHours < 24) return `vor ${diffHours} Stunden`;
    return `vor ${diffDays} Tagen`;
  };

  return (
    <div className="flex-1 overflow-y-auto p-4 md:p-8">
      {/* Header */}
      <div className="mb-8 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-slate-900 mb-2">Arbeitsprofile verwalten</h1>
          <p className="text-slate-500">
            Erstelle und verwalte Profile für verschiedene Arbeits-Workflows
          </p>
        </div>
        <button
          onClick={handleCreate}
          className="px-4 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors flex items-center gap-2"
        >
          <Plus className="w-4 h-4" />
          Neues Profil
        </button>
      </div>

      {/* Profile Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {profiles.map((profile) => (
          <div
            key={profile.id}
            className="bg-white border border-slate-200 rounded-lg p-6 hover:shadow-lg transition-all relative group"
          >
            {/* Header */}
            <div className="flex items-start gap-3 mb-4">
              <span className="text-4xl">{profile.icon}</span>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <h3 className="font-semibold text-slate-900 truncate">{profile.name}</h3>
                  {profile.isDefault && (
                    <Star className="w-4 h-4 text-yellow-500 fill-yellow-500 flex-shrink-0" />
                  )}
                </div>
                <p className="text-sm text-slate-500 line-clamp-2 mt-1">{profile.description}</p>
              </div>
            </div>

            {/* Status Badge */}
            {profile.id === activeProfile?.id && (
              <div className="mb-3">
                <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-emerald-100 text-emerald-700">
                  Aktiv
                </span>
              </div>
            )}

            {/* Last Used */}
            <p className="text-xs text-slate-400 mb-4">
              Zuletzt verwendet: {formatLastUsed(profile.lastUsed)}
            </p>

            {/* Widget Preview */}
            <div className="mb-4 p-3 bg-slate-50 rounded border border-slate-100">
              <p className="text-xs text-slate-500 mb-2">Dashboard Widgets:</p>
              <div className="flex flex-wrap gap-1">
                {profile.dashboardWidgets.slice(0, 3).map((widget) => (
                  <span
                    key={widget.id}
                    className="inline-block px-2 py-0.5 bg-white border border-slate-200 rounded text-xs text-slate-600"
                  >
                    {widget.type}
                  </span>
                ))}
                {profile.dashboardWidgets.length > 3 && (
                  <span className="inline-block px-2 py-0.5 text-xs text-slate-400">
                    +{profile.dashboardWidgets.length - 3}
                  </span>
                )}
              </div>
            </div>

            {/* Module Stats */}
            <div className="mb-4 flex items-center gap-2 text-xs text-slate-500">
              <span>{profile.modules.filter(m => m.enabled).length} Module aktiv</span>
              <span>•</span>
              <span style={{ color: profile.color }}>●</span>
            </div>

            {/* Action Buttons */}
            <div className="flex items-center gap-2">
              {profile.id === activeProfile?.id ? (
                <button
                  disabled
                  className="flex-1 px-3 py-2 bg-slate-100 text-slate-400 rounded-lg text-sm font-medium cursor-not-allowed"
                >
                  Aktiv
                </button>
              ) : (
                <button
                  onClick={() => switchProfile(profile.id)}
                  className="flex-1 px-3 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors text-sm font-medium"
                >
                  Aktivieren
                </button>
              )}

              <button
                onClick={() => handleEdit(profile)}
                className="p-2 text-slate-600 hover:bg-slate-100 rounded-lg transition-colors"
                title="Bearbeiten"
              >
                <Edit2 className="w-4 h-4" />
              </button>

              <button
                onClick={() => duplicateProfile(profile.id)}
                className="p-2 text-slate-600 hover:bg-slate-100 rounded-lg transition-colors"
                title="Duplizieren"
              >
                <Copy className="w-4 h-4" />
              </button>

              {profile.id !== 'default' && (
                <button
                  onClick={() => handleDelete(profile.id)}
                  className="p-2 text-red-600 hover:bg-red-50 rounded-lg transition-colors"
                  title="Löschen"
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              )}
            </div>

            {/* Hover Action - Set as Default */}
            {!profile.isDefault && (
              <button
                onClick={() => setDefaultProfile(profile.id)}
                className="absolute top-4 right-4 opacity-0 group-hover:opacity-100 transition-opacity text-xs text-slate-500 hover:text-yellow-600 flex items-center gap-1"
              >
                <Star className="w-3 h-3" />
                Als Standard
              </button>
            )}
          </div>
        ))}
      </div>

      {/* Profile Modal */}
      <ProfileModal
        isOpen={isModalOpen}
        onClose={handleCloseModal}
        editingProfile={editingProfile}
      />
    </div>
  );
}
