import { useEffect, useState, useCallback } from 'react';

export function useUnsavedGuard(isDirty: boolean) {
  const [pendingAction, setPendingAction] = useState<(() => void) | null>(null);

  useEffect(() => {
    const handleBeforeUnload = (e: BeforeUnloadEvent) => {
      if (isDirty) {
        e.preventDefault();
        e.returnValue = '';
      }
    };

    window.addEventListener('beforeunload', handleBeforeUnload);
    return () => window.removeEventListener('beforeunload', handleBeforeUnload);
  }, [isDirty]);

  const guardedAction = useCallback(
    (action: () => void) => {
      if (!isDirty) {
        action();
      } else {
        setPendingAction(() => action);
      }
    },
    [isDirty]
  );

  const confirmLeave = useCallback(() => {
    if (pendingAction) {
      pendingAction();
      setPendingAction(null);
    }
  }, [pendingAction]);

  const cancelLeave = useCallback(() => {
    setPendingAction(null);
  }, []);

  return {
    isPromptOpen: pendingAction !== null,
    guardedAction,
    confirmLeave,
    cancelLeave,
  };
}
