export interface RoomMeta {
  layout_file: string;
  layout_image: string;
  map_url: string;
  images: string[];
}

export type RoomConfigMap = Record<string, RoomMeta>;

export type ItemType = 'seats' | 'obstruction' | 'gap';

export interface SeatItem {
  type: 'seats';
  char: string;
  start?: number;
  count?: number;
  inverse?: boolean;
  manual?: number[];
  step?: number;
  _calculatedNums?: number[];
}

export interface ObstructionItem {
  type: 'obstruction';
  label?: string;
  count?: number;
  width?: number;
  height?: number;
  transparent?: boolean;
}

export interface GapItem {
  type: 'gap';
  count: number;
}

export type LayoutItem = SeatItem | ObstructionItem | GapItem;

export interface ColumnBlock {
  type: 'column';
  label?: string;
  items: LayoutItem[];
}

export interface AisleBlock {
  type: 'aisle';
  width: number;
}

export type LayoutBlock = ColumnBlock | AisleBlock;

export interface RoomLayout {
  layout: LayoutBlock[];
  frontLabel?: string;
  backLabel?: string;
}

export interface ToastData {
  message: string;
  type: 'success' | 'error';
}
