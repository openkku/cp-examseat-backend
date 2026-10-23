import React, { useState } from 'react';
import { Plus, Trash2, Save, ExternalLink } from 'lucide-react';
import type { RoomMeta } from '../types';
import { Button } from '../ui/Button';
import { Input } from '../ui/Input';
import { Textarea } from '../ui/Textarea';

export interface DetailsFormProps {
  roomId: string;
  meta: RoomMeta;
  onChange: (meta: RoomMeta) => void;
  onSave: () => void;
  isSaving: boolean;
  isDirty: boolean;
}

export const DetailsForm: React.FC<DetailsFormProps> = ({
  roomId,
  meta,
  onChange,
  onSave,
  isSaving,
  isDirty,
}) => {
  const [newImageInput, setNewImageInput] = useState('');

  const addImage = () => {
    const trimmed = newImageInput.trim();
    if (trimmed) {
      onChange({
        ...meta,
        images: [...(meta.images || []), trimmed],
      });
      setNewImageInput('');
    }
  };

  const removeImage = (index: number) => {
    onChange({
      ...meta,
      images: (meta.images || []).filter((_, i) => i !== index),
    });
  };

  return (
    <div className="flex flex-col gap-6 p-6 max-w-3xl bg-white dark:bg-slate-900 border border-slate-200/80 dark:border-slate-800/90 rounded-2xl shadow-sm shadow-teal-950/[0.025] dark:shadow-none">
      <div className="flex items-center justify-between border-b border-slate-200 dark:border-slate-800 pb-4">
        <div>
          <h2 className="text-base font-extrabold text-slate-800 dark:text-slate-100">
            Room Metadata: {roomId}
          </h2>
          <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
            Configure layout files, images, campus map links, and metadata stored in room/metadata.json.
          </p>
        </div>
        <Button
          variant="primary"
          size="sm"
          onClick={onSave}
          loading={isSaving}
          disabled={!isDirty}
        >
          <Save className="w-4 h-4" />
          Save Details
        </Button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Input
          label="Layout File"
          value={meta.layout_file || ''}
          placeholder="e.g. SC1101.json"
          onChange={(e) => onChange({ ...meta, layout_file: e.target.value })}
          helperText="Referenced JSON in data/room/map/"
        />
        <Input
          label="Layout Image Web Path"
          value={meta.layout_image || ''}
          placeholder="/room/image/layout/SC.1101.webp"
          onChange={(e) => onChange({ ...meta, layout_image: e.target.value })}
          helperText="Web image URL or local /room/image/ path"
        />
      </div>

      <div className="flex flex-col gap-1">
        <div className="flex items-center justify-between">
          <label className="text-xs font-semibold text-slate-700 dark:text-slate-300">
            Map URL
          </label>
          {meta.map_url && (
            <a
              href={meta.map_url}
              target="_blank"
              rel="noopener noreferrer"
              className="text-xs text-faculty hover:text-faculty-dark dark:text-blue-400 inline-flex items-center gap-1"
            >
              Open map <ExternalLink className="w-3 h-3" />
            </a>
          )}
        </div>
        <Input
          value={meta.map_url || ''}
          placeholder="https://maps.app.goo.gl/..."
          onChange={(e) => onChange({ ...meta, map_url: e.target.value })}
        />
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Input
          label="Display Title (optional)"
          value={meta.title || ''}
          placeholder="e.g. SC.1101 (Computer Lab 1)"
          onChange={(e) => onChange({ ...meta, title: e.target.value })}
        />
        <div className="grid grid-cols-2 gap-2">
          <Input
            label="Latitude (optional)"
            type="number"
            step="any"
            value={meta.lat !== undefined && meta.lat !== null ? meta.lat : ''}
            placeholder="16.474..."
            onChange={(e) => {
              const val = e.target.value === '' ? undefined : parseFloat(e.target.value);
              onChange({ ...meta, lat: val });
            }}
          />
          <Input
            label="Longitude (optional)"
            type="number"
            step="any"
            value={meta.lng !== undefined && meta.lng !== null ? meta.lng : ''}
            placeholder="102.82..."
            onChange={(e) => {
              const val = e.target.value === '' ? undefined : parseFloat(e.target.value);
              onChange({ ...meta, lng: val });
            }}
          />
        </div>
      </div>

      <Textarea
        label="Description (optional)"
        rows={3}
        value={meta.description || ''}
        placeholder="Additional details about equipment, projector, entrance..."
        onChange={(e) => onChange({ ...meta, description: e.target.value })}
      />

      {/* Room Pictures List Editor */}
      <div className="flex flex-col gap-2 pt-2 border-t border-slate-200 dark:border-slate-800">
        <label className="text-xs font-semibold text-slate-700 dark:text-slate-300">
          Room Pictures (images list)
        </label>
        <div className="flex gap-2">
          <Input
            value={newImageInput}
            placeholder="/room/image/pic/SC.1101/_1.webp"
            onChange={(e) => setNewImageInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault();
                addImage();
              }
            }}
          />
          <Button variant="secondary" size="md" onClick={addImage} className="shrink-0">
            <Plus className="w-4 h-4" />
            Add Image
          </Button>
        </div>

        {(meta.images || []).length > 0 ? (
          <div className="flex flex-col gap-1.5 mt-2">
            {(meta.images || []).map((img, idx) => (
              <div
                key={`${img}-${idx}`}
                className="flex items-center justify-between p-2 rounded-lg bg-slate-50 dark:bg-slate-800/60 border border-slate-200 dark:border-slate-700 text-xs"
              >
                <span className="font-mono text-slate-700 dark:text-slate-300 truncate mr-2">
                  {img}
                </span>
                <button
                  onClick={() => removeImage(idx)}
                  className="text-slate-400 hover:text-rose-600 p-1 rounded"
                  title="Remove image"
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
            ))}
          </div>
        ) : (
          <p className="text-xs text-slate-400 dark:text-slate-500 italic mt-1">
            No secondary pictures configured.
          </p>
        )}
      </div>
    </div>
  );
};
