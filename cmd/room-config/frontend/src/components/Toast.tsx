import React from 'react';
import type { ToastData } from '../types';

interface ToastProps {
  toast: ToastData;
}

export const Toast: React.FC<ToastProps> = ({ toast }) => {
  return (
    <div className={`toast ${toast.type}`}>
      {toast.type === 'success' ? '✅' : '❌'} {toast.message}
    </div>
  );
};
