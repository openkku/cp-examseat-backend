import React from 'react';

interface ToolbarProps {
  selectedId: string;
  layoutFile: string;
  tab: 'meta' | 'layout';
  onDeleteClick: () => void;
  onSaveClick: () => void;
}

export const Toolbar: React.FC<ToolbarProps> = ({
  selectedId,
  layoutFile,
  tab,
  onDeleteClick,
  onSaveClick,
}) => {
  return (
    <div className="toolbar">
      <div className="toolbar-left">
        <span className="toolbar-title">{selectedId}</span>
        <span className="toolbar-sub">room/map/{layoutFile || '—'}</span>
      </div>
      <div className="toolbar-actions">
        <button className="btn btn-danger btn-sm" onClick={onDeleteClick}>
          Delete
        </button>
        <button className="btn btn-primary btn-sm" onClick={onSaveClick}>
          {tab === 'meta' ? 'Save Properties' : 'Save Layout'}
        </button>
      </div>
    </div>
  );
};
