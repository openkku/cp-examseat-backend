import React, { useState } from 'react';
import { Upload, FileText, CheckCircle2, AlertTriangle, ArrowRight } from 'lucide-react';
import { roundsApi, LIVE_SOON } from '../api';
import type { ImportPreview } from '../types';
import { Button } from '../ui/Button';
import { Input } from '../ui/Input';
import { Dialog } from '../ui/Dialog';
import { Table, Thead, Tbody, Tr, Th, Td } from '../ui/Table';
import { useToast } from '../ui/useToast';

export interface ImportWizardProps {
  open: boolean;
  onClose: () => void;
  onImportComplete: () => void;
}

export const ImportWizard: React.FC<ImportWizardProps> = ({
  open,
  onClose,
  onImportComplete,
}) => {
  const { toast } = useToast();

  const [file, setFile] = useState<File | null>(null);
  const [roundId, setRoundId] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [labels, setLabels] = useState('');
  const [roomLayout, setRoomLayout] = useState('');
  const [customId, setCustomId] = useState('');

  const [preview, setPreview] = useState<ImportPreview | null>(null);
  const [isPreviewing, setIsPreviewing] = useState(false);
  const [previewError, setPreviewError] = useState<string | null>(null);

  const [isImporting, setIsImporting] = useState(false);

  // Clear preview whenever any form parameter changes
  const handleParamChange = (setter: (val: string) => void, val: string) => {
    setter(val);
    setPreview(null);
    setPreviewError(null);
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const selected = e.target.files?.[0] || null;
    setFile(selected);
    setPreview(null);
    setPreviewError(null);
  };

  const buildFormData = (): FormData | null => {
    if (!file || !roundId.trim()) return null;
    const form = new FormData();
    form.append('file', file);
    form.append('round', roundId.trim());
    if (displayName.trim()) form.append('display', displayName.trim());
    if (labels.trim()) form.append('labels', labels.trim());
    if (roomLayout.trim()) form.append('room_layout', roomLayout.trim());
    if (customId.trim()) form.append('custom_id', customId.trim());
    return form;
  };

  const handlePreview = async () => {
    const form = buildFormData();
    if (!form) return;

    try {
      setIsPreviewing(true);
      setPreviewError(null);
      const res = await roundsApi.previewImport(form);
      setPreview(res);
    } catch (err: unknown) {
      const msg = (err as Error).message || 'Failed to parse upload file';
      setPreviewError(msg);
      setPreview(null);
    } finally {
      setIsPreviewing(false);
    }
  };

  const handleConfirmImport = async () => {
    const form = buildFormData();
    if (!form || !preview) return;

    try {
      setIsImporting(true);
      const res = await roundsApi.importFile(form);
      toast.success(`Successfully imported ${res.seats} seats into round ${res.round} ${LIVE_SOON}`);
      onImportComplete();
      onClose();
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Import failed');
    } finally {
      setIsImporting(false);
    }
  };

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title="Import Seating File"
      description="Upload an official exam timetable or custom enrollment file (.xlsx, .xls, .pdf, .json)."
      maxWidth="4xl"
      footer={
        <div className="flex items-center justify-between w-full">
          <div className="text-xs text-slate-500">
            {preview ? (
              <span className="text-teal-600 dark:text-teal-400 font-medium">
                ✓ Ready to import {preview.seats} seats
              </span>
            ) : (
              <span>Step 1: Configure & Preview file before importing</span>
            )}
          </div>
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="sm" onClick={onClose} disabled={isImporting}>
              Cancel
            </Button>
            {!preview ? (
              <Button
                variant="primary"
                size="sm"
                onClick={handlePreview}
                loading={isPreviewing}
                disabled={!file || !roundId.trim()}
              >
                Inspect & Preview
                <ArrowRight className="w-3.5 h-3.5" />
              </Button>
            ) : (
              <Button
                variant="primary"
                size="sm"
                onClick={handleConfirmImport}
                loading={isImporting}
              >
                <CheckCircle2 className="w-4 h-4" />
                Confirm & Import
              </Button>
            )}
          </div>
        </div>
      }
    >
      <div className="flex flex-col gap-6">
        {/* Step 1: Upload and Parameters */}
        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5 text-left">
            <label className="text-xs font-semibold text-slate-700 dark:text-slate-300">
              Exam Seating File (.xlsx, .xls, .pdf, .json)
            </label>
            <div className="flex items-center gap-3">
              <label className="flex items-center gap-2 px-3 py-2 border border-slate-300 dark:border-slate-700 hover:border-faculty rounded-lg cursor-pointer bg-white dark:bg-slate-900 text-xs font-medium text-slate-700 dark:text-slate-200 transition-colors">
                <Upload className="w-4 h-4 text-faculty dark:text-blue-300" />
                <span>{file ? file.name : 'Choose File'}</span>
                <input
                  type="file"
                  accept=".xlsx,.xls,.pdf,.json"
                  onChange={handleFileChange}
                  className="hidden"
                />
              </label>
              {file && (
                <span className="text-xs text-slate-500 font-mono">
                  {(file.size / 1024).toFixed(1)} KB
                </span>
              )}
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <Input
              label="Round ID *"
              placeholder="e.g. mid_1_2569"
              value={roundId}
              onChange={(e) => handleParamChange(setRoundId, e.target.value)}
              helperText="Safe identifier (letters, digits, _ and -)"
            />
            <Input
              label="Display Label (optional)"
              placeholder="e.g. กลางภาค 1/2569"
              value={displayName}
              onChange={(e) => handleParamChange(setDisplayName, e.target.value)}
              helperText="Derived automatically from Round ID if omitted"
            />
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
            <Input
              label="Labels (optional)"
              placeholder="e.g. นอกตาราง"
              value={labels}
              onChange={(e) => handleParamChange(setLabels, e.target.value)}
              helperText="Comma-separated labels"
            />
            <Input
              label="Room Layout Override (optional)"
              placeholder="e.g. SC1101"
              value={roomLayout}
              onChange={(e) => handleParamChange(setRoomLayout, e.target.value)}
            />
            <Input
              label="Custom Dataset ID (optional)"
              placeholder="e.g. LAB_A"
              value={customId}
              onChange={(e) => handleParamChange(setCustomId, e.target.value)}
              helperText="For out-of-schedule datasets"
            />
          </div>
        </div>

        {/* Error message */}
        {previewError && (
          <div className="flex items-start gap-2 p-3 bg-rose-50 dark:bg-rose-950/60 border border-rose-200 dark:border-rose-800 rounded-lg text-xs text-rose-800 dark:text-rose-200">
            <AlertTriangle className="w-4 h-4 text-rose-600 shrink-0 mt-0.5" />
            <div>
              <span className="font-semibold">Inspection Failed:</span>
              <p className="mt-0.5 font-mono text-[11px]">{previewError}</p>
            </div>
          </div>
        )}

        {/* Step 2: Preview Results */}
        {preview && (
          <div className="flex flex-col gap-4 border-t border-slate-200 dark:border-slate-800 pt-4 animate-in fade-in duration-150">
            <div className="flex items-center gap-2">
              <FileText className="w-4 h-4 text-faculty dark:text-blue-300" />
              <h3 className="text-xs font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300">
                File Preview & Impact Analysis
              </h3>
            </div>

            {/* Counts Grid */}
            <div className="grid grid-cols-2 md:grid-cols-5 gap-2">
              <div className="p-2.5 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700">
                <span className="text-[10px] uppercase font-bold text-slate-400">Total Seats</span>
                <div className="text-lg font-bold text-slate-900 dark:text-slate-100">
                  {preview.seats}
                </div>
              </div>
              <div className="p-2.5 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700">
                <span className="text-[10px] uppercase font-bold text-slate-400">Students</span>
                <div className="text-lg font-bold text-slate-900 dark:text-slate-100">
                  {preview.students}
                </div>
              </div>
              <div className="p-2.5 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700">
                <span className="text-[10px] uppercase font-bold text-slate-400">Sessions</span>
                <div className="text-lg font-bold text-slate-900 dark:text-slate-100">
                  {preview.sessions}
                </div>
              </div>
              <div className="p-2.5 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700">
                <span className="text-[10px] uppercase font-bold text-slate-400">Rooms</span>
                <div className="text-lg font-bold text-slate-900 dark:text-slate-100">
                  {preview.rooms.length}
                </div>
              </div>
              <div className="p-2.5 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700">
                <span className="text-[10px] uppercase font-bold text-slate-400">Exam Dates</span>
                <div className="text-lg font-bold text-slate-900 dark:text-slate-100">
                  {preview.dates.length}
                </div>
              </div>
            </div>

            {/* Replaces Box */}
            <div className="p-3 bg-amber-50/70 dark:bg-amber-950/40 border border-amber-200 dark:border-amber-800 rounded-lg flex flex-col gap-1.5 text-xs">
              <span className="font-bold text-amber-900 dark:text-amber-200">
                Target Data Replacement:
              </span>
              {preview.replaces && preview.replaces.length > 0 ? (
                preview.replaces.map((r, idx) => (
                  <div key={idx} className="flex items-center gap-1.5 text-amber-800 dark:text-amber-300">
                    <span className="font-semibold">•</span>
                    {r.type === 'in_schedule' ? (
                      <span>
                        Will replace official in-schedule seats of round <strong>{r.id}</strong> (currently{' '}
                        <strong>{r.seats}</strong> seats stored).
                      </span>
                    ) : (
                      <span>
                        Will replace custom dataset <strong>{r.id}</strong> (currently{' '}
                        <strong>{r.seats}</strong> seats stored).
                      </span>
                    )}
                  </div>
                ))
              ) : (
                <span className="text-amber-700 dark:text-amber-400">
                  Will add new seats into round {roundId}.
                </span>
              )}
            </div>

            {/* First 20 Sample Rows */}
            <div className="flex flex-col gap-1.5">
              <span className="text-xs font-semibold text-slate-700 dark:text-slate-300">
                Sample Data (first {preview.rows.length} rows):
              </span>
              <div className="max-h-48 overflow-y-auto border border-slate-200 dark:border-slate-800 rounded-lg">
                <Table className="text-xs">
                  <Thead>
                    <Tr>
                      <Th>Student ID</Th>
                      <Th>Room</Th>
                      <Th>Seat</Th>
                      <Th>Subject</Th>
                      <Th>Date</Th>
                      <Th>Time</Th>
                    </Tr>
                  </Thead>
                  <Tbody>
                    {preview.rows.map((row, idx) => (
                      <Tr key={idx}>
                        <Td className="font-mono">{row.student_id || '-'}</Td>
                        <Td>{row.room || '-'}</Td>
                        <Td className="font-bold">{row.seat || '-'}</Td>
                        <Td>{row.subject || '-'}</Td>
                        <Td>{row.date || '-'}</Td>
                        <Td>{row.time || '-'}</Td>
                      </Tr>
                    ))}
                  </Tbody>
                </Table>
              </div>
            </div>
          </div>
        )}
      </div>
    </Dialog>
  );
};
