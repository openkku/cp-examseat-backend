import type {
  HealthReport,
  ImportPreview,
  PreviewSeat,
  RoomConfigMap,
  RoomLayout,
  RoomMeta,
  RoundSummary,
} from './types';

/** Notice displayed on change: public server refreshes within ~2s (.reload signal). */
export const LIVE_SOON = 'Changes go live within ~2s';

/** Calls the operator API and returns its JSON, throwing the server's error message on failure. */
export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(path, init);
  let body: unknown = null;
  try {
    body = await res.json();
  } catch {
    // Non-JSON response
  }
  if (!res.ok) {
    const message = (body as { error?: string } | null)?.error;
    throw new Error(message || `Request failed (${res.status})`);
  }
  return body as T;
}

const json = (method: string, body?: unknown): RequestInit => ({
  method,
  headers: { 'Content-Type': 'application/json' },
  body: body === undefined ? undefined : JSON.stringify(body),
});

export const roomsApi = {
  list: () => api<RoomConfigMap>('/api/rooms'),
  save: (roomId: string, meta: RoomMeta) =>
    api<{ success: boolean }>(`/api/rooms/${encodeURIComponent(roomId)}`, json('POST', meta)),
  remove: (roomId: string) =>
    api<{ success: boolean }>(`/api/rooms/${encodeURIComponent(roomId)}`, { method: 'DELETE' }),
  layout: (filename: string) =>
    api<RoomLayout>(`/api/rooms/layouts/${encodeURIComponent(filename)}`),
  saveLayout: (filename: string, layout: RoomLayout) =>
    api<{ success: boolean; seats: number }>(`/api/rooms/layouts/${encodeURIComponent(filename)}`, json('POST', layout)),
};

export const roundsApi = {
  list: () => api<RoundSummary[]>('/api/rounds'),
  rename: (round: string, label: string) =>
    api<{ round: string; label: string }>(`/api/rounds/${encodeURIComponent(round)}`, json('PATCH', { label })),
  remove: (round: string, customId?: string) =>
    api(`/api/rounds/${encodeURIComponent(round)}${customId ? `?custom_id=${encodeURIComponent(customId)}` : ''}`, { method: 'DELETE' }),
  previewImport: (form: FormData) =>
    api<ImportPreview>('/api/rounds/import/preview', {
      method: 'POST',
      body: form,
      headers: { 'X-Operator': '1' },
    }),
  importFile: (form: FormData) =>
    api<{ round: string; seats: number }>('/api/rounds/import', {
      method: 'POST',
      body: form,
      headers: { 'X-Operator': '1' },
    }),
  reload: () => api('/api/rounds/reload', json('POST')),
};

export const previewApi = {
  options: (type: 'dates' | 'times', round: string, room?: string, date?: string) => {
    const params = new URLSearchParams({ type, round });
    if (room) params.set('room', room);
    if (date) params.set('date', date);
    return api<string[]>(`/api/preview/options?${params.toString()}`);
  },
  seats: (round: string, room: string, date: string, time: string) => {
    const params = new URLSearchParams({ round, room, date, time });
    return api<PreviewSeat[]>(`/api/preview/seats?${params.toString()}`);
  },
};

export const healthApi = {
  check: () => api<HealthReport>('/api/health'),
};
