import { useCallback, useSyncExternalStore } from 'react';

export type Theme = 'light' | 'dark';

const THEME_EVENT = 'themechange';

// The inline script in index.html applies the stored or system theme to
// <html> before first paint, so the DOM is the source of truth.
function getSnapshot(): Theme {
  return document.documentElement.classList.contains('dark') ? 'dark' : 'light';
}

function subscribe(onChange: () => void) {
  window.addEventListener(THEME_EVENT, onChange);
  return () => window.removeEventListener(THEME_EVENT, onChange);
}

export const useTheme = () => {
  const theme = useSyncExternalStore(subscribe, getSnapshot);

  const toggleTheme = useCallback(() => {
    const next: Theme = getSnapshot() === 'light' ? 'dark' : 'light';
    document.documentElement.classList.toggle('dark', next === 'dark');
    try {
      localStorage.setItem('theme', next);
    } catch {
      // Storage unavailable (private mode); the theme still applies for this page view.
    }
    window.dispatchEvent(new Event(THEME_EVENT));
  }, []);

  return { theme, toggleTheme, isDark: theme === 'dark' };
};
