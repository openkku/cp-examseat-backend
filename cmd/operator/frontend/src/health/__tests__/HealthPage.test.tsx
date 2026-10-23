import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { HealthPage } from '../HealthPage';
import { ToastProvider } from '../../ui/Toast';
import type { HealthReport } from '../../types';

describe('HealthPage', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('renders all problem kinds and triggers onSelectRoom on click', async () => {
    const fakeReport: HealthReport = {
      problems: [
        {
          kind: 'missing_metadata',
          severity: 'error',
          room: 'UNKNOWN_ROOM',
          round: 'mid_1_2569',
          message: 'Room "UNKNOWN_ROOM" is used in round mid_1_2569 but has no metadata entry',
        },
        {
          kind: 'invalid_layout',
          severity: 'error',
          room: 'SC1102',
          message: 'Layout file "SC1102.json" is unparsable',
        },
        {
          kind: 'missing_seats',
          severity: 'warning',
          room: 'SC1101',
          round: 'mid_1_2569',
          message: '3 seats used in round mid_1_2569 missing from layout (A4, A5, A6)',
          details: ['A4', 'A5', 'A6'],
        },
        {
          kind: 'missing_image',
          severity: 'warning',
          room: 'SC1104',
          message: 'Layout image "/room/image/layout/missing.webp" not found',
        },
        {
          kind: 'unused_room',
          severity: 'info',
          room: 'SC9999',
          message: 'Room SC9999 has no seats in any round',
        },
      ],
    };

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(fakeReport),
    });

    const onSelectRoom = vi.fn();

    render(
      <ToastProvider>
        <HealthPage onSelectRoom={onSelectRoom} />
      </ToastProvider>
    );

    // Wait for all 5 problem kind sections to render
    await waitFor(() => {
      expect(screen.getByText(/rooms in rounds with missing metadata/i)).toBeInTheDocument();
      expect(screen.getByText(/missing or unparsable layout files/i)).toBeInTheDocument();
      expect(screen.getByText(/seats in rounds missing from layout/i)).toBeInTheDocument();
      expect(screen.getByText(/missing room or layout images/i)).toBeInTheDocument();
      expect(screen.getByText(/configured rooms with no seats/i)).toBeInTheDocument();
    });

    // Check specific problem messages
    expect(screen.getByText(/Room "UNKNOWN_ROOM" is used/i)).toBeInTheDocument();
    expect(screen.getByText(/Layout file "SC1102.json" is unparsable/i)).toBeInTheDocument();
    expect(screen.getAllByText(/A4/i).length).toBeGreaterThan(0);

    // Click "Open room" for SC1101
    const openRoomButtons = screen.getAllByRole('button', { name: /open room/i });
    expect(openRoomButtons.length).toBeGreaterThan(0);
    fireEvent.click(openRoomButtons[0]);

    expect(onSelectRoom).toHaveBeenCalled();
  });

  it('renders all healthy state when zero problems detected', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ problems: [] }),
    });

    render(
      <ToastProvider>
        <HealthPage />
      </ToastProvider>
    );

    await waitFor(() => {
      expect(screen.getByText(/all systems healthy/i)).toBeInTheDocument();
      expect(screen.getByText(/zero discrepancies detected/i)).toBeInTheDocument();
    });
  });
});
