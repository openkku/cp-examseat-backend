import React from 'react';
import type { RoomMeta } from '../types';

interface AddRoomModalProps {
  newId: string;
  onNewIdChange: (id: string) => void;
  newMeta: RoomMeta;
  onNewMetaChange: (meta: RoomMeta) => void;
  onClose: () => void;
  onSubmit: (e: React.FormEvent) => void;
}

export const AddRoomModal: React.FC<AddRoomModalProps> = ({
  newId,
  onNewIdChange,
  newMeta,
  onNewMetaChange,
  onClose,
  onSubmit,
}) => {
  return (
    <div className="overlay" onClick={onClose}>
      <form
        className="modal"
        onClick={(e) => e.stopPropagation()}
        onSubmit={onSubmit}
      >
        <div className="modal-head">
          <h3>Create Room</h3>
          <button type="button" className="icon-btn" onClick={onClose}>
            &times;
          </button>
        </div>
        <div className="modal-body">
          <div className="field">
            <span className="field-label">Room ID</span>
            <input
              required
              className="field-input"
              placeholder="e.g. CP.9999"
              value={newId}
              onChange={(e) => onNewIdChange(e.target.value)}
            />
          </div>
          <div className="field">
            <span className="field-label">Layout File (optional)</span>
            <input
              className="field-input mono"
              placeholder="e.g. CP9999.json"
              value={newMeta.layout_file}
              onChange={(e) =>
                onNewMetaChange({ ...newMeta, layout_file: e.target.value })
              }
            />
          </div>
          <div className="field">
            <span className="field-label">Maps Link (optional)</span>
            <input
              className="field-input"
              placeholder="https://maps.app.goo.gl/..."
              value={newMeta.map_url}
              onChange={(e) =>
                onNewMetaChange({ ...newMeta, map_url: e.target.value })
              }
            />
          </div>
          <div className="field">
            <span className="field-label">Layout Image (optional)</span>
            <input
              className="field-input"
              placeholder="/room/image/CP.9999.jpg"
              value={newMeta.layout_image}
              onChange={(e) =>
                onNewMetaChange({ ...newMeta, layout_image: e.target.value })
              }
            />
          </div>
        </div>
        <div className="modal-foot">
          <button type="button" className="btn btn-ghost" onClick={onClose}>
            Cancel
          </button>
          <button type="submit" className="btn btn-primary">
            Create
          </button>
        </div>
      </form>
    </div>
  );
};
