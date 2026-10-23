import React, { useState } from 'react';
import { Calendar, DoorOpen, Activity, Sun, Moon } from 'lucide-react';
import { useTheme } from './hooks/useTheme';
import { LIVE_SOON } from './api';
import { ToastProvider } from './ui/Toast';
import { RoundsPage } from './rounds/RoundsPage';
import { RoomsPage } from './rooms/RoomsPage';
import { HealthPage } from './health/HealthPage';
import { ConfirmDialog } from './ui/Dialog';
import { useUnsavedGuard } from './hooks/useUnsavedGuard';

export type NavSection = 'rounds' | 'rooms' | 'health';

export const AppContent: React.FC = () => {
  const [section, setSection] = useState<NavSection>('rooms');
  const [targetRoomId, setTargetRoomId] = useState<string | undefined>(undefined);
  // Unsaved room edits; switching section would discard them.
  const [roomsDirty, setRoomsDirty] = useState(false);
  const { isPromptOpen, guardedAction, confirmLeave, cancelLeave } = useUnsavedGuard(roomsDirty);

  const { theme, toggleTheme } = useTheme();

  const goTo = (next: NavSection) => {
    if (next !== section) guardedAction(() => setSection(next));
  };

  const handleSelectRoomFromHealth = (roomId: string) => {
    setTargetRoomId(roomId);
    setSection('rooms');
  };

  const navItem = (id: NavSection, label: string, Icon: typeof Calendar) => {
    const active = section === id;
    return (
      <button
        onClick={() => goTo(id)}
        className={`w-full flex items-center gap-2.5 px-3 py-2.5 rounded-xl text-xs font-bold transition-all ${
          active
            ? 'bg-faculty-light text-faculty dark:bg-blue-950/50 dark:text-blue-300 shadow-sm shadow-blue-950/5 dark:shadow-none'
            : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100/80 dark:hover:bg-slate-800/60 hover:text-slate-900 dark:hover:text-slate-100'
        }`}
      >
        <Icon className="w-4 h-4 shrink-0" />
        <span>{label}</span>
      </button>
    );
  };

  return (
    <div className="relative flex h-screen w-screen overflow-hidden bg-slate-50 dark:bg-slate-950 text-slate-800 dark:text-slate-200 font-sans transition-colors duration-300">
      {/* Daylight glow, as on the public site */}
      <div className="absolute top-[-18%] left-[-14%] w-[54%] h-[54%] rounded-full bg-sky-300/14 dark:bg-sky-900/8 blur-[140px] pointer-events-none z-0" />
      <div className="absolute bottom-[-18%] right-[-14%] w-[54%] h-[54%] rounded-full bg-teal-300/14 dark:bg-teal-900/8 blur-[140px] pointer-events-none z-0" />

      {/* Left Navigation Bar */}
      <aside className="relative z-10 w-56 border-r border-slate-200/70 dark:border-slate-800/70 bg-white/82 dark:bg-slate-900/82 backdrop-blur-xl flex flex-col shrink-0">
        <div className="h-16 px-4 border-b border-slate-200/70 dark:border-slate-800/70 flex items-center gap-2.5">
          <div className="bg-faculty text-white w-9 h-9 rounded-2xl flex items-center justify-center font-black text-sm shadow-sm shadow-faculty/25 dark:shadow-none">
            CP
          </div>
          <div className="flex flex-col">
            <span className="font-black text-sm leading-none tracking-tight text-slate-800 dark:text-slate-200">Exam Seat</span>
            <span className="text-[9px] font-bold text-slate-400 uppercase tracking-widest leading-none mt-1">Operator</span>
          </div>
        </div>

        <nav className="flex-1 p-2.5 space-y-1">
          {navItem('rounds', 'Rounds & Import', Calendar)}
          {navItem('rooms', 'Rooms & Layouts', DoorOpen)}
          {navItem('health', 'Data Health', Activity)}
        </nav>

        {/* Loopback Security Notice */}
        <div className="p-3 border-t border-slate-200/70 dark:border-slate-800/70 text-[10px] text-slate-400 leading-tight">
          <span className="font-bold">Loopback tool only</span>
          <div className="mt-0.5">Port 8081 · SSH Tunnel</div>
        </div>
      </aside>

      {/* Main Content Area */}
      <div className="relative z-10 flex-1 flex flex-col h-full overflow-hidden">
        {/* Top Header Live Status */}
        <header className="h-16 border-b border-slate-200/70 dark:border-slate-800/70 bg-white/82 dark:bg-slate-900/82 backdrop-blur-xl px-6 flex items-center justify-between shrink-0">
          <div className="flex items-center gap-2 text-xs font-semibold text-slate-500 dark:text-slate-400">
            <span className="w-2 h-2 rounded-full bg-teal-500 animate-pulse" />
            <span>{LIVE_SOON}</span>
          </div>

          <button
            onClick={toggleTheme}
            className="min-w-10 min-h-10 p-2 rounded-xl border border-slate-200/70 dark:border-slate-800/80 hover:bg-faculty-light/60 dark:hover:bg-blue-950/30 text-slate-500 dark:text-slate-400 hover:text-faculty dark:hover:text-blue-200 transition-all cursor-pointer shadow-sm dark:shadow-none active:scale-95 flex items-center justify-center"
            title={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
            aria-label="Toggle dark mode"
          >
            {theme === 'dark' ? <Sun className="w-4 h-4" /> : <Moon className="w-4 h-4" />}
          </button>
        </header>

        {/* Viewport */}
        <main className="flex-1 overflow-auto">
          {section === 'rounds' && <RoundsPage />}
          {section === 'rooms' && <RoomsPage initialRoomId={targetRoomId} onDirtyChange={setRoomsDirty} />}
          {section === 'health' && <HealthPage onSelectRoom={handleSelectRoomFromHealth} />}
        </main>
      </div>

      <ConfirmDialog
        open={isPromptOpen}
        onClose={cancelLeave}
        onConfirm={confirmLeave}
        title="Unsaved Changes"
        message="You have unsaved room changes that will be lost if you leave this view. Are you sure you want to discard them?"
        confirmText="Discard and Leave"
        cancelText="Stay"
        variant="danger"
      />
    </div>
  );
};

export const App: React.FC = () => {
  return (
    <ToastProvider>
      <AppContent />
    </ToastProvider>
  );
};

export default App;
