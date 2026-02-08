import { useState } from 'react';
import { X, Check, ArrowRight, ArrowLeft, Sparkles, Shield, Zap } from 'lucide-react';
import { Progress } from './ui/progress';

interface OnboardingWizardProps {
  onComplete: () => void;
  onSkip: () => void;
}

const steps = [
  {
    id: 1,
    title: 'Willkommen im KMU Digital Hub',
    description: 'Ihre All-in-One Plattform für Schweizer KMU',
    icon: Sparkles,
    content: (
      <div className="text-center py-8">
        <div className="w-20 h-20 bg-emerald-100 rounded-full flex items-center justify-center mx-auto mb-6">
          <Sparkles className="w-10 h-10 text-emerald-600" />
        </div>
        <h3 className="text-xl text-slate-900 mb-3">Herzlich Willkommen!</h3>
        <p className="text-slate-600 mb-6 max-w-md mx-auto">
          Der KMU Digital Hub vereint Projektverwaltung, Dokumentenmanagement, Team-Kommunikation und Buchhaltung
          in einer einzigen Plattform.
        </p>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 max-w-2xl mx-auto">
          <div className="bg-slate-50 rounded-lg p-4">
            <Shield className="w-8 h-8 text-emerald-600 mx-auto mb-2" />
            <p className="text-sm text-slate-700">100% Swiss Hosting</p>
            <p className="text-xs text-slate-500 mt-1">DSG/DSGVO-konform</p>
          </div>
          <div className="bg-slate-50 rounded-lg p-4">
            <Zap className="w-8 h-8 text-blue-600 mx-auto mb-2" />
            <p className="text-sm text-slate-700">Einfach & Intuitiv</p>
            <p className="text-xs text-slate-500 mt-1">Für nicht-technische Nutzer</p>
          </div>
          <div className="bg-slate-50 rounded-lg p-4">
            <Check className="w-8 h-8 text-purple-600 mx-auto mb-2" />
            <p className="text-sm text-slate-700">Alles in Einem</p>
            <p className="text-xs text-slate-500 mt-1">Keine Zettelwirtschaft mehr</p>
          </div>
        </div>
      </div>
    ),
  },
  {
    id: 2,
    title: 'Wählen Sie Ihre Module',
    description: 'Aktivieren Sie nur die Features, die Sie benötigen',
    icon: Check,
    content: (
      <div className="py-8">
        <h3 className="text-xl text-slate-900 mb-6 text-center">Welche Module möchten Sie nutzen?</h3>
        <div className="space-y-3 max-w-2xl mx-auto">
          {[
            { name: 'Projektverwaltung', description: 'Kanban, Gantt, Zeiterfassung', default: true },
            { name: 'Aufgabenverwaltung', description: 'To-Dos und Deadlines', default: true },
            { name: 'Dokumentenmanagement', description: 'Zentrale Ablage', default: true },
            { name: 'Buchhaltung', description: 'Bexio, Abacus Integration', default: false },
            { name: 'Kommunikation', description: 'Team-Chat', default: true },
            { name: 'Team & CRM', description: 'Mitarbeiter und Kontakte', default: true },
          ].map((module, idx) => (
            <label
              key={idx}
              className="flex items-center justify-between p-4 bg-white border border-slate-200 rounded-lg cursor-pointer hover:bg-slate-50 transition-colors"
            >
              <div className="flex items-center gap-3">
                <input
                  type="checkbox"
                  defaultChecked={module.default}
                  className="w-5 h-5 text-emerald-600 rounded"
                />
                <div>
                  <p className="text-sm text-slate-900">{module.name}</p>
                  <p className="text-xs text-slate-500">{module.description}</p>
                </div>
              </div>
            </label>
          ))}
        </div>
        <p className="text-xs text-slate-500 text-center mt-6">
          Sie können Module jederzeit in den Einstellungen aktivieren oder deaktivieren
        </p>
      </div>
    ),
  },
  {
    id: 3,
    title: 'Erstes Projekt anlegen',
    description: 'Starten Sie mit Ihrem ersten Projekt',
    icon: Check,
    content: (
      <div className="py-8">
        <h3 className="text-xl text-slate-900 mb-6 text-center">Erstellen Sie Ihr erstes Projekt</h3>
        <div className="max-w-md mx-auto space-y-4">
          <div>
            <label className="block text-sm text-slate-700 mb-2">Projekt-Name</label>
            <input
              type="text"
              placeholder="z.B. Website Relaunch"
              className="w-full px-4 py-2 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-emerald-500"
            />
          </div>
          <div>
            <label className="block text-sm text-slate-700 mb-2">Beschreibung</label>
            <textarea
              placeholder="Kurze Projektbeschreibung..."
              rows={3}
              className="w-full px-4 py-2 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-emerald-500"
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-slate-700 mb-2">Priorität</label>
              <select className="w-full px-4 py-2 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-emerald-500">
                <option>Niedrig</option>
                <option>Mittel</option>
                <option>Hoch</option>
              </select>
            </div>
            <div>
              <label className="block text-sm text-slate-700 mb-2">Deadline</label>
              <input
                type="date"
                className="w-full px-4 py-2 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-emerald-500"
              />
            </div>
          </div>
        </div>
        <p className="text-xs text-slate-500 text-center mt-6">
          Sie können diesen Schritt überspringen und später Projekte anlegen
        </p>
      </div>
    ),
  },
  {
    id: 4,
    title: 'Alles bereit!',
    description: 'Sie können jetzt loslegen',
    icon: Check,
    content: (
      <div className="text-center py-8">
        <div className="w-20 h-20 bg-emerald-100 rounded-full flex items-center justify-center mx-auto mb-6">
          <Check className="w-10 h-10 text-emerald-600" />
        </div>
        <h3 className="text-xl text-slate-900 mb-3">Perfekt! Sie sind startklar.</h3>
        <p className="text-slate-600 mb-8 max-w-md mx-auto">
          Ihr KMU Digital Hub ist eingerichtet und bereit für den Einsatz. Entdecken Sie die verschiedenen Module
          und optimieren Sie Ihre Workflows.
        </p>
        <div className="bg-blue-50 border border-blue-200 rounded-lg p-6 max-w-lg mx-auto">
          <h4 className="text-sm text-blue-900 mb-2">💡 Tipp für den Start</h4>
          <p className="text-sm text-blue-700">
            Beginnen Sie mit dem Dashboard, um einen Überblick über alle Module zu erhalten. Bei Fragen steht
            Ihnen unser Support-Team jederzeit zur Verfügung.
          </p>
        </div>
        <div className="mt-8 flex flex-wrap justify-center gap-4 text-sm">
          <a href="#" className="text-emerald-600 hover:text-emerald-700 underline">
            Dokumentation ansehen
          </a>
          <span className="text-slate-300">•</span>
          <a href="#" className="text-emerald-600 hover:text-emerald-700 underline">
            Video-Tutorials
          </a>
          <span className="text-slate-300">•</span>
          <a href="#" className="text-emerald-600 hover:text-emerald-700 underline">
            Support kontaktieren
          </a>
        </div>
      </div>
    ),
  },
];

export function OnboardingWizard({ onComplete, onSkip }: OnboardingWizardProps) {
  const [currentStep, setCurrentStep] = useState(0);

  const handleNext = () => {
    if (currentStep < steps.length - 1) {
      setCurrentStep(currentStep + 1);
    } else {
      onComplete();
    }
  };

  const handleBack = () => {
    if (currentStep > 0) {
      setCurrentStep(currentStep - 1);
    }
  };

  const progress = ((currentStep + 1) / steps.length) * 100;
  const currentStepData = steps[currentStep];

  return (
    <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
      <div className="bg-white rounded-lg shadow-xl max-w-3xl w-full max-h-[90vh] overflow-hidden flex flex-col">
        {/* Header */}
        <div className="p-6 border-b border-slate-200">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 bg-emerald-100 rounded-lg flex items-center justify-center">
                {currentStep + 1}
              </div>
              <div>
                <h2 className="text-lg text-slate-900">{currentStepData.title}</h2>
                <p className="text-sm text-slate-500">{currentStepData.description}</p>
              </div>
            </div>
            <button
              onClick={onSkip}
              className="p-2 hover:bg-slate-100 rounded-lg transition-colors"
            >
              <X className="w-5 h-5 text-slate-400" />
            </button>
          </div>
          <Progress value={progress} className="h-2" />
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6">
          {currentStepData.content}
        </div>

        {/* Footer */}
        <div className="p-6 border-t border-slate-200 flex items-center justify-between">
          <button
            onClick={onSkip}
            className="text-sm text-slate-500 hover:text-slate-700 transition-colors"
          >
            Überspringen
          </button>

          <div className="flex items-center gap-3">
            {currentStep > 0 && (
              <button
                onClick={handleBack}
                className="flex items-center gap-2 px-4 py-2 border border-slate-200 text-slate-700 rounded-lg hover:bg-slate-50 transition-colors"
              >
                <ArrowLeft className="w-4 h-4" />
                Zurück
              </button>
            )}
            <button
              onClick={handleNext}
              className="flex items-center gap-2 px-6 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors"
            >
              {currentStep === steps.length - 1 ? 'Fertig' : 'Weiter'}
              {currentStep < steps.length - 1 && <ArrowRight className="w-4 h-4" />}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
