import { describe, it, expect } from 'vitest';
import fs from 'node:fs';
import path from 'node:path';
import { computeSeatIDs, getItemSeatNumbers, addColumn, addAisle, removeBlock } from '../model';
import type { RoomLayout, SeatItem } from '../../../types';

describe('Layout Model and Seat Numbering', () => {
  it('matches Go seat numbering against testdata/layouts fixtures', () => {
    // Find testdata directory
    const candidates = [
      path.resolve(process.cwd(), '../../../testdata/layouts'),
      path.resolve(process.cwd(), '../../testdata/layouts'),
      path.resolve(process.cwd(), 'testdata/layouts'),
    ];
    const testdataDir = candidates.find((dir) => fs.existsSync(dir));
    expect(testdataDir, 'testdata/layouts directory should exist').toBeDefined();

    const jsonFiles = fs.readdirSync(testdataDir!).filter((f: string) => f.endsWith('.json'));
    expect(jsonFiles.length).toBeGreaterThan(0);

    for (const file of jsonFiles) {
      const fullPath = path.join(testdataDir!, file);
      const layoutData: RoomLayout = JSON.parse(fs.readFileSync(fullPath, 'utf-8'));
      const gotSeats = computeSeatIDs(layoutData);

      const seatsFile = path.join(testdataDir!, file.replace(/\.json$/, '.seats.txt'));
      expect(fs.existsSync(seatsFile), `Expected ${seatsFile} to exist`).toBe(true);

      const wantSeats = fs
        .readFileSync(seatsFile, 'utf-8')
        .split('\n')
        .map((s: string) => s.trim())
        .filter(Boolean);

      expect(gotSeats, `Mismatch for fixture ${file}`).toEqual(wantSeats);
    }
  });

  it('handles start, step, count, and inverse', () => {
    const counters: Record<string, number> = {};
    const item: SeatItem = {
      type: 'seats',
      char: 'B',
      start: 10,
      step: 2,
      count: 3,
      inverse: false,
    };
    const nums = getItemSeatNumbers(item, counters);
    expect(nums).toEqual([10, 12, 14]);
    expect(counters['B']).toBe(16);
  });

  it('handles manual seat numbers with inverse flag', () => {
    const counters: Record<string, number> = {};
    const item: SeatItem = {
      type: 'seats',
      char: 'A',
      manual: [1, 2, 5],
      inverse: true,
    };
    const nums = getItemSeatNumbers(item, counters);
    expect(nums).toEqual([5, 2, 1]);
  });

  it('manipulates columns and aisles', () => {
    let layout: RoomLayout = { layout: [] };
    layout = addColumn(layout, 'C1');
    layout = addAisle(layout, 30);
    expect(layout.layout.length).toBe(2);
    expect(layout.layout[0].type).toBe('column');
    expect(layout.layout[1].type).toBe('aisle');

    layout = removeBlock(layout, 0);
    expect(layout.layout.length).toBe(1);
    expect(layout.layout[0].type).toBe('aisle');
  });
});
