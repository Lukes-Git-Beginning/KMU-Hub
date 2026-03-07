import { useState } from 'react';
import { 
  Shield, X, Lock, Key, Smartphone, CheckCircle, AlertTriangle, 
  Download, Upload, Clock, Users, Monitor, Eye, Share2, FileText,
  Archive, Trash2, Cloud, HardDrive, Home, Settings, HelpCircle,
  ChevronRight, Copy, RefreshCw, LogOut, BarChart3
} from 'lucide-react';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from './ui/dialog';
import { Tabs, TabsContent, TabsList, TabsTrigger } from './ui/tabs';
import { Progress } from './ui/progress';
import { Badge } from './ui/badge';
import { Label } from './ui/label';
import { Input } from './ui/input';

interface VaultSettingsProps {
  isOpen: boolean;
  onClose: () => void;
}

export function VaultSettings({ isOpen, onClose }: VaultSettingsProps) {
  const [activeTab, setActiveTab] = useState('security');
  const [showToast, setShowToast] = useState(false);
  const [toastMessage, setToastMessage] = useState('');

  const toast = (message: string) => {
    setToastMessage(message);
    setShowToast(true);
    setTimeout(() => setShowToast(false), 3000);
  };

  return (
    <>
      <Dialog open={isOpen} onOpenChange={onClose}>
        <DialogContent className="max-w-4xl max-h-[90vh] overflow-hidden p-0">
          <DialogHeader className="sr-only">
            <DialogTitle>Datentresor-Sicherheit</DialogTitle>
            <DialogDescription>
              Verwalte Verschlüsselung, Zugriff & Sicherheit für deinen Datentresor
            </DialogDescription>
          </DialogHeader>
          
          {/* Header */}
          <div className="px-6 pt-6 pb-4 border-b border-slate-200 dark:border-slate-700 bg-gradient-to-br from-emerald-50 to-white dark:from-emerald-950 dark:to-slate-900">
            <div className="flex items-start justify-between mb-2">
              <div className="flex items-center gap-3">
                <div className="w-12 h-12 bg-emerald-100 dark:bg-emerald-900 rounded-xl flex items-center justify-center">
                  <Shield className="w-6 h-6 text-emerald-600 dark:text-emerald-400" />
                </div>
                <div>
                  <h2 className="text-xl font-semibold text-slate-900 dark:text-white flex items-center gap-2" aria-hidden="true">
                    🔐 Datentresor-Sicherheit
                  </h2>
                  <p className="text-sm text-slate-600 dark:text-slate-400" aria-hidden="true">
                    Verwalte Verschlüsselung, Zugriff & Sicherheit
                  </p>
                </div>
              </div>
              <button
                onClick={onClose}
                className="p-2 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg transition-colors"
                aria-label="Schließen"
              >
                <X className="w-5 h-5 text-slate-500" />
              </button>
            </div>
          </div>

          {/* Tabs */}
          <Tabs value={activeTab} onValueChange={setActiveTab} className="flex flex-col h-full">
            <TabsList className="w-full justify-start border-b border-slate-200 dark:border-slate-700 rounded-none h-auto p-0 bg-transparent px-6">
              <TabsTrigger 
                value="security" 
                className="rounded-none border-b-2 border-transparent data-[state=active]:border-emerald-600 data-[state=active]:text-emerald-600 px-4 py-3 text-sm"
              >
                Sicherheit
              </TabsTrigger>
              <TabsTrigger 
                value="encryption"
                className="rounded-none border-b-2 border-transparent data-[state=active]:border-emerald-600 data-[state=active]:text-emerald-600 px-4 py-3 text-sm"
              >
                Verschlüsselung
              </TabsTrigger>
              <TabsTrigger 
                value="2fa"
                className="rounded-none border-b-2 border-transparent data-[state=active]:border-emerald-600 data-[state=active]:text-emerald-600 px-4 py-3 text-sm"
              >
                Zwei-Faktor-Auth
              </TabsTrigger>
              <TabsTrigger 
                value="access"
                className="rounded-none border-b-2 border-transparent data-[state=active]:border-emerald-600 data-[state=active]:text-emerald-600 px-4 py-3 text-sm"
              >
                Zugriff & Audit
              </TabsTrigger>
              <TabsTrigger 
                value="backups"
                className="rounded-none border-b-2 border-transparent data-[state=active]:border-emerald-600 data-[state=active]:text-emerald-600 px-4 py-3 text-sm"
              >
                Backups
              </TabsTrigger>
            </TabsList>

            <div className="flex-1 overflow-y-auto p-6">
              {/* TAB 1: SECURITY */}
              <TabsContent value="security" className="m-0 space-y-6">
                {/* Security Status Card */}
                <div className="bg-gradient-to-br from-green-50 to-emerald-50 dark:from-green-950 dark:to-emerald-950 border border-green-200 dark:border-green-800 rounded-xl p-6">
                  <div className="flex items-start gap-4 mb-6">
                    <div className="w-12 h-12 bg-green-500 rounded-full flex items-center justify-center">
                      <CheckCircle className="w-6 h-6 text-white" />
                    </div>
                    <div className="flex-1">
                      <h3 className="text-lg font-semibold text-green-900 dark:text-green-100 mb-1">
                        Sicherheit: Optimal
                      </h3>
                      <p className="text-sm text-green-700 dark:text-green-300">
                        Dein Datentresor ist vollständig gesichert
                      </p>
                    </div>
                    <div className="text-center">
                      <div className="relative w-24 h-24">
                        <svg className="w-full h-full transform -rotate-90">
                          <circle
                            cx="48"
                            cy="48"
                            r="42"
                            stroke="currentColor"
                            strokeWidth="8"
                            fill="none"
                            className="text-green-200 dark:text-green-900"
                          />
                          <circle
                            cx="48"
                            cy="48"
                            r="42"
                            stroke="currentColor"
                            strokeWidth="8"
                            fill="none"
                            strokeDasharray={`${2 * Math.PI * 42}`}
                            strokeDashoffset={`${2 * Math.PI * 42 * (1 - 0.95)}`}
                            className="text-green-500"
                            strokeLinecap="round"
                          />
                        </svg>
                        <div className="absolute inset-0 flex flex-col items-center justify-center">
                          <span className="text-2xl font-semibold text-green-900 dark:text-green-100">95</span>
                          <span className="text-xs text-green-700 dark:text-green-300">Sehr Sicher</span>
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* Security Breakdown */}
                  <div className="grid grid-cols-3 gap-4">
                    <div className="bg-white/50 dark:bg-slate-900/50 rounded-lg p-4">
                      <Lock className="w-5 h-5 text-green-600 dark:text-green-400 mb-2" />
                      <div className="text-sm font-medium text-slate-900 dark:text-white mb-1">
                        Verschlüsselung
                      </div>
                      <div className="text-xs text-green-600 dark:text-green-400 mb-1">🟢 Stark</div>
                      <div className="text-xs text-slate-600 dark:text-slate-400">AES-256-GCM aktiviert</div>
                    </div>
                    <div className="bg-white/50 dark:bg-slate-900/50 rounded-lg p-4">
                      <Smartphone className="w-5 h-5 text-green-600 dark:text-green-400 mb-2" />
                      <div className="text-sm font-medium text-slate-900 dark:text-white mb-1">
                        Authentifizierung
                      </div>
                      <div className="text-xs text-green-600 dark:text-green-400 mb-1">🟢 Aktiviert</div>
                      <div className="text-xs text-slate-600 dark:text-slate-400">2FA mit Authenticator</div>
                    </div>
                    <div className="bg-white/50 dark:bg-slate-900/50 rounded-lg p-4">
                      <Users className="w-5 h-5 text-orange-600 dark:text-orange-400 mb-2" />
                      <div className="text-sm font-medium text-slate-900 dark:text-white mb-1">
                        Zugriff
                      </div>
                      <div className="text-xs text-orange-600 dark:text-orange-400 mb-1">🟡 Moderat</div>
                      <div className="text-xs text-slate-600 dark:text-slate-400">5 aktive Sessions</div>
                    </div>
                  </div>
                </div>

                {/* Master Password */}
                <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl p-5">
                  <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-1">Master-Passwort</h3>
                  <p className="text-xs text-slate-600 dark:text-slate-400 mb-4">
                    Schützt alle deine verschlüsselten Dateien
                  </p>
                  <div className="flex items-center justify-between p-3 bg-slate-50 dark:bg-slate-900 rounded-lg mb-4">
                    <div className="flex items-center gap-3">
                      <Key className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
                      <div>
                        <div className="text-sm font-medium text-green-700 dark:text-green-300">✓ Gesetzt</div>
                        <div className="text-xs text-slate-500">Zuletzt geändert: vor 6 Monaten</div>
                      </div>
                    </div>
                    <div className="flex gap-2">
                      <button
                        onClick={() => toast('Passwort-Änderung gestartet')}
                        className="px-3 py-1.5 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg text-sm transition-colors"
                      >
                        Passwort ändern
                      </button>
                      <button className="px-3 py-1.5 bg-slate-100 dark:bg-slate-700 hover:bg-slate-200 dark:hover:bg-slate-600 text-slate-700 dark:text-slate-300 rounded-lg text-sm transition-colors">
                        Zurücksetzen
                      </button>
                    </div>
                  </div>
                </div>

                {/* Encryption Status */}
                <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl p-5">
                  <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4">Verschlüsselungs-Status</h3>
                  <div className="space-y-4">
                    <div className="flex items-start gap-3 p-3 bg-emerald-50 dark:bg-emerald-950 rounded-lg">
                      <Shield className="w-5 h-5 text-emerald-600 dark:text-emerald-400 mt-0.5" />
                      <div className="flex-1">
                        <div className="text-sm font-semibold text-slate-900 dark:text-white">AES-256-GCM</div>
                        <div className="text-xs text-slate-600 dark:text-slate-400 mt-1">Galois/Counter Mode (GCM)</div>
                        <div className="text-xs text-slate-600 dark:text-slate-400">256-Bit Schlüssel</div>
                        <Badge className="mt-2 bg-green-100 dark:bg-green-900 text-green-700 dark:text-green-300 text-xs">
                          ✓ Aktiv
                        </Badge>
                      </div>
                    </div>

                    {/* Key Rotation */}
                    <div className="border-t border-slate-200 dark:border-slate-700 pt-4">
                      <div className="flex items-center justify-between mb-2">
                        <div className="text-sm text-slate-900 dark:text-white">Schlüssel automatisch rotiert</div>
                        <button
                          onClick={() => toast('Schlüssel wird rotiert...')}
                          className="text-xs text-emerald-600 dark:text-emerald-400 hover:underline"
                        >
                          Sofort rotieren
                        </button>
                      </div>
                      <div className="space-y-1">
                        <div className="text-xs text-slate-600 dark:text-slate-400">Jeden 90 Tage</div>
                        <div className="text-xs text-slate-600 dark:text-slate-400">Letzte Rotation: vor 15 Tagen</div>
                        <div className="text-xs text-slate-600 dark:text-slate-400">Nächste Rotation: in 75 Tagen</div>
                      </div>
                      <Progress value={17} className="h-2 mt-2" />
                    </div>
                  </div>
                </div>

                {/* Threat Detection */}
                <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl p-5">
                  <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4">Bedrohungserkennung</h3>
                  <div className="flex items-center gap-3 p-3 bg-green-50 dark:bg-green-950 rounded-lg mb-4">
                    <CheckCircle className="w-5 h-5 text-green-600 dark:text-green-400" />
                    <div>
                      <div className="text-sm font-medium text-green-900 dark:text-green-100">
                        Keine Bedrohungen erkannt
                      </div>
                      <div className="text-xs text-green-700 dark:text-green-300">In den letzten 30 Tagen</div>
                    </div>
                  </div>
                  <div className="space-y-2">
                    <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
                      <input type="checkbox" defaultChecked className="rounded border-slate-300" />
                      Verdächtige Anmeldungen erkennen
                    </label>
                    <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
                      <input type="checkbox" defaultChecked className="rounded border-slate-300" />
                      Unerwartete Zugriffsmuster
                    </label>
                    <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
                      <input type="checkbox" defaultChecked className="rounded border-slate-300" />
                      Brute-Force Schutz
                    </label>
                  </div>
                </div>

                {/* Quick Actions */}
                <div className="space-y-2">
                  <button className="w-full px-4 py-3 bg-slate-100 dark:bg-slate-700 hover:bg-slate-200 dark:hover:bg-slate-600 text-slate-700 dark:text-slate-300 rounded-lg transition-colors flex items-center justify-center gap-2">
                    <RefreshCw className="w-4 h-4" />
                    🔄 Alle Daten re-verschlüsseln
                  </button>
                  <button className="w-full px-4 py-3 bg-slate-100 dark:bg-slate-700 hover:bg-slate-200 dark:hover:bg-slate-600 text-slate-700 dark:text-slate-300 rounded-lg transition-colors flex items-center justify-center gap-2">
                    <Download className="w-4 h-4" />
                    📋 Sicherheitsbericht herunterladen
                  </button>
                  <button className="w-full px-4 py-3 bg-red-100 dark:bg-red-900/30 hover:bg-red-200 dark:hover:bg-red-900/50 text-red-700 dark:text-red-300 rounded-lg transition-colors flex items-center justify-center gap-2">
                    <LogOut className="w-4 h-4" />
                    ⚠️ Notfall: Alle Sessions beenden
                  </button>
                </div>
              </TabsContent>

              {/* TAB 2: ENCRYPTION */}
              <TabsContent value="encryption" className="m-0 space-y-6">
                {/* Encryption Overview */}
                <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl p-5">
                  <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4">Verschlüsselungs-Übersicht</h3>
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <div className="text-xs text-slate-500 dark:text-slate-400 mb-1">Algorithmus</div>
                      <div className="text-sm text-slate-900 dark:text-white flex items-center gap-2">
                        <Lock className="w-4 h-4 text-emerald-600" />
                        AES-256 (Advanced Encryption Standard)
                      </div>
                    </div>
                    <div>
                      <div className="text-xs text-slate-500 dark:text-slate-400 mb-1">Modus</div>
                      <div className="text-sm text-slate-900 dark:text-white">GCM (Galois/Counter Mode) - AEAD</div>
                    </div>
                    <div>
                      <div className="text-xs text-slate-500 dark:text-slate-400 mb-1">Schlüssel-Ableitung</div>
                      <div className="text-sm text-slate-900 dark:text-white">PBKDF2-SHA256 (1,000,000 Iterationen)</div>
                    </div>
                    <div>
                      <div className="text-xs text-slate-500 dark:text-slate-400 mb-1">Nachrichtenintegrität</div>
                      <div className="text-sm text-slate-900 dark:text-white">SHA-256 HMAC</div>
                    </div>
                  </div>
                </div>

                {/* Master Key */}
                <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl p-5">
                  <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4">Master-Schlüssel</h3>
                  <div className="space-y-3">
                    <div className="flex justify-between text-sm">
                      <span className="text-slate-600 dark:text-slate-400">Erstellt:</span>
                      <span className="text-slate-900 dark:text-white">20. Januar 2024</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-slate-600 dark:text-slate-400">Schlüssel-Größe:</span>
                      <span className="text-slate-900 dark:text-white">256 Bit (32 Bytes)</span>
                    </div>
                    <div className="flex justify-between text-sm items-center">
                      <span className="text-slate-600 dark:text-slate-400">Schlüssel-ID:</span>
                      <div className="flex items-center gap-2">
                        <code className="text-xs bg-slate-100 dark:bg-slate-900 px-2 py-1 rounded">
                          K-2024-001-ABC123XYZ
                        </code>
                        <button
                          onClick={() => {
                            navigator.clipboard.writeText('K-2024-001-ABC123XYZ');
                            toast('Schlüssel-ID kopiert');
                          }}
                          className="p-1 hover:bg-slate-100 dark:hover:bg-slate-700 rounded"
                        >
                          <Copy className="w-3 h-3" />
                        </button>
                      </div>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-slate-600 dark:text-slate-400">Status:</span>
                      <span className="text-green-600 dark:text-green-400">✓ Aktiv & Geschützt</span>
                    </div>
                    <div className="pt-3 border-t border-slate-200 dark:border-slate-700">
                      <div className="flex items-center gap-2 mb-2">
                        <Shield className="w-4 h-4 text-green-600 dark:text-green-400" />
                        <span className="text-sm text-green-700 dark:text-green-300">Mit Tresor-Passwort geschützt</span>
                      </div>
                      <button className="w-full px-4 py-2 bg-slate-100 dark:bg-slate-700 hover:bg-slate-200 dark:hover:bg-slate-600 text-slate-700 dark:text-slate-300 rounded-lg text-sm transition-colors">
                        Neue Sicherungskopie erstellen
                      </button>
                    </div>
                  </div>
                </div>

                {/* Data Stats */}
                <div className="grid grid-cols-3 gap-4">
                  <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl p-4 text-center">
                    <FileText className="w-6 h-6 text-emerald-600 dark:text-emerald-400 mx-auto mb-2" />
                    <div className="text-2xl font-semibold text-slate-900 dark:text-white">127</div>
                    <div className="text-xs text-slate-600 dark:text-slate-400 mb-1">Aktiv verschlüsselt</div>
                    <div className="text-xs text-slate-500">1.2 GB</div>
                  </div>
                  <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl p-4 text-center">
                    <Archive className="w-6 h-6 text-blue-600 dark:text-blue-400 mx-auto mb-2" />
                    <div className="text-2xl font-semibold text-slate-900 dark:text-white">34</div>
                    <div className="text-xs text-slate-600 dark:text-slate-400 mb-1">In Archive gespeichert</div>
                    <div className="text-xs text-slate-500">250 MB</div>
                  </div>
                  <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl p-4 text-center">
                    <Trash2 className="w-6 h-6 text-slate-600 dark:text-slate-400 mx-auto mb-2" />
                    <div className="text-2xl font-semibold text-slate-900 dark:text-white">12</div>
                    <div className="text-xs text-slate-600 dark:text-slate-400 mb-1">Im Papierkorb</div>
                    <div className="text-xs text-slate-500">45 MB</div>
                  </div>
                </div>

                {/* Advanced Options */}
                <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl p-5">
                  <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4">Erweiterte Optionen</h3>
                  <div className="space-y-3">
                    <label className="flex items-start gap-3">
                      <input type="checkbox" defaultChecked className="mt-1 rounded border-slate-300" />
                      <div className="flex-1">
                        <div className="text-sm text-slate-900 dark:text-white">End-to-End Verschlüsselung für Freigaben</div>
                        <div className="text-xs text-slate-600 dark:text-slate-400">Nur Empfänger können Dateien öffnen</div>
                      </div>
                    </label>
                    <label className="flex items-start gap-3">
                      <input type="checkbox" className="mt-1 rounded border-slate-300" />
                      <div className="flex-1">
                        <div className="text-sm text-slate-900 dark:text-white">Automatische Datei-Löschung nach Zugriff</div>
                        <div className="text-xs text-slate-600 dark:text-slate-400">Dateien werden nach Herunterladen automatisch gelöscht</div>
                      </div>
                    </label>
                    <label className="flex items-start gap-3">
                      <input type="checkbox" defaultChecked className="mt-1 rounded border-slate-300" />
                      <div className="flex-1">
                        <div className="text-sm text-slate-900 dark:text-white">Zero-Knowledge Modus aktivieren</div>
                        <div className="text-xs text-slate-600 dark:text-slate-400">Server hat keinen Zugriff auf Verschlüsselungsschlüssel</div>
                      </div>
                    </label>
                    <label className="flex items-start gap-3">
                      <input type="checkbox" defaultChecked className="mt-1 rounded border-slate-300" />
                      <div className="flex-1">
                        <div className="text-sm text-slate-900 dark:text-white">Audit-Log Verschlüsselung</div>
                        <div className="text-xs text-slate-600 dark:text-slate-400">Selbst Zugriffslogs sind verschlüsselt</div>
                      </div>
                    </label>
                  </div>
                </div>
              </TabsContent>

              {/* TAB 3: 2FA */}
              <TabsContent value="2fa" className="m-0 space-y-6">
                {/* 2FA Status */}
                <div className="bg-gradient-to-br from-green-50 to-emerald-50 dark:from-green-950 dark:to-emerald-950 border border-green-200 dark:border-green-800 rounded-xl p-5">
                  <div className="flex items-start gap-3">
                    <CheckCircle className="w-6 h-6 text-green-600 dark:text-green-400" />
                    <div className="flex-1">
                      <h3 className="text-sm font-semibold text-green-900 dark:text-green-100 mb-1">
                        ✓ Zwei-Faktor-Authentifizierung aktiv
                      </h3>
                      <p className="text-xs text-green-700 dark:text-green-300 mb-1">
                        Ihr Konto ist mit 2FA geschützt
                      </p>
                      <p className="text-xs text-green-600 dark:text-green-400">
                        Zuletzt verifiziert: vor 2h
                      </p>
                    </div>
                  </div>
                </div>

                {/* Active Methods */}
                <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl p-5">
                  <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4">Aktivierte Methoden</h3>
                  <div className="space-y-4">
                    {/* Authenticator App */}
                    <div className="flex items-start gap-3 p-3 bg-slate-50 dark:bg-slate-900 rounded-lg">
                      <Smartphone className="w-5 h-5 text-emerald-600 dark:text-emerald-400 mt-0.5" />
                      <div className="flex-1">
                        <div className="text-sm font-medium text-slate-900 dark:text-white mb-1">Authenticator App</div>
                        <div className="text-xs text-slate-600 dark:text-slate-400 mb-1">Time-based One-Time Password (TOTP)</div>
                        <div className="text-xs text-slate-500">Google Authenticator / Authy / Microsoft Authenticator</div>
                        <div className="flex items-center gap-2 mt-2">
                          <Badge className="bg-green-100 dark:bg-green-900 text-green-700 dark:text-green-300 text-xs">
                            ✓ Aktiviert
                          </Badge>
                          <span className="text-xs text-slate-500">vor 6 Monaten hinzugefügt</span>
                        </div>
                      </div>
                      <button className="px-3 py-1.5 bg-red-100 dark:bg-red-900/30 hover:bg-red-200 dark:hover:bg-red-900/50 text-red-700 dark:text-red-300 rounded-lg text-xs transition-colors">
                        Entfernen
                      </button>
                    </div>

                    {/* Backup Codes */}
                    <div className="flex items-start gap-3 p-3 bg-slate-50 dark:bg-slate-900 rounded-lg">
                      <Key className="w-5 h-5 text-emerald-600 dark:text-emerald-400 mt-0.5" />
                      <div className="flex-1">
                        <div className="text-sm font-medium text-slate-900 dark:text-white mb-1">Backup-Codes</div>
                        <div className="text-xs text-slate-600 dark:text-slate-400 mb-1">10-stellige Wiederherstellungscodes</div>
                        <div className="flex items-center gap-2 mt-2">
                          <Badge className="bg-orange-100 dark:bg-orange-900/30 text-orange-700 dark:text-orange-300 text-xs">
                            ✓ 8 verbleibend von 10
                          </Badge>
                          <span className="text-xs text-slate-500">2 bereits verwendet</span>
                        </div>
                      </div>
                      <button className="px-3 py-1.5 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg text-xs transition-colors">
                        Neue Codes generieren
                      </button>
                    </div>

                    {/* Security Key */}
                    <div className="flex items-start gap-3 p-3 bg-slate-50 dark:bg-slate-900 rounded-lg">
                      <Lock className="w-5 h-5 text-slate-400 mt-0.5" />
                      <div className="flex-1">
                        <div className="text-sm font-medium text-slate-900 dark:text-white mb-1">FIDO2/WebAuthn Security Key</div>
                        <div className="text-xs text-slate-600 dark:text-slate-400 mb-1">Hardware-Sicherheitsschlüssel (z.B. YubiKey)</div>
                        <div className="flex items-center gap-2 mt-2">
                          <Badge className="bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300 text-xs">
                            ❌ Nicht aktiviert
                          </Badge>
                        </div>
                      </div>
                      <button className="px-3 py-1.5 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg text-xs transition-colors">
                        Aktivieren
                      </button>
                    </div>
                  </div>
                </div>

                {/* 2FA Settings */}
                <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl p-5">
                  <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4">2FA-Einstellungen</h3>
                  <div className="space-y-3">
                    <label className="flex items-start gap-3">
                      <input type="checkbox" defaultChecked className="mt-1 rounded border-slate-300" />
                      <div className="flex-1">
                        <div className="text-sm text-slate-900 dark:text-white">2FA für Vault-Zugriff erforderlich</div>
                        <div className="text-xs text-slate-600 dark:text-slate-400">2FA immer beim Öffnen des Tresors</div>
                      </div>
                    </label>
                    <label className="flex items-start gap-3">
                      <input type="checkbox" defaultChecked className="mt-1 rounded border-slate-300" />
                      <div className="flex-1">
                        <div className="text-sm text-slate-900 dark:text-white">2FA für neue Geräte erforderlich</div>
                        <div className="text-xs text-slate-600 dark:text-slate-400">2FA bei Login von unbekanntem Gerät</div>
                      </div>
                    </label>
                    <label className="flex items-start gap-3">
                      <input type="checkbox" defaultChecked className="mt-1 rounded border-slate-300" />
                      <div className="flex-1">
                        <div className="text-sm text-slate-900 dark:text-white">Vertrauenswürdige Geräte merken</div>
                        <div className="text-xs text-slate-600 dark:text-slate-400">30 Tage lang keine 2FA auf diesem Gerät</div>
                      </div>
                    </label>
                  </div>
                </div>
              </TabsContent>

              {/* TAB 4: ACCESS & AUDIT */}
              <TabsContent value="access" className="m-0 space-y-6">
                {/* Active Sessions */}
                <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl p-5">
                  <div className="flex items-center justify-between mb-4">
                    <h3 className="text-sm font-semibold text-slate-900 dark:text-white">Aktive Sitzungen</h3>
                    <button className="text-xs text-red-600 dark:text-red-400 hover:underline">
                      Alle anderen beenden
                    </button>
                  </div>
                  <div className="space-y-3">
                    {[
                      { device: 'MacBook Pro (Safari)', ip: '192.168.1.100', location: 'Zürich, Schweiz', time: '14:30 heute', active: true, current: true },
                      { device: 'iPhone 12 (App)', ip: '192.168.1.101', location: 'Zürich, Schweiz', time: 'vor 2h', active: false, current: false },
                      { device: 'Windows Desktop (Chrome)', ip: '192.168.1.102', location: 'Zürich, Schweiz', time: 'vor 1 Tag', active: false, current: false },
                    ].map((session, idx) => (
                      <div key={idx} className="flex items-start gap-3 p-3 bg-slate-50 dark:bg-slate-900 rounded-lg">
                        <Monitor className="w-5 h-5 text-slate-600 dark:text-slate-400 mt-0.5" />
                        <div className="flex-1">
                          <div className="flex items-center gap-2 mb-1">
                            <span className="text-sm font-medium text-slate-900 dark:text-white">{session.device}</span>
                            {session.current && (
                              <Badge className="bg-emerald-100 dark:bg-emerald-900 text-emerald-700 dark:text-emerald-300 text-xs">
                                Aktuell
                              </Badge>
                            )}
                          </div>
                          <div className="text-xs text-slate-600 dark:text-slate-400">
                            {session.ip} • {session.location}
                          </div>
                          <div className="text-xs text-slate-500 mt-1">
                            Login: {session.time}
                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          <span className={`text-xs ${session.active ? 'text-green-600 dark:text-green-400' : 'text-slate-400'}`}>
                            {session.active ? '🟢 Aktiv' : '⚪ Inaktiv'}
                          </span>
                          {!session.current && (
                            <button className="px-2 py-1 bg-slate-200 dark:bg-slate-700 hover:bg-slate-300 dark:hover:bg-slate-600 text-slate-700 dark:text-slate-300 rounded text-xs transition-colors">
                              Beenden
                            </button>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>

                {/* Access Logs */}
                <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl p-5">
                  <div className="flex items-center justify-between mb-4">
                    <h3 className="text-sm font-semibold text-slate-900 dark:text-white">Zugriffs-Protokoll</h3>
                    <button className="text-xs text-emerald-600 dark:text-emerald-400 hover:underline flex items-center gap-1">
                      <Download className="w-3 h-3" />
                      Als CSV exportieren
                    </button>
                  </div>
                  <div className="space-y-2 max-h-64 overflow-y-auto">
                    {[
                      { icon: Lock, action: 'Tresor geöffnet', user: 'Achim Weber', time: 'vor 2 min', success: true },
                      { icon: Download, action: 'Datei heruntergeladen "Vertrag.pdf"', user: 'Achim Weber', time: 'vor 1h', success: true },
                      { icon: Share2, action: 'Zugriff gewährt an Lisa Meier', user: 'Achim Weber', time: 'vor 3h', success: true },
                      { icon: RefreshCw, action: 'Schlüssel rotiert', user: 'System', time: 'vor 1 Tag', success: true },
                      { icon: AlertTriangle, action: 'Mehrere fehlgeschlagene Login-Versuche', user: 'Unbekannt', time: 'vor 2 Tage', success: false },
                    ].map((log, idx) => {
                      const Icon = log.icon;
                      return (
                        <div key={idx} className="flex items-start gap-2 p-2 hover:bg-slate-50 dark:hover:bg-slate-900 rounded">
                          <Icon className={`w-4 h-4 mt-0.5 ${log.success ? 'text-slate-600 dark:text-slate-400' : 'text-red-600 dark:text-red-400'}`} />
                          <div className="flex-1 min-w-0">
                            <div className="text-xs font-medium text-slate-900 dark:text-white truncate">{log.action}</div>
                            <div className="text-xs text-slate-600 dark:text-slate-400">{log.user} • {log.time}</div>
                          </div>
                          <span className={`text-xs ${log.success ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'}`}>
                            {log.success ? '✓' : '❌'}
                          </span>
                        </div>
                      );
                    })}
                  </div>
                </div>

                {/* Suspicious Activity */}
                <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl p-5">
                  <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4">Verdächtige Aktivität</h3>
                  <div className="flex items-center gap-3 p-3 bg-green-50 dark:bg-green-950 rounded-lg">
                    <CheckCircle className="w-5 h-5 text-green-600 dark:text-green-400" />
                    <div>
                      <div className="text-sm text-green-900 dark:text-green-100">Keine verdächtige Aktivität erkannt</div>
                      <div className="text-xs text-green-700 dark:text-green-300">In den letzten 30 Tagen</div>
                    </div>
                  </div>
                </div>
              </TabsContent>

              {/* TAB 5: BACKUPS */}
              <TabsContent value="backups" className="m-0 space-y-6">
                {/* Backup Status */}
                <div className="bg-gradient-to-br from-blue-50 to-indigo-50 dark:from-blue-950 dark:to-indigo-950 border border-blue-200 dark:border-blue-800 rounded-xl p-5">
                  <div className="flex items-start gap-3">
                    <CheckCircle className="w-6 h-6 text-blue-600 dark:text-blue-400" />
                    <div className="flex-1">
                      <h3 className="text-sm font-semibold text-blue-900 dark:text-blue-100 mb-1">
                        ✓ Automatische Sicherungen aktiv
                      </h3>
                      <p className="text-xs text-blue-700 dark:text-blue-300 mb-2">
                        Tägliche Sicherungen werden erstellt
                      </p>
                      <div className="flex items-center gap-4 text-xs">
                        <div>
                          <Clock className="w-3 h-3 inline mr-1" />
                          Letzte Sicherung: vor 2 Stunden
                        </div>
                        <div>Größe: 1.2 GB</div>
                      </div>
                    </div>
                    <button
                      onClick={() => toast('Sicherung wird erstellt...')}
                      className="px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-xs transition-colors"
                    >
                      Jetzt sichern
                    </button>
                  </div>
                </div>

                {/* Backup Locations */}
                <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl p-5">
                  <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4">Sicherungsorte</h3>
                  <div className="space-y-3">
                    <div className="flex items-start gap-3 p-3 bg-slate-50 dark:bg-slate-900 rounded-lg">
                      <Cloud className="w-5 h-5 text-blue-600 dark:text-blue-400 mt-0.5" />
                      <div className="flex-1">
                        <div className="text-sm font-medium text-slate-900 dark:text-white mb-1">Cloud Speicher (Regional EU)</div>
                        <div className="text-xs text-slate-600 dark:text-slate-400 mb-1">Frankfurt, Deutschland (3x redundant)</div>
                        <div className="text-xs text-slate-500">AES-256 verschlüsselt</div>
                        <Badge className="mt-2 bg-green-100 dark:bg-green-900 text-green-700 dark:text-green-300 text-xs">
                          ✓ Aktiv
                        </Badge>
                      </div>
                    </div>
                    <div className="flex items-start gap-3 p-3 bg-slate-50 dark:bg-slate-900 rounded-lg">
                      <HardDrive className="w-5 h-5 text-orange-600 dark:text-orange-400 mt-0.5" />
                      <div className="flex-1">
                        <div className="text-sm font-medium text-slate-900 dark:text-white mb-1">USB Sicherungslaufwerk</div>
                        <div className="text-xs text-slate-600 dark:text-slate-400 mb-1">Letzte Sicherung: vor 1 Woche</div>
                        <Badge className="mt-2 bg-orange-100 dark:bg-orange-900/30 text-orange-700 dark:text-orange-300 text-xs">
                          ⚠️ Nicht verbunden
                        </Badge>
                      </div>
                      <button className="px-3 py-1.5 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg text-xs transition-colors">
                        Verbinden
                      </button>
                    </div>
                  </div>
                </div>

                {/* Restore Options */}
                <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl p-5">
                  <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4">Wiederherstellungsoptionen</h3>
                  <div className="space-y-3">
                    <button className="w-full px-4 py-3 bg-slate-100 dark:bg-slate-700 hover:bg-slate-200 dark:hover:bg-slate-600 text-slate-700 dark:text-slate-300 rounded-lg transition-colors flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <Download className="w-4 h-4" />
                        <span className="text-sm">📥 Sicherung durchsuchen & wiederherstellen</span>
                      </div>
                      <ChevronRight className="w-4 h-4" />
                    </button>
                    <button className="w-full px-4 py-3 bg-red-100 dark:bg-red-900/30 hover:bg-red-200 dark:hover:bg-red-900/50 text-red-700 dark:text-red-300 rounded-lg transition-colors flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <RefreshCw className="w-4 h-4" />
                        <span className="text-sm">🔄 Gesamte Sicherung wiederherstellen</span>
                      </div>
                      <ChevronRight className="w-4 h-4" />
                    </button>
                  </div>
                  <p className="text-xs text-slate-600 dark:text-slate-400 mt-3">
                    ⚠️ Wiederherstellung ersetzt alle aktuellen Daten
                  </p>
                </div>
              </TabsContent>
            </div>
          </Tabs>

          {/* Footer */}
          <div className="px-6 py-4 border-t border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900">
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-4 text-xs text-slate-600 dark:text-slate-400">
                <button className="flex items-center gap-1 hover:text-emerald-600 dark:hover:text-emerald-400">
                  <HelpCircle className="w-3 h-3" />
                  Sicherheits-FAQ
                </button>
                <button className="flex items-center gap-1 hover:text-emerald-600 dark:hover:text-emerald-400">
                  Support kontaktieren
                </button>
              </div>
            </div>
            <div className="flex gap-3">
              <button
                onClick={onClose}
                className="flex-1 px-4 py-2 bg-slate-200 dark:bg-slate-700 hover:bg-slate-300 dark:hover:bg-slate-600 text-slate-700 dark:text-slate-300 rounded-lg transition-colors"
              >
                Abbrechen
              </button>
              <button
                onClick={() => {
                  toast('✅ Einstellungen gespeichert');
                  setTimeout(() => onClose(), 1000);
                }}
                className="flex-1 px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg transition-colors"
              >
                Änderungen speichern
              </button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* Toast */}
      {showToast && (
        <div className="fixed bottom-4 right-4 z-50 animate-in slide-in-from-bottom duration-300">
          <div className="px-4 py-3 bg-green-50 dark:bg-green-900/30 border border-green-200 dark:border-green-800 text-green-700 dark:text-green-300 rounded-xl shadow-lg flex items-center gap-2">
            <CheckCircle className="w-4 h-4" />
            <p className="text-sm font-medium">{toastMessage}</p>
          </div>
        </div>
      )}
    </>
  );
}