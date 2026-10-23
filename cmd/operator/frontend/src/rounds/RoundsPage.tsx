import React, { useState, useEffect, useCallback } from 'react';
import {
  Upload,
  RotateCcw,
  Pencil,
  Check,
  X,
  Trash2,
  Calendar,
  Layers,
} from 'lucide-react';
import { roundsApi, LIVE_SOON } from '../api';
import type { RoundSummary } from '../types';
import { Button } from '../ui/Button';
import { ConfirmDialog } from '../ui/Dialog';
import { Table, Thead, Tbody, Tr, Th, Td } from '../ui/Table';
import { useToast } from '../ui/useToast';
import { ImportWizard } from './ImportWizard';

export const RoundsPage: React.FC = () => {
  const { toast } = useToast();

  const [rounds, setRounds] = useState<RoundSummary[]>([]);
  const [loading, setLoading] = useState(true);

  // Import wizard
  const [wizardOpen, setWizardOpen] = useState(false);

  // Inline rename state
  const [editingRoundId, setEditingRoundId] = useState<string | null>(null);
  const [editLabelValue, setEditLabelValue] = useState<string>('');
  const [isRenaming, setIsRenaming] = useState(false);

  // Reload state
  const [isReloading, setIsReloading] = useState(false);

  // Delete modal states
  const [roundToDelete, setRoundToDelete] = useState<string | null>(null);
  const [datasetToDelete, setDatasetToDelete] = useState<{ round: string; customId: string } | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);

  const fetchRounds = useCallback(async () => {
    try {
      setLoading(true);
      const data = await roundsApi.list();
      setRounds(data);
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to load rounds');
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    let ignore = false;
    roundsApi
      .list()
      .then((data) => {
        if (!ignore) {
          setRounds(data);
          setLoading(false);
        }
      })
      .catch((err: unknown) => {
        if (!ignore) {
          toast.error((err as Error).message || 'Failed to load rounds');
          setLoading(false);
        }
      });
    return () => {
      ignore = true;
    };
  }, [toast]);

  const handleStartRename = (round: RoundSummary) => {
    setEditingRoundId(round.id);
    setEditLabelValue(round.label || '');
  };

  const handleCancelRename = () => {
    setEditingRoundId(null);
    setEditLabelValue('');
  };

  const handleSaveRename = async (roundId: string) => {
    const trimmed = editLabelValue.trim();
    if (!trimmed) return;

    try {
      setIsRenaming(true);
      await roundsApi.rename(roundId, trimmed);
      toast.success(`Renamed round ${roundId} ${LIVE_SOON}`);
      setEditingRoundId(null);
      await fetchRounds();
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to rename round');
    } finally {
      setIsRenaming(false);
    }
  };

  const handleDeleteRound = async () => {
    if (!roundToDelete) return;
    try {
      setIsDeleting(true);
      await roundsApi.remove(roundToDelete);
      toast.success(`Deleted round ${roundToDelete} ${LIVE_SOON}`);
      setRoundToDelete(null);
      await fetchRounds();
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to delete round');
    } finally {
      setIsDeleting(false);
    }
  };

  const handleDeleteDataset = async () => {
    if (!datasetToDelete) return;
    try {
      setIsDeleting(true);
      await roundsApi.remove(datasetToDelete.round, datasetToDelete.customId);
      toast.success(`Deleted dataset ${datasetToDelete.customId} from ${datasetToDelete.round} ${LIVE_SOON}`);
      setDatasetToDelete(null);
      await fetchRounds();
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to delete dataset');
    } finally {
      setIsDeleting(false);
    }
  };

  const handleReload = async () => {
    try {
      setIsReloading(true);
      await roundsApi.reload();
      toast.success('Signalled server reload successfully');
      await fetchRounds();
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Reload request failed');
    } finally {
      setIsReloading(false);
    }
  };

  return (
    <div className="p-6 max-w-7xl mx-auto flex flex-col gap-6">
      {/* Top Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            <Calendar className="w-5 h-5 text-faculty dark:text-blue-400" />
            <h1 className="text-xl font-black tracking-tight text-slate-800 dark:text-slate-100">
              Exam Rounds & Datasets
            </h1>
          </div>
          <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
            Manage official timetables, student enrollments, out-of-schedule exams, and trigger cache reloads.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <Button
            variant="secondary"
            size="sm"
            onClick={handleReload}
            loading={isReloading}
            title="Ask public server to reload memory caches immediately"
          >
            <RotateCcw className="w-3.5 h-3.5" />
            Reload Server
          </Button>
          <Button variant="primary" size="sm" onClick={() => setWizardOpen(true)}>
            <Upload className="w-3.5 h-3.5" />
            Import File
          </Button>
        </div>
      </div>

      {/* Rounds Table */}
      {loading ? (
        <div className="p-8 text-center text-xs text-slate-400">Loading exam rounds...</div>
      ) : rounds.length === 0 ? (
        <div className="p-12 text-center border-2 border-dashed border-slate-200 dark:border-slate-800 rounded-2xl">
          <Calendar className="w-8 h-8 text-slate-400 mx-auto mb-2" />
          <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-200">No rounds found</h3>
          <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
            Import an Excel or JSON schedule file to create the first exam round.
          </p>
          <div className="mt-4">
            <Button variant="primary" size="sm" onClick={() => setWizardOpen(true)}>
              <Upload className="w-3.5 h-3.5" />
              Import Now
            </Button>
          </div>
        </div>
      ) : (
        <Table>
          <Thead>
            <Tr>
              <Th>Round ID</Th>
              <Th>Display Label</Th>
              <Th className="text-right">Total Seats</Th>
              <Th className="text-right">Students</Th>
              <Th className="text-right">In-Schedule</Th>
              <Th>Custom Datasets</Th>
              <Th className="text-right">Actions</Th>
            </Tr>
          </Thead>
          <Tbody>
            {rounds.map((round) => {
              const isEditing = editingRoundId === round.id;

              return (
                <Tr key={round.id}>
                  {/* Round ID */}
                  <Td className="font-mono font-semibold text-xs text-faculty dark:text-blue-400">
                    {round.id}
                  </Td>

                  {/* Inline Editable Label */}
                  <Td>
                    {isEditing ? (
                      <div className="flex items-center gap-1.5 max-w-sm">
                        <input
                          type="text"
                          value={editLabelValue}
                          onChange={(e) => setEditLabelValue(e.target.value)}
                          onKeyDown={(e) => {
                            if (e.key === 'Enter') handleSaveRename(round.id);
                            if (e.key === 'Escape') handleCancelRename();
                          }}
                          className="text-xs rounded border border-faculty px-2 py-1 bg-white dark:bg-slate-900 focus:outline-none w-full"
                          autoFocus
                        />
                        <button
                          onClick={() => handleSaveRename(round.id)}
                          disabled={isRenaming}
                          className="p-1 text-teal-600 hover:text-teal-700 rounded"
                          title="Save label"
                        >
                          <Check className="w-4 h-4" />
                        </button>
                        <button
                          onClick={handleCancelRename}
                          disabled={isRenaming}
                          className="p-1 text-slate-400 hover:text-slate-600 rounded"
                          title="Cancel"
                        >
                          <X className="w-4 h-4" />
                        </button>
                      </div>
                    ) : (
                      <div className="flex items-center gap-1.5 group">
                        <span className="font-medium text-slate-900 dark:text-slate-100">
                          {round.label || '-'}
                        </span>
                        <button
                          onClick={() => handleStartRename(round)}
                          className="opacity-0 group-hover:opacity-100 p-1 text-slate-400 hover:text-faculty dark:hover:text-blue-300 transition-opacity rounded"
                          title="Rename label"
                        >
                          <Pencil className="w-3 h-3" />
                        </button>
                      </div>
                    )}
                  </Td>

                  {/* Seat and Student Counts */}
                  <Td className="text-right font-mono">{round.seats.toLocaleString()}</Td>
                  <Td className="text-right font-mono">{round.students.toLocaleString()}</Td>
                  <Td className="text-right font-mono">{round.in_schedule_seats.toLocaleString()}</Td>

                  {/* Custom Datasets with Individual Delete */}
                  <Td>
                    {round.custom_datasets && round.custom_datasets.length > 0 ? (
                      <div className="flex flex-wrap gap-1.5 items-center">
                        {round.custom_datasets.map((ds) => (
                          <div
                            key={ds.id}
                            className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[11px] bg-slate-100 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 text-slate-700 dark:text-slate-300"
                          >
                            <Layers className="w-3 h-3 text-amber-500" />
                            <span className="font-semibold">{ds.id}:</span>
                            <span className="font-mono text-slate-500">{ds.seats}</span>
                            <button
                              onClick={() => setDatasetToDelete({ round: round.id, customId: ds.id })}
                              className="ml-1 text-slate-400 hover:text-rose-600"
                              title={`Delete dataset ${ds.id}`}
                            >
                              <X className="w-3 h-3" />
                            </button>
                          </div>
                        ))}
                      </div>
                    ) : (
                      <span className="text-xs text-slate-400 italic">None</span>
                    )}
                  </Td>

                  {/* Round Delete Action */}
                  <Td className="text-right">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setRoundToDelete(round.id)}
                      className="text-slate-400 hover:text-rose-600"
                      title={`Delete round ${round.id}`}
                    >
                      <Trash2 className="w-4 h-4" />
                    </Button>
                  </Td>
                </Tr>
              );
            })}
          </Tbody>
        </Table>
      )}

      {/* Import Wizard Modal */}
      <ImportWizard
        open={wizardOpen}
        onClose={() => setWizardOpen(false)}
        onImportComplete={fetchRounds}
      />

      {/* Delete Round Confirm Dialog */}
      <ConfirmDialog
        open={roundToDelete !== null}
        onClose={() => setRoundToDelete(null)}
        onConfirm={handleDeleteRound}
        title="Delete Round"
        message={`Are you sure you want to permanently delete round "${roundToDelete}" and all its scheduled seats?`}
        confirmText="Delete Round"
        loading={isDeleting}
      />

      {/* Delete Dataset Confirm Dialog */}
      <ConfirmDialog
        open={datasetToDelete !== null}
        onClose={() => setDatasetToDelete(null)}
        onConfirm={handleDeleteDataset}
        title="Delete Custom Dataset"
        message={`Are you sure you want to delete custom dataset "${datasetToDelete?.customId}" from round "${datasetToDelete?.round}"?`}
        confirmText="Delete Dataset"
        loading={isDeleting}
      />
    </div>
  );
};
