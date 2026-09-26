import React, { useMemo, useRef, useEffect } from 'react';
import type { RoomLayout } from '../types';

interface CanvasProps {
  layout: RoomLayout;
  selBlock: number | null;
  dragBlock: number | null;
  loadingLayout: boolean;
  onSetLayout: (layout: RoomLayout) => void;
  onSetSelBlock: (idx: number | null) => void;
  onSetDragBlock: (idx: number | null) => void;
  onAddAisle: () => void;
  onAddColumn: () => void;
  onDeleteBlock: () => void;
  moveBlock: (from: number, to: number) => void;
}

export const Canvas: React.FC<CanvasProps> = ({
  layout,
  selBlock,
  dragBlock,
  loadingLayout,
  onSetLayout,
  onSetSelBlock,
  onSetDragBlock,
  onAddAisle,
  onAddColumn,
  onDeleteBlock,
  moveBlock,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);

  const transform = useRef({ x: 0, y: 0, k: 1 });
  const isDragging = useRef(false);
  const lastPos = useRef({ x: 0, y: 0 });
  const lastTouchDistance = useRef<number | null>(null);

  const updateTransform = () => {
    if (contentRef.current) {
      const { x, y, k } = transform.current;
      contentRef.current.style.transform = `translate3d(${x}px, ${y}px, 0) scale(${k})`;
    }
  };

  const centerTarget = (animate = true) => {
    if (!containerRef.current || !contentRef.current) return;

    const containerRect = containerRef.current.getBoundingClientRect();
    const stage = contentRef.current.querySelector('.canvas-stage') as HTMLElement;
    
    let stageOffsetLeft = 160;
    let stageOffsetTop = 160;
    let stageWidth = 600;
    let stageHeight = 400;

    if (stage) {
      stageOffsetLeft = stage.offsetLeft;
      stageOffsetTop = stage.offsetTop;
      stageWidth = stage.offsetWidth;
      stageHeight = stage.offsetHeight;
    }

    const targetX = stageOffsetLeft + stageWidth / 2;
    const targetY = stageOffsetTop + stageHeight / 2;

    const centerX = containerRect.width / 2 - targetX;
    const centerY = containerRect.height / 2 - targetY;

    transform.current = { x: centerX, y: centerY, k: 1 };

    contentRef.current.style.transition = animate
      ? 'transform 0.4s cubic-bezier(0.25, 1, 0.5, 1)'
      : 'none';
    updateTransform();
    if (animate) {
      setTimeout(() => {
        if (contentRef.current) contentRef.current.style.transition = '';
      }, 400);
    }
  };

  // Center target when layout loads or changes
  useEffect(() => {
    setTimeout(() => centerTarget(false), 50);
  }, [layout.frontLabel, layout.backLabel, loadingLayout]);

  // Handle wheel zoom
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const onWheel = (e: WheelEvent) => {
      e.preventDefault();
      e.stopPropagation();
      const ZOOM_SPEED = 0.0015;
      const currentK = transform.current.k;
      let newK = Math.min(
        Math.max(currentK - e.deltaY * ZOOM_SPEED * currentK, 0.25),
        3.0
      );

      const rect = container.getBoundingClientRect();
      const mouseX = e.clientX - rect.left;
      const mouseY = e.clientY - rect.top;
      const scaleRatio = newK / currentK;

      transform.current.x = mouseX - (mouseX - transform.current.x) * scaleRatio;
      transform.current.y = mouseY - (mouseY - transform.current.y) * scaleRatio;
      transform.current.k = newK;

      updateTransform();
    };

    container.addEventListener('wheel', onWheel, { passive: false });
    return () => container.removeEventListener('wheel', onWheel);
  }, []);

  const handleMouseDown = (e: React.MouseEvent) => {
    const target = e.target as HTMLElement;
    // Don't pan if clicking layout nodes or toolbar elements
    if (
      target.closest('.col-block') ||
      target.closest('.aisle-block') ||
      target.closest('.canvas-toolbar') ||
      target.closest('.canvas-zoom-controls') ||
      target.closest('input') ||
      target.closest('button')
    ) {
      return;
    }
    isDragging.current = true;
    lastPos.current = { x: e.clientX, y: e.clientY };
    if (containerRef.current) containerRef.current.style.cursor = 'grabbing';
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    if (!isDragging.current) return;
    e.preventDefault();
    const dx = e.clientX - lastPos.current.x;
    const dy = e.clientY - lastPos.current.y;
    transform.current.x += dx;
    transform.current.y += dy;
    lastPos.current = { x: e.clientX, y: e.clientY };
    updateTransform();
  };

  const handleMouseUp = () => {
    isDragging.current = false;
    if (containerRef.current) containerRef.current.style.cursor = 'grab';
  };

  const getDistance = (t1: React.Touch, t2: React.Touch) => {
    const dx = t1.clientX - t2.clientX;
    const dy = t1.clientY - t2.clientY;
    return Math.sqrt(dx * dx + dy * dy);
  };

  const handleTouchStart = (e: React.TouchEvent) => {
    const target = e.target as HTMLElement;
    if (
      target.closest('.col-block') ||
      target.closest('.aisle-block') ||
      target.closest('.canvas-toolbar') ||
      target.closest('.canvas-zoom-controls') ||
      target.closest('input') ||
      target.closest('button')
    ) {
      return;
    }
    if (e.touches.length === 1) {
      isDragging.current = true;
      lastPos.current = { x: e.touches[0].clientX, y: e.touches[0].clientY };
    } else if (e.touches.length === 2) {
      isDragging.current = false;
      lastTouchDistance.current = getDistance(e.touches[0], e.touches[1]);
    }
  };

  const handleTouchMove = (e: React.TouchEvent) => {
    if (e.touches.length === 1 && isDragging.current) {
      const dx = e.touches[0].clientX - lastPos.current.x;
      const dy = e.touches[0].clientY - lastPos.current.y;
      transform.current.x += dx;
      transform.current.y += dy;
      lastPos.current = { x: e.touches[0].clientX, y: e.touches[0].clientY };
      updateTransform();
    } else if (e.touches.length === 2 && lastTouchDistance.current && containerRef.current) {
      const newDist = getDistance(e.touches[0], e.touches[1]);
      const currentK = transform.current.k;
      let newK = Math.min(Math.max(currentK * (newDist / lastTouchDistance.current), 0.25), 3.0);

      const rect = containerRef.current.getBoundingClientRect();
      const centerX = (e.touches[0].clientX + e.touches[1].clientX) / 2 - rect.left;
      const centerY = (e.touches[0].clientY + e.touches[1].clientY) / 2 - rect.top;
      const ratio = newK / currentK;

      transform.current.x = centerX - (centerX - transform.current.x) * ratio;
      transform.current.y = centerY - (centerY - transform.current.y) * ratio;
      transform.current.k = newK;

      updateTransform();
      lastTouchDistance.current = newDist;
    }
  };

  const handleTouchEnd = () => {
    isDragging.current = false;
    lastTouchDistance.current = null;
  };

  const handleZoom = (factor: number) => {
    if (!containerRef.current) return;
    const currentK = transform.current.k;
    let newK = Math.min(Math.max(currentK + factor, 0.25), 3.0);

    const rect = containerRef.current.getBoundingClientRect();
    const mouseX = rect.width / 2;
    const mouseY = rect.height / 2;
    const scaleRatio = newK / currentK;

    transform.current.x = mouseX - (mouseX - transform.current.x) * scaleRatio;
    transform.current.y = mouseY - (mouseY - transform.current.y) * scaleRatio;
    transform.current.k = newK;

    if (contentRef.current) {
      contentRef.current.style.transition = 'transform 0.2s ease-out';
      updateTransform();
      setTimeout(() => {
        if (contentRef.current) contentRef.current.style.transition = '';
      }, 200);
    }
  };

  // Derived state to calculate seat numbers matching SeatMap.tsx engine
  const computedLayout = useMemo(() => {
    const globalCounters: Record<string, number> = {};
    const computedBlocks = layout.layout.map((block) => {
      if (block.type !== 'column') return block;
      const items = block.items.map((item) => {
        if (item.type !== 'seats') return item;

        const nums: number[] = [];
        if (item.manual) {
          nums.push(...item.manual);
          if (item.inverse) nums.reverse();
        } else {
          const start = item.start ?? (globalCounters[item.char] || 1);
          const count = item.count || 0;
          const step = item.step ?? (item.inverse ? -1 : 1);
          let current = start;
          for (let i = 0; i < count; i++) {
            nums.push(current);
            current += step;
          }
          globalCounters[item.char] = current;
        }
        return { ...item, _calculatedNums: nums };
      });
      return { ...block, items };
    });
    return { ...layout, layout: computedBlocks };
  }, [layout]);

  return (
    <div className="canvas" onClick={() => onSetSelBlock(null)}>
      <div className="canvas-header">
        <span className="label">Design Board</span>
        <span className="stats">
          {computedLayout.layout.filter((x) => x.type === 'column').length} cols ·{' '}
          {computedLayout.layout.filter((x) => x.type === 'aisle').length} aisles
        </span>
      </div>

      <div
        ref={containerRef}
        className="canvas-viewport"
        onMouseDown={handleMouseDown}
        onMouseMove={handleMouseMove}
        onMouseUp={handleMouseUp}
        onMouseLeave={handleMouseUp}
        onTouchStart={handleTouchStart}
        onTouchMove={handleTouchMove}
        onTouchEnd={handleTouchEnd}
      >
        <div ref={contentRef} className="canvas-content" style={{ padding: '160px' }}>
          {loadingLayout ? (
            <div className="canvas-stage" style={{ minWidth: '400px', height: '200px', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
              <div className="empty">
                <p>Loading Layout...</p>
              </div>
            </div>
          ) : (
            <div className="canvas-stage">
              <input
                className="stage-label-input top"
                value={layout.frontLabel || ''}
                placeholder="FRONT LABEL"
                onClick={(e) => e.stopPropagation()}
                onChange={(e) => onSetLayout({ ...layout, frontLabel: e.target.value })}
              />

              <div className="grid-row">
                {computedLayout.layout.map((block, bIdx) => {
                  if (block.type === 'aisle') {
                    return (
                      <div
                        key={`b-${bIdx}`}
                        draggable
                        onDragStart={(e) => {
                          onSetDragBlock(bIdx);
                          e.dataTransfer.effectAllowed = 'move';
                        }}
                        onDragOver={(e) => e.preventDefault()}
                        onDrop={() => {
                          if (dragBlock !== null) {
                            moveBlock(dragBlock, bIdx);
                            onSetDragBlock(null);
                          }
                        }}
                        onClick={(e) => {
                          e.stopPropagation();
                          onSetSelBlock(bIdx);
                        }}
                        style={{ width: `${block.width}px` }}
                        className={`aisle-block ${bIdx === selBlock ? 'selected' : ''} ${
                          bIdx === dragBlock ? 'block-dragging' : ''
                        }`}
                      >
                        <div className="grip">⠿</div>
                        <div className="aisle-line" />
                      </div>
                    );
                  }
                  if (block.type === 'column') {
                    return (
                      <div
                        key={`b-${bIdx}`}
                        draggable
                        onDragStart={(e) => {
                          onSetDragBlock(bIdx);
                          e.dataTransfer.effectAllowed = 'move';
                        }}
                        onDragOver={(e) => e.preventDefault()}
                        onDrop={() => {
                          if (dragBlock !== null) {
                            moveBlock(dragBlock, bIdx);
                            onSetDragBlock(null);
                          }
                        }}
                        onClick={(e) => {
                          e.stopPropagation();
                          onSetSelBlock(bIdx);
                        }}
                        className={`col-block ${bIdx === selBlock ? 'selected' : ''} ${
                          bIdx === dragBlock ? 'block-dragging' : ''
                        }`}
                      >
                        <div className="grip">⠿</div>
                        <input
                          className="col-label-input"
                          value={block.label || ''}
                          placeholder={`C${bIdx + 1}`}
                          onClick={(e) => e.stopPropagation()}
                          onChange={(e) => {
                            const u = { ...layout };
                            const col = u.layout[bIdx];
                            if (col && col.type === 'column') {
                              col.label = e.target.value;
                              onSetLayout(u);
                            }
                          }}
                        />
                        {block.items.map((item, iIdx) => {
                          if (item.type === 'seats') {
                            const nums = item._calculatedNums || [];
                            return nums.map((n) => (
                              <div
                                key={`${item.char}${n}`}
                                className={`seat ${item.inverse ? 'inverse' : ''}`}
                              >
                                {item.char}{n}
                              </div>
                            ));
                          }
                          if (item.type === 'obstruction') {
                            const h = item.height || (item.count ? item.count * 40 - 6 : 34);
                            const w = item.width ? `${item.width}px` : '42px';
                            if (item.transparent)
                              return (
                                <div
                                  key={`o-${bIdx}-${iIdx}`}
                                  style={{ height: `${h}px`, width: w }}
                                />
                              );
                            return (
                              <div
                                key={`o-${bIdx}-${iIdx}`}
                                style={{ height: `${h}px`, width: w }}
                                className="obs-box"
                              >
                                {item.label || ''}
                              </div>
                            );
                          }
                          if (item.type === 'gap') {
                            const h = item.count ? item.count * 40 - 6 : 34;
                            return (
                              <div
                                key={`g-${bIdx}-${iIdx}`}
                                style={{ height: `${h}px`, flexShrink: 0 }}
                              />
                            );
                          }
                          return null;
                        })}
                      </div>
                    );
                  }
                  return null;
                })}
              </div>

              <input
                className="stage-label-input bottom"
                value={layout.backLabel || ''}
                placeholder="BACK LABEL"
                onClick={(e) => e.stopPropagation()}
                onChange={(e) => onSetLayout({ ...layout, backLabel: e.target.value })}
              />
            </div>
          )}
        </div>
      </div>

      <div className="canvas-zoom-controls">
        <button className="zoom-btn" onClick={() => handleZoom(0.15)} title="Zoom In">
          +
        </button>
        <button
          className="zoom-btn reset"
          onClick={() => centerTarget(true)}
          title="Reset Pan/Zoom"
        >
          ⌖
        </button>
        <button className="zoom-btn" onClick={() => handleZoom(-0.15)} title="Zoom Out">
          −
        </button>
      </div>

      <div className="canvas-toolbar">
        <button className="btn btn-ghost btn-xs" onClick={onAddAisle}>
          + Aisle
        </button>
        <button className="btn btn-ghost btn-xs" onClick={onAddColumn}>
          + Column
        </button>
        <div className="toolbar-divider" />
        <button
          className="btn btn-danger btn-xs"
          disabled={selBlock === null}
          onClick={onDeleteBlock}
        >
          Delete
        </button>
      </div>
    </div>
  );
};
