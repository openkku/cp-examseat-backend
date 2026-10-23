import React, { useState, useRef, useCallback, useEffect, useLayoutEffect, useMemo } from 'react';
import { ZoomIn, ZoomOut, Maximize2, RotateCcw, GripVertical } from 'lucide-react';
import type {
  AisleBlock,
  ColumnBlock,
  LayoutBlock,
  LayoutItem,
  PreviewSeat,
  RoomLayout,
  SeatItem,
} from '../../types';
import { getItemSeatNumbers } from './model';
import { SEAT_PALETTE, subjectColors } from './seatPalette';

export type Selection =
  | { type: 'block'; index: number }
  | { type: 'item'; colIndex: number; itemIndex: number }
  | null;

export interface LayoutCanvasProps {
  layout: RoomLayout;
  selection: Selection;
  onSelect: (selection: Selection) => void;
  onMoveBlock: (fromIndex: number, toIndex: number) => void;
  onMoveItem: (fromCol: number, fromIdx: number, toCol: number, toIdx: number) => void;
  studentRoster?: Record<string, PreviewSeat>;
}

const NO_STUDENTS: Record<string, PreviewSeat> = {};
const MIN_ZOOM = 0.2;
const MAX_ZOOM = 3;

function clampZoom(z: number): number {
  return Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, Number(z.toFixed(3))));
}

export const LayoutCanvas: React.FC<LayoutCanvasProps> = ({
  layout,
  selection,
  onSelect,
  onMoveBlock,
  onMoveItem,
  studentRoster = NO_STUDENTS,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const [view, setView] = useState<{ zoom: number; pan: { x: number; y: number } }>({
    zoom: 1,
    pan: { x: 0, y: 0 },
  });
  // Animate only button-driven changes; wheel and drag must follow the pointer.
  const [animate, setAnimate] = useState(false);
  const [isPanning, setIsPanning] = useState(false);
  const panStartRef = useRef<{ x: number; y: number }>({ x: 0, y: 0 });
  const { zoom, pan } = view;

  // Drag-and-drop state
  const [draggingBlockIndex, setDraggingBlockIndex] = useState<number | null>(null);
  const [draggingItem, setDraggingItem] = useState<{ colIndex: number; itemIndex: number } | null>(null);

  // zoomAt scales by factor while keeping the point under (clientX, clientY) still.
  // The content is centred in the viewport, so offsets are taken from its centre.
  const zoomAt = useCallback((factor: number, clientX?: number, clientY?: number) => {
    const rect = containerRef.current?.getBoundingClientRect();
    const mx = rect && clientX !== undefined ? clientX - (rect.left + rect.width / 2) : 0;
    const my = rect && clientY !== undefined ? clientY - (rect.top + rect.height / 2) : 0;
    setView((v) => {
      const next = clampZoom(v.zoom * factor);
      const k = next / v.zoom;
      return {
        zoom: next,
        pan: { x: mx - k * (mx - v.pan.x), y: my - k * (my - v.pan.y) },
      };
    });
  }, []);

  // Wheel zooms toward the cursor (a trackpad pinch arrives as a Ctrl + wheel).
  // React's onWheel is passive and can't preventDefault, so attach it directly.
  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const onWheel = (e: WheelEvent) => {
      e.preventDefault();
      setAnimate(false);
      const unit = e.deltaMode === 1 ? 16 : e.deltaMode === 2 ? el.clientHeight : 1;
      const dy = (e.deltaY || e.deltaX) * unit;
      zoomAt(Math.exp(-dy * 0.002), e.clientX, e.clientY);
    };
    el.addEventListener('wheel', onWheel, { passive: false });
    return () => el.removeEventListener('wheel', onWheel);
  }, [zoomAt]);

  // Pan handlers
  const handleMouseDown = (e: React.MouseEvent) => {
    // Only pan if clicking canvas background
    if ((e.target as HTMLElement).closest('.interactive-seat-element')) {
      return;
    }
    setAnimate(false);
    setIsPanning(true);
    panStartRef.current = { x: e.clientX - pan.x, y: e.clientY - pan.y };
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    if (!isPanning) return;
    setView((v) => ({
      ...v,
      pan: { x: e.clientX - panStartRef.current.x, y: e.clientY - panStartRef.current.y },
    }));
  };

  const handleMouseUp = () => {
    setIsPanning(false);
  };

  const zoomButton = (factor: number) => {
    setAnimate(true);
    zoomAt(factor);
  };

  const resetView = () => {
    setAnimate(true);
    setView({ zoom: 1, pan: { x: 0, y: 0 } });
  };

  // fitView scales the whole layout to fit the viewport, never above 100%.
  const fitView = useCallback(() => {
    const viewport = containerRef.current;
    const content = contentRef.current;
    if (!viewport || !content || !content.offsetWidth || !content.offsetHeight) return;
    const pad = 64; // p-8 on both sides
    const fit = Math.min(
      (viewport.clientWidth - pad) / content.offsetWidth,
      (viewport.clientHeight - pad) / content.offsetHeight,
      1
    );
    setView({ zoom: clampZoom(fit), pan: { x: 0, y: 0 } });
  }, []);

  // Start every room fitted, so wide rooms aren't cut off.
  useLayoutEffect(() => {
    fitView();
  }, [fitView]);

  // Subject colours, assigned as on the public seat map
  const colors = useMemo(() => subjectColors(studentRoster), [studentRoster]);

  // Compute counters for drawing
  const counters: Record<string, number> = {};

  return (
    <div className="relative w-full h-full flex flex-col bg-slate-50 dark:bg-slate-950 border border-slate-200/50 dark:border-slate-800/80 rounded-2xl overflow-hidden select-none transition-colors">
      {/* Zoom / View controls */}
      <div className="absolute top-3 right-3 z-30 flex items-center gap-0.5 p-1 bg-white/90 dark:bg-slate-900/90 backdrop-blur border border-slate-200/50 dark:border-slate-800 rounded-full shadow-lg dark:shadow-none">
        <button
          onClick={() => zoomButton(1.2)}
          className="w-8 h-8 flex items-center justify-center text-slate-600 dark:text-slate-300 rounded-full hover:bg-faculty-light/70 dark:hover:bg-blue-950/30 hover:text-faculty dark:hover:text-blue-200 transition-all active:scale-95"
          title="Zoom In"
        >
          <ZoomIn className="w-4 h-4" />
        </button>
        <span className="text-xs font-bold font-mono px-1 text-slate-500 w-12 text-center">
          {Math.round(zoom * 100)}%
        </span>
        <button
          onClick={() => zoomButton(1 / 1.2)}
          className="w-8 h-8 flex items-center justify-center text-slate-600 dark:text-slate-300 rounded-full hover:bg-faculty-light/70 dark:hover:bg-blue-950/30 hover:text-faculty dark:hover:text-blue-200 transition-all active:scale-95"
          title="Zoom Out"
        >
          <ZoomOut className="w-4 h-4" />
        </button>
        <button
          onClick={resetView}
          className="w-8 h-8 flex items-center justify-center text-slate-600 dark:text-slate-300 rounded-full hover:bg-faculty-light/70 dark:hover:bg-blue-950/30 hover:text-faculty dark:hover:text-blue-200 transition-all active:scale-95"
          title="Reset View (100%)"
        >
          <RotateCcw className="w-4 h-4" />
        </button>
        <button
          onClick={() => {
            setAnimate(true);
            fitView();
          }}
          className="w-8 h-8 flex items-center justify-center text-white rounded-full bg-faculty hover:bg-faculty-dark shadow-sm shadow-faculty/20 dark:shadow-none transition-all active:scale-95"
          title="Fit View"
        >
          <Maximize2 className="w-4 h-4" />
        </button>
      </div>

      <div className="absolute bottom-3 left-3 z-30 text-[11px] text-slate-400 dark:text-slate-500 pointer-events-none">
        Drag to move · Scroll to zoom
      </div>

      {/* Canvas Viewport */}
      <div
        ref={containerRef}
        onMouseDown={handleMouseDown}
        onMouseMove={handleMouseMove}
        onMouseUp={handleMouseUp}
        onMouseLeave={handleMouseUp}
        onClick={() => onSelect(null)}
        className="flex-1 w-full h-full overflow-hidden relative cursor-grab active:cursor-grabbing flex items-center justify-center p-8"
      >
        <div
          style={{
            transform: `translate(${pan.x}px, ${pan.y}px) scale(${zoom})`,
            transformOrigin: 'center center',
            transition: animate && !isPanning ? 'transform 0.15s ease-out' : 'none',
          }}
          ref={contentRef}
          className="relative inline-flex flex-col items-center p-8"
          onClick={(e) => e.stopPropagation()}
        >
          {/* Front Label */}
          <div className="w-full text-center text-xs font-bold text-slate-300 dark:text-slate-700 uppercase tracking-[0.5em] mb-4">
            {layout.frontLabel || 'Front'}
          </div>

          {/* Blocks Layout Row */}
          <div className="flex items-start gap-1.5">
            {layout.layout.map((block: LayoutBlock, colIdx: number) => {
              if (block.type === 'aisle') {
                const aisle = block as AisleBlock;
                const isSelected =
                  selection?.type === 'block' && selection.index === colIdx;

                return (
                  <div
                    key={`aisle-${colIdx}`}
                    draggable
                    onDragStart={(e) => {
                      e.stopPropagation();
                      setDraggingBlockIndex(colIdx);
                    }}
                    onDragOver={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                    }}
                    onDrop={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      if (draggingBlockIndex !== null && draggingBlockIndex !== colIdx) {
                        onMoveBlock(draggingBlockIndex, colIdx);
                        setDraggingBlockIndex(null);
                      }
                    }}
                    onClick={(e) => {
                      e.stopPropagation();
                      onSelect({ type: 'block', index: colIdx });
                    }}
                    style={{ width: `${Math.max(16, aisle.width || 20)}px` }}
                    className={`interactive-seat-element shrink-0 self-stretch min-h-[140px] rounded border border-dashed transition-all flex flex-col items-center justify-center cursor-pointer ${
                      isSelected
                        ? 'border-faculty bg-faculty-light/50 dark:bg-blue-950/40 ring-2 ring-faculty'
                        : 'border-slate-300 dark:border-slate-700 hover:border-slate-400 dark:hover:border-slate-600 bg-slate-50/30 dark:bg-slate-900/30'
                    }`}
                    title={`Aisle (width: ${aisle.width || 20}px)`}
                  >
                    <span className="text-[10px] font-mono text-slate-400 rotate-90 whitespace-nowrap">
                      {aisle.width || 20}px
                    </span>
                  </div>
                );
              }

              // Column block
              const col = block as ColumnBlock;
              const isColSelected =
                selection?.type === 'block' && selection.index === colIdx;

              return (
                <div
                  key={`col-${colIdx}`}
                  onDragOver={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                  }}
                  onDrop={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    if (draggingBlockIndex !== null && draggingBlockIndex !== colIdx) {
                      onMoveBlock(draggingBlockIndex, colIdx);
                      setDraggingBlockIndex(null);
                    } else if (draggingItem !== null) {
                      onMoveItem(draggingItem.colIndex, draggingItem.itemIndex, colIdx, col.items.length);
                      setDraggingItem(null);
                    }
                  }}
                  className={`interactive-seat-element flex flex-col gap-1.5 shrink-0 p-1.5 rounded-xl border transition-all ${
                    isColSelected
                      ? 'border-faculty bg-faculty-light/20 dark:bg-blue-950/20 ring-2 ring-faculty/50'
                      : 'border-transparent hover:border-slate-200 dark:hover:border-slate-800'
                  }`}
                >
                  {/* Column Header */}
                  <div
                    draggable
                    onDragStart={(e) => {
                      e.stopPropagation();
                      setDraggingBlockIndex(colIdx);
                    }}
                    onClick={(e) => {
                      e.stopPropagation();
                      onSelect({ type: 'block', index: colIdx });
                    }}
                    className="flex items-center justify-center gap-1 text-xs font-bold text-slate-400 dark:text-slate-655 h-6 px-1 rounded-lg cursor-grab active:cursor-grabbing hover:bg-slate-200/50 dark:hover:bg-slate-800/50"
                  >
                    <GripVertical className="w-3 h-3 text-slate-400" />
                    <span>{col.label || `Col ${colIdx + 1}`}</span>
                  </div>

                  {/* Items list in Column */}
                  <div className="flex flex-col gap-1.5 min-h-[60px]">
                    {col.items.map((item: LayoutItem, itemIdx: number) => {
                      const isItemSelected =
                        selection?.type === 'item' &&
                        selection.colIndex === colIdx &&
                        selection.itemIndex === itemIdx;

                      if (item.type === 'seats') {
                        const seatItem = item as SeatItem;
                        const seatNumbers = getItemSeatNumbers(seatItem, counters);

                        return (
                          <div
                            key={`item-${colIdx}-${itemIdx}`}
                            draggable
                            onDragStart={(e) => {
                              e.stopPropagation();
                              setDraggingItem({ colIndex: colIdx, itemIndex: itemIdx });
                            }}
                            onDragOver={(e) => {
                              e.preventDefault();
                              e.stopPropagation();
                            }}
                            onDrop={(e) => {
                              e.preventDefault();
                              e.stopPropagation();
                              if (draggingItem !== null) {
                                onMoveItem(
                                  draggingItem.colIndex,
                                  draggingItem.itemIndex,
                                  colIdx,
                                  itemIdx
                                );
                                setDraggingItem(null);
                              }
                            }}
                            onClick={(e) => {
                              e.stopPropagation();
                              onSelect({ type: 'item', colIndex: colIdx, itemIndex: itemIdx });
                            }}
                            className={`interactive-seat-element flex flex-col gap-1.5 p-1 rounded-lg border transition-all cursor-pointer ${
                              isItemSelected
                                ? 'border-faculty bg-faculty-light/40 dark:bg-blue-950/40 ring-2 ring-faculty'
                                : 'border-transparent hover:border-slate-300 dark:hover:border-slate-700'
                            }`}
                          >
                            {seatNumbers.map((num) => {
                              const seatId = `${seatItem.char}${num}`;
                              const student = studentRoster[seatId];
                              const isOccupied = !!student;

                              return (
                                <div
                                  key={seatId}
                                  title={
                                    student
                                      ? `${seatId}: ${student.student_id}\n${student.subject} Sec ${student.section}`
                                      : seatId
                                  }
                                  style={{ width: '42px', height: '34px' }}
                                  className={`flex flex-col items-center justify-center rounded border font-bold shrink-0 transition-all duration-200 ${
                                    student
                                      ? (() => {
                                          const c = colors[student.subject] ?? SEAT_PALETTE[0];
                                          return `${c.bg} ${c.text} ${c.border}`;
                                        })()
                                      : 'bg-white dark:bg-slate-900 text-slate-400 dark:text-slate-500 border-slate-200 dark:border-slate-800/80'
                                  }`}
                                >
                                  <span className="leading-none text-[10px] font-bold truncate max-w-[38px]">
                                    {seatId}
                                  </span>
                                  {isOccupied && (
                                    <span className="text-[9px] leading-none opacity-70 truncate max-w-[38px] mt-0.5">
                                      {student.student_id.slice(-4)}
                                    </span>
                                  )}
                                </div>
                              );
                            })}
                          </div>
                        );
                      }

                      if (item.type === 'obstruction') {
                        const h = item.height || (item.count ? item.count * 40 - 6 : 34);
                        const w = item.width ? `${item.width}px` : '42px';

                        if (item.transparent) {
                          return (
                            <div
                              key={`item-${colIdx}-${itemIdx}`}
                              draggable
                              onDragStart={(e) => {
                                e.stopPropagation();
                                setDraggingItem({ colIndex: colIdx, itemIndex: itemIdx });
                              }}
                              onClick={(e) => {
                                e.stopPropagation();
                                onSelect({ type: 'item', colIndex: colIdx, itemIndex: itemIdx });
                              }}
                              style={{ height: `${h}px`, width: w }}
                              className={`interactive-seat-element rounded border border-dashed transition-all cursor-pointer ${
                                isItemSelected
                                  ? 'border-faculty bg-faculty-light/50 dark:bg-blue-950/40 ring-2 ring-faculty'
                                  : 'border-slate-300 dark:border-slate-700 opacity-40'
                              }`}
                              title="Transparent Obstruction"
                            />
                          );
                        }

                        return (
                          <div
                            key={`item-${colIdx}-${itemIdx}`}
                            draggable
                            onDragStart={(e) => {
                              e.stopPropagation();
                              setDraggingItem({ colIndex: colIdx, itemIndex: itemIdx });
                            }}
                            onClick={(e) => {
                              e.stopPropagation();
                              onSelect({ type: 'item', colIndex: colIdx, itemIndex: itemIdx });
                            }}
                            style={{ height: `${h}px`, width: w }}
                            className={`interactive-seat-element rounded border flex items-center justify-center text-xs font-bold pattern-stripes writing-vertical shrink-0 transition-all cursor-pointer ${
                              isItemSelected
                                ? 'border-faculty bg-faculty-light/60 dark:bg-blue-950/60 ring-2 ring-faculty text-faculty dark:text-blue-300'
                                : 'border-slate-200 dark:border-slate-800/80 bg-slate-100 dark:bg-slate-900/60 text-slate-400 dark:text-slate-500 hover:border-slate-300 dark:hover:border-slate-700'
                            }`}
                            title={`Obstruction: ${item.label || 'Pillar'}`}
                          >
                            <span className="uppercase tracking-widest leading-none truncate">
                              {item.label}
                            </span>
                          </div>
                        );
                      }

                      if (item.type === 'gap') {
                        const h = item.count ? item.count * 40 - 6 : 34;
                        return (
                          <div
                            key={`item-${colIdx}-${itemIdx}`}
                            draggable
                            onDragStart={(e) => {
                              e.stopPropagation();
                              setDraggingItem({ colIndex: colIdx, itemIndex: itemIdx });
                            }}
                            onClick={(e) => {
                              e.stopPropagation();
                              onSelect({ type: 'item', colIndex: colIdx, itemIndex: itemIdx });
                            }}
                            style={{ height: `${h}px`, width: '42px' }}
                            className={`interactive-seat-element rounded border border-dashed transition-all cursor-pointer flex items-center justify-center ${
                              isItemSelected
                                ? 'border-faculty bg-faculty-light/50 dark:bg-blue-950/40 ring-2 ring-faculty'
                                : 'border-slate-200 dark:border-slate-800 opacity-30 hover:opacity-75'
                            }`}
                            title={`Gap (count: ${item.count})`}
                          >
                            <span className="text-[9px] text-slate-400">gap</span>
                          </div>
                        );
                      }

                      return null;
                    })}
                  </div>
                </div>
              );
            })}
          </div>

          {/* Back Label */}
          <div className="w-full text-center text-xs font-bold text-slate-300 dark:text-slate-700 uppercase tracking-[0.5em] mt-4">
            {layout.backLabel || 'Back'}
          </div>
        </div>
      </div>
    </div>
  );
};
