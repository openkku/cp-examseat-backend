import { useReducer, useEffect, useCallback } from 'react';
import type { RoomLayout } from '../../types';

export interface HistoryState {
  past: RoomLayout[];
  present: RoomLayout;
  future: RoomLayout[];
  initialJson: string;
}

export type HistoryAction =
  | { type: 'LOAD'; layout: RoomLayout }
  | { type: 'MODIFY'; layout: RoomLayout }
  | { type: 'UNDO' }
  | { type: 'REDO' }
  | { type: 'RESET_DIRTY' };

const MAX_HISTORY = 50;

export function historyReducer(state: HistoryState, action: HistoryAction): HistoryState {
  switch (action.type) {
    case 'LOAD': {
      return {
        past: [],
        present: action.layout,
        future: [],
        initialJson: JSON.stringify(action.layout),
      };
    }
    case 'MODIFY': {
      if (JSON.stringify(state.present) === JSON.stringify(action.layout)) {
        return state;
      }
      const newPast = [...state.past, state.present].slice(-MAX_HISTORY);
      return {
        past: newPast,
        present: action.layout,
        future: [],
        initialJson: state.initialJson,
      };
    }
    case 'UNDO': {
      if (state.past.length === 0) return state;
      const previous = state.past[state.past.length - 1];
      const newPast = state.past.slice(0, state.past.length - 1);
      return {
        past: newPast,
        present: previous,
        future: [state.present, ...state.future],
        initialJson: state.initialJson,
      };
    }
    case 'REDO': {
      if (state.future.length === 0) return state;
      const next = state.future[0];
      const newFuture = state.future.slice(1);
      return {
        past: [...state.past, state.present],
        present: next,
        future: newFuture,
        initialJson: state.initialJson,
      };
    }
    case 'RESET_DIRTY': {
      return {
        ...state,
        initialJson: JSON.stringify(state.present),
      };
    }
    default:
      return state;
  }
}

export function useLayoutHistory(initialLayout: RoomLayout = { layout: [] }) {
  const [state, dispatch] = useReducer(historyReducer, {
    past: [],
    present: initialLayout,
    future: [],
    initialJson: JSON.stringify(initialLayout),
  });

  const load = useCallback((layout: RoomLayout) => {
    dispatch({ type: 'LOAD', layout });
  }, []);

  const modify = useCallback((layout: RoomLayout) => {
    dispatch({ type: 'MODIFY', layout });
  }, []);

  const undo = useCallback(() => {
    dispatch({ type: 'UNDO' });
  }, []);

  const redo = useCallback(() => {
    dispatch({ type: 'REDO' });
  }, []);

  const markSaved = useCallback(() => {
    dispatch({ type: 'RESET_DIRTY' });
  }, []);

  const isDirty = JSON.stringify(state.present) !== state.initialJson;
  const canUndo = state.past.length > 0;
  const canRedo = state.future.length > 0;

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement;
      if (
        target.tagName === 'INPUT' ||
        target.tagName === 'TEXTAREA' ||
        target.tagName === 'SELECT' ||
        target.isContentEditable
      ) {
        return;
      }

      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'z') {
        if (e.shiftKey) {
          e.preventDefault();
          if (canRedo) redo();
        } else {
          e.preventDefault();
          if (canUndo) undo();
        }
      } else if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'y') {
        e.preventDefault();
        if (canRedo) redo();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [canUndo, canRedo, undo, redo]);

  return {
    layout: state.present,
    load,
    modify,
    undo,
    redo,
    markSaved,
    isDirty,
    canUndo,
    canRedo,
  };
}
