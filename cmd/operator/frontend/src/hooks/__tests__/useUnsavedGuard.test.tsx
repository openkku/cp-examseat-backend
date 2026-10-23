import { describe, it, expect, vi } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useUnsavedGuard } from '../useUnsavedGuard';

describe('useUnsavedGuard', () => {
  it('executes action immediately when not dirty', () => {
    const { result } = renderHook(() => useUnsavedGuard(false));
    const action = vi.fn();

    act(() => {
      result.current.guardedAction(action);
    });

    expect(action).toHaveBeenCalledTimes(1);
    expect(result.current.isPromptOpen).toBe(false);
  });

  it('blocks action and opens confirmation dialog when dirty', () => {
    const { result } = renderHook(() => useUnsavedGuard(true));
    const action = vi.fn();

    act(() => {
      result.current.guardedAction(action);
    });

    expect(action).not.toHaveBeenCalled();
    expect(result.current.isPromptOpen).toBe(true);

    // Cancel leaves action uncalled
    act(() => {
      result.current.cancelLeave();
    });
    expect(action).not.toHaveBeenCalled();
    expect(result.current.isPromptOpen).toBe(false);
  });

  it('runs pending action when confirmLeave is called', () => {
    const { result } = renderHook(() => useUnsavedGuard(true));
    const action = vi.fn();

    act(() => {
      result.current.guardedAction(action);
    });

    expect(result.current.isPromptOpen).toBe(true);
    expect(action).not.toHaveBeenCalled();

    act(() => {
      result.current.confirmLeave();
    });

    expect(action).toHaveBeenCalledTimes(1);
    expect(result.current.isPromptOpen).toBe(false);
  });
});
