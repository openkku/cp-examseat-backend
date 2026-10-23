import React, { useState, useCallback, useMemo } from 'react';
import { CheckCircle2, AlertCircle, Info, X } from 'lucide-react';
import { ToastContext, type ToastItem, type ToastType } from './useToast';

export const ToastProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [toasts, setToasts] = useState<ToastItem[]>([]);

  const addToast = useCallback((type: ToastType, message: string) => {
    const id = Math.random().toString(36).slice(2, 9);
    setToasts((prev) => [...prev, { id, type, message }]);

    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, 4000);
  }, []);

  const removeToast = (id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  };

  // Memoized so consumers can list `toast` as an effect dependency.
  const value = useMemo(
    () => ({
      toast: {
        success: (msg: string) => addToast('success', msg),
        error: (msg: string) => addToast('error', msg),
        info: (msg: string) => addToast('info', msg),
      },
    }),
    [addToast]
  );

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="fixed bottom-4 right-4 z-50 flex flex-col gap-2 max-w-sm w-full pointer-events-none">
        {toasts.map((t) => (
          <div
            key={t.id}
            className={`pointer-events-auto flex items-center justify-between p-3 rounded-xl shadow-lg dark:shadow-none border text-sm animate-in slide-in-from-bottom-2 duration-150 ${
              t.type === 'success'
                ? 'bg-teal-50 dark:bg-teal-950/80 border-teal-300 dark:border-teal-800 text-teal-800 dark:text-teal-200'
                : t.type === 'error'
                ? 'bg-rose-50 dark:bg-rose-950/80 border-rose-300 dark:border-rose-800 text-rose-800 dark:text-rose-200'
                : 'bg-slate-50 dark:bg-slate-900 border-slate-300 dark:border-slate-700 text-slate-800 dark:text-slate-200'
            }`}
          >
            <div className="flex items-center gap-2">
              {t.type === 'success' && <CheckCircle2 className="w-4 h-4 text-teal-600 shrink-0" />}
              {t.type === 'error' && <AlertCircle className="w-4 h-4 text-rose-600 shrink-0" />}
              {t.type === 'info' && <Info className="w-4 h-4 text-faculty dark:text-blue-300 shrink-0" />}
              <span className="font-semibold text-xs break-all">{t.message}</span>
            </div>
            <button
              onClick={() => removeToast(t.id)}
              className="text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 ml-2"
              aria-label="Close notification"
            >
              <X className="w-3.5 h-3.5" />
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
};
