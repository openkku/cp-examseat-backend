import React from 'react';

interface TabBarProps {
  activeTab: 'meta' | 'layout';
  onChangeTab: (tab: 'meta' | 'layout') => void;
}

export const TabBar: React.FC<TabBarProps> = ({ activeTab, onChangeTab }) => {
  return (
    <div className="tab-bar">
      <button
        className={`tab ${activeTab === 'meta' ? 'active' : ''}`}
        onClick={() => onChangeTab('meta')}
      >
        Properties
      </button>
      <button
        className={`tab ${activeTab === 'layout' ? 'active' : ''}`}
        onClick={() => onChangeTab('layout')}
      >
        Layout Editor
      </button>
    </div>
  );
};
