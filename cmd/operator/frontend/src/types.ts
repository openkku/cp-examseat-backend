export interface RoomMeta {
  layout_file: string;
  layout_image: string;
  map_url: string;
  images: string[];
  title?: string;
  description?: string;
  lat?: number;
  lng?: number;
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

export interface CustomDataset {
  id: string;
  seats: number;
}

export interface RoundSummary {
  id: string;
  label: string;
  seats: number;
  students: number;
  in_schedule_seats: number;
  custom_datasets: CustomDataset[];
}

export interface ReplaceTarget {
  type: 'in_schedule' | 'custom_dataset';
  id: string;
  seats: number;
}

export interface PreviewRow {
  sheet: string;
  date: string;
  time: string;
  room: string;
  subject: string;
  subject_name: string;
  section: string;
  student_id: string;
  seat: string;
  custom_id: string;
  labels: string[];
}

export interface ImportPreview {
  seats: number;
  students: number;
  rooms: string[];
  dates: string[];
  sessions: number;
  custom_ids: string[];
  replaces: ReplaceTarget[];
  rows: PreviewRow[];
}

export interface PreviewSeat {
  seat: string;
  student_id: string;
  subject: string;
  section: string;
}

export type ProblemKind =
  | 'missing_metadata'
  | 'invalid_layout'
  | 'missing_seats'
  | 'missing_image'
  | 'unused_room';

export type ProblemSeverity = 'error' | 'warning' | 'info';

export interface HealthProblem {
  kind: ProblemKind;
  severity: ProblemSeverity;
  room?: string;
  round?: string;
  message: string;
  details?: string[];
}

export interface HealthReport {
  problems: HealthProblem[];
}
