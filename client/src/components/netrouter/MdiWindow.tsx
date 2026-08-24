/**
 * Design reminder — child panes behave like restrained native MDI windows:
 * thin frames, small title bars, direct manipulation, and no oversized controls.
 */
import { Maximize2, Minimize2, Square, X } from "lucide-react";
import { type PointerEvent, type ReactNode, useRef } from "react";

export type MdiWindowProps = {
  id: string;
  title: string;
  active: boolean;
  minimized: boolean;
  maximized: boolean;
  zIndex: number;
  position: [number, number];
  size: [number, number];
  children: ReactNode;
  onFocus: (id: string) => void;
  onClose: (id: string) => void;
  onMinimize: (id: string) => void;
  onMaximize: (id: string) => void;
  onMove: (id: string, position: [number, number]) => void;
};

export function MdiWindow({
  id,
  title,
  active,
  minimized,
  maximized,
  zIndex,
  position,
  size,
  children,
  onFocus,
  onClose,
  onMinimize,
  onMaximize,
  onMove,
}: MdiWindowProps) {
  const dragStart = useRef<{ x: number; y: number; left: number; top: number } | null>(null);

  function startDrag(event: PointerEvent<HTMLDivElement>) {
    if ((event.target as HTMLElement).closest("button")) return;
    onFocus(id);
    dragStart.current = { x: event.clientX, y: event.clientY, left: position[0], top: position[1] };
    event.currentTarget.setPointerCapture(event.pointerId);
  }

  function moveDrag(event: PointerEvent<HTMLDivElement>) {
    if (!dragStart.current || maximized) return;
    const nextLeft = Math.max(-100, dragStart.current.left + event.clientX - dragStart.current.x);
    const nextTop = Math.max(0, dragStart.current.top + event.clientY - dragStart.current.y);
    onMove(id, [nextLeft, nextTop]);
  }

  function endDrag() {
    dragStart.current = null;
  }

  if (minimized) return null;

  return (
    <section
      className={`nr-mdi-window ${active ? "is-active" : ""} ${maximized ? "is-maximized" : ""}`}
      style={
        maximized
          ? { zIndex }
          : { left: position[0], top: position[1], width: size[0], height: size[1], zIndex }
      }
      onPointerDown={() => onFocus(id)}
      aria-label={`${title} child window`}
    >
      <div className="nr-window-titlebar" onPointerDown={startDrag} onPointerMove={moveDrag} onPointerUp={endDrag}>
        <span className="nr-window-title">{title}</span>
        <div className="nr-window-actions" aria-label={`${title} window actions`}>
          <button type="button" aria-label={`Minimize ${title}`} onClick={() => onMinimize(id)}><Minimize2 size={13} /></button>
          <button type="button" aria-label={`Maximize ${title}`} onClick={() => onMaximize(id)}>{maximized ? <Square size={11} /> : <Maximize2 size={12} />}</button>
          <button type="button" aria-label={`Close ${title}`} onClick={() => onClose(id)}><X size={13} /></button>
        </div>
      </div>
      <div className="nr-window-body">{children}</div>
    </section>
  );
}
