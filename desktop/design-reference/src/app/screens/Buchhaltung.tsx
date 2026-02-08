import { useState } from 'react';
import { Plus, TrendingUp, TrendingDown, Calendar, CheckCircle, XCircle, ExternalLink, Clock } from 'lucide-react';
import { Badge } from '../components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../components/ui/tabs';
import { Progress } from '../components/ui/progress';

const integrations = [
  {
    id: 'bexio',
    name: 'Bexio',
    description: 'Cloud-Buchhaltung für KMU – der Schweizer Standard',
    logo: '🇨🇭',
    status: 'available',
    features: ['Rechnungsstellung', 'Zeiterfassung', 'Projektverwaltung', 'Banking'],
    pricing: 'Ab CHF 30/Monat',
  },
  {
    id: 'abacus',
    name: 'Abacus',
    description: 'Professionelle Business-Software aus der Schweiz',
    logo: '📊',
    status: 'available',
    features: ['Finanzbuchhaltung', 'Lohnbuchhaltung', 'Leistungserfassung', 'CRM'],
    pricing: 'Auf Anfrage',
  },
  {
    id: 'runmyaccounts',
    name: 'Run my Accounts',
    description: 'Online-Buchhaltung mit integriertem E-Banking',
    logo: '💼',
    status: 'available',
    features: ['Online-Buchhaltung', 'E-Banking', 'Mehrwertsteuer', 'Reporting'],
    pricing: 'Ab CHF 29/Monat',
  },
];

const recentTransactions = [
  {
    id: 1,
    type: 'income',
    description: 'Rechnung #2024-001 - Kunde ABC AG',
    amount: 12500.0,
    date: '2024-01-20',
    status: 'paid',
  },
  {
    id: 2,
    type: 'expense',
    description: 'Büromaterial - Office World AG',
    amount: -450.0,
    date: '2024-01-19',
    status: 'paid',
  },
  {
    id: 3,
    type: 'income',
    description: 'Rechnung #2024-002 - Kunde XYZ GmbH',
    amount: 8900.0,
    date: '2024-01-18',
    status: 'pending',
  },
  {
    id: 4,
    type: 'expense',
    description: 'Software-Lizenzen - Microsoft',
    amount: -1200.0,
    date: '2024-01-17',
    status: 'paid',
  },
];

const invoices = [
  {
    id: '2024-001',
    client: 'ABC AG',
    amount: 12500.0,
    date: '2024-01-15',
    dueDate: '2024-02-14',
    status: 'paid',
  },
  {
    id: '2024-002',
    client: 'XYZ GmbH',
    amount: 8900.0,
    date: '2024-01-18',
    dueDate: '2024-02-17',
    status: 'pending',
  },
  {
    id: '2024-003',
    client: 'Test AG',
    amount: 5600.0,
    date: '2024-01-19',
    dueDate: '2024-02-18',
    status: 'overdue',
  },
];

export function Buchhaltung() {
  const [connectedIntegration, setConnectedIntegration] = useState<string | null>(null);

  const totalIncome = recentTransactions
    .filter((t) => t.type === 'income')
    .reduce((sum, t) => sum + t.amount, 0);

  const totalExpenses = Math.abs(
    recentTransactions
      .filter((t) => t.type === 'expense')
      .reduce((sum, t) => sum + t.amount, 0)
  );

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'paid':
        return 'bg-green-100 text-green-700';
      case 'pending':
        return 'bg-yellow-100 text-yellow-700';
      case 'overdue':
        return 'bg-red-100 text-red-700';
      default:
        return 'bg-slate-100 text-slate-700';
    }
  };

  const getStatusLabel = (status: string) => {
    switch (status) {
      case 'paid':
        return 'Bezahlt';
      case 'pending':
        return 'Ausstehend';
      case 'overdue':
        return 'Überfällig';
      default:
        return status;
    }
  };

  return (
    <div className="flex-1 overflow-y-auto p-4 md:p-8 bg-slate-50 dark:bg-slate-900">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between mb-6 md:mb-8 gap-4">
        <div>
          <h1 className="text-slate-900 dark:text-white mb-1">Buchhaltung</h1>
          <p className="text-sm text-slate-500 dark:text-slate-400">Integrieren Sie Ihre Schweizer Buchhaltungssoftware</p>
        </div>
        {connectedIntegration && (
          <button className="flex items-center justify-center gap-2 px-4 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors">
            <Plus className="w-4 h-4" />
            <span className="hidden sm:inline">Neue Rechnung</span>
            <span className="sm:hidden">Rechnung</span>
          </button>
        )}
      </div>

      {!connectedIntegration ? (
        <>
          {/* Info Banner */}
          <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-6 mb-8">
            <h3 className="text-blue-900 dark:text-blue-300 mb-2">Schweizer Buchhaltungs-Integration</h3>
            <p className="text-sm text-blue-700 dark:text-blue-400 mb-4">
              Verbinden Sie Ihren KMU Digital Hub mit Ihrer bestehenden Buchhaltungssoftware für nahtlose Workflows.
              Alle Daten bleiben in der Schweiz und sind DSG/DSGVO-konform.
            </p>
            <div className="flex flex-wrap gap-2">
              <Badge className="bg-emerald-100 dark:bg-emerald-900 text-emerald-700 dark:text-emerald-300">100% Swiss Hosting</Badge>
              <Badge className="bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300">DSG-konform</Badge>
              <Badge className="bg-purple-100 dark:bg-purple-900 text-purple-700 dark:text-purple-300">Echtzeit-Sync</Badge>
            </div>
          </div>

          {/* Integration Cards */}
          <div className="mb-8">
            <h2 className="text-lg text-slate-900 dark:text-white mb-6">Verfügbare Integrationen</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
              {integrations.map((integration) => (
                <div
                  key={integration.id}
                  className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-6 hover:shadow-lg dark:hover:shadow-emerald-900/10 transition-shadow"
                >
                  <div className="flex items-start justify-between mb-4">
                    <div className="flex items-center gap-3">
                      <div className="text-4xl">{integration.logo}</div>
                      <div>
                        <h3 className="text-slate-900 dark:text-white">{integration.name}</h3>
                        <p className="text-xs text-slate-500 dark:text-slate-400">{integration.pricing}</p>
                      </div>
                    </div>
                  </div>

                  <p className="text-sm text-slate-600 dark:text-slate-300 mb-4">{integration.description}</p>

                  <div className="mb-4">
                    <p className="text-xs text-slate-500 dark:text-slate-400 mb-2">Features:</p>
                    <div className="space-y-1">
                      {integration.features.map((feature, idx) => (
                        <div key={idx} className="flex items-center gap-2">
                          <CheckCircle className="w-3 h-3 text-emerald-600 dark:text-emerald-400 flex-shrink-0" />
                          <span className="text-xs text-slate-700 dark:text-slate-300">{feature}</span>
                        </div>
                      ))}
                    </div>
                  </div>

                  <button
                    onClick={() => setConnectedIntegration(integration.id)}
                    className="w-full px-4 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors text-sm font-medium"
                  >
                    Jetzt verbinden
                  </button>
                </div>
              ))}
            </div>
          </div>

          {/* Help Section */}
          <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-6">
            <h3 className="text-slate-900 dark:text-white mb-2">Benötigen Sie Hilfe bei der Integration?</h3>
            <p className="text-sm text-slate-600 dark:text-slate-300 mb-4">
              Unser Support-Team hilft Ihnen gerne bei der Einrichtung Ihrer Buchhaltungs-Integration. Wir bieten
              persönliche Beratung und technischen Support.
            </p>
            <div className="flex flex-wrap gap-3">
              <button className="px-4 py-2 bg-slate-50 dark:bg-slate-700 border border-slate-200 dark:border-slate-600 text-slate-700 dark:text-slate-200 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-600 transition-colors text-sm">
                Dokumentation
              </button>
              <button className="px-4 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors text-sm">
                Support kontaktieren
              </button>
            </div>
          </div>
        </>
      ) : (
        <>
          {/* Connected Banner */}
          <div className="bg-emerald-50 border border-emerald-200 rounded-lg p-6 mb-8">
            <div className="flex items-start justify-between">
              <div className="flex items-center gap-3">
                <div className="text-4xl">
                  {integrations.find((i) => i.id === connectedIntegration)?.logo}
                </div>
                <div>
                  <h3 className="text-emerald-900 mb-1 flex items-center gap-2">
                    Verbunden mit {integrations.find((i) => i.id === connectedIntegration)?.name}
                    <CheckCircle className="w-5 h-5 text-emerald-600" />
                  </h3>
                  <p className="text-sm text-emerald-700">Letzte Synchronisation: vor 5 Minuten</p>
                </div>
              </div>
              <button
                onClick={() => setConnectedIntegration(null)}
                className="text-sm text-emerald-700 hover:text-emerald-800 underline"
              >
                Trennen
              </button>
            </div>
          </div>

          {/* Stats */}
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 md:gap-6 mb-8">
            <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-6">
              <div className="flex items-center gap-3 mb-2">
                <TrendingUp className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
                <p className="text-sm text-slate-500 dark:text-slate-400">Einnahmen (MTD)</p>
              </div>
              <p className="text-2xl text-slate-900 dark:text-white">CHF {totalIncome.toLocaleString('de-CH')}</p>
              <p className="text-xs text-emerald-600 dark:text-emerald-400 mt-1">+12% zum Vormonat</p>
            </div>

            <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-6">
              <div className="flex items-center gap-3 mb-2">
                <TrendingDown className="w-5 h-5 text-red-600 dark:text-red-400" />
                <p className="text-sm text-slate-500 dark:text-slate-400">Ausgaben (MTD)</p>
              </div>
              <p className="text-2xl text-slate-900 dark:text-white">CHF {totalExpenses.toLocaleString('de-CH')}</p>
              <p className="text-xs text-red-600 dark:text-red-400 mt-1">-5% zum Vormonat</p>
            </div>

            <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-6">
              <div className="flex items-center gap-3 mb-2">
                <Calendar className="w-5 h-5 text-blue-600 dark:text-blue-400" />
                <p className="text-sm text-slate-500 dark:text-slate-400">Rechnungen</p>
              </div>
              <p className="text-2xl text-slate-900 dark:text-white">{invoices.length}</p>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">Diesen Monat</p>
            </div>

            <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-6">
              <div className="flex items-center gap-3 mb-2">
                <Clock className="w-5 h-5 text-yellow-600 dark:text-yellow-400" />
                <p className="text-sm text-slate-500 dark:text-slate-400">Überfällig</p>
              </div>
              <p className="text-2xl text-slate-900 dark:text-white">
                {invoices.filter((i) => i.status === 'overdue').length}
              </p>
              <p className="text-xs text-yellow-600 dark:text-yellow-400 mt-1">Sofort handeln</p>
            </div>
          </div>

          {/* Tabs */}
          <Tabs defaultValue="transactions" className="w-full">
            <TabsList className="mb-6">
              <TabsTrigger value="transactions">Transaktionen</TabsTrigger>
              <TabsTrigger value="invoices">Rechnungen</TabsTrigger>
              <TabsTrigger value="reports">Berichte</TabsTrigger>
            </TabsList>

            <TabsContent value="transactions">
              <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg overflow-hidden">
                <div className="overflow-x-auto">
                  <table className="w-full">
                    <thead className="bg-slate-50 dark:bg-slate-900/50 border-b border-slate-200 dark:border-slate-700">
                      <tr>
                        <th className="text-left px-4 md:px-6 py-3 text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">
                          Datum
                        </th>
                        <th className="text-left px-4 md:px-6 py-3 text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">
                          Beschreibung
                        </th>
                        <th className="text-left px-4 md:px-6 py-3 text-xs font-medium text-slate-500 dark:text-slate-400 uppercase hidden md:table-cell">
                          Status
                        </th>
                        <th className="text-right px-4 md:px-6 py-3 text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">
                          Betrag
                        </th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
                      {recentTransactions.map((transaction) => (
                        <tr key={transaction.id} className="hover:bg-slate-50 dark:hover:bg-slate-700">
                          <td className="px-4 md:px-6 py-4 text-sm text-slate-900 dark:text-white">
                            {new Date(transaction.date).toLocaleDateString('de-CH')}
                          </td>
                          <td className="px-4 md:px-6 py-4 text-sm text-slate-900 dark:text-white">
                            <div className="flex items-center gap-2">
                              {transaction.type === 'income' ? (
                                <TrendingUp className="w-4 h-4 text-emerald-600 dark:text-emerald-400 flex-shrink-0" />
                              ) : (
                                <TrendingDown className="w-4 h-4 text-red-600 dark:text-red-400 flex-shrink-0" />
                              )}
                              <span className="truncate">{transaction.description}</span>
                            </div>
                          </td>
                          <td className="px-4 md:px-6 py-4 text-sm hidden md:table-cell">
                            <Badge className={getStatusColor(transaction.status)}>
                              {getStatusLabel(transaction.status)}
                            </Badge>
                          </td>
                          <td
                            className={`px-4 md:px-6 py-4 text-sm font-medium text-right ${
                              transaction.type === 'income' ? 'text-emerald-600 dark:text-emerald-400' : 'text-red-600 dark:text-red-400'
                            }`}
                          >
                            CHF {Math.abs(transaction.amount).toLocaleString('de-CH', { minimumFractionDigits: 2 })}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            </TabsContent>

            <TabsContent value="invoices">
              <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg overflow-hidden">
                <div className="overflow-x-auto">
                  <table className="w-full">
                    <thead className="bg-slate-50 dark:bg-slate-900/50 border-b border-slate-200 dark:border-slate-700">
                      <tr>
                        <th className="text-left px-4 md:px-6 py-3 text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">
                          Rechnungs-Nr.
                        </th>
                        <th className="text-left px-4 md:px-6 py-3 text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">
                          Kunde
                        </th>
                        <th className="text-left px-4 md:px-6 py-3 text-xs font-medium text-slate-500 dark:text-slate-400 uppercase hidden md:table-cell">
                          Fällig am
                        </th>
                        <th className="text-left px-4 md:px-6 py-3 text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">
                          Status
                        </th>
                        <th className="text-right px-4 md:px-6 py-3 text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">
                          Betrag
                        </th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
                      {invoices.map((invoice) => (
                        <tr key={invoice.id} className="hover:bg-slate-50 dark:hover:bg-slate-700">
                          <td className="px-4 md:px-6 py-4 text-sm text-slate-900 dark:text-white font-medium">
                            #{invoice.id}
                          </td>
                          <td className="px-4 md:px-6 py-4 text-sm text-slate-900 dark:text-white">{invoice.client}</td>
                          <td className="px-4 md:px-6 py-4 text-sm text-slate-600 dark:text-slate-300 hidden md:table-cell">
                            {new Date(invoice.dueDate).toLocaleDateString('de-CH')}
                          </td>
                          <td className="px-4 md:px-6 py-4 text-sm">
                            <Badge className={getStatusColor(invoice.status)}>
                              {getStatusLabel(invoice.status)}
                            </Badge>
                          </td>
                          <td className="px-4 md:px-6 py-4 text-sm font-medium text-slate-900 dark:text-white text-right">
                            CHF {invoice.amount.toLocaleString('de-CH', { minimumFractionDigits: 2 })}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            </TabsContent>

            <TabsContent value="reports">
              <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-8 text-center">
                <div className="max-w-md mx-auto">
                  <div className="w-16 h-16 bg-slate-100 dark:bg-slate-700 rounded-full flex items-center justify-center mx-auto mb-4">
                    <Calendar className="w-8 h-8 text-slate-400 dark:text-slate-500" />
                  </div>
                  <h3 className="text-slate-900 dark:text-white mb-2">Berichte werden generiert</h3>
                  <p className="text-sm text-slate-500 dark:text-slate-400 mb-6">
                    Finanzberichte, Umsatzanalysen und Steuer-Reports werden automatisch aus Ihren Buchhaltungsdaten
                    erstellt.
                  </p>
                  <div className="space-y-3">
                    <div className="flex items-center justify-between p-3 bg-slate-50 dark:bg-slate-700 rounded-lg">
                      <span className="text-sm text-slate-700 dark:text-slate-300">Monatlicher Umsatzbericht</span>
                      <Badge className="bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300">In Entwicklung</Badge>
                    </div>
                    <div className="flex items-center justify-between p-3 bg-slate-50 dark:bg-slate-700 rounded-lg">
                      <span className="text-sm text-slate-700 dark:text-slate-300">Mehrwertsteuer-Abrechnung</span>
                      <Badge className="bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300">In Entwicklung</Badge>
                    </div>
                    <div className="flex items-center justify-between p-3 bg-slate-50 dark:bg-slate-700 rounded-lg">
                      <span className="text-sm text-slate-700 dark:text-slate-300">Jahresabschluss</span>
                      <Badge className="bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300">In Entwicklung</Badge>
                    </div>
                  </div>
                </div>
              </div>
            </TabsContent>
          </Tabs>
        </>
      )}
    </div>
  );
}