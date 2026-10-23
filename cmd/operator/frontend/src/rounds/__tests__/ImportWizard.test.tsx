import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { ImportWizard } from '../ImportWizard';
import { ToastProvider } from '../../ui/Toast';

describe('ImportWizard Flow', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  const renderWizard = (props = {}) => {
    const defaultProps = {
      open: true,
      onClose: vi.fn(),
      onImportComplete: vi.fn(),
      ...props,
    };
    return render(
      <ToastProvider>
        <ImportWizard {...defaultProps} />
      </ToastProvider>
    );
  };

  it('renders with Inspect & Preview disabled until file and round ID are provided', () => {
    renderWizard();

    const inspectBtn = screen.getByRole('button', { name: /inspect & preview/i });
    expect(inspectBtn).toBeDisabled();
    expect(screen.queryByRole('button', { name: /confirm & import/i })).not.toBeInTheDocument();
  });

  it('runs inspection preview, enables confirm, and clears preview on input change', async () => {
    // Mock fetch for preview and import
    const fakePreview = {
      seats: 50,
      students: 48,
      rooms: ['SC1101'],
      dates: ['2026-10-01'],
      sessions: 1,
      custom_ids: [],
      replaces: [{ type: 'in_schedule', id: 'mid_1_2569', seats: 20 }],
      rows: [{ sheet: 'S1', date: '2026-09-01', time: '08.30-11.30', room: 'SC1101', subject: 'CP1001', subject_name: '', section: '1', student_id: '653380001', seat: 'A1', custom_id: '', labels: [] }],
    };

    const fetchMock = vi.fn().mockImplementation((url: string) => {
      if (url === '/api/rounds/import/preview') {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve(fakePreview),
        });
      }
      if (url === '/api/rounds/import') {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ round: 'mid_1_2569', seats: 50 }),
        });
      }
      return Promise.reject(new Error(`Unknown URL: ${url}`));
    });

    globalThis.fetch = fetchMock;

    const onImportComplete = vi.fn();
    renderWizard({ onImportComplete });

    // Enter round ID
    const roundInput = screen.getByLabelText(/round id \*/i);
    fireEvent.change(roundInput, { target: { value: 'mid_1_2569' } });

    // Choose file
    const file = new File(['{"test": true}'], 'seats.json', { type: 'application/json' });
    const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement;
    fireEvent.change(fileInput, { target: { files: [file] } });

    const inspectBtn = screen.getByRole('button', { name: /inspect & preview/i });
    expect(inspectBtn).not.toBeDisabled();

    // Click Inspect & Preview
    fireEvent.click(inspectBtn);

    // Wait for preview content
    await waitFor(() => {
      expect(screen.getByText(/file preview & impact analysis/i)).toBeInTheDocument();
      expect(screen.getByText('50')).toBeInTheDocument(); // total seats
      expect(screen.getByText(/Target Data Replacement:/i)).toBeInTheDocument();
      expect(screen.getByText(/Will replace official in-schedule seats of round/i)).toBeInTheDocument();
    });

    // Confirm button should now be visible
    const confirmBtn = screen.getByRole('button', { name: /confirm & import/i });
    expect(confirmBtn).toBeInTheDocument();

    // Changing any input field clears the preview and returns to Inspect & Preview button
    fireEvent.change(roundInput, { target: { value: 'mid_1_2569_changed' } });

    expect(screen.queryByText(/file preview & impact analysis/i)).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /confirm & import/i })).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: /inspect & preview/i })).toBeInTheDocument();
  });
});
