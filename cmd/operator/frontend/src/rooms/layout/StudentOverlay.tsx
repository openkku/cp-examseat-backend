import React, { useState, useEffect } from 'react';
import { AlertTriangle, Users, X } from 'lucide-react';
import { previewApi, roundsApi } from '../../api';
import type { PreviewSeat, RoundSummary } from '../../types';
import { Button } from '../../ui/Button';
import { Select } from '../../ui/Select';
import { Badge } from '../../ui/Badge';

export interface StudentOverlayProps {
  roomId: string;
  layoutSeatSet: Set<string>;
  onRosterLoaded: (roster: Record<string, PreviewSeat>) => void;
  onClose: () => void;
}

export const StudentOverlay: React.FC<StudentOverlayProps> = ({
  roomId,
  layoutSeatSet,
  onRosterLoaded,
  onClose,
}) => {
  const [rounds, setRounds] = useState<RoundSummary[]>([]);
  const [selectedRound, setSelectedRound] = useState<string>('');
  const [dates, setDates] = useState<string[]>([]);
  const [selectedDate, setSelectedDate] = useState<string>('');
  const [times, setTimes] = useState<string[]>([]);
  const [selectedTime, setSelectedTime] = useState<string>('');
  const [loading, setLoading] = useState(false);
  const [unmappedSeats, setUnmappedSeats] = useState<string[]>([]);

  useEffect(() => {
    roundsApi.list().then((list) => {
      setRounds(list);
      if (list.length > 0) {
        setSelectedRound(list[0].id);
      }
    }).catch(console.error);
  }, []);

  useEffect(() => {
    let active = true;
    if (!selectedRound || !roomId) {
      void Promise.resolve().then(() => {
        if (active) {
          setDates([]);
          setSelectedDate('');
        }
      });
      return;
    }
    previewApi.options('dates', selectedRound, roomId).then((d) => {
      if (!active) return;
      setDates(d);
      setSelectedDate(d[0] || '');
    }).catch(() => {
      if (!active) return;
      setDates([]);
      setSelectedDate('');
    });
    return () => {
      active = false;
    };
  }, [selectedRound, roomId]);

  useEffect(() => {
    let active = true;
    if (!selectedRound || !roomId || !selectedDate) {
      void Promise.resolve().then(() => {
        if (active) {
          setTimes([]);
          setSelectedTime('');
        }
      });
      return;
    }
    previewApi.options('times', selectedRound, roomId, selectedDate).then((t) => {
      if (!active) return;
      setTimes(t);
      setSelectedTime(t[0] || '');
    }).catch(() => {
      if (!active) return;
      setTimes([]);
      setSelectedTime('');
    });
    return () => {
      active = false;
    };
  }, [selectedRound, roomId, selectedDate]);

  useEffect(() => {
    let active = true;
    if (!selectedRound || !roomId || !selectedDate || !selectedTime) {
      void Promise.resolve().then(() => {
        if (active) {
          onRosterLoaded({});
          setUnmappedSeats([]);
        }
      });
      return;
    }

    void Promise.resolve().then(() => {
      if (active) setLoading(true);
    });
    previewApi.seats(selectedRound, roomId, selectedDate, selectedTime)
      .then((seats) => {
        if (!active) return;
        const roster: Record<string, PreviewSeat> = {};
        const missing: string[] = [];

        for (const s of seats) {
          roster[s.seat] = s;
          if (!layoutSeatSet.has(s.seat)) {
            missing.push(s.seat);
          }
        }

        onRosterLoaded(roster);
        setUnmappedSeats(missing);
      })
      .catch((err) => {
        if (!active) return;
        console.error('Failed to load roster:', err);
        onRosterLoaded({});
        setUnmappedSeats([]);
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [selectedRound, roomId, selectedDate, selectedTime, layoutSeatSet, onRosterLoaded]);

  return (
    <div className="bg-white dark:bg-slate-900 border border-slate-200/80 dark:border-slate-800/90 rounded-2xl shadow-sm shadow-teal-950/[0.025] dark:shadow-none p-3 shadow-md mb-3 flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1.5 text-xs font-semibold text-slate-800 dark:text-slate-200">
          <Users className="w-4 h-4 text-faculty dark:text-blue-300" />
          <span>Real-Student Live Preview</span>
          {loading && <span className="text-[10px] text-slate-400 animate-pulse">(Loading...)</span>}
        </div>
        <Button variant="ghost" size="sm" onClick={onClose} aria-label="Close overlay">
          <X className="w-4 h-4" />
        </Button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-2">
        <Select
          label="Round"
          value={selectedRound}
          onChange={(e) => setSelectedRound(e.target.value)}
        >
          {rounds.map((r) => (
            <option key={r.id} value={r.id}>
              {r.label || r.id}
            </option>
          ))}
        </Select>

        <Select
          label="Exam Date"
          value={selectedDate}
          disabled={dates.length === 0}
          onChange={(e) => setSelectedDate(e.target.value)}
        >
          {dates.length === 0 ? (
            <option value="">No exam dates for this room</option>
          ) : (
            dates.map((d) => (
              <option key={d} value={d}>
                {d}
              </option>
            ))
          )}
        </Select>

        <Select
          label="Time Slot"
          value={selectedTime}
          disabled={times.length === 0}
          onChange={(e) => setSelectedTime(e.target.value)}
        >
          {times.length === 0 ? (
            <option value="">No slots</option>
          ) : (
            times.map((t) => (
              <option key={t} value={t}>
                {t}
              </option>
            ))
          )}
        </Select>
      </div>

      {unmappedSeats.length > 0 && (
        <div className="mt-1 flex items-start gap-2 p-2 bg-amber-50 dark:bg-amber-950/50 border border-amber-200 dark:border-amber-800/80 rounded-lg text-xs text-amber-800 dark:text-amber-200">
          <AlertTriangle className="w-4 h-4 text-amber-600 shrink-0 mt-0.5" />
          <div className="flex-1">
            <span className="font-semibold">
              {unmappedSeats.length} student seat(s) in this session are missing from the layout:
            </span>
            <div className="flex flex-wrap gap-1 mt-1">
              {unmappedSeats.slice(0, 15).map((s) => (
                <Badge key={s} variant="warning" size="sm">
                  {s}
                </Badge>
              ))}
              {unmappedSeats.length > 15 && (
                <span className="text-[10px] text-amber-600">+{unmappedSeats.length - 15} more</span>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
