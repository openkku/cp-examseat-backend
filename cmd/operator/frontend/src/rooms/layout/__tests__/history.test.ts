import { describe, it, expect } from 'vitest';
import { historyReducer, type HistoryState } from '../history';
import type { RoomLayout } from '../../../types';

describe('Layout History Reducer (undo/redo/dirty)', () => {
  const initialLayout: RoomLayout = { layout: [{ type: 'column', label: 'C1', items: [] }] };

  const getInitialState = (): HistoryState => ({
    past: [],
    present: initialLayout,
    future: [],
    initialJson: JSON.stringify(initialLayout),
  });

  it('initializes clean with no undo/redo', () => {
    const state = getInitialState();
    expect(state.past.length).toBe(0);
    expect(state.future.length).toBe(0);
    expect(JSON.stringify(state.present) === state.initialJson).toBe(true);
  });

  it('records modifications and sets dirty flag', () => {
    let state = getInitialState();
    const modifiedLayout: RoomLayout = {
      layout: [
        { type: 'column', label: 'C1', items: [{ type: 'seats', char: 'A', count: 5 }] },
      ],
    };

    state = historyReducer(state, { type: 'MODIFY', layout: modifiedLayout });
    expect(state.past.length).toBe(1);
    expect(state.present).toEqual(modifiedLayout);
    expect(state.future.length).toBe(0);
    expect(JSON.stringify(state.present) !== state.initialJson).toBe(true);
  });

  it('handles undo and redo correctly', () => {
    let state = getInitialState();
    const layout1: RoomLayout = { layout: [{ type: 'column', label: 'C1_MOD', items: [] }] };
    const layout2: RoomLayout = { layout: [{ type: 'column', label: 'C2_MOD', items: [] }] };

    state = historyReducer(state, { type: 'MODIFY', layout: layout1 });
    state = historyReducer(state, { type: 'MODIFY', layout: layout2 });
    expect(state.past.length).toBe(2);

    // Undo step 1
    state = historyReducer(state, { type: 'UNDO' });
    expect(state.present).toEqual(layout1);
    expect(state.past.length).toBe(1);
    expect(state.future.length).toBe(1);
    expect(state.future[0]).toEqual(layout2);

    // Undo step 2
    state = historyReducer(state, { type: 'UNDO' });
    expect(state.present).toEqual(initialLayout);
    expect(state.past.length).toBe(0);
    expect(state.future.length).toBe(2);

    // Redo step 1
    state = historyReducer(state, { type: 'REDO' });
    expect(state.present).toEqual(layout1);
    expect(state.past.length).toBe(1);
    expect(state.future.length).toBe(1);

    // Redo step 2
    state = historyReducer(state, { type: 'REDO' });
    expect(state.present).toEqual(layout2);
    expect(state.past.length).toBe(2);
    expect(state.future.length).toBe(0);
  });

  it('resets dirty flag when RESET_DIRTY is dispatched on save', () => {
    let state = getInitialState();
    const modifiedLayout: RoomLayout = { layout: [] };
    state = historyReducer(state, { type: 'MODIFY', layout: modifiedLayout });
    expect(JSON.stringify(state.present) !== state.initialJson).toBe(true);

    state = historyReducer(state, { type: 'RESET_DIRTY' });
    expect(JSON.stringify(state.present) === state.initialJson).toBe(true);
  });
});
