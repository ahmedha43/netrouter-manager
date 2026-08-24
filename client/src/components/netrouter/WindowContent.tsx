/**
 * Design reminder — every configuration surface favors compact form rows,
 * technical tables, explicit state, and low-latency desktop interaction.
 */
import { StatusIndicator } from "@/components/netrouter/StatusIndicator";
import { TrafficGraph } from "@/components/netrouter/TrafficGraph";
import { interfaceRows, neighbors, type InterfaceRow, type Metrics, type WindowKind } from "@/lib/netrouter-model";
import {
  AlertTriangle,
  Check,
  ChevronDown,
  Copy,
  Download,
  File,
  Folder,
  Info,
  MoreHorizontal,
  Plus,
  RefreshCw,
  Search,
  Trash2,
  Upload,
} from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";

export type WindowContentProps = {
  kind: WindowKind;
  metrics: Metrics;
  onOpen: (kind: WindowKind) => void;
  onRequestConfirm: (kind: "reboot" | "factory") => void;
};

type SectionProps = { title: string; children: React.ReactNode; className?: string };

function Section({ title, children, className = "" }: SectionProps) {
  return <section className={`nr-section ${className}`}><div className="nr-section-title">{title}</div>{children}</section>;
}

function ActionButton({ children, tone = "default", onClick, icon }: { children: React.ReactNode; tone?: "default" | "primary" | "danger"; onClick?: () => void; icon?: React.ReactNode }) {
  return <button type="button" className={`nr-button ${tone}`} onClick={onClick}>{icon}{children}</button>;
}

function Field({ label, children, hint }: { label: string; children: React.ReactNode; hint?: string }) {
  return <label className="nr-field"><span>{label}</span><div>{children}</div>{hint && <small>{hint}</small>}</label>;
}

function ReadonlyRow({ label, value }: { label: string; value: React.ReactNode }) {
  return <div className="nr-readonly-row"><span>{label}</span><strong>{value}</strong></div>;
}

function Dashboard({ metrics }: { metrics: Metrics }) {
  return (
    <div className="nr-dashboard">
      <Section title="System">
        <div className="nr-mini-grid">
          <ReadonlyRow label="Identity" value="edge-hq-01" />
          <ReadonlyRow label="Architecture" value="arm64" />
          <ReadonlyRow label="Version" value="NetRouter OS 1.4.2" />
          <ReadonlyRow label="Uptime" value="12d 04:18:22" />
        </div>
      </Section>
      <div className="nr-dashboard-pair">
        <Section title="WAN">
          <ReadonlyRow label="Connection" value={<StatusIndicator state="CONNECTED" compact />} />
          <ReadonlyRow label="Type" value="DHCP Client" />
          <ReadonlyRow label="IP" value="100.64.18.22" />
          <ReadonlyRow label="Gateway" value="100.64.18.1" />
          <ReadonlyRow label="DNS" value="1.1.1.1, 9.9.9.9" />
        </Section>
        <Section title="LAN">
          <ReadonlyRow label="Gateway" value="192.168.88.1" />
          <ReadonlyRow label="Subnet" value="192.168.88.0/24" />
          <ReadonlyRow label="DHCP" value="Enabled" />
          <ReadonlyRow label="Clients" value="14 active" />
        </Section>
      </div>
      <div className="nr-dashboard-pair">
        <Section title="Resources">
          <div className="nr-meter"><span>CPU</span><div><i style={{ width: `${metrics.cpu}%` }} /></div><b>{metrics.cpu.toFixed(0)}%</b></div>
          <div className="nr-meter"><span>Memory</span><div><i style={{ width: `${metrics.memory}%` }} /></div><b>{metrics.memory.toFixed(0)}%</b></div>
        </Section>
        <Section title="Traffic">
          <ReadonlyRow label="WAN RX / TX" value={`${metrics.wanRx.toFixed(1)} / ${metrics.wanTx.toFixed(1)} Mbps`} />
          <ReadonlyRow label="LAN RX / TX" value={`${metrics.lanRx.toFixed(1)} / ${metrics.lanTx.toFixed(1)} Mbps`} />
        </Section>
      </div>
    </div>
  );
}

function Interfaces() {
  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState("ether1");
  const [sortKey, setSortKey] = useState<keyof InterfaceRow>("name");
  const [contextFor, setContextFor] = useState<string | null>(null);
  const filteredRows = useMemo(() => {
    const matched = interfaceRows.filter((row) => Object.values(row).join(" ").toLowerCase().includes(query.toLowerCase()));
    return [...matched].sort((a, b) => String(a[sortKey]).localeCompare(String(b[sortKey])));
  }, [query, sortKey]);
  const columns: { key: keyof InterfaceRow; label: string }[] = [
    { key: "name", label: "Name" }, { key: "role", label: "Role" }, { key: "type", label: "Type" },
    { key: "mac", label: "MAC Address" }, { key: "mtu", label: "MTU" }, { key: "ip", label: "IP Address" },
    { key: "rx", label: "RX" }, { key: "tx", label: "TX" }, { key: "status", label: "Status" },
  ];
  return (
    <div className="nr-fill-column">
      <div className="nr-commandbar">
        <ActionButton icon={<Plus size={13} />} onClick={() => toast("Interface editor opened", { description: "A new interface can be configured here." })}>Add</ActionButton>
        <ActionButton icon={<Trash2 size={13} />} tone="danger" onClick={() => toast("Remove requires confirmation")}>Remove</ActionButton>
        <ActionButton onClick={() => toast("Selected interface enabled")}>Enable</ActionButton>
        <ActionButton onClick={() => toast("Selected interface disabled")}>Disable</ActionButton>
        <ActionButton onClick={() => toast("Comment mode enabled")}>Comment</ActionButton>
        <ActionButton icon={<RefreshCw size={13} />} onClick={() => toast.success("Interface counters refreshed")}>Refresh</ActionButton>
        <div className="nr-search"><Search size={13} /><input aria-label="Search interfaces" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search" /></div>
      </div>
      <div className="nr-table-wrap" onClick={() => setContextFor(null)}>
        <table className="nr-table nr-interface-table">
          <thead><tr>{columns.map((column) => <th key={column.key} className="nr-resizable-head"><button type="button" onClick={() => setSortKey(column.key)}>{column.label}{sortKey === column.key && <ChevronDown size={12} />}</button></th>)}</tr></thead>
          <tbody>{filteredRows.map((row) => <tr key={row.name} className={selected === row.name ? "is-selected" : ""} onClick={() => setSelected(row.name)} onContextMenu={(event) => { event.preventDefault(); setSelected(row.name); setContextFor(row.name); }}>
            <td>{row.name}</td><td>{row.role}</td><td>{row.type}</td><td>{row.mac}</td><td>{row.mtu}</td><td>{row.ip}</td><td>{row.rx}</td><td>{row.tx}</td><td><StatusIndicator state={row.status} compact /></td>
          </tr>)}</tbody>
        </table>
        {contextFor && <div className="nr-context-menu" role="menu"><button type="button" onClick={() => toast.success(`${contextFor} enabled`)}>Enable</button><button type="button" onClick={() => toast(`${contextFor} disabled`)}>Disable</button><button type="button" onClick={() => toast("Properties opened")}>Properties</button><button type="button" onClick={() => toast("Comment editor opened")}>Comment</button><button type="button" onClick={() => toast.success("Counters reset")}>Reset Counters</button></div>}
      </div>
      <div className="nr-table-footer">{filteredRows.length} items | Selected: {selected}</div>
    </div>
  );
}

function Wan({ metrics }: { metrics: Metrics }) {
  const [tab, setTab] = useState<"General" | "Status">("General");
  const [wanType, setWanType] = useState("DHCP Client");
  return <div className="nr-fill-column">
    <div className="nr-tabs">{["General", "Status"].map((item) => <button key={item} type="button" className={tab === item ? "is-active" : ""} onClick={() => setTab(item as typeof tab)}>{item}</button>)}</div>
    {tab === "General" ? <div className="nr-form-grid">
      <Field label="WAN Type"><select value={wanType} onChange={(event) => setWanType(event.target.value)}><option>DHCP Client</option><option>Static IP</option><option>PPPoE</option></select></Field>
      <Field label="Interface"><select><option>ether1</option><option>pppoe-wan</option></select></Field>
      {wanType === "DHCP Client" && <><Field label="Hostname"><input defaultValue="edge-hq-01" /></Field><Field label="Client ID"><input defaultValue="netrouter-hq" /></Field><Field label="DNS"><select><option>Use peer DNS</option><option>Manual</option></select></Field><Field label="Default Route"><select><option>Yes</option><option>No</option></select></Field></>}
      {wanType === "Static IP" && <><Field label="IP Address"><input placeholder="100.64.18.22/24" /></Field><Field label="Netmask"><input placeholder="255.255.255.0" /></Field><Field label="Gateway"><input placeholder="100.64.18.1" /></Field><Field label="DNS"><input placeholder="1.1.1.1, 9.9.9.9" /></Field></>}
      {wanType === "PPPoE" && <><Field label="Username"><input placeholder="subscriber@example" /></Field><Field label="Password"><input type="password" placeholder="••••••••" /></Field><Field label="Service Name"><input placeholder="Optional" /></Field><Field label="MTU / MRU"><input defaultValue="1492 / 1492" /></Field><Field label="Keepalive"><input defaultValue="10" /></Field><Field label="Default Route"><select><option>Yes</option><option>No</option></select></Field></>}
    </div> : <div className="nr-readonly-list"><ReadonlyRow label="Connection State" value={<StatusIndicator state="CONNECTED" compact />} /><ReadonlyRow label="Uptime" value="12d 04:18:22" /><ReadonlyRow label="Local Address" value="100.64.18.22/24" /><ReadonlyRow label="Remote Address" value="100.64.18.1" /><ReadonlyRow label="Gateway" value="100.64.18.1" /><ReadonlyRow label="DNS" value="1.1.1.1, 9.9.9.9" /><ReadonlyRow label="RX" value={`${metrics.wanRx.toFixed(1)} Mbps`} /><ReadonlyRow label="TX" value={`${metrics.wanTx.toFixed(1)} Mbps`} /></div>}
    <div className="nr-footer-actions"><ActionButton tone="primary" onClick={() => toast.success("WAN configuration applied")}>Apply</ActionButton><ActionButton onClick={() => toast.success("WAN configuration saved")}>OK</ActionButton><ActionButton onClick={() => toast("Changes discarded")}>Cancel</ActionButton></div>
  </div>;
}

function Lan({ onOpen }: { onOpen: (kind: WindowKind) => void }) {
  const [tab, setTab] = useState<"General" | "DHCP" | "Clients">("General");
  return <div className="nr-fill-column">
    <div className="nr-tabs">{["General", "DHCP", "Clients"].map((item) => <button key={item} type="button" className={tab === item ? "is-active" : ""} onClick={() => setTab(item as typeof tab)}>{item}</button>)}</div>
    {tab === "General" && <div className="nr-form-grid"><Field label="Interface"><select><option>bridge-lan</option></select></Field><Field label="Gateway Address"><input defaultValue="192.168.88.1" /></Field><Field label="Subnet Mask"><input defaultValue="255.255.255.0" /></Field><Field label="MAC Address"><input readOnly defaultValue="AA:BB:CC:DD:EE:02" /></Field><Field label="MTU"><input defaultValue="1500" /></Field><Field label="Status"><StatusIndicator state="CONNECTED" /></Field></div>}
    {tab === "DHCP" && <div className="nr-form-grid"><Field label="DHCP Server"><select><option>Enabled</option><option>Disabled</option></select></Field><Field label="Pool Start"><input defaultValue="192.168.88.10" /></Field><Field label="Pool End"><input defaultValue="192.168.88.254" /></Field><Field label="Lease Time"><input defaultValue="12h" /></Field><Field label="DNS Server"><input defaultValue="192.168.88.1" /></Field><Field label="Gateway"><input defaultValue="192.168.88.1" /></Field></div>}
    {tab === "Clients" && <div className="nr-fill-column"><div className="nr-commandbar"><ActionButton onClick={() => onOpen("leases")}>Open DHCP Leases</ActionButton><ActionButton icon={<RefreshCw size={13} />}>Refresh</ActionButton></div><table className="nr-table"><thead><tr><th>IP Address</th><th>MAC Address</th><th>Hostname</th><th>Lease Status</th><th>Remaining Time</th></tr></thead><tbody><tr><td>192.168.88.18</td><td>9C:3D:CF:21:1B:90</td><td>workstation-01</td><td>bound</td><td>10h 42m</td></tr><tr><td>192.168.88.31</td><td>48:4D:7E:33:A9:10</td><td>scanner-east</td><td>bound</td><td>06h 11m</td></tr><tr><td>192.168.88.50</td><td>3E:2B:11:F3:91:04</td><td>lab-sensor</td><td>waiting</td><td>—</td></tr></tbody></table></div>}
    <div className="nr-footer-actions"><ActionButton tone="primary" onClick={() => toast.success("LAN configuration applied")}>Apply</ActionButton><ActionButton onClick={() => toast.success("LAN configuration saved")}>OK</ActionButton><ActionButton>Cancel</ActionButton></div>
  </div>;
}

function Dhcp({ onOpen }: { onOpen: (kind: WindowKind) => void }) {
  return <div className="nr-fill-column"><div className="nr-commandbar"><ActionButton icon={<Plus size={13} />}>Add Pool</ActionButton><ActionButton icon={<RefreshCw size={13} />}>Refresh</ActionButton><ActionButton onClick={() => onOpen("leases")}>Leases</ActionButton></div><Section title="Server"><div className="nr-form-grid"><Field label="Interface"><select><option>bridge-lan</option></select></Field><Field label="Address Pool"><select><option>lan-pool</option></select></Field><Field label="Authoritative"><select><option>After 2s delay</option></select></Field><Field label="Lease Time"><input defaultValue="12h" /></Field></div></Section><Section title="Networks"><table className="nr-table"><thead><tr><th>Address</th><th>Gateway</th><th>DNS Servers</th><th>Domain</th></tr></thead><tbody><tr><td>192.168.88.0/24</td><td>192.168.88.1</td><td>192.168.88.1</td><td>lan.netrouter</td></tr></tbody></table></Section><div className="nr-footer-actions"><ActionButton tone="primary" onClick={() => toast.success("DHCP server applied")}>Apply</ActionButton><ActionButton>OK</ActionButton><ActionButton>Cancel</ActionButton></div></div>;
}

function Leases() {
  return <div className="nr-fill-column"><div className="nr-commandbar"><ActionButton icon={<Plus size={13} />}>Make Static</ActionButton><ActionButton icon={<Trash2 size={13} />} tone="danger">Remove</ActionButton><ActionButton icon={<RefreshCw size={13} />}>Refresh</ActionButton><div className="nr-search"><Search size={13} /><input placeholder="Search lease" /></div></div><div className="nr-table-wrap"><table className="nr-table"><thead><tr><th>IP Address</th><th>MAC Address</th><th>Hostname</th><th>Lease Status</th><th>Remaining Time</th><th>Comment</th></tr></thead><tbody><tr className="is-selected"><td>192.168.88.18</td><td>9C:3D:CF:21:1B:90</td><td>workstation-01</td><td>bound</td><td>10h 42m</td><td>Office</td></tr><tr><td>192.168.88.31</td><td>48:4D:7E:33:A9:10</td><td>scanner-east</td><td>bound</td><td>06h 11m</td><td>Scanner</td></tr><tr><td>192.168.88.50</td><td>3E:2B:11:F3:91:04</td><td>lab-sensor</td><td>waiting</td><td>—</td><td>Lab</td></tr></tbody></table></div></div>;
}

function Traffic({ metrics }: { metrics: Metrics }) {
  const [range, setRange] = useState("1 minute");
  return <div className="nr-fill-column"><div className="nr-commandbar"><span className="nr-command-label">Time range</span><select className="nr-compact-select" value={range} onChange={(event) => setRange(event.target.value)}><option>10 seconds</option><option>1 minute</option><option>5 minutes</option><option>15 minutes</option></select><span className="nr-quiet">Refresh: live</span></div><div className="nr-monitor-grid"><Section title="WAN"><TrafficGraph label="RX" color="blue" value={metrics.wanRx} /><TrafficGraph label="TX" color="green" value={metrics.wanTx} /></Section><Section title="LAN"><TrafficGraph label="RX" color="blue" value={metrics.lanRx} /><TrafficGraph label="TX" color="green" value={metrics.lanTx} /></Section></div><div className="nr-table-footer">Sampling range: {range} · values update asynchronously</div></div>;
}

function SystemPanel({ onOpen, onRequestConfirm }: Pick<WindowContentProps, "onOpen" | "onRequestConfirm">) {
  const [tab, setTab] = useState<"General" | "Resources" | "Maintenance" | "Upgrade">("General");
  return <div className="nr-fill-column"><div className="nr-tabs">{["General", "Resources", "Maintenance", "Upgrade"].map((item) => <button key={item} type="button" className={tab === item ? "is-active" : ""} onClick={() => setTab(item as typeof tab)}>{item}</button>)}</div>{tab === "General" && <div className="nr-readonly-list"><ReadonlyRow label="Identity" value="edge-hq-01" /><ReadonlyRow label="Firmware Version" value="NetRouter OS 1.4.2" /><ReadonlyRow label="Kernel Version" value="6.8.12-nr" /><ReadonlyRow label="Uptime" value="12d 04:18:22" /><ReadonlyRow label="Boot Time" value="2026-08-13 10:21:38" /></div>}{tab === "Resources" && <div className="nr-readonly-list"><ReadonlyRow label="CPU" value="ARM Cortex-A72 · 4 cores" /><ReadonlyRow label="Memory" value="2.0 GB / 4.0 GB" /><ReadonlyRow label="Storage" value="1.8 GB / 8.0 GB" /><ReadonlyRow label="Architecture" value="arm64" /></div>}{tab === "Maintenance" && <div className="nr-maintenance"><p>Destructive maintenance operations require an explicit confirmation.</p><div><ActionButton tone="danger" onClick={() => onRequestConfirm("reboot")}>Reboot</ActionButton><ActionButton tone="danger" onClick={() => onRequestConfirm("factory")}>Factory Reset</ActionButton><ActionButton icon={<Download size={13} />} onClick={() => toast.success("Backup downloaded")}>Backup</ActionButton><ActionButton icon={<Upload size={13} />} onClick={() => toast("Select a backup to restore")}>Restore</ActionButton></div></div>}{tab === "Upgrade" && <div className="nr-upgrade-summary"><ReadonlyRow label="Current Version" value="NetRouter OS 1.4.2" /><ReadonlyRow label="Available Package" value="No package selected" /><ActionButton tone="primary" onClick={() => onOpen("firmware")}>Open Firmware Upgrade</ActionButton></div>}</div>;
}

function Firmware({ onRequestConfirm }: { onRequestConfirm: (kind: "reboot" | "factory") => void }) {
  const [uploaded, setUploaded] = useState(false);
  const [verified, setVerified] = useState(false);
  return <div className="nr-fill-column"><Section title="Firmware Package"><div className="nr-upload-row"><File size={19} /><div><strong>{uploaded ? "netrouter-os-1.4.3.npk" : "No firmware package selected"}</strong><span>{uploaded ? "6.8 MB · signature pending" : "Upload a signed NetRouter OS package."}</span></div><ActionButton icon={<Upload size={13} />} onClick={() => setUploaded(true)}>Upload Firmware</ActionButton></div></Section><Section title="Verification"><ReadonlyRow label="Checksum" value={uploaded ? "SHA-256 detected" : "—"} /><ReadonlyRow label="Signature" value={verified ? <span className="nr-ok"><Check size={13} /> Verified</span> : "Not verified"} /></Section><div className="nr-footer-actions"><ActionButton onClick={() => { setVerified(true); toast.success("Firmware signature verified"); }}>Verify</ActionButton><ActionButton tone="primary" onClick={() => uploaded && verified ? onRequestConfirm("reboot") : toast.error("Upload and verify firmware first")}>Upgrade & Reboot</ActionButton><ActionButton>Cancel</ActionButton></div></div>;
}

function Neighbors() {
  return <div className="nr-fill-column"><div className="nr-commandbar"><ActionButton icon={<RefreshCw size={13} />} onClick={() => toast.success("Neighbor discovery refreshed")}>Refresh</ActionButton><ActionButton onClick={() => toast("Connect via MAC selected")}>Connect via MAC</ActionButton><ActionButton onClick={() => toast("Connect via IP selected")}>Connect via IP</ActionButton><span className="nr-quiet">L2 MAC + L3 IPv4 discovery</span></div><div className="nr-table-wrap"><table className="nr-table"><thead><tr><th>Identity</th><th>MAC</th><th>IPv4</th><th>Architecture</th><th>Version</th><th>Uptime</th><th>Status</th></tr></thead><tbody>{neighbors.map((neighbor) => <tr key={neighbor.mac}><td>{neighbor.identity}</td><td>{neighbor.mac}</td><td>{neighbor.ip}</td><td>{neighbor.architecture}</td><td>{neighbor.version}</td><td>{neighbor.uptime}</td><td><span className={neighbor.status === "Available" ? "nr-available" : "nr-busy"}>{neighbor.status}</span></td></tr>)}</tbody></table></div></div>;
}

function TerminalPanel() {
  const [command, setCommand] = useState("");
  const [lines, setLines] = useState(["NetRouter OS 1.4.2 (stable)", "[admin@edge-hq-01] > "]);
  function submit() { if (!command.trim()) return; setLines((items) => [...items, `[admin@edge-hq-01] > ${command}`, "command accepted (simulation)", "[admin@edge-hq-01] > "]); setCommand(""); }
  return <div className="nr-terminal"><pre>{lines.join("\n")}</pre><div><span>[admin@edge-hq-01] &gt;</span><input aria-label="Terminal command" value={command} onChange={(event) => setCommand(event.target.value)} onKeyDown={(event) => event.key === "Enter" && submit()} /></div></div>;
}

function FilesPanel() {
  return <div className="nr-fill-column"><div className="nr-commandbar"><ActionButton icon={<Upload size={13} />}>Upload</ActionButton><ActionButton icon={<Download size={13} />}>Download</ActionButton><ActionButton icon={<Trash2 size={13} />} tone="danger">Remove</ActionButton><ActionButton icon={<RefreshCw size={13} />}>Refresh</ActionButton></div><div className="nr-file-list"><div><Folder size={15} /><span>backup</span><small>directory</small></div><div><File size={15} /><span>router-backup-2026-08-25.backup</span><small>1.3 MB</small></div><div><File size={15} /><span>netrouter-os-1.4.2.npk</span><small>6.7 MB</small></div><div><File size={15} /><span>startup.rsc</span><small>2.1 KB</small></div></div></div>;
}

function LogPanel() {
  return <div className="nr-fill-column"><div className="nr-commandbar"><ActionButton icon={<RefreshCw size={13} />}>Refresh</ActionButton><ActionButton icon={<Copy size={13} />}>Copy</ActionButton><div className="nr-search"><Search size={13} /><input placeholder="Filter log" /></div></div><div className="nr-log-list"><p><time>10:24:07</time><span className="is-info">system</span> configuration applied by admin</p><p><time>10:23:58</time><span className="is-ok">dhcp</span> lease bound: 192.168.88.18</p><p><time>10:22:16</time><span className="is-info">interface</span> ether1 link up (1 Gbps full duplex)</p><p><time>10:18:42</time><span className="is-warn">firewall</span> dropped invalid packet from 198.51.100.22</p><p><time>10:16:20</time><span className="is-ok">system</span> time synchronized with NTP</p></div></div>;
}

function SettingsPanel() {
  return <div className="nr-fill-column"><Section title="Application"><div className="nr-form-grid"><Field label="Language"><select><option>English</option><option>Arabic</option></select></Field><Field label="Appearance"><select><option>Classic light</option><option>High contrast</option></select></Field><Field label="Hide Passwords"><select><option>Enabled</option><option>Disabled</option></select></Field><Field label="Inline Comments"><select><option>Enabled</option><option>Disabled</option></select></Field></div></Section><Section title="Session"><div className="nr-form-grid"><Field label="Autosave"><select><option>Every 5 minutes</option><option>Off</option></select></Field><Field label="Recent Session Limit"><input defaultValue="10" /></Field></div></Section><div className="nr-footer-actions"><ActionButton tone="primary" onClick={() => toast.success("Settings saved")}>Apply</ActionButton><ActionButton>OK</ActionButton><ActionButton>Cancel</ActionButton></div></div>;
}

function QuickSet() {
  const [mode, setMode] = useState("DHCP Client");
  return <div className="nr-fill-column"><Section title="Router Setup"><div className="nr-form-grid"><Field label="Router Identity"><input defaultValue="edge-hq-01" /></Field><Field label="WAN Mode"><select value={mode} onChange={(event) => setMode(event.target.value)}><option>DHCP Client</option><option>Static IP</option><option>PPPoE</option></select></Field>{mode === "Static IP" && <Field label="WAN IP"><input placeholder="100.64.18.22/24" /></Field>}{mode === "PPPoE" && <><Field label="Username"><input placeholder="subscriber@example" /></Field><Field label="Password"><input type="password" /></Field></>}<Field label="LAN Address"><input defaultValue="192.168.88.1" /></Field><Field label="LAN Subnet"><input defaultValue="255.255.255.0" /></Field><Field label="DHCP Server"><select><option>Enabled</option><option>Disabled</option></select></Field><Field label="DNS"><input defaultValue="1.1.1.1, 9.9.9.9" /></Field></div></Section><Section title="Internet Status"><div className="nr-internet-status"><StatusIndicator state="CONNECTED" /><span>WAN DHCP lease valid · gateway reachable</span></div></Section><div className="nr-footer-actions"><ActionButton tone="primary" onClick={() => toast.success("Quick Set configuration applied")}>Apply</ActionButton><ActionButton onClick={() => toast("Quick Set values reset")}>Reset</ActionButton></div></div>;
}

export function WindowContent({ kind, metrics, onOpen, onRequestConfirm }: WindowContentProps) {
  switch (kind) {
    case "dashboard": return <Dashboard metrics={metrics} />;
    case "interfaces": return <Interfaces />;
    case "wan": return <Wan metrics={metrics} />;
    case "lan": return <Lan onOpen={onOpen} />;
    case "dhcp": return <Dhcp onOpen={onOpen} />;
    case "leases": return <Leases />;
    case "traffic": return <Traffic metrics={metrics} />;
    case "system": return <SystemPanel onOpen={onOpen} onRequestConfirm={onRequestConfirm} />;
    case "firmware": return <Firmware onRequestConfirm={onRequestConfirm} />;
    case "neighbors": return <Neighbors />;
    case "terminal": return <TerminalPanel />;
    case "files": return <FilesPanel />;
    case "log": return <LogPanel />;
    case "settings": return <SettingsPanel />;
    case "quickset": return <QuickSet />;
  }
}
