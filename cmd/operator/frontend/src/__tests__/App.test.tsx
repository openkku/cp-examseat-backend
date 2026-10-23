import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { App } from '../App';

/** Renders the app and opens the Details tab of the first room. */
async function openDetails() {
  render(<App />);
  fireEvent.click(await screen.findByRole('button', { name: /^details$/i }));
  return screen.findByLabelText(/display title/i);
}

const json = (body: unknown) => Promise.resolve(new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } }));

describe('App navigation', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = String(input);
      if (url.includes('/api/rooms/layouts/')) return json({ layout: [] });
      if (url.includes('/api/rooms')) return json({ 'CP.9127': { layout_file: 'CP9127.json', layout_image: '', map_url: '', images: [] } });
      if (url.includes('/api/rounds')) return json([]);
      if (url.includes('/api/health')) return json({ problems: [] });
      return json({});
    });
  });

  it('asks before leaving Rooms with unsaved edits', async () => {
    const title = await openDetails();
    fireEvent.change(title, { target: { value: 'Lecture hall' } });

    fireEvent.click(screen.getByRole('button', { name: /rounds & import/i }));
    expect(await screen.findByText(/unsaved room changes/i)).toBeInTheDocument();

    // Stay keeps the draft on the Rooms page.
    fireEvent.click(screen.getByRole('button', { name: /^stay$/i }));
    await waitFor(() => expect(screen.queryByText(/unsaved room changes/i)).not.toBeInTheDocument());
    expect(screen.getByLabelText(/display title/i)).toHaveValue('Lecture hall');

    // Discarding switches section.
    fireEvent.click(screen.getByRole('button', { name: /data health/i }));
    fireEvent.click(await screen.findByRole('button', { name: /discard and leave/i }));
    await waitFor(() => expect(screen.queryByLabelText(/display title/i)).not.toBeInTheDocument());
  });

  it('switches freely when nothing is unsaved', async () => {
    await openDetails();
    fireEvent.click(screen.getByRole('button', { name: /rounds & import/i }));
    expect(screen.queryByText(/unsaved room changes/i)).not.toBeInTheDocument();
    await waitFor(() => expect(screen.queryByLabelText(/display title/i)).not.toBeInTheDocument());
  });
});

describe('Rooms page loading', () => {
  it('fetches the room list once, not in a loop', async () => {
    const urls: string[] = [];
    vi.restoreAllMocks();
    vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
      const url = String(input);
      urls.push(url);
      if (url.includes('/api/rooms/layouts/')) return json({ layout: [] });
      return json({ 'CP.9127': { layout_file: 'CP9127.json', layout_image: '', map_url: '', images: [] } });
    });
    render(<App />);
    await screen.findByRole('button', { name: /^details$/i });
    await new Promise((r) => setTimeout(r, 300));
    expect(urls.filter((u) => u.endsWith('/api/rooms')).length).toBe(1);
  });
});
