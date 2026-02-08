import { User, Globe, Palette, Lock, Bell, Link2, CreditCard, Info, Heart, Zap, Shield, Check, Sparkles, Clock, Plus, X } from 'lucide-react';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../components/ui/tabs';
import { Switch } from '../components/ui/switch';
import { Avatar, AvatarFallback } from '../components/ui/avatar';
import { Label } from '../components/ui/label';
import { Input } from '../components/ui/input';
import { Textarea } from '../components/ui/textarea';
import { useTheme } from '../contexts/ThemeContext';
import { useState } from 'react';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../components/ui/select';

export function Einstellungen() {
  const { theme, setTheme } = useTheme();
  const [timePresets, setTimePresets] = useState<string[]>([
    'Projekt Dokumentation',
    'Team Meeting',
    'Code Review',
    'Kundengespräch',
    'E-Mail Bearbeitung',
  ]);
  const [newPreset, setNewPreset] = useState('');

  const addPreset = () => {
    if (newPreset.trim() && !timePresets.includes(newPreset.trim())) {
      setTimePresets([...timePresets, newPreset.trim()]);
      setNewPreset('');
    }
  };

  const removePreset = (preset: string) => {
    setTimePresets(timePresets.filter(p => p !== preset));
  };

  return (
    <div className="flex-1 overflow-y-auto p-4 md:p-8 bg-slate-50 dark:bg-slate-900">
      <div className="max-w-4xl mx-auto">
        <div className="mb-8">
          <h1 className="text-slate-900 dark:text-white mb-1">Einstellungen</h1>
          <p className="text-sm text-slate-500 dark:text-slate-400">Verwalte deine Konto- und Anwendungseinstellungen</p>
        </div>

        <Tabs defaultValue="profile" className="w-full">
          <TabsList className="mb-8">
            <TabsTrigger value="profile" className="gap-2">
              <User className="w-4 h-4" />
              Profil
            </TabsTrigger>
            <TabsTrigger value="timetracking" className="gap-2">
              <Clock className="w-4 h-4" />
              Zeiterfassung
            </TabsTrigger>
            <TabsTrigger value="language" className="gap-2">
              <Globe className="w-4 h-4" />
              Sprache & Region
            </TabsTrigger>
            <TabsTrigger value="appearance" className="gap-2">
              <Palette className="w-4 h-4" />
              Darstellung
            </TabsTrigger>
            <TabsTrigger value="security" className="gap-2">
              <Lock className="w-4 h-4" />
              Sicherheit
            </TabsTrigger>
            <TabsTrigger value="notifications" className="gap-2">
              <Bell className="w-4 h-4" />
              Benachrichtigungen
            </TabsTrigger>
            <TabsTrigger value="about" className="gap-2">
              <Info className="w-4 h-4" />
              Über uns
            </TabsTrigger>
          </TabsList>

          <TabsContent value="profile" className="space-y-6">
            <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-6">
              <h3 className="text-slate-900 dark:text-white mb-4">Profil Informationen</h3>
              
              <div className="flex items-center gap-6 mb-6">
                <Avatar className="w-20 h-20">
                  <AvatarFallback className="bg-emerald-100 dark:bg-emerald-900 text-emerald-700 dark:text-emerald-300 text-xl">
                    AM
                  </AvatarFallback>
                </Avatar>
                <button className="px-4 py-2 border border-slate-200 dark:border-slate-700 text-slate-700 dark:text-slate-300 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors text-sm">
                  Foto ändern
                </button>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="firstname">Vorname</Label>
                  <Input id="firstname" defaultValue="Anna" />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="lastname">Nachname</Label>
                  <Input id="lastname" defaultValue="Müller" />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="email">E-Mail</Label>
                  <Input id="email" type="email" defaultValue="anna.mueller@swissbiz.ch" />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="phone">Telefon</Label>
                  <Input id="phone" defaultValue="+41 79 123 45 67" />
                </div>
                <div className="space-y-2 col-span-2">
                  <Label htmlFor="position">Position</Label>
                  <Input id="position" defaultValue="Product Manager" />
                </div>
                <div className="space-y-2 col-span-2">
                  <Label htmlFor="bio">Bio</Label>
                  <Textarea id="bio" rows={3} placeholder="Erzähle etwas über dich..." />
                </div>
              </div>

              <div className="mt-6">
                <button className="px-6 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors">
                  Änderungen speichern
                </button>
              </div>
            </div>
          </TabsContent>

          <TabsContent value="timetracking" className="space-y-6">
            <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-6">
              <h3 className="text-slate-900 dark:text-white mb-4">Aufgaben-Presets</h3>
              <p className="text-sm text-slate-500 dark:text-slate-400 mb-4">
                Erstelle Vorlagen für häufig genutzte Aufgaben in der Zeiterfassung
              </p>
              
              <div className="space-y-4">
                {/* Add New Preset */}
                <div className="flex gap-2">
                  <Input
                    value={newPreset}
                    onChange={(e) => setNewPreset(e.target.value)}
                    placeholder="Neue Aufgabe hinzufügen..."
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') {
                        addPreset();
                      }
                    }}
                  />
                  <button
                    onClick={addPreset}
                    disabled={!newPreset.trim()}
                    className="flex items-center gap-2 px-4 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    <Plus className="w-4 h-4" />
                    Hinzufügen
                  </button>
                </div>

                {/* Preset List */}
                <div className="space-y-2">
                  {timePresets.map((preset, index) => (
                    <div
                      key={index}
                      className="flex items-center justify-between p-3 bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg"
                    >
                      <span className="text-sm text-slate-900 dark:text-white">{preset}</span>
                      <button
                        onClick={() => removePreset(preset)}
                        className="p-1.5 hover:bg-red-100 dark:hover:bg-red-900/20 rounded transition-colors"
                      >
                        <X className="w-4 h-4 text-red-600 dark:text-red-400" />
                      </button>
                    </div>
                  ))}
                  {timePresets.length === 0 && (
                    <div className="text-center py-8 text-slate-500 dark:text-slate-400">
                      <Clock className="w-12 h-12 mx-auto mb-2 opacity-50" />
                      <p className="text-sm">Noch keine Presets erstellt</p>
                    </div>
                  )}
                </div>

                <div className="mt-6">
                  <button 
                    onClick={() => {
                      localStorage.setItem('time_tracking_presets', JSON.stringify(timePresets));
                    }}
                    className="px-6 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors"
                  >
                    Änderungen speichern
                  </button>
                </div>
              </div>
            </div>
          </TabsContent>

          <TabsContent value="language" className="space-y-6">
            <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-6">
              <h3 className="text-slate-900 dark:text-white mb-4">Sprache & Region</h3>
              
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label>Sprache</Label>
                  <Select defaultValue="de">
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="de">🇩🇪 Deutsch</SelectItem>
                      <SelectItem value="en">🇬🇧 English</SelectItem>
                      <SelectItem value="fr">🇫🇷 Français</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div className="space-y-2">
                  <Label>Zeitzone</Label>
                  <Select defaultValue="zurich">
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="zurich">Europe/Zurich</SelectItem>
                      <SelectItem value="berlin">Europe/Berlin</SelectItem>
                      <SelectItem value="paris">Europe/Paris</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div className="space-y-2">
                  <Label>Datumsformat</Label>
                  <Select defaultValue="dmy">
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="dmy">DD.MM.YYYY</SelectItem>
                      <SelectItem value="mdy">MM/DD/YYYY</SelectItem>
                      <SelectItem value="ymd">YYYY-MM-DD</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div className="flex items-center justify-between py-2">
                  <Label htmlFor="24h">24-Stunden Format verwenden</Label>
                  <Switch id="24h" defaultChecked />
                </div>
              </div>
            </div>
          </TabsContent>

          <TabsContent value="appearance" className="space-y-6">
            <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-6">
              <h3 className="text-slate-900 dark:text-white mb-4">Darstellung</h3>
              
              <div className="space-y-6">
                <div className="space-y-3">
                  <Label>Theme</Label>
                  <div className="grid grid-cols-3 gap-4">
                    {[
                      { value: 'light', label: 'Hell', icon: '☀️' },
                      { value: 'dark', label: 'Dunkel', icon: '🌙' },
                      { value: 'auto', label: 'Automatisch', icon: '🌗' },
                    ].map((themeOption) => (
                      <button
                        key={themeOption.value}
                        className={`p-4 border-2 rounded-lg hover:bg-emerald-50 dark:hover:bg-emerald-900/20 transition-colors ${
                          theme === themeOption.value
                            ? 'border-emerald-600 bg-emerald-50 dark:bg-emerald-900/30'
                            : 'border-slate-200 dark:border-slate-700'
                        }`}
                        onClick={() => setTheme(themeOption.value as 'light' | 'dark' | 'auto')}
                      >
                        <div className="text-2xl mb-2">{themeOption.icon}</div>
                        <div className="text-sm text-slate-900 dark:text-white">{themeOption.label}</div>
                      </button>
                    ))}
                  </div>
                </div>

                <div className="flex items-center justify-between py-2">
                  <Label htmlFor="compact">Kompakte Ansicht</Label>
                  <Switch id="compact" />
                </div>

                <div className="flex items-center justify-between py-2">
                  <Label htmlFor="animations">Animationen reduzieren</Label>
                  <Switch id="animations" />
                </div>
              </div>
            </div>
          </TabsContent>

          <TabsContent value="security" className="space-y-6">
            <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-6">
              <h3 className="text-slate-900 dark:text-white mb-4">Passwort</h3>
              
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="current-password">Aktuelles Passwort</Label>
                  <Input id="current-password" type="password" />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="new-password">Neues Passwort</Label>
                  <Input id="new-password" type="password" />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="confirm-password">Passwort bestätigen</Label>
                  <Input id="confirm-password" type="password" />
                </div>
                <button className="px-6 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors">
                  Passwort ändern
                </button>
              </div>
            </div>

            <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-6">
              <h3 className="text-slate-900 dark:text-white mb-4">Zwei-Faktor-Authentifizierung</h3>
              <p className="text-sm text-slate-500 dark:text-slate-400 mb-4">
                Schütze dein Konto mit zusätzlicher Sicherheit
              </p>
              <button className="px-6 py-2 border border-slate-200 dark:border-slate-700 text-slate-700 dark:text-slate-300 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors">
                Einrichten
              </button>
            </div>
          </TabsContent>

          <TabsContent value="notifications" className="space-y-6">
            <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-6">
              <h3 className="text-slate-900 dark:text-white mb-4">E-Mail-Benachrichtigungen</h3>
              
              <div className="space-y-4">
                <div className="flex items-center justify-between py-2">
                  <div>
                    <Label>Neue Aufgabe zugewiesen</Label>
                    <p className="text-sm text-slate-500 dark:text-slate-400">Erhalte eine E-Mail wenn dir eine Aufgabe zugewiesen wird</p>
                  </div>
                  <Switch defaultChecked />
                </div>
                <div className="flex items-center justify-between py-2">
                  <div>
                    <Label>Projekt-Updates</Label>
                    <p className="text-sm text-slate-500 dark:text-slate-400">Updates zu Projekten in denen du Mitglied bist</p>
                  </div>
                  <Switch defaultChecked />
                </div>
                <div className="flex items-center justify-between py-2">
                  <div>
                    <Label>Nachrichten-Erwähnungen</Label>
                    <p className="text-sm text-slate-500 dark:text-slate-400">Wenn du in einer Nachricht erwähnt wirst</p>
                  </div>
                  <Switch defaultChecked />
                </div>
                <div className="flex items-center justify-between py-2">
                  <div>
                    <Label>Wöchentliche Zusammenfassung</Label>
                    <p className="text-sm text-slate-500 dark:text-slate-400">Erhalte eine wöchentliche Übersicht deiner Aktivitäten</p>
                  </div>
                  <Switch />
                </div>
              </div>
            </div>

            <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-6">
              <h3 className="text-slate-900 dark:text-white mb-4">Push-Benachrichtigungen</h3>
              
              <div className="space-y-4">
                <div className="flex items-center justify-between py-2">
                  <Label>Desktop-Benachrichtigungen</Label>
                  <Switch defaultChecked />
                </div>
                <div className="flex items-center justify-between py-2">
                  <Label>Mobile Benachrichtigungen</Label>
                  <Switch defaultChecked />
                </div>
              </div>
            </div>
          </TabsContent>

          <TabsContent value="about" className="space-y-6">
            {/* Hero Section */}
            <div className="bg-gradient-to-br from-emerald-500 to-emerald-600 rounded-lg p-8 text-white">
              <div className="flex items-center gap-3 mb-4">
                <Sparkles className="w-8 h-8" />
                <h2 className="text-2xl">KMU Digital Hub</h2>
              </div>
              <p className="text-emerald-100 mb-2">Die Schweizer Business-Plattform für KMU</p>
              <p className="text-sm text-emerald-50">
                Entwickelt in der Schweiz, für Schweizer Unternehmen – mit Fokus auf Einfachheit, Fairness und Zuverlässigkeit.
              </p>
            </div>

            {/* 3 Main Differentiators */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
              <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-6">
                <div className="w-12 h-12 bg-emerald-100 dark:bg-emerald-900 rounded-lg flex items-center justify-center mb-4">
                  <Heart className="w-6 h-6 text-emerald-600 dark:text-emerald-400" />
                </div>
                <h3 className="text-slate-900 dark:text-white mb-2">Fair & Transparent</h3>
                <p className="text-sm text-slate-600 dark:text-slate-300 mb-4">
                  Keine versteckten Kosten, keine Abo-Fallen. Klare Preise und einfache Kündigungsmöglichkeiten.
                </p>
                <ul className="space-y-2 text-sm text-slate-600 dark:text-slate-300">
                  <li className="flex items-start gap-2">
                    <Check className="w-4 h-4 text-emerald-600 dark:text-emerald-400 flex-shrink-0 mt-0.5" />
                    <span>Transparente Preisgestaltung</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <Check className="w-4 h-4 text-emerald-600 dark:text-emerald-400 flex-shrink-0 mt-0.5" />
                    <span>30 Tage vor Verlängerung informiert</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <Check className="w-4 h-4 text-emerald-600 dark:text-emerald-400 flex-shrink-0 mt-0.5" />
                    <span>Einfache Kündigung ohne Hürden</span>
                  </li>
                </ul>
              </div>

              <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-6">
                <div className="w-12 h-12 bg-blue-100 dark:bg-blue-900 rounded-lg flex items-center justify-center mb-4">
                  <Zap className="w-6 h-6 text-blue-600 dark:text-blue-400" />
                </div>
                <h3 className="text-slate-900 dark:text-white mb-2">Einfach & Schnell</h3>
                <p className="text-sm text-slate-600 dark:text-slate-300 mb-4">
                  Intuitive Bedienung für nicht-technische Nutzer. Alle Features funktionieren sofort – keine komplexen Setups.
                </p>
                <ul className="space-y-2 text-sm text-slate-600 dark:text-slate-300">
                  <li className="flex items-start gap-2">
                    <Check className="w-4 h-4 text-blue-600 dark:text-blue-400 flex-shrink-0 mt-0.5" />
                    <span>Blitzschnelle Ladezeiten unter 500ms</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <Check className="w-4 h-4 text-blue-600 dark:text-blue-400 flex-shrink-0 mt-0.5" />
                    <span>Keine steile Lernkurve</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <Check className="w-4 h-4 text-blue-600 dark:text-blue-400 flex-shrink-0 mt-0.5" />
                    <span>Native Features ohne Workarounds</span>
                  </li>
                </ul>
              </div>

              <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-6">
                <div className="w-12 h-12 bg-purple-100 dark:bg-purple-900 rounded-lg flex items-center justify-center mb-4">
                  <Shield className="w-6 h-6 text-purple-600 dark:text-purple-400" />
                </div>
                <h3 className="text-slate-900 dark:text-white mb-2">Schweizer Qualität</h3>
                <p className="text-sm text-slate-600 dark:text-slate-300 mb-4">
                  100% Schweizer Hosting mit DSG/DSGVO-Konformität. Persönlicher Support ohne Sales-Druck.
                </p>
                <ul className="space-y-2 text-sm text-slate-600 dark:text-slate-300">
                  <li className="flex items-start gap-2">
                    <Check className="w-4 h-4 text-purple-600 dark:text-purple-400 flex-shrink-0 mt-0.5" />
                    <span>Alle Daten in der Schweiz</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <Check className="w-4 h-4 text-purple-600 dark:text-purple-400 flex-shrink-0 mt-0.5" />
                    <span>Lokale Ansprechpartner</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <Check className="w-4 h-4 text-purple-600 dark:text-purple-400 flex-shrink-0 mt-0.5" />
                    <span>Rechtssichere Buchhaltungsintegration</span>
                  </li>
                </ul>
              </div>
            </div>

            {/* 5 Detailed Features */}
            <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-6">
              <h3 className="text-slate-900 dark:text-white mb-6">Was uns auszeichnet</h3>
              
              <div className="space-y-6">
                <div className="flex gap-4">
                  <div className="w-10 h-10 bg-emerald-50 dark:bg-emerald-900 rounded-lg flex items-center justify-center flex-shrink-0">
                    <span className="text-emerald-600 dark:text-emerald-400 font-medium">1</span>
                  </div>
                  <div>
                    <h4 className="text-sm font-medium text-slate-900 dark:text-white mb-1">Faire Preise & Transparenz</h4>
                    <p className="text-sm text-slate-600 dark:text-slate-300">
                      Anders als andere Lösungen verzichten wir auf versteckte Kosten und Abo-Fallen. Sie erhalten 30 Tage vor 
                      Verlängerung eine E-Mail und können jederzeit mit 30 Tagen Frist kündigen – ohne Rückgewinnungs-Anrufe.
                    </p>
                  </div>
                </div>

                <div className="flex gap-4">
                  <div className="w-10 h-10 bg-blue-50 dark:bg-blue-900 rounded-lg flex items-center justify-center flex-shrink-0">
                    <span className="text-blue-600 dark:text-blue-400 font-medium">2</span>
                  </div>
                  <div>
                    <h4 className="text-sm font-medium text-slate-900 dark:text-white mb-1">Einfache, klare Benutzeroberfläche</h4>
                    <p className="text-sm text-slate-600 dark:text-slate-300">
                      Unsere UI ist speziell für nicht-technische Nutzer konzipiert. Inspiriert von modernen Tools wie Linear, 
                      aber ohne unnötige Komplexität. Alle wichtigen Features sind sofort sichtbar.
                    </p>
                  </div>
                </div>

                <div className="flex gap-4">
                  <div className="w-10 h-10 bg-purple-50 dark:bg-purple-900 rounded-lg flex items-center justify-center flex-shrink-0">
                    <span className="text-purple-600 dark:text-purple-400 font-medium">3</span>
                  </div>
                  <div>
                    <h4 className="text-sm font-medium text-slate-900 dark:text-white mb-1">Native Features ohne Workarounds</h4>
                    <p className="text-sm text-slate-600 dark:text-slate-300">
                      Wiederkehrende Aufgaben, Zeiterfassung, Gantt-Charts – alles funktioniert out-of-the-box. Keine 
                      umständlichen Automationen oder Custom-Datenbanken nötig.
                    </p>
                  </div>
                </div>

                <div className="flex gap-4">
                  <div className="w-10 h-10 bg-yellow-50 dark:bg-yellow-900 rounded-lg flex items-center justify-center flex-shrink-0">
                    <span className="text-yellow-600 dark:text-yellow-400 font-medium">4</span>
                  </div>
                  <div>
                    <h4 className="text-sm font-medium text-slate-900 dark:text-white mb-1">Performance & Offline-Fähigkeit</h4>
                    <p className="text-sm text-slate-600 dark:text-slate-300">
                      Unser Ziel: Dashboard-Ladezeiten unter 500ms. Progressive Web App mit Offline-Synchronisation 
                      für zuverlässige Arbeit auch in Gebieten mit schlechter Netzabdeckung.
                    </p>
                  </div>
                </div>

                <div className="flex gap-4">
                  <div className="w-10 h-10 bg-green-50 dark:bg-green-900 rounded-lg flex items-center justify-center flex-shrink-0">
                    <span className="text-green-600 dark:text-green-400 font-medium">5</span>
                  </div>
                  <div>
                    <h4 className="text-sm font-medium text-slate-900 dark:text-white mb-1">Persönlicher Support</h4>
                    <p className="text-sm text-slate-600 dark:text-slate-300">
                      Unser Support-Team hilft Ihnen weiter – ohne Sales-Druck. Wir sind eine lokale Schweizer GmbH 
                      mit persönlichen Ansprechpartnern, die Ihre Branche verstehen.
                    </p>
                  </div>
                </div>
              </div>
            </div>

            {/* Version & Contact */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-6">
                <h4 className="text-sm font-medium text-slate-900 dark:text-white mb-3">Version & Technologie</h4>
                <div className="space-y-2 text-sm text-slate-600 dark:text-slate-300">
                  <div className="flex justify-between">
                    <span>Version:</span>
                    <span className="font-medium text-slate-900 dark:text-white">1.0.0</span>
                  </div>
                  <div className="flex justify-between">
                    <span>Letztes Update:</span>
                    <span className="font-medium text-slate-900 dark:text-white">Dezember 2024</span>
                  </div>
                  <div className="flex justify-between">
                    <span>Hosting:</span>
                    <span className="font-medium text-slate-900 dark:text-white">🇨🇭 Schweiz</span>
                  </div>
                </div>
              </div>

              <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-6">
                <h4 className="text-sm font-medium text-slate-900 dark:text-white mb-3">Kontakt & Support</h4>
                <div className="space-y-3">
                  <a href="#" className="block text-sm text-emerald-600 dark:text-emerald-400 hover:text-emerald-700 dark:hover:text-emerald-300">
                    📧 support@kmu-digital-hub.ch
                  </a>
                  <a href="#" className="block text-sm text-emerald-600 dark:text-emerald-400 hover:text-emerald-700 dark:hover:text-emerald-300">
                    📞 +41 XX XXX XX XX
                  </a>
                  <a href="#" className="block text-sm text-emerald-600 dark:text-emerald-400 hover:text-emerald-700 dark:hover:text-emerald-300">
                    📄 Dokumentation ansehen
                  </a>
                </div>
              </div>
            </div>
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}