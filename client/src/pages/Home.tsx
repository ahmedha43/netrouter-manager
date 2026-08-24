/**
 * Design reminder — “Operations Station” is a compact native-style router GUI.
 * Prioritize MDI workflow, slim controls, dense readable data, and minimal motion.
 */
import { MdiWindow } from "@/components/netrouter/MdiWindow";
import { StatusIndicator } from "@/components/netrouter/StatusIndicator";
import { WindowContent } from "@/components/netrouter/WindowContent";
import { connectionError } from "@/lib/netrouter-helpers";
import { formatUptime, initialMetrics, interfaceRows, jitterMetric, neighbors, type Metrics, type WindowKind, windowMetadata } from "@/lib/netrouter-model";
import {
  Activity,
  AlertTriangle,
  ArrowLeft,
  ArrowRight,
  Cable,
  ChevronDown,
  CircleDot,
  Clock3,
  Cpu,
  File,
  Files,
  Folder,
  Gauge,
  HardDrive,
  LayoutDashboard,
  ListTree,
  LockKeyhole,
  LogOut,
  Menu,
  MonitorCog,
  Network,
  PanelLeft,
  Play,
  Plug,
  Power,
  Radio,
  RefreshCw,
  RotateCcw,
  Router,
  Save,
  Search,
  Settings,
  ShieldCheck,
  SlidersHorizontal,
  Terminal,
  Upload,
  Users,
  X,
  type LucideIcon,
} from "lucide-react";
import { type FormEvent, useEffect, useMemo, useState } from "react";
import { toast } from "sonner";

type ManagedWindow = {
  id: string;
  kind: WindowKind;
  position: [number, number];
  size: [number, number];
  zIndex: number;
  minimized: boolean;
  maximized: boolean;
};

type ConfirmState = "reboot" | "factory" | null;

const initialWindows: ManagedWindow[] = [
  { id: "dashboard-1", kind: "dashboard", position: [18, 16], size: [510, 470], zIndex: 2, minimized: false, maximized: false },
  { id: "interfaces-1", kind: "interfaces", position: [116, 74], size: [760, 410], zIndex: 3, minimized: false, maximized: false },
  { id: "traffic-1", kind: "traffic", position: [472, 52], size: [445, 430], zIndex: 1, minimized: false, maximized: false },
];

const navigationItems: { label: string; kind?: WindowKind; icon: LucideIcon; divider?: boolean; action?: "reboot" | "exit" }[] = [
  { label: "Quick Set", kind: "quickset", icon: SlidersHorizontal },
  { label: "Interfaces", kind: "interfaces", icon: Cable },
  { label: "WAN", kind: "wan", icon: Network },
  { label: "LAN", kind: "lan", icon: Router },
  { label: "DHCP Server", kind: "dhcp", icon: Users },
  { label: "Firewall", icon: ShieldCheck },
  { label: "Traffic Monitor", kind: "traffic", icon: Activity },
  { label: "System", kind: "system", icon: MonitorCog, divider: true },
  { label: "Tools", kind: "neighbors", icon: Gauge },
  { label: "Files", kind: "files", icon: Folder },
  { label: "Log", kind: "log", icon: ListTree },
  { label: "New Terminal", kind: "terminal", icon: Terminal },
  { label: "Reboot", icon: Power, divider: true, action: "reboot" },
  { label: "Exit", icon: LogOut, action: "exit" },
];

const menus = {
  Session: ["New", "Open", "Save", "Save As", "Autosave", "Close All Windows", "Disconnect", "Exit"],
  Settings: ["Settings", "Hide Passwords", "Inline Comments", "Appearance"],
  Dashboard: ["Add CPU", "Add Memory", "Add Uptime", "Add Date", "Add Time"],
};

function ToolbarButton({ label, children, onClick, active = false }: { label: string; children: React.ReactNode; onClick?: () => void; active?: boolean }) {
  return <button type="button" className={`nr-toolbar-button ${active ? "is-active" : ""}`} title={label} aria-label={label} onClick={onClick}>{children}</button>;
}

function MetricChip({ icon: Icon, label, value }: { icon: LucideIcon; label: string; value: string }) {
  return <div className="nr-metric-chip" title={`${label}: ${value}`}><Icon size={14} /><span>{label}</span><strong>{value}</strong></div>;
}

export default function Home() {
  const [windows, setWindows] = useState<ManagedWindow[]>(initialWindows);
  const [activeWindow, setActiveWindow] = useState("interfaces-1");
  const [metrics, setMetrics] = useState<Metrics>(initialMetrics);
  const [menu, setMenu] = useState<keyof typeof menus | null>(null);
  const [connectionOpen, setConnectionOpen] = useState(true);
  const [connectionTarget, setConnectionTarget] = useState("192.168.88.1");
  const [username, setUsername] = useState("admin");
  const [password, setPassword] = useState("");
  const [connectionTab, setConnectionTab] = useState<"Managed Routers" | "Neighbors">("Neighbors");
  const [confirm, setConfirm] = useState<ConfirmState>(null);
  const [errorDialog, setErrorDialog] = useState<string | null>(null);
  const [safeMode, setSafeMode] = useState(false);

  useEffect(() => {
    const timer = window.setInterval(() => {
      setMetrics((current) => ({
        cpu: jitterMetric(current.cpu, 6, 4, 96),
        memory: jitterMetric(current.memory, 2, 18, 84),
        wanRx: jitterMetric(current.wanRx, 15, 4, 920),
        wanTx: jitterMetric(current.wanTx, 7, 1, 640),
        lanRx: jitterMetric(current.lanRx, 12, 4, 920),
        lanTx: jitterMetric(current.lanTx, 7, 1, 640),
        secondsOnline: current.secondsOnline + 2,
      }));
    }, 2000);
    return () => window.clearInterval(timer);
  }, []);

  const minimizedWindows = useMemo(() => windows.filter((item) => item.minimized), [windows]);

  function focusWindow(id: string) {
    setActiveWindow(id);
    setWindows((items) => {
      const highest = Math.max(...items.map((item) => item.zIndex), 1);
      return items.map((item) => item.id === id ? { ...item, zIndex: highest + 1, minimized: false } : item);
    });
  }

  function openWindow(kind: WindowKind) {
    const existing = windows.find((item) => item.kind === kind);
    if (existing) { focusWindow(existing.id); return; }
    const metadata = windowMetadata[kind];
    const id = `${kind}-${Date.now()}`;
    const highest = Math.max(...windows.map((item) => item.zIndex), 1);
    setWindows((items) => [...items, { id, kind, position: metadata.defaultPosition, size: metadata.defaultSize, zIndex: highest + 1, minimized: false, maximized: false }]);
    setActiveWindow(id);
  }

  function closeWindow(id: string) {
    setWindows((items) => items.filter((item) => item.id !== id));
  }

  function toggleMinimize(id: string) {
    setWindows((items) => items.map((item) => item.id === id ? { ...item, minimized: !item.minimized } : item));
  }

  function toggleMaximize(id: string) {
    setWindows((items) => items.map((item) => item.id === id ? { ...item, maximized: !item.maximized, minimized: false } : item));
    setActiveWindow(id);
  }

  function moveWindow(id: string, position: [number, number]) {
    setWindows((items) => items.map((item) => item.id === id ? { ...item, position } : item));
  }

  function invokeMenu(item: string) {
    setMenu(null);
    if (item === "Close All Windows") { setWindows([]); toast.success("All child windows closed"); return; }
    if (item === "Disconnect") { setConnectionOpen(true); return; }
    if (item === "Settings") { openWindow("settings"); return; }
    if (item === "Exit") { toast("Exit is unavailable in the browser prototype."); return; }
    if (item === "New") { openWindow("quickset"); return; }
    if (item === "Save" || item === "Save As") { toast.success("Session configuration saved locally"); return; }
    toast(`${item} selected`);
  }

  function submitConnection(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const error = connectionError(connectionTarget, username);
    if (error) { setErrorDialog(error); return; }
    setConnectionOpen(false);
    toast.success("Connected to edge-hq-01", { description: "Secure session established over TLS." });
  }

  function resolveConfirm() {
    if (confirm === "reboot") toast.success("Reboot request queued", { description: "The router would restart after configuration is persisted." });
    if (confirm === "factory") toast.warning("Factory reset request queued", { description: "This is a non-destructive browser demonstration." });
    setConfirm(null);
  }

  return (
    <div className="nr-app" onClick={() => menu && setMenu(null)}>
      <header className="nr-titlebar">
        <div className="nr-brand"><img src="/manus-storage/netrouter-mark_06493364.png" alt="NetRouter Manager mark" /><span className="nr-brand-word"><b>NETROUTER</b><em>MANAGER</em></span><small>NetRouter OS</small></div>
        <div className="nr-titlebar-state"><StatusIndicator state="CONNECTED" compact /><span>edge-hq-01</span></div>
        <div className="nr-window-chrome"><button type="button" aria-label="Minimize application">—</button><button type="button" aria-label="Maximize application">□</button><button type="button" aria-label="Close application"><X size={13} /></button></div>
      </header>
      <div className="nr-menubar" role="menubar">
        {(Object.keys(menus) as (keyof typeof menus)[]).map((name) => <div className="nr-menu-wrap" key={name} onClick={(event) => event.stopPropagation()}><button type="button" role="menuitem" className={menu === name ? "is-open" : ""} onClick={() => setMenu(menu === name ? null : name)}>{name}<ChevronDown size={11} /></button>{menu === name && <div className="nr-menu-popover" role="menu">{menus[name].map((item) => <button key={item} type="button" role="menuitem" onClick={() => invokeMenu(item)}>{item}</button>)}</div>}</div>)}
      </div>
      <div className="nr-toolbar">
        <div className="nr-toolbar-actions"><ToolbarButton label="Back"><ArrowLeft size={15} /></ToolbarButton><ToolbarButton label="Forward"><ArrowRight size={15} /></ToolbarButton><span className="nr-toolbar-separator" /><ToolbarButton label="Save session" onClick={() => toast.success("Session saved")}><Save size={15} /></ToolbarButton><ToolbarButton label="Refresh data" onClick={() => toast.success("Live data refreshed")}><RefreshCw size={15} /></ToolbarButton><ToolbarButton label="Safe Mode" active={safeMode} onClick={() => setSafeMode((value) => !value)}><ShieldCheck size={15} /></ToolbarButton></div>
        <div className="nr-toolbar-metrics"><button type="button" className="nr-session-button" onClick={() => setConnectionOpen(true)}><img src="/manus-storage/netrouter-mark_06493364.png" alt="" /><Plug size={14} /><span>Session</span><StatusIndicator state="CONNECTED" compact /></button><MetricChip icon={Cpu} label="CPU" value={`${metrics.cpu.toFixed(0)}%`} /><MetricChip icon={HardDrive} label="Memory" value={`${metrics.memory.toFixed(0)}%`} /><MetricChip icon={Clock3} label="Uptime" value={formatUptime(metrics.secondsOnline)} /><span className="nr-secure"><LockKeyhole size={13} />TLS</span></div>
      </div>
      <div className="nr-shell">
        <aside className="nr-sidebar" aria-label="Router navigation">
          <div className="nr-nav-header"><PanelLeft size={14} />Navigation</div>
          <div className="nr-nav-list">{navigationItems.map((item) => { const Icon = item.icon; return <div className={item.divider ? "nr-nav-divider" : ""} key={item.label}><button type="button" onClick={() => { if (item.kind) openWindow(item.kind); if (item.action === "reboot") setConfirm("reboot"); if (item.action === "exit") toast("Exit is unavailable in the browser prototype."); }}><Icon size={15} /><span>{item.label}</span>{item.label === "Tools" && <ChevronDown size={12} className="nr-nav-end" />}</button></div>; })}</div>
          <div className="nr-sidebar-footer"><CircleDot size={12} /><span>Router reachable</span></div>
        </aside>
        <main className="nr-mdi-workspace" aria-label="MDI workspace">
          <div className="nr-workspace-label"><Router size={13} />Workspace · drag child windows by title bar</div>
          <div className="nr-workspace-mark" aria-hidden="true"><img src="/manus-storage/netrouter-mark_06493364.png" alt="" /><span>NETROUTER<br />OPERATIONS</span></div>
          {windows.map((windowState) => <MdiWindow key={windowState.id} {...windowState} title={windowMetadata[windowState.kind].title} active={activeWindow === windowState.id} onFocus={focusWindow} onClose={closeWindow} onMinimize={toggleMinimize} onMaximize={toggleMaximize} onMove={moveWindow}><WindowContent kind={windowState.kind} metrics={metrics} onOpen={openWindow} onRequestConfirm={setConfirm} /></MdiWindow>)}
          {windows.length === 0 && <div className="nr-empty-workspace"><Router size={28} /><strong>No child windows open</strong><span>Use the navigation tree or Session → New to open a configuration surface.</span></div>}
        </main>
      </div>
      <footer className="nr-statusbar"><div><StatusIndicator state="CONNECTED" compact /><span>Connected via 192.168.88.1</span><span>Safe Mode: {safeMode ? "ON" : "OFF"}</span></div><div>{minimizedWindows.map((item) => <button key={item.id} type="button" onClick={() => focusWindow(item.id)}>{windowMetadata[item.kind].title}</button>)}<span>3 neighbors · 14 DHCP leases · {interfaceRows.length} interfaces</span></div></footer>

      {connectionOpen && <div className="nr-dialog-layer" role="presentation"><section className="nr-connect-dialog" role="dialog" aria-modal="true" aria-labelledby="connection-title"><div className="nr-dialog-titlebar"><div><img src="/manus-storage/netrouter-mark_06493364.png" alt="" /><span id="connection-title">Connect to NetRouter</span></div><button type="button" aria-label="Close connection dialog" onClick={() => setConnectionOpen(false)}><X size={14} /></button></div><form onSubmit={submitConnection}><div className="nr-connect-main"><div className="nr-connect-fields"><label>Connect To<input value={connectionTarget} onChange={(event) => setConnectionTarget(event.target.value)} placeholder="MAC Address / IP Address" autoFocus /></label><label>Username<input value={username} onChange={(event) => setUsername(event.target.value)} /></label><label>Password<input type="password" value={password} onChange={(event) => setPassword(event.target.value)} /></label><div className="nr-check-row"><label><input type="checkbox" defaultChecked />Keep Password</label><label><input type="checkbox" />Open New Session</label></div><div className="nr-dialog-actions"><button type="button" className="nr-button" onClick={() => setConnectionOpen(false)}>Cancel</button><button type="submit" className="nr-button primary"><Plug size={13} />Connect</button></div></div><div className="nr-connect-surface" /></div><div className="nr-dialog-tabs">{(["Managed Routers", "Neighbors"] as const).map((tab) => <button key={tab} type="button" className={connectionTab === tab ? "is-active" : ""} onClick={() => setConnectionTab(tab)}>{tab}</button>)}</div><div className="nr-connect-table"><table className="nr-table"><thead><tr><th>Identity</th><th>MAC Address</th><th>IP Address</th><th>Architecture</th><th>Version</th><th>Uptime</th></tr></thead><tbody>{connectionTab === "Neighbors" ? neighbors.map((neighbor) => <tr key={neighbor.mac} onDoubleClick={() => setConnectionTarget(neighbor.ip)} title="Double-click to populate Connect To"><td>{neighbor.identity}</td><td>{neighbor.mac}</td><td>{neighbor.ip}</td><td>{neighbor.architecture}</td><td>{neighbor.version}</td><td>{neighbor.uptime}</td></tr>) : <tr><td>edge-hq-01</td><td>AA:BB:CC:DD:EE:01</td><td>192.168.88.1</td><td>arm64</td><td>NetRouter OS 1.4.2</td><td>12d 04:18:22</td></tr>}</tbody></table></div></form></section></div>}
      {confirm && <div className="nr-dialog-layer" role="presentation"><section className="nr-confirm-dialog" role="dialog" aria-modal="true" aria-labelledby="confirm-title"><div className="nr-dialog-titlebar"><div><AlertTriangle size={15} /><span id="confirm-title">{confirm === "reboot" ? "Reboot Router" : "Factory Reset Router"}</span></div><button type="button" aria-label="Close confirmation" onClick={() => setConfirm(null)}><X size={14} /></button></div><div className="nr-confirm-content"><img src="/manus-storage/netrouter-maintenance-glyph_8fbe55ea.png" alt="Maintenance warning" /><div><strong>{confirm === "reboot" ? "Are you sure you want to reboot the router?" : "Are you sure you want to factory reset the router?"}</strong><p>{confirm === "reboot" ? "Active router sessions will be interrupted." : "All router configuration would be permanently removed."}</p></div></div><div className="nr-dialog-actions"><button type="button" className="nr-button" onClick={() => setConfirm(null)}>Cancel</button><button type="button" className="nr-button danger" onClick={resolveConfirm}>{confirm === "reboot" ? "Reboot" : "Factory Reset"}</button></div></section></div>}
      {errorDialog && <div className="nr-dialog-layer" role="presentation"><section className="nr-error-dialog" role="dialog" aria-modal="true" aria-labelledby="error-title"><div className="nr-dialog-titlebar"><div><AlertTriangle size={15} /><span id="error-title">Connection Error</span></div><button type="button" aria-label="Close error dialog" onClick={() => setErrorDialog(null)}><X size={14} /></button></div><div className="nr-error-content"><AlertTriangle size={31} /><div><strong>{errorDialog}</strong><p>Unable to connect to router. Check the address and credentials, then try again.</p></div></div><div className="nr-dialog-actions"><button type="button" className="nr-button" onClick={() => toast("Connection details: browser prototype uses local validation")}>Details</button><button type="button" className="nr-button primary" onClick={() => setErrorDialog(null)}>OK</button></div></section></div>}
    </div>
  );
}
