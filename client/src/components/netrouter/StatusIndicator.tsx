/**
 * Design reminder — status must always use a compact textual label plus color.
 */
import type { ConnectionState } from "@/lib/netrouter-model";

const stateClass: Record<ConnectionState, string> = {
  CONNECTED: "is-connected",
  CONNECTING: "is-connecting",
  DISCONNECTED: "is-disconnected",
  ERROR: "is-error",
};

export function StatusIndicator({ state, compact = false }: { state: ConnectionState; compact?: boolean }) {
  return (
    <span className={`nr-status ${stateClass[state]} ${compact ? "is-compact" : ""}`}>
      <i aria-hidden="true" />
      <span>{state}</span>
    </span>
  );
}
