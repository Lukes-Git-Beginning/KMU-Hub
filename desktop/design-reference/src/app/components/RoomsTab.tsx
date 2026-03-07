import { MessageCircle, Phone } from 'lucide-react';
import { Avatar, AvatarFallback } from './ui/avatar';

const rooms = [
  {
    category: '🏢 BÜRO',
    count: 12,
    rooms: [
      {
        id: 'r1',
        name: '📍 Raum 101 - Entwicklung',
        employees: [
          { id: 'e1', name: 'Anna Müller', avatar: 'AM', status: 'online' },
          { id: 'e2', name: 'Michael Berg', avatar: 'MB', status: 'online' },
          { id: 'e3', name: 'Tom Weber', avatar: 'TW', status: 'busy' },
        ],
      },
      {
        id: 'r2',
        name: '📍 Raum 102 - Design',
        employees: [
          { id: 'e4', name: 'Sarah Klein', avatar: 'SK', status: 'online' },
          { id: 'e5', name: 'Lisa Fischer', avatar: 'LF', status: 'dnd' },
        ],
      },
      {
        id: 'r3',
        name: '📍 Raum 103 - Marketing',
        employees: [
          { id: 'e6', name: 'Laura Meier', avatar: 'LM', status: 'online' },
        ],
      },
    ],
  },
  {
    category: '🏠 HOMEOFFICE',
    count: 5,
    employees: [
      { id: 'h1', name: 'Julia Schmidt', avatar: 'JS', status: 'online' },
      { id: 'h2', name: 'Peter Klein', avatar: 'PK', status: 'busy' },
      { id: 'h3', name: 'Maria Huber', avatar: 'MH', status: 'dnd' },
      { id: 'h4', name: 'David Müller', avatar: 'DM', status: 'online' },
      { id: 'h5', name: 'Sandra Koch', avatar: 'SK', status: 'busy' },
    ],
  },
  {
    category: '🚗 AUßENDIENST',
    count: 3,
    employees: [
      { id: 'f1', name: 'Thomas Meyer', avatar: 'TM', status: 'online' },
      { id: 'f2', name: 'Robert Wagner', avatar: 'RW', status: 'busy' },
      { id: 'f3', name: 'Andreas Lang', avatar: 'AL', status: 'offline' },
    ],
  },
];

export function RoomsTab() {
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'online':
        return 'bg-green-500';
      case 'busy':
        return 'bg-yellow-500';
      case 'dnd':
        return 'bg-red-500';
      default:
        return 'bg-slate-600';
    }
  };

  return (
    <div className="space-y-8">
      {rooms.map((category, idx) => (
        <div key={idx}>
          <h3 className="text-sm font-bold text-slate-400 dark:text-slate-500 uppercase tracking-wide mb-4">
            {category.category} ({category.count} Mitarbeiter)
          </h3>

          {category.rooms ? (
            // Office Rooms
            <div className="space-y-4">
              {category.rooms.map((room) => (
                <div
                  key={room.id}
                  className="bg-white dark:bg-[#1e293b] border border-slate-200 dark:border-slate-700 rounded-xl p-5 shadow-sm"
                >
                  <h4 className="text-sm font-semibold text-emerald-600 dark:text-emerald-400 mb-4">
                    {room.name}
                  </h4>
                  <div className="space-y-3">
                    {room.employees.map((employee) => (
                      <div
                        key={employee.id}
                        className="flex items-center justify-between py-2 border-b border-slate-100 dark:border-slate-700 last:border-0"
                      >
                        <div className="flex items-center gap-3">
                          <div className="relative">
                            <Avatar className="w-9 h-9">
                              <AvatarFallback className="bg-emerald-100 dark:bg-emerald-900 text-emerald-700 dark:text-emerald-300 text-sm">
                                {employee.avatar}
                              </AvatarFallback>
                            </Avatar>
                            <div
                              className={`absolute -bottom-0.5 -right-0.5 w-3 h-3 rounded-full border-2 border-white dark:border-[#1e293b] ${getStatusColor(employee.status)}`}
                            />
                          </div>
                          <span className="text-sm font-medium text-slate-900 dark:text-slate-100">
                            {employee.name}
                          </span>
                        </div>
                        <div className="flex items-center gap-2">
                          <button className="px-3 py-1.5 border border-emerald-500 text-emerald-500 hover:bg-emerald-500/10 rounded-md text-xs font-medium transition-colors flex items-center gap-1">
                            <MessageCircle className="w-3.5 h-3.5" />
                            Chat
                          </button>
                          <button className="px-3 py-1.5 border border-green-500 text-green-500 hover:bg-green-500/10 rounded-md text-xs font-medium transition-colors flex items-center gap-1">
                            <Phone className="w-3.5 h-3.5" />
                            Anrufen
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                  <p className="text-xs text-slate-500 dark:text-slate-400 mt-3">
                    {room.employees.length} Person{room.employees.length !== 1 && 'en'} anwesend
                  </p>
                </div>
              ))}
            </div>
          ) : (
            // Homeoffice / Field Service
            <div className="bg-white dark:bg-[#1e293b] border border-slate-200 dark:border-slate-700 rounded-xl p-5 shadow-sm">
              <div className="space-y-3">
                {category.employees?.map((employee) => (
                  <div
                    key={employee.id}
                    className="flex items-center justify-between py-2 border-b border-slate-100 dark:border-slate-700 last:border-0"
                  >
                    <div className="flex items-center gap-3">
                      <div className="relative">
                        <Avatar className="w-9 h-9">
                          <AvatarFallback className="bg-emerald-100 dark:bg-emerald-900 text-emerald-700 dark:text-emerald-300 text-sm">
                            {employee.avatar}
                          </AvatarFallback>
                        </Avatar>
                        <div
                          className={`absolute -bottom-0.5 -right-0.5 w-3 h-3 rounded-full border-2 border-white dark:border-[#1e293b] ${getStatusColor(employee.status)}`}
                        />
                      </div>
                      <span className="text-sm font-medium text-slate-900 dark:text-slate-100">
                        {employee.name}
                      </span>
                    </div>
                    <div className="flex items-center gap-2">
                      <button className="px-3 py-1.5 border border-emerald-500 text-emerald-500 hover:bg-emerald-500/10 rounded-md text-xs font-medium transition-colors flex items-center gap-1">
                        <MessageCircle className="w-3.5 h-3.5" />
                        Chat
                      </button>
                      {category.category.includes('BÜRO') || category.category.includes('HOMEOFFICE') ? (
                        <button className="px-3 py-1.5 border border-green-500 text-green-500 hover:bg-green-500/10 rounded-md text-xs font-medium transition-colors flex items-center gap-1">
                          <Phone className="w-3.5 h-3.5" />
                          Anrufen
                        </button>
                      ) : null}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      ))}
    </div>
  );
}
