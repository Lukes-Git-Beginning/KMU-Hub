import { useState } from 'react';
import { Play, Pause, Clock, X, ChevronDown, ChevronUp } from 'lucide-react';

export function TimeTrackerWidget() {
  const [isTracking, setIsTracking] = useState(false);
  const [time, setTime] = useState(0);
  const [isMinimized, setIsMinimized] = useState(false);
  const [isVisible, setIsVisible] = useState(true);

  const formatTime = (seconds: number) => {
    const h = Math.floor(seconds / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    const s = seconds % 60;
    return `${h.toString().padStart(2, '0')}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
  };

  const handleToggle = () => {
    if (isTracking) {
      // Stop tracking
      setIsTracking(false);
    } else {
      // Start tracking
      setIsTracking(true);
      const interval = setInterval(() => {
        setTime((prev) => prev + 1);
      }, 1000);
      
      return () => clearInterval(interval);
    }
  };

  if (!isVisible) return null;

  if (isMinimized) {
    return (
      <div className="fixed top-20 right-4 z-40 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg shadow-lg p-3 flex items-center gap-2">
        <Clock className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
        <span className="text-sm font-medium text-slate-900 dark:text-white">{formatTime(time)}</span>
        <button
          onClick={() => setIsMinimized(false)}
          className="p-1 hover:bg-slate-100 dark:hover:bg-slate-700 rounded"
        >
          <ChevronDown className="w-4 h-4 text-slate-500 dark:text-slate-400" />
        </button>
      </div>
    );
  }

  return (
    <div className="fixed top-20 right-4 z-40 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg shadow-lg w-80">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b border-slate-200 dark:border-slate-700">
        <div className="flex items-center gap-2">
          <Clock className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
          <h3 className="text-sm text-slate-900 dark:text-white font-medium">Zeiterfassung</h3>
        </div>
        <div className="flex items-center gap-1">
          <button
            onClick={() => setIsMinimized(true)}
            className="p-1 hover:bg-slate-100 dark:hover:bg-slate-700 rounded"
          >
            <ChevronUp className="w-4 h-4 text-slate-500 dark:text-slate-400" />
          </button>
          <button
            onClick={() => setIsVisible(false)}
            className="p-1 hover:bg-slate-100 dark:hover:bg-slate-700 rounded"
          >
            <X className="w-4 h-4 text-slate-500 dark:text-slate-400" />
          </button>
        </div>
      </div>

      {/* Timer Display */}
      <div className="p-6">
        <div className="text-center mb-6">
          <div className="text-4xl text-slate-900 dark:text-white mb-2 font-mono">{formatTime(time)}</div>
          <p className="text-sm text-slate-500 dark:text-slate-400">
            {isTracking ? 'Aktive Zeiterfassung' : 'Bereit zum Starten'}
          </p>
        </div>

        {/* Project Selection */}
        <div className="mb-4">
          <label className="text-xs text-slate-600 dark:text-slate-400 mb-2 block">Projekt</label>
          <select className="w-full px-3 py-2 bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 text-slate-900 dark:text-white rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500">
            <option>Website Relaunch</option>
            <option>Brand Refresh</option>
            <option>App Development</option>
          </select>
        </div>

        {/* Task Input */}
        <div className="mb-6">
          <label className="text-xs text-slate-600 dark:text-slate-400 mb-2 block">Aufgabe</label>
          <input
            type="text"
            placeholder="Was machst du gerade?"
            className="w-full px-3 py-2 bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 text-slate-900 dark:text-white placeholder:text-slate-400 dark:placeholder:text-slate-500 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500"
          />
        </div>

        {/* Control Button */}
        <button
          onClick={handleToggle}
          className={`w-full py-3 rounded-lg flex items-center justify-center gap-2 text-white font-medium transition-colors ${
            isTracking
              ? 'bg-red-600 hover:bg-red-700'
              : 'bg-emerald-600 hover:bg-emerald-700'
          }`}
        >
          {isTracking ? (
            <>
              <Pause className="w-5 h-5" />
              Stoppen
            </>
          ) : (
            <>
              <Play className="w-5 h-5" />
              Starten
            </>
          )}
        </button>
      </div>

      {/* Today's Total */}
      <div className="px-4 py-3 bg-slate-50 dark:bg-slate-900 border-t border-slate-200 dark:border-slate-700">
        <div className="flex items-center justify-between text-sm">
          <span className="text-slate-600 dark:text-slate-400">Heute gesamt</span>
          <span className="text-slate-900 dark:text-white font-medium">6h 23m</span>
        </div>
      </div>
    </div>
  );
}