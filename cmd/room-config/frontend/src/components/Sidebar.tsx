import React, { useMemo } from 'react';
import type { RoomConfigMap } from '../types';

interface SidebarProps {
  rooms: RoomConfigMap;
  selectedId: string;
  onSelectId: (id: string) => void;
  search: string;
  onSearchChange: (search: string) => void;
  onAddClick: () => void;
}

export const Sidebar: React.FC<SidebarProps> = ({
  rooms,
  selectedId,
  onSelectId,
  search,
  onSearchChange,
  onAddClick,
}) => {
  const filteredRooms = useMemo(() => {
    return Object.keys(rooms)
      .filter((id) => id.toLowerCase().includes(search.toLowerCase()))
      .sort();
  }, [rooms, search]);

  return (
    <aside className="sidebar" onClick={(e) => e.stopPropagation()}>
      <div className="sidebar-brand">
        <div className="brand-icon">CP</div>
        <div className="brand-title">Room Builder</div>
      </div>

      <div className="sidebar-search">
        <input
          className="search-box"
          placeholder="Search rooms..."
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
        />
      </div>

      <div className="sidebar-actions">
        <button
          className="btn btn-primary"
          style={{ width: '100%', justifyContent: 'center' }}
          onClick={onAddClick}
        >
          + Add Room
        </button>
      </div>

      <div className="room-list">
        {filteredRooms.map((id) => (
          <div
            key={id}
            className={`room-item ${id === selectedId ? 'active' : ''}`}
            onClick={() => onSelectId(id)}
          >
            <span className="room-name">{id}</span>
            <span className="room-file">{rooms[id].layout_file}</span>
          </div>
        ))}
        {filteredRooms.length === 0 && (
          <div className="empty" style={{ padding: '1rem' }}>
            <span style={{ fontSize: '1.2rem' }}>🔍</span>
            <p>No rooms matched</p>
          </div>
        )}
      </div>
    </aside>
  );
};
