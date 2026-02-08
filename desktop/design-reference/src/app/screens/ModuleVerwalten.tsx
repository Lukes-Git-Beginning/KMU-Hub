import { useState } from 'react';
import { useDrag, useDrop, DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { FolderKanban, CheckSquare, FileText, MessageSquare, Users, Calculator, GripVertical, Eye, EyeOff } from 'lucide-react';
import { Switch } from '../components/ui/switch';

interface Module {
  id: string;
  name: string;
  description: string;
  icon: any;
  color: string;
  isActive: boolean;
  order: number;
}

const initialModules: Module[] = [
  {
    id: 'projects',
    name: 'Projektverwaltung',
    description: 'Kanban-Boards, Zeiterfassung & Gantt-Charts',
    icon: FolderKanban,
    color: 'bg-blue-500',
    isActive: true,
    order: 1,
  },
  {
    id: 'tasks',
    name: 'Aufgabenverwaltung',
    description: 'To-Dos, Deadlines & Team-Zuweisungen',
    icon: CheckSquare,
    color: 'bg-emerald-500',
    isActive: true,
    order: 2,
  },
  {
    id: 'documents',
    name: 'Dokumentenmanagement',
    description: 'Zentrale Ablage mit Versionierung',
    icon: FileText,
    color: 'bg-purple-500',
    isActive: true,
    order: 3,
  },
  {
    id: 'accounting',
    name: 'Buchhaltung',
    description: 'Bexio, Abacus & Run my Accounts Integration',
    icon: Calculator,
    color: 'bg-orange-500',
    isActive: false,
    order: 4,
  },
  {
    id: 'communication',
    name: 'Kommunikation',
    description: 'Team-Chat & Channels',
    icon: MessageSquare,
    color: 'bg-pink-500',
    isActive: true,
    order: 5,
  },
  {
    id: 'team',
    name: 'Team & CRM',
    description: 'Mitarbeiterverwaltung & Kontakte',
    icon: Users,
    color: 'bg-cyan-500',
    isActive: true,
    order: 6,
  },
];

interface DraggableModuleProps {
  module: Module;
  index: number;
  moveModule: (dragIndex: number, hoverIndex: number) => void;
  toggleModule: (id: string) => void;
}

function DraggableModule({ module, index, moveModule, toggleModule }: DraggableModuleProps) {
  const Icon = module.icon;

  const [{ isDragging }, drag] = useDrag({
    type: 'module',
    item: { index },
    collect: (monitor) => ({
      isDragging: monitor.isDragging(),
    }),
  });

  const [, drop] = useDrop({
    accept: 'module',
    hover: (item: { index: number }) => {
      if (item.index !== index) {
        moveModule(item.index, index);
        item.index = index;
      }
    },
  });

  return (
    <div
      ref={(node) => drag(drop(node))}
      className={`bg-white border border-slate-200 rounded-lg p-6 transition-all cursor-move ${
        isDragging ? 'opacity-50' : 'hover:shadow-md'
      } ${!module.isActive ? 'opacity-60' : ''}`}
    >
      <div className="flex items-start gap-4">
        <GripVertical className="w-5 h-5 text-slate-400 flex-shrink-0 mt-1" />
        
        <div className={`w-12 h-12 rounded-lg ${module.color} bg-opacity-10 flex items-center justify-center flex-shrink-0`}>
          <Icon className={`w-6 h-6 ${module.color.replace('bg-', 'text-')}`} />
        </div>

        <div className="flex-1 min-w-0">
          <h3 className="text-slate-900 mb-1">{module.name}</h3>
          <p className="text-sm text-slate-500">{module.description}</p>
        </div>

        <div className="flex items-center gap-2 flex-shrink-0">
          {module.isActive ? (
            <Eye className="w-5 h-5 text-emerald-600" />
          ) : (
            <EyeOff className="w-5 h-5 text-slate-400" />
          )}
          <Switch
            checked={module.isActive}
            onCheckedChange={() => toggleModule(module.id)}
          />
        </div>
      </div>
    </div>
  );
}

export function ModuleVerwalten() {
  const [modules, setModules] = useState(initialModules);

  const moveModule = (dragIndex: number, hoverIndex: number) => {
    const newModules = [...modules];
    const draggedModule = newModules[dragIndex];
    newModules.splice(dragIndex, 1);
    newModules.splice(hoverIndex, 0, draggedModule);
    
    // Update order
    const updatedModules = newModules.map((mod, idx) => ({
      ...mod,
      order: idx + 1,
    }));
    
    setModules(updatedModules);
  };

  const toggleModule = (id: string) => {
    setModules(modules.map(mod => 
      mod.id === id ? { ...mod, isActive: !mod.isActive } : mod
    ));
  };

  const activeCount = modules.filter(m => m.isActive).length;

  return (
    <DndProvider backend={HTML5Backend}>
      <div className="flex-1 overflow-y-auto p-8">
        <div className="mb-8">
          <h1 className="text-slate-900 mb-1">Module verwalten</h1>
          <p className="text-sm text-slate-500">
            Aktivieren Sie Module und sortieren Sie diese per Drag & Drop nach Ihren Wünschen
          </p>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
          <div className="bg-white border border-slate-200 rounded-lg p-6">
            <p className="text-sm text-slate-500 mb-1">Verfügbare Module</p>
            <p className="text-3xl text-slate-900">{modules.length}</p>
          </div>
          <div className="bg-white border border-slate-200 rounded-lg p-6">
            <p className="text-sm text-slate-500 mb-1">Aktive Module</p>
            <p className="text-3xl text-emerald-600">{activeCount}</p>
          </div>
          <div className="bg-white border border-slate-200 rounded-lg p-6">
            <p className="text-sm text-slate-500 mb-1">Deaktivierte Module</p>
            <p className="text-3xl text-slate-400">{modules.length - activeCount}</p>
          </div>
        </div>

        {/* Info Box */}
        <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-8">
          <div className="flex items-start gap-3">
            <GripVertical className="w-5 h-5 text-blue-600 flex-shrink-0 mt-0.5" />
            <div>
              <p className="text-sm text-slate-900 font-medium mb-1">
                So funktioniert's
              </p>
              <p className="text-sm text-slate-600">
                Ziehen Sie die Module mit der Maus, um die Reihenfolge im Dashboard zu ändern. 
                Mit dem Schalter rechts können Sie Module aktivieren oder deaktivieren.
              </p>
            </div>
          </div>
        </div>

        {/* Module List */}
        <div className="space-y-4">
          {modules.map((module, index) => (
            <DraggableModule
              key={module.id}
              module={module}
              index={index}
              moveModule={moveModule}
              toggleModule={toggleModule}
            />
          ))}
        </div>

        {/* Actions */}
        <div className="mt-8 flex items-center justify-between">
          <button className="px-6 py-2 border border-slate-200 text-slate-700 rounded-lg hover:bg-slate-50 transition-colors">
            Auf Standard zurücksetzen
          </button>
          <button className="px-6 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors">
            Änderungen speichern
          </button>
        </div>
      </div>
    </DndProvider>
  );
}
