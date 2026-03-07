import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from './ui/table';
import { Badge } from './ui/badge';

const activities = [
  {
    id: 1,
    user: 'Thomas Weber',
    action: 'Dokument hochgeladen',
    project: 'Website Redesign',
    time: 'vor 5 Minuten',
    status: 'completed',
  },
  {
    id: 2,
    user: 'Sarah Schmidt',
    action: 'Aufgabe abgeschlossen',
    project: 'Mobile App',
    time: 'vor 23 Minuten',
    status: 'completed',
  },
  {
    id: 3,
    user: 'Michael Keller',
    action: 'Kommentar hinzugefügt',
    project: 'Marketing Kampagne',
    time: 'vor 1 Stunde',
    status: 'pending',
  },
  {
    id: 4,
    user: 'Julia Meier',
    action: 'Neues Projekt erstellt',
    project: 'CRM Integration',
    time: 'vor 2 Stunden',
    status: 'in-progress',
  },
  {
    id: 5,
    user: 'David Fischer',
    action: 'Deadline aktualisiert',
    project: 'Website Redesign',
    time: 'vor 3 Stunden',
    status: 'pending',
  },
];

const statusLabels: Record<string, { label: string; variant: 'default' | 'secondary' | 'outline' | 'destructive' }> = {
  completed: { label: 'Abgeschlossen', variant: 'default' },
  pending: { label: 'Ausstehend', variant: 'secondary' },
  'in-progress': { label: 'In Arbeit', variant: 'outline' },
};

export function ActivityTable() {
  return (
    <div className="bg-white border border-slate-200 rounded-lg">
      <div className="p-6 border-b border-slate-200">
        <h2 className="text-slate-900">Letzte Aktivitäten</h2>
      </div>
      
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="text-slate-500">Benutzer</TableHead>
            <TableHead className="text-slate-500">Aktion</TableHead>
            <TableHead className="text-slate-500">Projekt</TableHead>
            <TableHead className="text-slate-500">Status</TableHead>
            <TableHead className="text-slate-500">Zeit</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {activities.map((activity) => (
            <TableRow key={activity.id}>
              <TableCell>
                <div className="flex items-center gap-3">
                  <div className="w-8 h-8 bg-slate-100 rounded-full flex items-center justify-center text-sm text-slate-600">
                    {activity.user.split(' ').map(n => n[0]).join('')}
                  </div>
                  <span className="text-sm text-slate-900">{activity.user}</span>
                </div>
              </TableCell>
              <TableCell className="text-sm text-slate-600">{activity.action}</TableCell>
              <TableCell className="text-sm text-slate-600">{activity.project}</TableCell>
              <TableCell>
                <Badge 
                  variant={statusLabels[activity.status].variant}
                  className={activity.status === 'completed' ? 'bg-emerald-100 text-emerald-700 hover:bg-emerald-100' : ''}
                >
                  {statusLabels[activity.status].label}
                </Badge>
              </TableCell>
              <TableCell className="text-sm text-slate-500">{activity.time}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
