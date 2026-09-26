import React, { useState, useEffect } from 'react';
import type {
  RoomMeta,
  RoomConfigMap,
  RoomLayout,
  LayoutItem,
  ToastData,
  ItemType,
} from './types';
import { Sidebar } from './components/Sidebar';
import { Toolbar } from './components/Toolbar';
import { TabBar } from './components/TabBar';
import { MetaEditor } from './components/MetaEditor';
import { Canvas } from './components/Canvas';
import { Inspector } from './components/Inspector';
import { AddRoomModal } from './components/AddRoomModal';
import { Toast } from './components/Toast';

const API = '';

export default function App() {
  // --- STATE ---
  const [rooms, setRooms] = useState<RoomConfigMap>({});
  const [selectedId, setSelectedId] = useState('');
  const [search, setSearch] = useState('');
  const [tab, setTab] = useState<'meta' | 'layout'>('meta');

  const [meta, setMeta] = useState<RoomMeta>({
    layout_file: '',
    layout_image: '',
    map_url: '',
    images: [],
  });
  const [layout, setLayout] = useState<RoomLayout>({
    layout: [],
    frontLabel: 'Front',
    backLabel: 'Back',
  });
  const [loadingLayout, setLoadingLayout] = useState(false);

  const [selBlock, setSelBlock] = useState<number | null>(null);
  const [dragBlock, setDragBlock] = useState<number | null>(null);
  const [dragItem, setDragItem] = useState<number | null>(null);
  const [expandItem, setExpandItem] = useState<number | null>(null);

  const [showAddModal, setShowAddModal] = useState(false);
  const [newId, setNewId] = useState('');
  const [newMeta, setNewMeta] = useState<RoomMeta>({
    layout_file: '',
    layout_image: '',
    map_url: '',
    images: [],
  });
  const [toast, setToast] = useState<ToastData | null>(null);

  const notify = (message: string, type: 'success' | 'error' = 'success') => {
    setToast({ message, type });
    setTimeout(() => setToast(null), 3000);
  };

  // --- DATA FETCHING ---
  const fetchConfig = async () => {
    try {
      const res = await fetch(`${API}/api/config`);
      const data = await res.json();
      setRooms(data);
      if (Object.keys(data).length > 0 && !selectedId) {
        setSelectedId(Object.keys(data).sort()[0]);
      }
    } catch {
      notify('Failed to load config', 'error');
    }
  };

  useEffect(() => {
    fetchConfig();
  }, []);

  useEffect(() => {
    if (selectedId && rooms[selectedId]) {
      setMeta({ ...rooms[selectedId] });
      setSelBlock(null);
      fetchLayout(rooms[selectedId].layout_file);
    }
  }, [selectedId, rooms]);

  const fetchLayout = async (file: string) => {
    if (!file) {
      setLayout({ layout: [], frontLabel: 'Front', backLabel: 'Back' });
      return;
    }
    setLoadingLayout(true);
    try {
      const res = await fetch(`${API}/api/layout/${file}`);
      const d = await res.json();
      setLayout({
        layout: d.layout || [],
        frontLabel: d.frontLabel || 'Front',
        backLabel: d.backLabel || 'Back',
      });
    } catch {
      notify('Could not load layout: ' + file, 'error');
      setLayout({ layout: [], frontLabel: 'Front', backLabel: 'Back' });
    } finally {
      setLoadingLayout(false);
    }
  };

  // --- SAVE / ADD / DELETE ---
  const saveMeta = async () => {
    if (!selectedId) return;
    try {
      const res = await fetch(`${API}/api/config/${selectedId}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(meta),
      });
      const d = await res.json();
      if (d.success) {
        notify('Metadata saved');
        fetchConfig();
      } else notify('Save failed', 'error');
    } catch {
      notify('Server error', 'error');
    }
  };

  const saveLayout = async () => {
    if (!meta.layout_file) {
      notify('Set layout filename first', 'error');
      return;
    }
    try {
      const res = await fetch(`${API}/api/layout/${meta.layout_file}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(layout),
      });
      const d = await res.json();
      if (d.success) notify('Layout saved');
      else notify('Save failed', 'error');
    } catch {
      notify('Server error', 'error');
    }
  };

  const addRoom = async (e: React.FormEvent) => {
    e.preventDefault();
    const id = newId.trim();
    if (!id) {
      notify('Room ID required', 'error');
      return;
    }
    if (rooms[id]) {
      notify('Already exists', 'error');
      return;
    }
    const file =
      newMeta.layout_file.trim() ||
      id.replace(/[^a-zA-Z0-9]/g, '') + '.json';
    try {
      const res = await fetch(`${API}/api/config/${id}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...newMeta, layout_file: file }),
      });
      const d = await res.json();
      if (d.success) {
        notify(`Room ${id} created`);
        setShowAddModal(false);
        setNewId('');
        setNewMeta({ layout_file: '', layout_image: '', map_url: '', images: [] });
        setSelectedId(id);
        fetchConfig();
      } else notify('Failed', 'error');
    } catch {
      notify('Server error', 'error');
    }
  };

  const deleteRoom = async () => {
    if (!selectedId || !window.confirm(`Delete room "${selectedId}"?`)) return;
    try {
      const res = await fetch(`${API}/api/config/${selectedId}`, {
        method: 'DELETE',
      });
      const d = await res.json();
      if (d.success) {
        notify(`${selectedId} deleted`);
        const rest = Object.keys(rooms)
          .filter((k) => k !== selectedId)
          .sort();
        setSelectedId(rest[0] || '');
        fetchConfig();
      } else notify('Delete failed', 'error');
    } catch {
      notify('Server error', 'error');
    }
  };

  // --- BLOCK OPERATIONS ---
  const moveBlock = (from: number, to: number) => {
    if (from === to) return;
    const u = { ...layout };
    const [m] = u.layout.splice(from, 1);
    u.layout.splice(to, 0, m);
    setLayout(u);
    if (selBlock === from) setSelBlock(to);
    else if (selBlock !== null) {
      if (from < selBlock && to >= selBlock) setSelBlock(selBlock - 1);
      else if (from > selBlock && to <= selBlock) setSelBlock(selBlock + 1);
    }
  };

  const addAisle = () => {
    const u = { ...layout };
    u.layout.push({ type: 'aisle', width: 20 });
    setLayout(u);
    setSelBlock(u.layout.length - 1);
  };

  const addColumn = () => {
    const u = { ...layout };
    u.layout.push({
      type: 'column',
      label: `C${u.layout.filter((x) => x.type === 'column').length + 1}`,
      items: [],
    });
    setLayout(u);
    setSelBlock(u.layout.length - 1);
  };

  const deleteBlock = () => {
    if (selBlock === null || !window.confirm('Delete this block?')) return;
    const u = { ...layout };
    u.layout.splice(selBlock, 1);
    setLayout(u);
    setSelBlock(null);
  };

  // --- ITEM OPERATIONS ---
  const moveItem = (bIdx: number, from: number, to: number) => {
    if (from === to) return;
    const u = { ...layout };
    const col = u.layout[bIdx];
    if (col?.type === 'column') {
      const [m] = col.items.splice(from, 1);
      col.items.splice(to, 0, m);
      setLayout(u);
      if (expandItem === from) setExpandItem(to);
      else if (expandItem === to) setExpandItem(from);
    }
  };

  const addItem = (bIdx: number, type: ItemType) => {
    const u = { ...layout };
    const b = u.layout[bIdx];
    if (b?.type === 'column') {
      if (type === 'seats')
        b.items.push({ type: 'seats', char: 'A', start: 1, count: 10 });
      else if (type === 'obstruction')
        b.items.push({
          type: 'obstruction',
          label: 'Pillar',
          count: 1,
          transparent: false,
        });
      else b.items.push({ type: 'gap', count: 1 });
      setLayout(u);
      setExpandItem(b.items.length - 1);
    }
  };

  const updateItem = (
    bIdx: number,
    iIdx: number,
    patch: Partial<LayoutItem>
  ) => {
    const u = { ...layout };
    const b = u.layout[bIdx];
    if (b?.type === 'column') {
      b.items[iIdx] = { ...b.items[iIdx], ...patch } as LayoutItem;
      setLayout(u);
    }
  };

  const deleteItem = (bIdx: number, iIdx: number) => {
    const u = { ...layout };
    const b = u.layout[bIdx];
    if (b?.type === 'column') {
      b.items.splice(iIdx, 1);
      setLayout(u);
      setExpandItem(null);
    }
  };

  // --- RENDER ---
  return (
    <div className="app-shell" onClick={() => setSelBlock(null)}>
      {/* SIDEBAR */}
      <Sidebar
        rooms={rooms}
        selectedId={selectedId}
        onSelectId={setSelectedId}
        search={search}
        onSearchChange={setSearch}
        onAddClick={() => setShowAddModal(true)}
      />

      {/* MAIN */}
      <main className="main" onClick={(e) => e.stopPropagation()}>
        {selectedId ? (
          <>
            {/* Toolbar */}
            <Toolbar
              selectedId={selectedId}
              layoutFile={meta.layout_file}
              tab={tab}
              onDeleteClick={deleteRoom}
              onSaveClick={tab === 'meta' ? saveMeta : saveLayout}
            />

            {/* Tab bar */}
            <TabBar activeTab={tab} onChangeTab={setTab} />

            {/* Workspace */}
            <div className="workspace">
              {tab === 'meta' && (
                <MetaEditor meta={meta} onChangeMeta={setMeta} />
              )}

              {tab === 'layout' && (
                <div className="layout-workspace">
                  {/* Canvas */}
                  <Canvas
                    layout={layout}
                    selBlock={selBlock}
                    dragBlock={dragBlock}
                    loadingLayout={loadingLayout}
                    onSetLayout={setLayout}
                    onSetSelBlock={setSelBlock}
                    onSetDragBlock={setDragBlock}
                    onAddAisle={addAisle}
                    onAddColumn={addColumn}
                    onDeleteBlock={deleteBlock}
                    moveBlock={moveBlock}
                  />

                  {/* Inspector */}
                  <Inspector
                    layout={layout}
                    selBlock={selBlock}
                    expandItem={expandItem}
                    dragItem={dragItem}
                    onSetLayout={setLayout}
                    onSetSelBlock={setSelBlock}
                    onSetExpandItem={setExpandItem}
                    onSetDragItem={setDragItem}
                    moveItem={moveItem}
                    addItem={addItem}
                    updateItem={updateItem}
                    deleteItem={deleteItem}
                  />
                </div>
              )}
            </div>
          </>
        ) : (
          <div className="empty">
            <span className="empty-icon">🗺️</span>
            <h2>Room Config Manager</h2>
            <p>Select a room from the sidebar or click "Add Room" to get started.</p>
          </div>
        )}
      </main>

      {/* ADD ROOM MODAL */}
      {showAddModal && (
        <AddRoomModal
          newId={newId}
          onNewIdChange={setNewId}
          newMeta={newMeta}
          onNewMetaChange={setNewMeta}
          onClose={() => setShowAddModal(false)}
          onSubmit={addRoom}
        />
      )}

      {/* TOAST */}
      {toast && <Toast toast={toast} />}
    </div>
  );
}
