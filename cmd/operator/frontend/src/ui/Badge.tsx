import React from 'react';

export interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  variant?: 'neutral' | 'success' | 'warning' | 'error' | 'info';
  size?: 'sm' | 'md';
}

export const Badge: React.FC<BadgeProps> = ({
  variant = 'neutral',
  size = 'md',
  children,
  className = '',
  ...props
}) => {
  const variants = {
    neutral: 'bg-slate-100 dark:bg-slate-800 border-slate-200 dark:border-slate-700 text-slate-750 dark:text-slate-200',
    success: 'bg-teal-50 dark:bg-teal-950/80 border-teal-200 dark:border-teal-800 text-teal-800 dark:text-teal-200',
    warning: 'bg-amber-50 dark:bg-amber-950/80 border-amber-200 dark:border-amber-800 text-amber-800 dark:text-amber-300',
    error: 'bg-rose-50 dark:bg-rose-950/80 border-rose-200 dark:border-rose-800 text-rose-800 dark:text-rose-300',
    info: 'bg-faculty-light dark:bg-blue-950/80 border-blue-200 dark:border-blue-800 text-faculty dark:text-blue-200',
  };

  const sizes = {
    sm: 'text-[10px] px-2 py-0.5',
    md: 'text-xs px-2.5 py-1',
  };

  return (
    <span
      className={`inline-flex items-center font-bold rounded-full border whitespace-nowrap leading-none shadow-xs ${variants[variant]} ${sizes[size]} ${className}`}
      {...props}
    >
      {children}
    </span>
  );
};
