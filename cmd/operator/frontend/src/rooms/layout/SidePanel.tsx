import React from 'react';
import {
  ArrowLeft,
  ArrowRight,
  ArrowUp,
  ArrowDown,
  Trash2,
  Columns,
  SplitSquareVertical,
  Armchair,
  ShieldAlert,
  Space,
} from 'lucide-react';
import type {
  AisleBlock,
  ColumnBlock,
  GapItem,
  ObstructionItem,
  RoomLayout,
  SeatItem,
} from '../../types';
import { Button } from '../../ui/Button';
import { Input } from '../../ui/Input';
import type { Selection } from './LayoutCanvas';
import {
  addColumn,
  addAisle,
  addItem,
  moveBlock,
  moveItem,
  removeBlock,
  removeItem,
  updateBlock,
  updateFrontBackLabels,
  updateItem,
} from './model';

export interface SidePanelProps {
  layout: RoomLayout;
  selection: Selection;
  onChange: (layout: RoomLayout) => void;
  onSelect: (selection: Selection) => void;
}

export const SidePanel: React.FC<SidePanelProps> = ({
  layout,
  selection,
  onChange,
  onSelect,
}) => {
  // 1. If an item is selected
  if (selection?.type === 'item') {
    const { colIndex, itemIndex } = selection;
    const block = layout.layout[colIndex];
    if (block && block.type === 'column') {
      const col = block as ColumnBlock;
      const item = col.items[itemIndex];

      if (item) {
        return (
          <div className="flex flex-col gap-4 p-4 bg-white dark:bg-slate-900 border border-slate-200/80 dark:border-slate-800/90 rounded-2xl shadow-sm shadow-teal-950/[0.025] dark:shadow-none h-full overflow-y-auto">
            <div className="flex items-center justify-between border-b border-slate-200 dark:border-slate-800 pb-3">
              <div>
                <span className="text-[10px] font-bold uppercase tracking-wider text-faculty dark:text-blue-400">
                  {item.type} item
                </span>
                <h3 className="text-sm font-bold text-slate-800 dark:text-slate-100">
                  Item #{itemIndex + 1} in {col.label || `Col ${colIndex + 1}`}
                </h3>
              </div>
              <div className="flex items-center gap-1">
                <Button
                  variant="ghost"
                  size="sm"
                  disabled={itemIndex === 0}
                  onClick={() => {
                    const next = moveItem(layout, colIndex, itemIndex, colIndex, itemIndex - 1);
                    onChange(next);
                    onSelect({ type: 'item', colIndex, itemIndex: itemIndex - 1 });
                  }}
                  title="Move Up"
                >
                  <ArrowUp className="w-3.5 h-3.5" />
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  disabled={itemIndex === col.items.length - 1}
                  onClick={() => {
                    const next = moveItem(layout, colIndex, itemIndex, colIndex, itemIndex + 1);
                    onChange(next);
                    onSelect({ type: 'item', colIndex, itemIndex: itemIndex + 1 });
                  }}
                  title="Move Down"
                >
                  <ArrowDown className="w-3.5 h-3.5" />
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    const next = removeItem(layout, colIndex, itemIndex);
                    onChange(next);
                    onSelect(null);
                  }}
                  className="text-rose-600 hover:text-rose-700 hover:bg-rose-50 dark:hover:bg-rose-950/40"
                  title="Delete Item"
                >
                  <Trash2 className="w-3.5 h-3.5" />
                </Button>
              </div>
            </div>

            {/* Seat Item Editor */}
            {item.type === 'seats' && (
              <div className="flex flex-col gap-3">
                <Input
                  label="Row Letter (char)"
                  value={item.char || ''}
                  maxLength={4}
                  onChange={(e) =>
                    onChange(updateItem(layout, colIndex, itemIndex, { char: e.target.value.toUpperCase() }))
                  }
                  helperText="Prefix character, e.g. 'A' or 'B'"
                />
                <Input
                  label="Number of Seats (count)"
                  type="number"
                  min={1}
                  max={100}
                  value={item.count || 1}
                  onChange={(e) =>
                    onChange(updateItem(layout, colIndex, itemIndex, { count: Math.max(1, parseInt(e.target.value, 10) || 1) }))
                  }
                />
                <div className="grid grid-cols-2 gap-2">
                  <Input
                    label="Start (optional)"
                    type="number"
                    value={item.start !== undefined ? item.start : ''}
                    placeholder="Auto (next)"
                    onChange={(e) => {
                      const val = e.target.value === '' ? undefined : parseInt(e.target.value, 10);
                      onChange(updateItem(layout, colIndex, itemIndex, { start: val }));
                    }}
                  />
                  <Input
                    label="Step (optional)"
                    type="number"
                    value={item.step !== undefined ? item.step : ''}
                    placeholder="1"
                    onChange={(e) => {
                      const val = e.target.value === '' ? undefined : parseInt(e.target.value, 10);
                      onChange(updateItem(layout, colIndex, itemIndex, { step: val }));
                    }}
                  />
                </div>
                <div className="flex items-center gap-2 pt-1">
                  <input
                    type="checkbox"
                    id="seat-inverse"
                    checked={!!item.inverse}
                    onChange={(e) =>
                      onChange(updateItem(layout, colIndex, itemIndex, { inverse: e.target.checked }))
                    }
                    className="rounded border-slate-300 text-faculty focus:ring-faculty h-4 w-4"
                  />
                  <label htmlFor="seat-inverse" className="text-xs font-medium text-slate-700 dark:text-slate-300">
                    Inverse numbering (count down / reverse)
                  </label>
                </div>
                <Input
                  label="Manual Seat Numbers"
                  placeholder="e.g. 53, 52, 50"
                  value={item.manual ? item.manual.join(', ') : ''}
                  onChange={(e) => {
                    const text = e.target.value.trim();
                    if (!text) {
                      onChange(updateItem(layout, colIndex, itemIndex, { manual: undefined }));
                    } else {
                      const nums = text
                        .split(',')
                        .map((s) => parseInt(s.trim(), 10))
                        .filter((n) => !isNaN(n));
                      onChange(updateItem(layout, colIndex, itemIndex, { manual: nums }));
                    }
                  }}
                  helperText="Comma-separated manual numbers overrides start/step"
                />
              </div>
            )}

            {/* Obstruction Item Editor */}
            {item.type === 'obstruction' && (
              <div className="flex flex-col gap-3">
                <Input
                  label="Obstruction Label"
                  value={item.label || ''}
                  placeholder="e.g. Pillar, Podium"
                  onChange={(e) =>
                    onChange(updateItem(layout, colIndex, itemIndex, { label: e.target.value }))
                  }
                />
                <div className="grid grid-cols-2 gap-2">
                  <Input
                    label="Span Count"
                    type="number"
                    min={1}
                    value={item.count || 1}
                    onChange={(e) =>
                      onChange(updateItem(layout, colIndex, itemIndex, { count: Math.max(1, parseInt(e.target.value, 10) || 1) }))
                    }
                  />
                  <Input
                    label="Custom Width (px)"
                    type="number"
                    value={item.width || 42}
                    onChange={(e) =>
                      onChange(updateItem(layout, colIndex, itemIndex, { width: parseInt(e.target.value, 10) || undefined }))
                    }
                  />
                </div>
                <Input
                  label="Custom Height (px, optional)"
                  type="number"
                  value={item.height || ''}
                  placeholder="Auto"
                  onChange={(e) => {
                    const val = e.target.value === '' ? undefined : parseInt(e.target.value, 10);
                    onChange(updateItem(layout, colIndex, itemIndex, { height: val }));
                  }}
                />
                <div className="flex items-center gap-2 pt-1">
                  <input
                    type="checkbox"
                    id="obs-transparent"
                    checked={!!item.transparent}
                    onChange={(e) =>
                      onChange(updateItem(layout, colIndex, itemIndex, { transparent: e.target.checked }))
                    }
                    className="rounded border-slate-300 text-faculty focus:ring-faculty h-4 w-4"
                  />
                  <label htmlFor="obs-transparent" className="text-xs font-medium text-slate-700 dark:text-slate-300">
                    Transparent obstruction (invisible spacer)
                  </label>
                </div>
              </div>
            )}

            {/* Gap Item Editor */}
            {item.type === 'gap' && (
              <div className="flex flex-col gap-3">
                <Input
                  label="Gap Seat Count"
                  type="number"
                  min={1}
                  value={item.count || 1}
                  onChange={(e) =>
                    onChange(updateItem(layout, colIndex, itemIndex, { count: Math.max(1, parseInt(e.target.value, 10) || 1) }))
                  }
                  helperText="Vertical spacing in seat heights"
                />
              </div>
            )}
          </div>
        );
      }
    }
  }

  // 2. If a block is selected
  if (selection?.type === 'block') {
    const blockIndex = selection.index;
    const block = layout.layout[blockIndex];

    if (block) {
      if (block.type === 'aisle') {
        const aisle = block as AisleBlock;
        return (
          <div className="flex flex-col gap-4 p-4 bg-white dark:bg-slate-900 border border-slate-200/80 dark:border-slate-800/90 rounded-2xl shadow-sm shadow-teal-950/[0.025] dark:shadow-none h-full overflow-y-auto">
            <div className="flex items-center justify-between border-b border-slate-200 dark:border-slate-800 pb-3">
              <div>
                <span className="text-[10px] font-bold uppercase tracking-wider text-faculty dark:text-blue-400">
                  Aisle
                </span>
                <h3 className="text-sm font-bold text-slate-800 dark:text-slate-100">
                  Aisle #{blockIndex + 1}
                </h3>
              </div>
              <div className="flex items-center gap-1">
                <Button
                  variant="ghost"
                  size="sm"
                  disabled={blockIndex === 0}
                  onClick={() => {
                    const next = moveBlock(layout, blockIndex, blockIndex - 1);
                    onChange(next);
                    onSelect({ type: 'block', index: blockIndex - 1 });
                  }}
                  title="Move Left"
                >
                  <ArrowLeft className="w-3.5 h-3.5" />
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  disabled={blockIndex === layout.layout.length - 1}
                  onClick={() => {
                    const next = moveBlock(layout, blockIndex, blockIndex + 1);
                    onChange(next);
                    onSelect({ type: 'block', index: blockIndex + 1 });
                  }}
                  title="Move Right"
                >
                  <ArrowRight className="w-3.5 h-3.5" />
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    const next = removeBlock(layout, blockIndex);
                    onChange(next);
                    onSelect(null);
                  }}
                  className="text-rose-600 hover:text-rose-700 hover:bg-rose-50 dark:hover:bg-rose-950/40"
                  title="Delete Aisle"
                >
                  <Trash2 className="w-3.5 h-3.5" />
                </Button>
              </div>
            </div>

            <Input
              label="Aisle Width (px)"
              type="number"
              min={10}
              max={200}
              value={aisle.width || 20}
              onChange={(e) =>
                onChange(updateBlock(layout, blockIndex, { width: parseInt(e.target.value, 10) || 20 }))
              }
              helperText="Pixel width of this hallway / walkway"
            />
          </div>
        );
      }

      // Column block selected
      const col = block as ColumnBlock;
      return (
        <div className="flex flex-col gap-4 p-4 bg-white dark:bg-slate-900 border border-slate-200/80 dark:border-slate-800/90 rounded-2xl shadow-sm shadow-teal-950/[0.025] dark:shadow-none h-full overflow-y-auto">
          <div className="flex items-center justify-between border-b border-slate-200 dark:border-slate-800 pb-3">
            <div>
              <span className="text-[10px] font-bold uppercase tracking-wider text-faculty dark:text-blue-400">
                Column Block
              </span>
              <h3 className="text-sm font-bold text-slate-800 dark:text-slate-100">
                {col.label || `Col ${blockIndex + 1}`} ({col.items.length} items)
              </h3>
            </div>
            <div className="flex items-center gap-1">
              <Button
                variant="ghost"
                size="sm"
                disabled={blockIndex === 0}
                onClick={() => {
                  const next = moveBlock(layout, blockIndex, blockIndex - 1);
                  onChange(next);
                  onSelect({ type: 'block', index: blockIndex - 1 });
                }}
                title="Move Left"
              >
                <ArrowLeft className="w-3.5 h-3.5" />
              </Button>
              <Button
                variant="ghost"
                size="sm"
                disabled={blockIndex === layout.layout.length - 1}
                onClick={() => {
                  const next = moveBlock(layout, blockIndex, blockIndex + 1);
                  onChange(next);
                  onSelect({ type: 'block', index: blockIndex + 1 });
                }}
                title="Move Right"
              >
                <ArrowRight className="w-3.5 h-3.5" />
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  const next = removeBlock(layout, blockIndex);
                  onChange(next);
                  onSelect(null);
                }}
                className="text-rose-600 hover:text-rose-700 hover:bg-rose-50 dark:hover:bg-rose-950/40"
                title="Delete Column"
              >
                <Trash2 className="w-3.5 h-3.5" />
              </Button>
            </div>
          </div>

          <Input
            label="Column Label"
            value={col.label || ''}
            placeholder={`e.g. C${blockIndex + 1}`}
            onChange={(e) =>
              onChange(updateBlock(layout, blockIndex, { label: e.target.value }))
            }
          />

          <div className="flex flex-col gap-2 pt-2 border-t border-slate-200 dark:border-slate-800">
            <span className="text-xs font-semibold text-slate-700 dark:text-slate-300">
              Add item to this column:
            </span>
            <div className="flex flex-col gap-1.5">
              <Button
                variant="secondary"
                size="sm"
                className="justify-start"
                onClick={() => {
                  const newItem: SeatItem = { type: 'seats', char: 'A', count: 5 };
                  const next = addItem(layout, blockIndex, newItem);
                  onChange(next);
                  onSelect({ type: 'item', colIndex: blockIndex, itemIndex: col.items.length });
                }}
              >
                <Armchair className="w-4 h-4 text-faculty dark:text-blue-300" />
                Add Row of Seats
              </Button>
              <Button
                variant="secondary"
                size="sm"
                className="justify-start"
                onClick={() => {
                  const newItem: ObstructionItem = { type: 'obstruction', count: 1, label: 'Pillar' };
                  const next = addItem(layout, blockIndex, newItem);
                  onChange(next);
                  onSelect({ type: 'item', colIndex: blockIndex, itemIndex: col.items.length });
                }}
              >
                <ShieldAlert className="w-4 h-4 text-amber-500" />
                Add Obstruction
              </Button>
              <Button
                variant="secondary"
                size="sm"
                className="justify-start"
                onClick={() => {
                  const newItem: GapItem = { type: 'gap', count: 1 };
                  const next = addItem(layout, blockIndex, newItem);
                  onChange(next);
                  onSelect({ type: 'item', colIndex: blockIndex, itemIndex: col.items.length });
                }}
              >
                <Space className="w-4 h-4 text-slate-400" />
                Add Spacer / Gap
              </Button>
            </div>
          </div>
        </div>
      );
    }
  }

  // 3. Nothing selected -> Room layout global settings
  return (
    <div className="flex flex-col gap-4 p-4 bg-white dark:bg-slate-900 border border-slate-200/80 dark:border-slate-800/90 rounded-2xl shadow-sm shadow-teal-950/[0.025] dark:shadow-none h-full overflow-y-auto">
      <div className="border-b border-slate-200 dark:border-slate-800 pb-3">
        <span className="text-[10px] font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400">
          Room Layout Canvas
        </span>
        <h3 className="text-sm font-bold text-slate-800 dark:text-slate-100">Global Settings</h3>
        <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
          Click any column, aisle, or seat to inspect and edit details.
        </p>
      </div>

      <div className="flex flex-col gap-3">
        <Input
          label="Front Label"
          value={layout.frontLabel || ''}
          placeholder="Front (default)"
          onChange={(e) => onChange(updateFrontBackLabels(layout, e.target.value, undefined))}
          helperText="e.g. 'Board', 'Screen', or 'Front'"
        />
        <Input
          label="Back Label"
          value={layout.backLabel || ''}
          placeholder="Back (default)"
          onChange={(e) => onChange(updateFrontBackLabels(layout, undefined, e.target.value))}
          helperText="e.g. 'Back' or 'Exit door'"
        />
      </div>

      <div className="flex flex-col gap-2 pt-2 border-t border-slate-200 dark:border-slate-800">
        <span className="text-xs font-semibold text-slate-700 dark:text-slate-300">
          Add blocks:
        </span>
        <div className="flex flex-col gap-1.5">
          <Button
            variant="secondary"
            size="sm"
            className="justify-start"
            onClick={() => {
              const next = addColumn(layout, `C${layout.layout.filter((b) => b.type === 'column').length + 1}`);
              onChange(next);
              onSelect({ type: 'block', index: next.layout.length - 1 });
            }}
          >
            <Columns className="w-4 h-4 text-faculty dark:text-blue-300" />
            Add New Column
          </Button>
          <Button
            variant="secondary"
            size="sm"
            className="justify-start"
            onClick={() => {
              const next = addAisle(layout, 20);
              onChange(next);
              onSelect({ type: 'block', index: next.layout.length - 1 });
            }}
          >
            <SplitSquareVertical className="w-4 h-4 text-slate-400" />
            Add Aisle
          </Button>
        </div>
      </div>
    </div>
  );
};
