import React, { useMemo } from 'react';
import type { RoomLayout, LayoutItem, AisleBlock, ColumnBlock, ItemType } from '../types';

interface InspectorProps {
  layout: RoomLayout;
  selBlock: number | null;
  expandItem: number | null;
  dragItem: number | null;
  onSetLayout: (layout: RoomLayout) => void;
  onSetSelBlock: (idx: number | null) => void;
  onSetExpandItem: (idx: number | null) => void;
  onSetDragItem: (idx: number | null) => void;
  moveItem: (bIdx: number, from: number, to: number) => void;
  addItem: (bIdx: number, type: ItemType) => void;
  updateItem: (bIdx: number, iIdx: number, patch: Partial<LayoutItem>) => void;
  deleteItem: (bIdx: number, iIdx: number) => void;
}

export const Inspector: React.FC<InspectorProps> = ({
  layout,
  selBlock,
  expandItem,
  dragItem,
  onSetLayout,
  onSetSelBlock,
  onSetExpandItem,
  onSetDragItem,
  moveItem,
  addItem,
  updateItem,
  deleteItem,
}) => {
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

  if (selBlock === null) {
    return (
      <aside className="inspector">
        <div className="inspector-head">
          <h3>Layout Properties</h3>
        </div>
        <div className="inspector-body">
          <div className="field">
            <span className="field-label">Front Label</span>
            <input
              className="field-input"
              placeholder="e.g. Front, Screen"
              value={layout.frontLabel || ''}
              onChange={(e) => onSetLayout({ ...layout, frontLabel: e.target.value })}
            />
          </div>
          <div className="field">
            <span className="field-label">Back Label</span>
            <input
              className="field-input"
              placeholder="e.g. Back, Entrance"
              value={layout.backLabel || ''}
              onChange={(e) => onSetLayout({ ...layout, backLabel: e.target.value })}
            />
          </div>
          <div className="empty" style={{ border: 'none', height: 'auto', padding: '1.5rem 0' }}>
            <span className="empty-icon">💡</span>
            <p>Click a column or aisle in the design board to configure items.</p>
          </div>
        </div>
      </aside>
    );
  }

  const block = computedLayout.layout[selBlock];
  if (!block) return null;

  return (
    <aside className="inspector">
      <div className="inspector-head">
        <h3>
          Properties{' '}
          <span
            className={`badge ${
              block.type === 'column' ? 'badge-blue' : 'badge-amber'
            }`}
          >
            {block.type}
          </span>
        </h3>
        <button className="icon-btn" onClick={() => onSetSelBlock(null)}>
          &times;
        </button>
      </div>
      <div className="inspector-body">
        {block.type === 'aisle' && (
          <div className="field">
            <span className="field-label">Width (px)</span>
            <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center' }}>
              <input
                type="range"
                min="5"
                max="120"
                step="5"
                style={{ flex: 1 }}
                value={block.width}
                onChange={(e) => {
                  const u = { ...layout };
                  (u.layout[selBlock] as AisleBlock).width =
                    parseInt(e.target.value) || 20;
                  onSetLayout(u);
                }}
              />
              <input
                type="number"
                className="field-input"
                style={{ width: '60px', padding: '0.3rem' }}
                value={block.width}
                onChange={(e) => {
                  const u = { ...layout };
                  (u.layout[selBlock] as AisleBlock).width =
                    parseInt(e.target.value) || 20;
                  onSetLayout(u);
                }}
              />
            </div>
          </div>
        )}

        {block.type === 'column' && (
          <>
            <div className="field">
              <span className="field-label">Label</span>
              <input
                className="field-input"
                placeholder="e.g. C1"
                value={block.label || ''}
                onChange={(e) => {
                  const u = { ...layout };
                  (u.layout[selBlock] as ColumnBlock).label = e.target.value;
                  onSetLayout(u);
                }}
              />
            </div>

            <div className="field">
              <span className="field-label">Items</span>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '0.35rem' }}>
                {block.items.map((item, iIdx) => {
                  const open = iIdx === expandItem;
                  return (
                    <div
                      key={iIdx}
                      className={`item-card ${iIdx === dragItem ? 'dragging' : ''}`}
                      draggable
                      onDragStart={(e) => {
                        onSetDragItem(iIdx);
                        e.stopPropagation();
                      }}
                      onDragOver={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                      }}
                      onDrop={(e) => {
                        e.stopPropagation();
                        if (dragItem !== null) {
                          moveItem(selBlock, dragItem, iIdx);
                          onSetDragItem(null);
                        }
                      }}
                    >
                      <div
                        className="item-head"
                        onClick={() => onSetExpandItem(open ? null : iIdx)}
                      >
                        <div className="item-title">
                          <span className="grip-icon">⠿</span>
                          <span className="item-badge">{item.type}</span>
                          <span>
                            {item.type === 'seats'
                              ? `${item.char} (${
                                  item.manual
                                    ? `man:${item.manual.length}`
                                    : item._calculatedNums &&
                                      item._calculatedNums.length > 0
                                    ? `${item._calculatedNums[0]}–${
                                        item._calculatedNums[
                                          item._calculatedNums.length - 1
                                        ]
                                      }`
                                    : ''
                                })`
                              : item.type === 'obstruction'
                              ? item.label || 'Pillar'
                              : `Gap ×${item.count}`}
                          </span>
                        </div>
                        <div className="item-actions">
                          <span style={{ fontSize: '0.55rem', color: '#94a3b8' }}>
                            {open ? '▼' : '▶'}
                          </span>
                          <button
                            className="icon-btn danger"
                            onClick={(e) => {
                              e.stopPropagation();
                              deleteItem(selBlock, iIdx);
                            }}
                          >
                            &times;
                          </button>
                        </div>
                      </div>

                      {open && (
                        <div className="item-body">
                          {item.type === 'seats' && (
                            <>
                              <div className="field">
                                <span className="field-label">Char</span>
                                <input
                                  className="field-input mono"
                                  maxLength={2}
                                  value={item.char || 'A'}
                                  onChange={(e) =>
                                    updateItem(selBlock, iIdx, {
                                      char: e.target.value.toUpperCase().trim(),
                                    })
                                  }
                                />
                              </div>

                              <div className="check-row">
                                <input
                                  type="checkbox"
                                  id={`man-${iIdx}`}
                                  checked={!!item.manual}
                                  onChange={(e) => {
                                    if (e.target.checked)
                                      updateItem(selBlock, iIdx, {
                                        manual: [1],
                                        start: undefined,
                                        count: undefined,
                                      });
                                    else
                                      updateItem(selBlock, iIdx, {
                                        start: 1,
                                        count: 10,
                                        manual: undefined,
                                      });
                                  }}
                                />
                                <label htmlFor={`man-${iIdx}`}>Manual list</label>
                              </div>

                              {item.manual ? (
                                <div className="field">
                                  <span className="field-label">Seats (comma-sep)</span>
                                  <input
                                    className="field-input mono"
                                    value={item.manual.join(', ')}
                                    onChange={(e) => {
                                      const nums = e.target.value
                                        .split(',')
                                        .map((v) => parseInt(v.trim()))
                                        .filter((v) => !isNaN(v));
                                      updateItem(selBlock, iIdx, { manual: nums });
                                    }}
                                  />
                                </div>
                              ) : (
                                <>
                                  <div
                                    className="inline-fields"
                                    style={{ gridTemplateColumns: 'repeat(3, 1fr)' }}
                                  >
                                    <div className="field">
                                      <span className="field-label">Start</span>
                                      <input
                                        type="number"
                                        className="field-input"
                                        value={item.start ?? ''}
                                        placeholder="auto"
                                        onChange={(e) => {
                                          const val =
                                            e.target.value === ''
                                              ? undefined
                                              : parseInt(e.target.value);
                                          updateItem(selBlock, iIdx, {
                                            start: isNaN(val as any) ? undefined : val,
                                          });
                                        }}
                                      />
                                    </div>
                                    <div className="field">
                                      <span className="field-label">Count</span>
                                      <input
                                        type="number"
                                        className="field-input"
                                        value={item.count ?? 10}
                                        onChange={(e) =>
                                          updateItem(selBlock, iIdx, {
                                            count: parseInt(e.target.value) || 1,
                                          })
                                        }
                                      />
                                    </div>
                                    <div className="field">
                                      <span className="field-label">Step</span>
                                      <input
                                        type="number"
                                        className="field-input"
                                        value={item.step ?? ''}
                                        placeholder="auto"
                                        onChange={(e) => {
                                          const val =
                                            e.target.value === ''
                                              ? undefined
                                              : parseInt(e.target.value);
                                          updateItem(selBlock, iIdx, {
                                            step: isNaN(val as any) ? undefined : val,
                                          });
                                        }}
                                      />
                                    </div>
                                  </div>
                                </>
                              )}

                              <div className="check-row">
                                <input
                                  type="checkbox"
                                  id={`inv-${iIdx}`}
                                  checked={!!item.inverse}
                                  onChange={(e) =>
                                    updateItem(selBlock, iIdx, {
                                      inverse: e.target.checked,
                                    })
                                  }
                                />
                                <label htmlFor={`inv-${iIdx}`}>Inverse order</label>
                              </div>
                            </>
                          )}

                          {item.type === 'obstruction' && (
                            <>
                              <div className="field">
                                <span className="field-label">Label</span>
                                <input
                                  className="field-input"
                                  value={item.label || ''}
                                  placeholder="Pillar, Desk..."
                                  onChange={(e) =>
                                    updateItem(selBlock, iIdx, {
                                      label: e.target.value,
                                    })
                                  }
                                />
                              </div>
                              <div className="field">
                                <span className="field-label">Rows Span (count)</span>
                                <input
                                  type="number"
                                  className="field-input"
                                  value={item.count ?? 1}
                                  onChange={(e) =>
                                    updateItem(selBlock, iIdx, {
                                      count: parseInt(e.target.value) || 1,
                                    })
                                  }
                                />
                              </div>
                              <div className="check-row">
                                <input
                                  type="checkbox"
                                  id={`trans-${iIdx}`}
                                  checked={!!item.transparent}
                                  onChange={(e) =>
                                    updateItem(selBlock, iIdx, {
                                      transparent: e.target.checked,
                                    })
                                  }
                                />
                                <label htmlFor={`trans-${iIdx}`}>Transparent</label>
                              </div>
                            </>
                          )}

                          {item.type === 'gap' && (
                            <div className="field">
                              <span className="field-label">Gap multiplier</span>
                              <input
                                type="number"
                                className="field-input"
                                value={item.count ?? 1}
                                onChange={(e) =>
                                  updateItem(selBlock, iIdx, {
                                    count: parseInt(e.target.value) || 1,
                                  })
                                }
                              />
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            </div>

            <div className="add-items-row">
              <button
                className="btn btn-ghost btn-xs"
                style={{ flex: 1 }}
                onClick={() => addItem(selBlock, 'seats')}
              >
                + Seats
              </button>
              <button
                className="btn btn-ghost btn-xs"
                style={{ flex: 1 }}
                onClick={() => addItem(selBlock, 'obstruction')}
              >
                + Pillar
              </button>
              <button
                className="btn btn-ghost btn-xs"
                style={{ flex: 1 }}
                onClick={() => addItem(selBlock, 'gap')}
              >
                + Gap
              </button>
            </div>
          </>
        )}
      </div>
    </aside>
  );
};
