import { describe, it, expect } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { LayoutCanvas } from '../LayoutCanvas';
import type { RoomLayout } from '../../../types';

const layout: RoomLayout = {
  layout: [{ type: 'column', items: [{ type: 'seats', char: 'A', count: 3 }] }],
};

function renderCanvas() {
  const { container } = render(
    <LayoutCanvas layout={layout} selection={null} onSelect={() => {}} onMoveBlock={() => {}} onMoveItem={() => {}} />
  );
  const viewport = container.querySelector('.cursor-grab') as HTMLElement;
  const content = viewport.firstElementChild as HTMLElement;
  return { viewport, content };
}

describe('LayoutCanvas view', () => {
  it('zooms with the wheel', () => {
    const { viewport } = renderCanvas();
    expect(screen.getByText('100%')).toBeTruthy();
    fireEvent.wheel(viewport, { deltaY: -100 });
    expect(screen.queryByText('100%')).toBeNull();
    fireEvent.wheel(viewport, { deltaY: 5000 });
    expect(screen.getByText('20%')).toBeTruthy();
  });

  it('moves by dragging the background', () => {
    const { viewport, content } = renderCanvas();
    fireEvent.mouseDown(viewport, { clientX: 100, clientY: 100 });
    fireEvent.mouseMove(viewport, { clientX: 70, clientY: 50 });
    fireEvent.mouseUp(viewport);
    expect(content.style.transform).toBe('translate(-30px, -50px) scale(1)');
  });

  it('zooms with the buttons', () => {
    renderCanvas();
    fireEvent.click(screen.getByTitle('Zoom In'));
    expect(screen.getByText('120%')).toBeTruthy();
    fireEvent.click(screen.getByTitle('Reset View (100%)'));
    expect(screen.getByText('100%')).toBeTruthy();
  });
});
