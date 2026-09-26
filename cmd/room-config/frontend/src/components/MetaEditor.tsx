import React from 'react';
import type { RoomMeta } from '../types';

interface MetaEditorProps {
  meta: RoomMeta;
  onChangeMeta: (meta: RoomMeta) => void;
}

export const MetaEditor: React.FC<MetaEditorProps> = ({ meta, onChangeMeta }) => {
  return (
    <div className="meta-pane">
      <div className="meta-grid">
        <div className="field">
          <span className="field-label">Layout Filename</span>
          <input
            className="field-input mono"
            placeholder="e.g. SC1102.json"
            value={meta.layout_file}
            onChange={(e) => onChangeMeta({ ...meta, layout_file: e.target.value })}
          />
        </div>
        <div className="field">
          <span className="field-label">Blueprint Image</span>
          <input
            className="field-input mono"
            placeholder="e.g. /room/image/SC.1102.jpg"
            value={meta.layout_image}
            onChange={(e) => onChangeMeta({ ...meta, layout_image: e.target.value })}
          />
        </div>
        <div className="field full-width">
          <span className="field-label">Google Maps Link</span>
          <input
            className="field-input"
            placeholder="e.g. https://maps.app.goo.gl/..."
            value={meta.map_url}
            onChange={(e) => onChangeMeta({ ...meta, map_url: e.target.value })}
          />
        </div>
        <div className="field full-width">
          <span className="field-label">Reference Images (comma-separated)</span>
          <textarea
            className="field-input"
            rows={3}
            placeholder="/room/image/photo1.jpg, /room/image/photo2.jpg"
            value={meta.images.join(', ')}
            onChange={(e) =>
              onChangeMeta({
                ...meta,
                images: e.target.value
                  .split(',')
                  .map((x) => x.trim())
                  .filter(Boolean),
              })
            }
          />
        </div>
      </div>
    </div>
  );
};
