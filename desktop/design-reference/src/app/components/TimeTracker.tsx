import { useState, useEffect } from 'react';
import { Play, Pause, Square, Clock, ChevronDown, ChevronUp } from 'lucide-react';

interface TimeEntry {
  id: string;
  task: string;
  startTime: Date;
  endTime?: Date;
  duration: number; // in seconds
}

interface TimeTrackerProps {
  isExpanded: boolean;
  onToggleExpanded: () => void;
  presets: string[];
}

export function TimeTracker({ isExpanded, onToggleExpanded, presets }: TimeTrackerProps) {
  const [isRunning, setIsRunning] = useState(false);
  const [isPaused, setIsPaused] = useState(false);
  const [currentTask, setCurrentTask] = useState('');
  const [currentSessionTime, setCurrentSessionTime] = useState(0);
  const [totalDayTime, setTotalDayTime] = useState(0);
  const [todayEntries, setTodayEntries] = useState<TimeEntry[]>([]);
  const [showTaskSelect, setShowTaskSelect] = useState(false);

  // Mock data for demonstration
  useEffect(() => {
    // Initialize with some mock entries for today
    const mockEntries: TimeEntry[] = [
      {
        id: '1',
        task: 'Projekt Dokumentation',
        startTime: new Date(Date.now() - 7200000),
        endTime: new Date(Date.now() - 3600000),
        duration: 3600,
      },
      {
        id: '2',
        task: 'Team Meeting',
        startTime: new Date(Date.now() - 3600000),
        endTime: new Date(Date.now() - 1800000),
        duration: 1800,
      },
    ];
    setTodayEntries(mockEntries);
    
    const total = mockEntries.reduce((sum, entry) => sum + entry.duration, 0);
    setTotalDayTime(total);
  }, []);

  // Timer effect
  useEffect(() => {
    let interval: NodeJS.Timeout;
    if (isRunning && !isPaused) {
      interval = setInterval(() => {
        setCurrentSessionTime((prev) => prev + 1);
        setTotalDayTime((prev) => prev + 1);
      }, 1000);
    }
    return () => clearInterval(interval);
  }, [isRunning, isPaused]);

  const formatTime = (seconds: number) => {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;
    return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
  };

  const handleStart = () => {
    if (!currentTask) {
      setShowTaskSelect(true);
      return;
    }
    setIsRunning(true);
    setIsPaused(false);
  };

  const handlePause = () => {
    setIsPaused(true);
  };

  const handleResume = () => {
    setIsPaused(false);
  };

  const handleStop = () => {
    if (currentTask && currentSessionTime > 0) {
      const newEntry: TimeEntry = {
        id: Date.now().toString(),
        task: currentTask,
        startTime: new Date(Date.now() - currentSessionTime * 1000),
        endTime: new Date(),
        duration: currentSessionTime,
      };
      setTodayEntries([...todayEntries, newEntry]);
    }
    setIsRunning(false);
    setIsPaused(false);
    setCurrentSessionTime(0);
    setCurrentTask('');
  };

  const handleTaskChange = (task: string) => {
    setCurrentTask(task);
    setShowTaskSelect(false);
  };

  // Mini view (in dropdown when collapsed)
  if (!isExpanded) {
    return (
      <div className="p-4 border-t border-slate-200 dark:border-slate-700">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <Clock className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
            <span className="text-sm font-medium text-slate-900 dark:text-white">
              Zeiterfassung
            </span>
          </div>
          <button
            onClick={onToggleExpanded}
            className="text-xs text-emerald-600 dark:text-emerald-400 hover:underline"
          >
            Details
          </button>
        </div>

        {/* Current Task Display */}
        {isRunning && currentTask && (
          <div className="mb-3 p-2 bg-emerald-50 dark:bg-emerald-900/20 rounded-lg border border-emerald-200 dark:border-emerald-800">
            <p className="text-xs text-emerald-700 dark:text-emerald-300 mb-1">Aktuelle Aufgabe:</p>
            <p className="text-sm font-medium text-emerald-900 dark:text-emerald-100">{currentTask}</p>
          </div>
        )}

        {/* Time Display */}
        <div className="flex items-center justify-between mb-3">
          <span className="text-xs text-slate-600 dark:text-slate-400">Heute gesamt:</span>
          <span className="text-lg font-mono font-semibold text-slate-900 dark:text-white">
            {formatTime(totalDayTime)}
          </span>
        </div>

        {/* Controls */}
        <div className="flex items-center gap-2">
          {!isRunning ? (
            <button
              onClick={handleStart}
              className="flex-1 flex items-center justify-center gap-2 px-3 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors text-sm"
            >
              <Play className="w-4 h-4" />
              Start
            </button>
          ) : (
            <>
              {isPaused ? (
                <button
                  onClick={handleResume}
                  className="flex-1 flex items-center justify-center gap-2 px-3 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors text-sm"
                >
                  <Play className="w-4 h-4" />
                  Weiter
                </button>
              ) : (
                <button
                  onClick={handlePause}
                  className="flex-1 flex items-center justify-center gap-2 px-3 py-2 bg-yellow-600 text-white rounded-lg hover:bg-yellow-700 transition-colors text-sm"
                >
                  <Pause className="w-4 h-4" />
                  Pause
                </button>
              )}
              <button
                onClick={handleStop}
                className="flex-1 flex items-center justify-center gap-2 px-3 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors text-sm"
              >
                <Square className="w-4 h-4" />
                Stop
              </button>
            </>
          )}
        </div>

        {/* Task Selection Modal */}
        {showTaskSelect && (
          <div className="mt-3 p-3 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg shadow-lg">
            <p className="text-sm font-medium text-slate-900 dark:text-white mb-2">
              Aufgabe auswählen:
            </p>
            <input
              type="text"
              value={currentTask}
              onChange={(e) => setCurrentTask(e.target.value)}
              placeholder="Aufgabe eingeben..."
              className="w-full px-3 py-2 mb-2 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 text-slate-900 dark:text-white placeholder:text-slate-400 dark:placeholder:text-slate-500 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500"
            />
            {presets.length > 0 && (
              <div className="mb-2">
                <p className="text-xs text-slate-600 dark:text-slate-400 mb-1">Oder Preset wählen:</p>
                <div className="space-y-1">
                  {presets.map((preset, index) => (
                    <button
                      key={index}
                      onClick={() => handleTaskChange(preset)}
                      className="w-full text-left px-2 py-1.5 text-sm text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800 rounded transition-colors"
                    >
                      {preset}
                    </button>
                  ))}
                </div>
              </div>
            )}
            <div className="flex gap-2">
              <button
                onClick={() => {
                  setShowTaskSelect(false);
                  if (currentTask) handleStart();
                }}
                disabled={!currentTask}
                className="flex-1 px-3 py-1.5 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors text-sm disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Starten
              </button>
              <button
                onClick={() => setShowTaskSelect(false)}
                className="flex-1 px-3 py-1.5 bg-slate-200 dark:bg-slate-700 text-slate-900 dark:text-white rounded-lg hover:bg-slate-300 dark:hover:bg-slate-600 transition-colors text-sm"
              >
                Abbrechen
              </button>
            </div>
          </div>
        )}
      </div>
    );
  }

  // Expanded view (separate panel)
  return (
    <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
      <div className="bg-white dark:bg-slate-800 rounded-lg shadow-2xl border border-slate-200 dark:border-slate-700 w-full max-w-2xl max-h-[90vh] overflow-hidden flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-slate-200 dark:border-slate-700 bg-emerald-600 text-white">
          <div className="flex items-center gap-2">
            <Clock className="w-5 h-5" />
            <h3 className="font-medium">Zeiterfassung</h3>
          </div>
          <button
            onClick={onToggleExpanded}
            className="p-1 hover:bg-emerald-700 rounded transition-colors"
          >
            <ChevronUp className="w-5 h-5" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6">
          {/* Current Session */}
          <div className="mb-6">
            <h4 className="text-sm font-medium text-slate-900 dark:text-white mb-3">
              Aktuelle Session
            </h4>
            <div className="p-4 bg-slate-50 dark:bg-slate-900 rounded-lg border border-slate-200 dark:border-slate-700">
              <div className="flex items-center justify-between mb-4">
                <span className="text-sm text-slate-600 dark:text-slate-400">Session-Zeit:</span>
                <span className="text-2xl font-mono font-semibold text-slate-900 dark:text-white">
                  {formatTime(currentSessionTime)}
                </span>
              </div>

              {/* Task Selection */}
              <div className="mb-4">
                <label className="block text-sm text-slate-700 dark:text-slate-300 mb-2">
                  Aufgabe:
                </label>
                <input
                  type="text"
                  value={currentTask}
                  onChange={(e) => setCurrentTask(e.target.value)}
                  placeholder="Aufgabe eingeben..."
                  disabled={isRunning && !isPaused}
                  className="w-full px-3 py-2 mb-2 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 text-slate-900 dark:text-white placeholder:text-slate-400 dark:placeholder:text-slate-500 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500 disabled:opacity-50 disabled:cursor-not-allowed"
                />
                {presets.length > 0 && !isRunning && (
                  <div className="flex flex-wrap gap-2">
                    {presets.map((preset, index) => (
                      <button
                        key={index}
                        onClick={() => setCurrentTask(preset)}
                        className="px-3 py-1 text-xs bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 text-slate-700 dark:text-slate-300 rounded-full hover:bg-emerald-50 dark:hover:bg-emerald-900/20 hover:border-emerald-500 dark:hover:border-emerald-400 transition-colors"
                      >
                        {preset}
                      </button>
                    ))}
                  </div>
                )}
              </div>

              {/* Controls */}
              <div className="flex gap-2">
                {!isRunning ? (
                  <button
                    onClick={handleStart}
                    disabled={!currentTask}
                    className="flex-1 flex items-center justify-center gap-2 px-4 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    <Play className="w-4 h-4" />
                    Start
                  </button>
                ) : (
                  <>
                    {isPaused ? (
                      <button
                        onClick={handleResume}
                        className="flex-1 flex items-center justify-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
                      >
                        <Play className="w-4 h-4" />
                        Weiter
                      </button>
                    ) : (
                      <button
                        onClick={handlePause}
                        className="flex-1 flex items-center justify-center gap-2 px-4 py-2 bg-yellow-600 text-white rounded-lg hover:bg-yellow-700 transition-colors"
                      >
                        <Pause className="w-4 h-4" />
                        Pause
                      </button>
                    )}
                    <button
                      onClick={handleStop}
                      className="flex-1 flex items-center justify-center gap-2 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors"
                    >
                      <Square className="w-4 h-4" />
                      Stop
                    </button>
                  </>
                )}
              </div>
            </div>
          </div>

          {/* Today's Summary */}
          <div>
            <div className="flex items-center justify-between mb-3">
              <h4 className="text-sm font-medium text-slate-900 dark:text-white">
                Heute (Übersicht)
              </h4>
              <div className="text-right">
                <p className="text-xs text-slate-600 dark:text-slate-400">Gesamtzeit:</p>
                <p className="text-xl font-mono font-semibold text-emerald-600 dark:text-emerald-400">
                  {formatTime(totalDayTime)}
                </p>
              </div>
            </div>

            {/* Entries List */}
            <div className="space-y-2">
              {todayEntries.map((entry) => (
                <div
                  key={entry.id}
                  className="p-3 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex-1">
                      <p className="text-sm font-medium text-slate-900 dark:text-white mb-1">
                        {entry.task}
                      </p>
                      <p className="text-xs text-slate-500 dark:text-slate-400">
                        {entry.startTime.toLocaleTimeString('de-CH', { hour: '2-digit', minute: '2-digit' })}
                        {' - '}
                        {entry.endTime?.toLocaleTimeString('de-CH', { hour: '2-digit', minute: '2-digit' })}
                      </p>
                    </div>
                    <div className="text-right">
                      <p className="text-sm font-mono font-semibold text-slate-900 dark:text-white">
                        {formatTime(entry.duration)}
                      </p>
                    </div>
                  </div>
                </div>
              ))}
              
              {todayEntries.length === 0 && (
                <div className="p-8 text-center text-slate-500 dark:text-slate-400">
                  <Clock className="w-12 h-12 mx-auto mb-2 opacity-50" />
                  <p className="text-sm">Noch keine Einträge heute</p>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
