import React, { useState, useEffect, useMemo, useCallback } from 'react';
import {
  Plus,
  Trash2,
  Search,
  Undo2,
  Redo2,
  Save,
  Users,
  Eye,
  Settings2,
  DoorOpen,
} from 'lucide-react';
import { roomsApi, LIVE_SOON } from '../api';
import type { PreviewSeat, RoomConfigMap, RoomMeta } from '../types';
import { Button } from '../ui/Button';
import { Input } from '../ui/Input';
import { Badge } from '../ui/Badge';
import { ConfirmDialog, Dialog } from '../ui/Dialog';
import { useToast } from '../ui/useToast';
import { useUnsavedGuard } from '../hooks/useUnsavedGuard';
import { DetailsForm } from './DetailsForm';
import { LayoutCanvas, type Selection } from './layout/LayoutCanvas';
import { SidePanel } from './layout/SidePanel';
import { StudentOverlay } from './layout/StudentOverlay';
import { useLayoutHistory } from './layout/history';
import { computeSeatIDs } from './layout/model';

interface RoomsPageProps {
  initialRoomId?: string;
  /** Reports whether the room has unsaved edits (details or layout), so the app can guard navigation. */
  onDirtyChange?: (dirty: boolean) => void;
}

export const RoomsPage: React.FC<RoomsPageProps> = ({ initialRoomId, onDirtyChange }) => {
  const { toast } = useToast();

  const [rooms, setRooms] = useState<RoomConfigMap>({});
  const [loading, setLoading] = useState(true);
  const [selectedRoomId, setSelectedRoomId] = useState<string>('');
  const [search, setSearch] = useState('');
  const [activeTab, setActiveTab] = useState<'layout' | 'details'>('layout');

  // New room modal
  const [addModalOpen, setAddModalOpen] = useState(false);
  const [newRoomId, setNewRoomId] = useState('');
  const [newRoomLayoutFile, setNewRoomLayoutFile] = useState('');
  const [isAddingRoom, setIsAddingRoom] = useState(false);

  // Delete room dialog
  const [roomToDelete, setRoomToDelete] = useState<string | null>(null);
  const [isDeletingRoom, setIsDeletingRoom] = useState(false);

  // Room metadata form draft
  const [metaDraft, setMetaDraft] = useState<RoomMeta>({
    layout_file: '',
    layout_image: '',
    map_url: '',
    images: [],
  });
  const [initialMetaJson, setInitialMetaJson] = useState<string>('');
  const [isSavingMeta, setIsSavingMeta] = useState(false);

  // Layout history hook
  const {
    layout,
    load: loadLayout,
    modify: modifyLayout,
    undo,
    redo,
    markSaved: markLayoutSaved,
    isDirty: isLayoutDirty,
    canUndo,
    canRedo,
  } = useLayoutHistory({ layout: [] });

  const [canvasSelection, setCanvasSelection] = useState<Selection>(null);
  const [isSavingLayout, setIsSavingLayout] = useState(false);

  // Student roster overlay
  const [showStudentOverlay, setShowStudentOverlay] = useState(false);
  const [studentRoster, setStudentRoster] = useState<Record<string, PreviewSeat>>({});

  const isMetaDirty = JSON.stringify(metaDraft) !== initialMetaJson;
  const isDirty = activeTab === 'layout' ? isLayoutDirty : isMetaDirty;

  const { isPromptOpen, guardedAction, confirmLeave, cancelLeave } = useUnsavedGuard(isDirty);

  const hasUnsavedEdits = isLayoutDirty || isMetaDirty;
  useEffect(() => {
    onDirtyChange?.(hasUnsavedEdits);
  }, [hasUnsavedEdits, onDirtyChange]);
  // Leaving the page drops its drafts, so it is clean again afterwards.
  useEffect(() => () => onDirtyChange?.(false), [onDirtyChange]);

  // Stable (no dependency on `rooms`): callers pass the room list, so the
  // load effect below runs once instead of refetching on every render.
  const selectRoom = useCallback((id: string, currentRooms: RoomConfigMap) => {
    setSelectedRoomId(id);
    const currentMeta = currentRooms[id] || { layout_file: '', layout_image: '', map_url: '', images: [] };
    setMetaDraft({ ...currentMeta });
    setInitialMetaJson(JSON.stringify(currentMeta));
    setCanvasSelection(null);
    setShowStudentOverlay(false);
    setStudentRoster({});
  }, []);

  // Fetch rooms list
  const fetchRooms = useCallback(async () => {
    try {
      setLoading(true);
      const data = await roomsApi.list();
      setRooms(data);
      const roomKeys = Object.keys(data);
      if (roomKeys.length > 0 && !selectedRoomId) {
        const nextId = initialRoomId && data[initialRoomId] ? initialRoomId : roomKeys[0];
        selectRoom(nextId, data);
      }
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to load rooms');
    } finally {
      setLoading(false);
    }
  }, [selectedRoomId, initialRoomId, selectRoom, toast]);

  useEffect(() => {
    let ignore = false;
    roomsApi
      .list()
      .then((data) => {
        if (!ignore) {
          setRooms(data);
          const roomKeys = Object.keys(data);
          if (roomKeys.length > 0) {
            const nextId = initialRoomId && data[initialRoomId] ? initialRoomId : roomKeys[0];
            selectRoom(nextId, data);
          }
          setLoading(false);
        }
      })
      .catch((err: unknown) => {
        if (!ignore) {
          toast.error((err as Error).message || 'Failed to load rooms');
          setLoading(false);
        }
      });
    return () => {
      ignore = true;
    };
  }, [initialRoomId, selectRoom, toast]);

  // Load selected room layout
  useEffect(() => {
    if (!selectedRoomId || !rooms[selectedRoomId]) return;

    const currentMeta = rooms[selectedRoomId];
    let ignore = false;
    if (currentMeta.layout_file) {
      roomsApi
        .layout(currentMeta.layout_file)
        .then((l) => {
          if (!ignore) loadLayout(l);
        })
        .catch(() => {
          if (!ignore) loadLayout({ layout: [] });
        });
    } else {
      loadLayout({ layout: [] });
    }
    return () => {
      ignore = true;
    };
  }, [selectedRoomId, rooms, loadLayout]);

  // Computed seat count
  const seatIDs = useMemo(() => computeSeatIDs(layout), [layout]);
  const seatSet = useMemo(() => new Set(seatIDs), [seatIDs]);

  // Save Layout
  const handleSaveLayout = async () => {
    const meta = rooms[selectedRoomId];
    if (!meta?.layout_file) {
      toast.error('This room does not have a layout_file configured. Please set it in Details.');
      return;
    }

    try {
      setIsSavingLayout(true);
      const res = await roomsApi.saveLayout(meta.layout_file, layout);
      markLayoutSaved();
      toast.success(`Saved layout (${res.seats} seats) ${LIVE_SOON}`);
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to save layout');
    } finally {
      setIsSavingLayout(false);
    }
  };

  // Save Meta Details
  const handleSaveMeta = async () => {
    try {
      setIsSavingMeta(true);
      await roomsApi.save(selectedRoomId, metaDraft);
      setInitialMetaJson(JSON.stringify(metaDraft));
      toast.success(`Saved details for ${selectedRoomId} ${LIVE_SOON}`);
      await fetchRooms();
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to save room details');
    } finally {
      setIsSavingMeta(false);
    }
  };

  // Add Room
  const handleAddRoom = async () => {
    const trimmedId = newRoomId.trim();
    if (!trimmedId) return;

    try {
      setIsAddingRoom(true);
      const defaultLayoutFile = newRoomLayoutFile.trim() || `${trimmedId.replace(/[^A-Za-z0-9]/g, '')}.json`;
      const meta: RoomMeta = {
        layout_file: defaultLayoutFile,
        layout_image: '',
        map_url: '',
        images: [],
      };
      await roomsApi.save(trimmedId, meta);
      toast.success(`Created room ${trimmedId}`);
      setAddModalOpen(false);
      setNewRoomId('');
      setNewRoomLayoutFile('');
      await fetchRooms();
      setSelectedRoomId(trimmedId);
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to create room');
    } finally {
      setIsAddingRoom(false);
    }
  };

  // Delete Room
  const handleDeleteRoom = async () => {
    if (!roomToDelete) return;
    try {
      setIsDeletingRoom(true);
      await roomsApi.remove(roomToDelete);
      toast.success(`Deleted room ${roomToDelete} ${LIVE_SOON}`);
      setRoomToDelete(null);
      await fetchRooms();
      const remaining = Object.keys(rooms).filter((id) => id !== roomToDelete);
      if (remaining.length > 0) {
        setSelectedRoomId(remaining[0]);
      }
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to delete room');
    } finally {
      setIsDeletingRoom(false);
    }
  };

  const filteredRoomIds = useMemo(() => {
    return Object.keys(rooms).filter((id) =>
      id.toLowerCase().includes(search.toLowerCase())
    );
  }, [rooms, search]);

  return (
    <div className="flex h-[calc(100vh-4rem)] w-full overflow-hidden">
      {/* Rooms List Left Sidebar */}
      <div className="w-64 border-r border-slate-200/70 dark:border-slate-800/70 bg-white/70 dark:bg-slate-900/70 backdrop-blur-xl flex flex-col shrink-0">
        <div className="p-3 border-b border-slate-200 dark:border-slate-800 flex items-center justify-between gap-2">
          <div className="flex items-center gap-1.5 font-bold text-xs uppercase tracking-wider text-slate-500">
            <DoorOpen className="w-4 h-4 text-faculty dark:text-blue-300" />
            <span>Rooms ({Object.keys(rooms).length})</span>
          </div>
          <Button variant="primary" size="sm" onClick={() => setAddModalOpen(true)}>
            <Plus className="w-3.5 h-3.5" />
            Add
          </Button>
        </div>

        <div className="p-2 border-b border-slate-200 dark:border-slate-800">
          <div className="relative">
            <Search className="w-3.5 h-3.5 absolute left-2.5 top-2.75 text-slate-400" />
            <input
              type="text"
              placeholder="Search rooms..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full text-xs rounded-xl pl-8 pr-2 py-2 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-800 shadow-sm dark:shadow-none text-slate-800 dark:text-slate-200 focus:outline-none focus:border-faculty focus:ring-2 focus:ring-faculty/25"
            />
          </div>
        </div>

        <div className="flex-1 overflow-y-auto p-1.5 space-y-0.5">
          {loading ? (
            <div className="p-4 text-center text-xs text-slate-400">Loading rooms...</div>
          ) : filteredRoomIds.length === 0 ? (
            <div className="p-4 text-center text-xs text-slate-400">No rooms found</div>
          ) : (
            filteredRoomIds.map((id) => {
              const isSelected = id === selectedRoomId;
              const rMeta = rooms[id];

              return (
                <div
                  key={id}
                  onClick={() => guardedAction(() => setSelectedRoomId(id))}
                  className={`group flex items-center justify-between px-2.5 py-2 rounded-xl cursor-pointer transition-all text-xs font-semibold ${
                    isSelected
                      ? 'bg-faculty-light text-faculty dark:bg-blue-950/50 dark:text-blue-300 font-bold shadow-sm shadow-blue-950/5 dark:shadow-none'
                      : 'text-slate-700 dark:text-slate-300 hover:bg-slate-100/80 dark:hover:bg-slate-800/60'
                  }`}
                >
                  <div className="truncate">
                    <div className="font-mono">{id}</div>
                    <div className="text-[10px] text-slate-400 font-normal truncate">
                      {rMeta.layout_file || 'No layout file'}
                    </div>
                  </div>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      setRoomToDelete(id);
                    }}
                    className="opacity-0 group-hover:opacity-100 p-1 text-slate-400 hover:text-rose-600 transition-opacity rounded"
                    title={`Delete room ${id}`}
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                </div>
              );
            })
          )}
        </div>
      </div>

      {/* Main Workspace */}
      <div className="flex-1 flex flex-col h-full overflow-hidden">
        {selectedRoomId ? (
          <>
            {/* Top Toolbar */}
            <div className="h-13 px-4 border-b border-slate-200/70 dark:border-slate-800/70 bg-white/70 dark:bg-slate-900/70 backdrop-blur-xl flex items-center justify-between shrink-0">
              <div className="flex items-center gap-3">
                <h1 className="text-sm font-black font-mono tracking-tight text-slate-800 dark:text-slate-100">
                  {selectedRoomId}
                </h1>

                {/* Tabs */}
                <div className="flex items-center p-1 bg-slate-100 dark:bg-slate-800 rounded-xl text-xs font-bold">
                  <button
                    onClick={() => guardedAction(() => setActiveTab('layout'))}
                    className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg transition-all ${
                      activeTab === 'layout'
                        ? 'bg-white dark:bg-slate-900 text-faculty dark:text-blue-300 shadow-sm dark:shadow-none'
                        : 'text-slate-500 hover:text-slate-800 dark:hover:text-slate-200'
                    }`}
                  >
                    <Eye className="w-3.5 h-3.5" />
                    Layout Canvas
                  </button>
                  <button
                    onClick={() => guardedAction(() => setActiveTab('details'))}
                    className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg transition-all ${
                      activeTab === 'details'
                        ? 'bg-white dark:bg-slate-900 text-faculty dark:text-blue-300 shadow-sm dark:shadow-none'
                        : 'text-slate-500 hover:text-slate-800 dark:hover:text-slate-200'
                    }`}
                  >
                    <Settings2 className="w-3.5 h-3.5" />
                    Details
                  </button>
                </div>

                {activeTab === 'layout' && (
                  <Badge variant="neutral" size="sm">
                    {seatIDs.length} Seats
                  </Badge>
                )}
                {isDirty && (
                  <Badge variant="warning" size="sm">
                    Unsaved Changes
                  </Badge>
                )}
              </div>

              {/* Action Buttons for Layout Tab */}
              {activeTab === 'layout' && (
                <div className="flex items-center gap-2">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={undo}
                    disabled={!canUndo}
                    title="Undo (Ctrl+Z)"
                  >
                    <Undo2 className="w-4 h-4" />
                    Undo
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={redo}
                    disabled={!canRedo}
                    title="Redo (Ctrl+Shift+Z)"
                  >
                    <Redo2 className="w-4 h-4" />
                    Redo
                  </Button>

                  <Button
                    variant={showStudentOverlay ? 'primary' : 'secondary'}
                    size="sm"
                    onClick={() => setShowStudentOverlay((v) => !v)}
                  >
                    <Users className="w-4 h-4" />
                    Student Overlay
                  </Button>

                  <Button
                    variant="primary"
                    size="sm"
                    onClick={handleSaveLayout}
                    loading={isSavingLayout}
                    disabled={!isLayoutDirty}
                  >
                    <Save className="w-4 h-4" />
                    Save Layout
                  </Button>
                </div>
              )}
            </div>

            {/* Tab Contents */}
            <div className="flex-1 overflow-hidden p-4">
              {activeTab === 'details' ? (
                <div className="h-full overflow-y-auto">
                  <DetailsForm
                    roomId={selectedRoomId}
                    meta={metaDraft}
                    onChange={setMetaDraft}
                    onSave={handleSaveMeta}
                    isSaving={isSavingMeta}
                    isDirty={isMetaDirty}
                  />
                </div>
              ) : (
                <div className="h-full flex flex-col overflow-hidden">
                  {showStudentOverlay && (
                    <StudentOverlay
                      roomId={selectedRoomId}
                      layoutSeatSet={seatSet}
                      onRosterLoaded={setStudentRoster}
                      onClose={() => setShowStudentOverlay(false)}
                    />
                  )}

                  <div className="flex-1 flex gap-4 overflow-hidden">
                    {/* Visual Canvas Area */}
                    <div className="flex-1 h-full overflow-hidden">
                      <LayoutCanvas
                        key={selectedRoomId}
                        layout={layout}
                        selection={canvasSelection}
                        onSelect={setCanvasSelection}
                        onMoveBlock={(fromIdx, toIdx) => {
                          const next = { ...layout };
                          const newBlocks = [...next.layout];
                          const [moved] = newBlocks.splice(fromIdx, 1);
                          newBlocks.splice(toIdx, 0, moved);
                          modifyLayout({ ...next, layout: newBlocks });
                          setCanvasSelection({ type: 'block', index: toIdx });
                        }}
                        onMoveItem={(fromCol, fromIdx, toCol, toIdx) => {
                          const sourceCol = layout.layout[fromCol];
                          const targetCol = layout.layout[toCol];
                          if (
                            sourceCol &&
                            sourceCol.type === 'column' &&
                            targetCol &&
                            targetCol.type === 'column'
                          ) {
                            if (fromCol === toCol) {
                              const newItems = [...sourceCol.items];
                              const [moved] = newItems.splice(fromIdx, 1);
                              newItems.splice(toIdx, 0, moved);
                              const newBlocks = [...layout.layout];
                              newBlocks[fromCol] = { ...sourceCol, items: newItems };
                              modifyLayout({ ...layout, layout: newBlocks });
                              setCanvasSelection({ type: 'item', colIndex: toCol, itemIndex: toIdx });
                            } else {
                              const newSourceItems = [...sourceCol.items];
                              const [moved] = newSourceItems.splice(fromIdx, 1);
                              const newTargetItems = [...targetCol.items];
                              newTargetItems.splice(toIdx, 0, moved);
                              const newBlocks = [...layout.layout];
                              newBlocks[fromCol] = { ...sourceCol, items: newSourceItems };
                              newBlocks[toCol] = { ...targetCol, items: newTargetItems };
                              modifyLayout({ ...layout, layout: newBlocks });
                              setCanvasSelection({ type: 'item', colIndex: toCol, itemIndex: toIdx });
                            }
                          }
                        }}
                        studentRoster={studentRoster}
                      />
                    </div>

                    {/* Inspector / Side Panel */}
                    <div className="w-80 h-full overflow-hidden shrink-0">
                      <SidePanel
                        layout={layout}
                        selection={canvasSelection}
                        onChange={modifyLayout}
                        onSelect={setCanvasSelection}
                      />
                    </div>
                  </div>
                </div>
              )}
            </div>
          </>
        ) : (
          <div className="flex items-center justify-center h-full text-slate-400 text-sm">
            Select a room from the left sidebar or create one to begin.
          </div>
        )}
      </div>

      {/* Add Room Dialog */}
      <Dialog
        open={addModalOpen}
        onClose={() => setAddModalOpen(false)}
        title="Add New Room"
        footer={
          <>
            <Button variant="ghost" size="sm" onClick={() => setAddModalOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="primary"
              size="sm"
              onClick={handleAddRoom}
              disabled={!newRoomId.trim()}
              loading={isAddingRoom}
            >
              Create Room
            </Button>
          </>
        }
      >
        <div className="flex flex-col gap-3">
          <Input
            label="Room ID"
            placeholder="e.g. SC.1105"
            value={newRoomId}
            onChange={(e) => setNewRoomId(e.target.value)}
            helperText="Canonical name used in exam schedules and URLs"
          />
          <Input
            label="Layout File Name (optional)"
            placeholder="e.g. SC1105.json"
            value={newRoomLayoutFile}
            onChange={(e) => setNewRoomLayoutFile(e.target.value)}
            helperText="Default: room ID without special characters + .json"
          />
        </div>
      </Dialog>

      {/* Delete Room Confirm Dialog */}
      <ConfirmDialog
        open={roomToDelete !== null}
        onClose={() => setRoomToDelete(null)}
        onConfirm={handleDeleteRoom}
        title="Delete Room"
        message={`Are you sure you want to delete room "${roomToDelete}"? If no other room shares its layout file, the layout file will also be removed.`}
        confirmText="Delete"
        loading={isDeletingRoom}
      />

      {/* Unsaved Changes Guard Dialog */}
      <ConfirmDialog
        open={isPromptOpen}
        onClose={cancelLeave}
        onConfirm={confirmLeave}
        title="Unsaved Changes"
        message="You have unsaved changes that will be lost if you leave this view. Are you sure you want to discard them?"
        confirmText="Discard and Leave"
        cancelText="Stay"
        variant="danger"
      />
    </div>
  );
};
