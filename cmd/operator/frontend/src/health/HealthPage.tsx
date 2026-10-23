import React, { useState, useEffect, useCallback } from 'react';
import {
  Activity,
  AlertCircle,
  AlertTriangle,
  Info,
  CheckCircle2,
  RotateCcw,
  ArrowRight,
  FileQuestion,
  ImageOff,
  HelpCircle,
  DoorClosed,
} from 'lucide-react';
import { healthApi } from '../api';
import type { HealthProblem, ProblemKind } from '../types';
import { Button } from '../ui/Button';
import { Badge } from '../ui/Badge';
import { useToast } from '../ui/useToast';

export interface HealthPageProps {
  onSelectRoom?: (roomId: string) => void;
}

const KIND_METADATA: Record<
  ProblemKind,
  { title: string; description: string; icon: React.FC<{ className?: string }> }
> = {
  missing_metadata: {
    title: 'Rooms in Rounds with Missing Metadata',
    description: 'Exam sessions reference rooms that have no entry in room/metadata.json.',
    icon: HelpCircle,
  },
  invalid_layout: {
    title: 'Missing or Unparsable Layout Files',
    description: 'Room configuration points to a layout file that is missing on disk or contains invalid JSON.',
    icon: FileQuestion,
  },
  missing_seats: {
    title: 'Seats in Rounds Missing from Layout',
    description: 'Students are seated in positions not defined in the room layout canvas.',
    icon: AlertTriangle,
  },
  missing_image: {
    title: 'Missing Room or Layout Images',
    description: 'Referenced layout_image or pictures do not exist on disk.',
    icon: ImageOff,
  },
  unused_room: {
    title: 'Configured Rooms with No Seats',
    description: 'Rooms listed in metadata that are not utilized in any active round.',
    icon: DoorClosed,
  },
};

export const HealthPage: React.FC<HealthPageProps> = ({ onSelectRoom }) => {
  const { toast } = useToast();
  const [problems, setProblems] = useState<HealthProblem[]>([]);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async () => {
    try {
      setLoading(true);
      const res = await healthApi.check();
      setProblems(res.problems || []);
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to execute health checks');
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    let ignore = false;
    healthApi
      .check()
      .then((res) => {
        if (!ignore) {
          setProblems(res.problems || []);
          setLoading(false);
        }
      })
      .catch((err: unknown) => {
        if (!ignore) {
          toast.error((err as Error).message || 'Failed to execute health checks');
          setLoading(false);
        }
      });
    return () => {
      ignore = true;
    };
  }, [toast]);

  const grouped = React.useMemo(() => {
    const map: Record<string, HealthProblem[]> = {};
    for (const p of problems) {
      if (!map[p.kind]) map[p.kind] = [];
      map[p.kind].push(p);
    }
    return map;
  }, [problems]);

  const problemKinds: ProblemKind[] = [
    'missing_metadata',
    'invalid_layout',
    'missing_seats',
    'missing_image',
    'unused_room',
  ];

  return (
    <div className="p-6 max-w-6xl mx-auto flex flex-col gap-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            <Activity className="w-5 h-5 text-faculty dark:text-blue-400" />
            <h1 className="text-xl font-black tracking-tight text-slate-800 dark:text-slate-100">
              Data Health & Integrity Checks
            </h1>
          </div>
          <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
            Automated verification across round sessions, room metadata, canvas layouts, and image assets.
          </p>
        </div>

        <Button variant="secondary" size="sm" onClick={refresh} loading={loading}>
          <RotateCcw className="w-3.5 h-3.5" />
          Re-run Checks
        </Button>
      </div>

      {loading ? (
        <div className="p-12 text-center text-xs text-slate-400">Running health diagnostics...</div>
      ) : problems.length === 0 ? (
        <div className="p-12 text-center border-2 border-dashed border-teal-200 dark:border-teal-800/60 bg-teal-50/30 dark:bg-teal-950/20 rounded-2xl">
          <CheckCircle2 className="w-10 h-10 text-teal-500 mx-auto mb-3" />
          <h2 className="text-base font-bold text-teal-900 dark:text-teal-100">
            All Systems Healthy
          </h2>
          <p className="text-xs text-teal-700 dark:text-teal-300 mt-1 max-w-md mx-auto">
            Zero discrepancies detected! All rooms have valid metadata, layout files parse correctly, seats align with exam rosters, and images exist.
          </p>
        </div>
      ) : (
        <div className="flex flex-col gap-6">
          {problemKinds.map((kind) => {
            const list = grouped[kind];
            if (!list || list.length === 0) return null;

            const meta = KIND_METADATA[kind];
            const Icon = meta.icon;

            return (
              <div
                key={kind}
                className="bg-white dark:bg-slate-900 border border-slate-200/80 dark:border-slate-800/90 rounded-2xl shadow-sm shadow-teal-950/[0.025] dark:shadow-none overflow-hidden"
              >
                <div className="p-4 border-b border-slate-200 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-900/50 flex items-center justify-between">
                  <div className="flex items-center gap-2.5">
                    <div className="p-2 rounded-lg bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-300">
                      <Icon className="w-4 h-4" />
                    </div>
                    <div>
                      <h2 className="text-sm font-bold text-slate-800 dark:text-slate-100">
                        {meta.title}
                      </h2>
                      <p className="text-xs text-slate-500 dark:text-slate-400">{meta.description}</p>
                    </div>
                  </div>
                  <Badge
                    variant={
                      kind === 'unused_room'
                        ? 'info'
                        : kind === 'missing_seats' || kind === 'missing_image'
                        ? 'warning'
                        : 'error'
                    }
                  >
                    {list.length} {list.length === 1 ? 'issue' : 'issues'}
                  </Badge>
                </div>

                <div className="divide-y divide-slate-100 dark:divide-slate-800">
                  {list.map((problem, idx) => (
                    <div
                      key={idx}
                      className="p-3.5 flex flex-col md:flex-row md:items-center justify-between gap-3 text-xs hover:bg-slate-50/60 dark:hover:bg-slate-800/30 transition-colors"
                    >
                      <div className="flex items-start gap-2.5">
                        {problem.severity === 'error' && (
                          <AlertCircle className="w-4 h-4 text-rose-500 shrink-0 mt-0.5" />
                        )}
                        {problem.severity === 'warning' && (
                          <AlertTriangle className="w-4 h-4 text-amber-500 shrink-0 mt-0.5" />
                        )}
                        {problem.severity === 'info' && (
                          <Info className="w-4 h-4 text-faculty dark:text-blue-300 shrink-0 mt-0.5" />
                        )}

                        <div className="flex flex-col gap-1">
                          <div className="flex items-center gap-2 flex-wrap">
                            {problem.room && (
                              <span className="font-semibold text-slate-900 dark:text-slate-100">
                                Room {problem.room}
                              </span>
                            )}
                            {problem.round && (
                              <Badge variant="neutral" size="sm">
                                Round: {problem.round}
                              </Badge>
                            )}
                          </div>
                          <p className="text-slate-600 dark:text-slate-300 font-mono text-[11px]">
                            {problem.message}
                          </p>

                          {problem.details && problem.details.length > 0 && (
                            <div className="flex flex-wrap gap-1 mt-1">
                              {problem.details.map((seat) => (
                                <Badge key={seat} variant="warning" size="sm">
                                  {seat}
                                </Badge>
                              ))}
                            </div>
                          )}
                        </div>
                      </div>

                      {problem.room && onSelectRoom && (
                        <button
                          onClick={() => onSelectRoom(problem.room!)}
                          className="self-start md:self-center shrink-0 inline-flex items-center gap-1 px-2.5 py-1 text-xs font-semibold text-faculty hover:text-faculty-dark dark:text-blue-400 dark:hover:text-blue-300 rounded hover:bg-faculty-light dark:hover:bg-blue-950/40 transition-colors"
                        >
                          Open room <ArrowRight className="w-3.5 h-3.5" />
                        </button>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};
