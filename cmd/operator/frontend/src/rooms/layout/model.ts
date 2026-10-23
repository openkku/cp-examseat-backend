import type {
  AisleBlock,
  ColumnBlock,
  LayoutBlock,
  LayoutItem,
  RoomLayout,
  SeatItem,
} from '../../types';

/**
 * Extracts seat numbers for a single seats item given the running per-character counters.
 * Mutates counters in place to match the sequential layout evaluation rules.
 */
export function getItemSeatNumbers(
  item: SeatItem,
  counters: Record<string, number>
): number[] {
  // Like the site's SeatMap, a manual list (even an empty one) replaces counting.
  if (item.manual) {
    const nums = [...item.manual];
    if (item.inverse) {
      nums.reverse();
    }
    return nums;
  }

  const start = item.start ?? (counters[item.char] || 1);
  const count = item.count || 0;
  const step = item.step ?? (item.inverse ? -1 : 1);

  const numbers: number[] = [];
  let current = start;
  for (let i = 0; i < count; i++) {
    numbers.push(current);
    current += step;
  }
  counters[item.char] = current;
  return numbers;
}

/**
 * Computes all seat IDs across a layout in canonical order matching roomconfig.SeatIDs in Go.
 */
export function computeSeatIDs(layout: RoomLayout): string[] {
  const counters: Record<string, number> = {};
  const seats: string[] = [];

  for (const block of layout.layout) {
    if (block.type !== 'column') {
      continue;
    }
    const col = block as ColumnBlock;
    if (!col.items) continue;

    for (const item of col.items) {
      if (item.type !== 'seats' || !item.char) {
        continue;
      }
      const numbers = getItemSeatNumbers(item as SeatItem, counters);
      for (const num of numbers) {
        seats.push(`${item.char}${num}`);
      }
    }
  }

  return seats;
}

export function addColumn(layout: RoomLayout, label = ''): RoomLayout {
  const newCol: ColumnBlock = {
    type: 'column',
    label,
    items: [],
  };
  return {
    ...layout,
    layout: [...layout.layout, newCol],
  };
}

export function addAisle(layout: RoomLayout, width = 20): RoomLayout {
  const newAisle: AisleBlock = {
    type: 'aisle',
    width,
  };
  return {
    ...layout,
    layout: [...layout.layout, newAisle],
  };
}

export function updateBlock(
  layout: RoomLayout,
  index: number,
  update: Partial<LayoutBlock>
): RoomLayout {
  if (index < 0 || index >= layout.layout.length) return layout;
  const newBlocks = [...layout.layout];
  newBlocks[index] = { ...newBlocks[index], ...update } as LayoutBlock;
  return { ...layout, layout: newBlocks };
}

export function removeBlock(layout: RoomLayout, index: number): RoomLayout {
  if (index < 0 || index >= layout.layout.length) return layout;
  return {
    ...layout,
    layout: layout.layout.filter((_, i) => i !== index),
  };
}

export function moveBlock(
  layout: RoomLayout,
  fromIndex: number,
  toIndex: number
): RoomLayout {
  if (
    fromIndex < 0 ||
    fromIndex >= layout.layout.length ||
    toIndex < 0 ||
    toIndex >= layout.layout.length ||
    fromIndex === toIndex
  ) {
    return layout;
  }
  const newBlocks = [...layout.layout];
  const [removed] = newBlocks.splice(fromIndex, 1);
  newBlocks.splice(toIndex, 0, removed);
  return { ...layout, layout: newBlocks };
}

export function addItem(
  layout: RoomLayout,
  colIndex: number,
  item: LayoutItem
): RoomLayout {
  const col = layout.layout[colIndex];
  if (!col || col.type !== 'column') return layout;

  const newCol: ColumnBlock = {
    ...col,
    items: [...col.items, item],
  };
  return updateBlock(layout, colIndex, newCol);
}

export function updateItem(
  layout: RoomLayout,
  colIndex: number,
  itemIndex: number,
  update: Partial<LayoutItem>
): RoomLayout {
  const col = layout.layout[colIndex];
  if (!col || col.type !== 'column') return layout;
  if (itemIndex < 0 || itemIndex >= col.items.length) return layout;

  const newItems = [...col.items];
  newItems[itemIndex] = { ...newItems[itemIndex], ...update } as LayoutItem;

  const newCol: ColumnBlock = {
    ...col,
    items: newItems,
  };
  return updateBlock(layout, colIndex, newCol);
}

export function removeItem(
  layout: RoomLayout,
  colIndex: number,
  itemIndex: number
): RoomLayout {
  const col = layout.layout[colIndex];
  if (!col || col.type !== 'column') return layout;
  if (itemIndex < 0 || itemIndex >= col.items.length) return layout;

  const newCol: ColumnBlock = {
    ...col,
    items: col.items.filter((_, i) => i !== itemIndex),
  };
  return updateBlock(layout, colIndex, newCol);
}

export function moveItem(
  layout: RoomLayout,
  fromCol: number,
  fromIdx: number,
  toCol: number,
  toIdx: number
): RoomLayout {
  const sourceCol = layout.layout[fromCol];
  const targetCol = layout.layout[toCol];
  if (!sourceCol || sourceCol.type !== 'column' || !targetCol || targetCol.type !== 'column') {
    return layout;
  }
  if (fromIdx < 0 || fromIdx >= sourceCol.items.length) return layout;

  if (fromCol === toCol) {
    if (fromIdx === toIdx) return layout;
    const newItems = [...sourceCol.items];
    const [moved] = newItems.splice(fromIdx, 1);
    newItems.splice(toIdx, 0, moved);
    return updateBlock(layout, fromCol, { ...sourceCol, items: newItems });
  }

  const newSourceItems = [...sourceCol.items];
  const [moved] = newSourceItems.splice(fromIdx, 1);
  const newTargetItems = [...targetCol.items];
  newTargetItems.splice(toIdx, 0, moved);

  let newLayout = updateBlock(layout, fromCol, { ...sourceCol, items: newSourceItems });
  newLayout = updateBlock(newLayout, toCol, { ...targetCol, items: newTargetItems });
  return newLayout;
}

export function updateFrontBackLabels(
  layout: RoomLayout,
  front?: string,
  back?: string
): RoomLayout {
  return {
    ...layout,
    frontLabel: front !== undefined ? front : layout.frontLabel,
    backLabel: back !== undefined ? back : layout.backLabel,
  };
}
