import { useState } from 'react';
import { HelpCircle, X, BookOpen, Video, MessageCircle, Send, Minimize2 } from 'lucide-react';

const helpArticles = [
  { id: 1, title: 'Erste Schritte mit KMU Digital Hub', category: 'Basics' },
  { id: 2, title: 'Wie erstelle ich ein Projekt?', category: 'Projekte' },
  { id: 3, title: 'Buchhaltung-Integration einrichten', category: 'Buchhaltung' },
  { id: 4, title: 'Team-Mitglieder einladen', category: 'Team' },
];

interface HelpWidgetProps {
  isOpen?: boolean;
  onToggle?: (open: boolean) => void;
}

export function HelpWidget({ isOpen: externalIsOpen, onToggle }: HelpWidgetProps = {}) {
  const [internalIsOpen, setInternalIsOpen] = useState(false);
  const [activeTab, setActiveTab] = useState<'help' | 'chat'>('help');
  const [isMinimized, setIsMinimized] = useState(false);
  const [chatMessage, setChatMessage] = useState('');

  // Use external control if provided, otherwise use internal state
  const isOpen = externalIsOpen !== undefined ? externalIsOpen : internalIsOpen;
  const setIsOpen = (open: boolean) => {
    if (onToggle) {
      onToggle(open);
    } else {
      setInternalIsOpen(open);
    }
  };

  const mockChatMessages = [
    { id: 1, sender: 'support', message: 'Hallo! Wie kann ich Ihnen helfen?', time: '14:30' },
    { id: 2, sender: 'user', message: 'Ich habe eine Frage zur Buchhaltung', time: '14:31' },
    { id: 3, sender: 'support', message: 'Gerne! Welche Buchhaltungssoftware möchten Sie integrieren?', time: '14:31' },
  ];

  // Don't render floating button - only show when opened via ProfileMenu
  if (!isOpen) {
    return null;
  }

  return (
    <div
      className={`fixed bottom-6 right-6 bg-white dark:bg-slate-800 rounded-lg shadow-2xl border border-slate-200 dark:border-slate-700 z-40 transition-all ${
        isMinimized ? 'w-80 h-16' : 'w-96 h-[600px]'
      }`}
    >
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b border-slate-200 dark:border-slate-700 bg-emerald-600 text-white rounded-t-lg">
        <div className="flex items-center gap-2">
          <HelpCircle className="w-5 h-5" />
          <h3 className="font-medium">Hilfe & Support</h3>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setIsMinimized(!isMinimized)}
            className="p-1 hover:bg-emerald-700 rounded transition-colors"
          >
            <Minimize2 className="w-4 h-4" />
          </button>
          <button
            onClick={() => setIsOpen(false)}
            className="p-1 hover:bg-emerald-700 rounded transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
      </div>

      {!isMinimized && (
        <>
          {/* Tabs */}
          <div className="flex border-b border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800">
            <button
              onClick={() => setActiveTab('help')}
              className={`flex-1 flex items-center justify-center gap-2 px-4 py-3 text-sm transition-colors ${
                activeTab === 'help'
                  ? 'border-b-2 border-emerald-600 text-emerald-600 dark:text-emerald-400'
                  : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
              }`}
            >
              <BookOpen className="w-4 h-4" />
              Hilfe
            </button>
            <button
              onClick={() => setActiveTab('chat')}
              className={`flex-1 flex items-center justify-center gap-2 px-4 py-3 text-sm transition-colors ${
                activeTab === 'chat'
                  ? 'border-b-2 border-emerald-600 text-emerald-600 dark:text-emerald-400'
                  : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
              }`}
            >
              <MessageCircle className="w-4 h-4" />
              Chat
              <span className="w-2 h-2 bg-emerald-500 dark:bg-emerald-400 rounded-full"></span>
            </button>
          </div>

          {/* Content */}
          <div className="flex-1 overflow-y-auto p-4 bg-white dark:bg-slate-800" style={{ height: 'calc(600px - 180px)' }}>
            {activeTab === 'help' ? (
              <div className="space-y-4">
                {/* Search */}
                <div>
                  <input
                    type="text"
                    placeholder="Suchen Sie nach Hilfe..."
                    className="w-full px-3 py-2 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 text-slate-900 dark:text-white placeholder:text-slate-400 dark:placeholder:text-slate-500 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500"
                  />
                </div>

                {/* Quick Links */}
                <div>
                  <h4 className="text-sm font-medium text-slate-900 dark:text-white mb-3">Schnellzugriff</h4>
                  <div className="space-y-2">
                    <a
                      href="#"
                      className="flex items-center gap-3 p-3 border border-slate-200 dark:border-slate-700 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors"
                    >
                      <BookOpen className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
                      <div className="flex-1">
                        <p className="text-sm text-slate-900 dark:text-white">Dokumentation</p>
                        <p className="text-xs text-slate-500 dark:text-slate-400">Alle Features im Detail</p>
                      </div>
                    </a>
                    <a
                      href="#"
                      className="flex items-center gap-3 p-3 border border-slate-200 dark:border-slate-700 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors"
                    >
                      <Video className="w-5 h-5 text-blue-600 dark:text-blue-400" />
                      <div className="flex-1">
                        <p className="text-sm text-slate-900 dark:text-white">Video-Tutorials</p>
                        <p className="text-xs text-slate-500 dark:text-slate-400">Schritt-für-Schritt Anleitungen</p>
                      </div>
                    </a>
                  </div>
                </div>

                {/* Popular Articles */}
                <div>
                  <h4 className="text-sm font-medium text-slate-900 dark:text-white mb-3">Beliebte Artikel</h4>
                  <div className="space-y-2">
                    {helpArticles.map((article) => (
                      <a
                        key={article.id}
                        href="#"
                        className="block p-3 border border-slate-200 dark:border-slate-700 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors"
                      >
                        <p className="text-sm text-slate-900 dark:text-white mb-1">{article.title}</p>
                        <span className="text-xs text-emerald-600 dark:text-emerald-400">{article.category}</span>
                      </a>
                    ))}
                  </div>
                </div>

                {/* Contact Support */}
                <div className="bg-emerald-50 dark:bg-emerald-900/20 border border-emerald-200 dark:border-emerald-800 rounded-lg p-4">
                  <h4 className="text-sm font-medium text-emerald-900 dark:text-emerald-300 mb-2">Noch Fragen?</h4>
                  <p className="text-xs text-emerald-700 dark:text-emerald-400 mb-3">
                    Unser Support-Team ist Mo-Fr von 8-18 Uhr für Sie da.
                  </p>
                  <button
                    onClick={() => setActiveTab('chat')}
                    className="w-full px-3 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors text-sm"
                  >
                    Chat starten
                  </button>
                </div>
              </div>
            ) : (
              <div className="flex flex-col h-full">
                {/* Chat Messages */}
                <div className="flex-1 space-y-3 mb-4">
                  {mockChatMessages.map((msg) => (
                    <div
                      key={msg.id}
                      className={`flex ${msg.sender === 'user' ? 'justify-end' : 'justify-start'}`}
                    >
                      <div
                        className={`max-w-[80%] rounded-lg p-3 ${
                          msg.sender === 'user'
                            ? 'bg-emerald-600 text-white'
                            : 'bg-slate-100 dark:bg-slate-700 text-slate-900 dark:text-white'
                        }`}
                      >
                        <p className="text-sm">{msg.message}</p>
                        <p
                          className={`text-xs mt-1 ${
                            msg.sender === 'user' ? 'text-emerald-100' : 'text-slate-500 dark:text-slate-400'
                          }`}
                        >
                          {msg.time}
                        </p>
                      </div>
                    </div>
                  ))}
                </div>

                {/* Typing Indicator */}
                <div className="mb-3">
                  <div className="flex items-center gap-2 text-xs text-slate-500 dark:text-slate-400">
                    <div className="flex gap-1">
                      <span className="w-2 h-2 bg-slate-400 dark:bg-slate-500 rounded-full animate-bounce"></span>
                      <span className="w-2 h-2 bg-slate-400 dark:bg-slate-500 rounded-full animate-bounce" style={{ animationDelay: '0.1s' }}></span>
                      <span className="w-2 h-2 bg-slate-400 dark:bg-slate-500 rounded-full animate-bounce" style={{ animationDelay: '0.2s' }}></span>
                    </div>
                    <span>Support schreibt...</span>
                  </div>
                </div>
              </div>
            )}
          </div>

          {/* Chat Input */}
          {activeTab === 'chat' && (
            <div className="p-4 border-t border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800">
              <div className="flex items-center gap-2">
                <input
                  type="text"
                  value={chatMessage}
                  onChange={(e) => setChatMessage(e.target.value)}
                  placeholder="Nachricht schreiben..."
                  className="flex-1 px-3 py-2 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 text-slate-900 dark:text-white placeholder:text-slate-400 dark:placeholder:text-slate-500 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500"
                />
                <button className="p-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors">
                  <Send className="w-4 h-4" />
                </button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}