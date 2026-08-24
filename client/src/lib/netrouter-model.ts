/**
 * Design reminder — NetRouter Manager uses a dense native desktop vocabulary:
 * compact engineering data, explicit statuses, and no SaaS card patterns.
 */
export type ConnectionState =
  | "CONNECTED"
  | "CONNECTING"
  | "DISCONNECTED"
  | "ERROR";

export type WindowKind =
  | "dashboard"
  | "interfaces"
  | "wan"
  | "lan"
  | "dhcp"
  | "leases"
  | "traffic"
  | "system"
  | "firmware"
  | "neighbors"
  | "terminal"
  | "files"
  | "log"
  | "settings"
  | "quickset";

export type Metrics = {
  cpu: number;
  memory: number;
  wanRx: number;
  wanTx: number;
  lanRx: number;
  lanTx: number;
  secondsOnline: number;
};

export type InterfaceRow = {
  name: string;
  role: "WAN" | "LAN";
  type: string;
  mac: string;
  mtu: string;
  ip: string;
  rx: string;
  tx: string;
  status: ConnectionState;
};

export type Neighbor = {
  identity: string;
  mac: string;
  ip: string;
  architecture: string;
  version: string;
  uptime: string;
  status: "Available" | "Busy";
};

export const interfaceRows: InterfaceRow[] = [
  {
    name: "ether1",
    role: "WAN",
    type: "Ethernet",
    mac: "AA:BB:CC:DD:EE:01",
    mtu: "1500",
    ip: "100.64.18.22/24",
    rx: "125.4 Mbps",
    tx: "38.2 Mbps",
    status: "CONNECTED",
  },
  {
    name: "ether2",
    role: "LAN",
    type: "Ethernet",
    mac: "AA:BB:CC:DD:EE:02",
    mtu: "1500",
    ip: "192.168.88.1/24",
    rx: "117.8 Mbps",
    tx: "35.4 Mbps",
    status: "CONNECTED",
  },
  {
    name: "pppoe-wan",
    role: "WAN",
    type: "PPPoE",
    mac: "—",
    mtu: "1492",
    ip: "100.64.18.22",
    rx: "125.4 Mbps",
    tx: "38.2 Mbps",
    status: "CONNECTED",
  },
  {
    name: "bridge-lan",
    role: "LAN",
    type: "Bridge",
    mac: "AA:BB:CC:DD:EE:02",
    mtu: "1500",
    ip: "192.168.88.1",
    rx: "117.8 Mbps",
    tx: "35.4 Mbps",
    status: "CONNECTED",
  },
];

export const neighbors: Neighbor[] = [
  {
    identity: "edge-hq-01",
    mac: "AA:BB:CC:DD:EE:01",
    ip: "192.168.88.1",
    architecture: "arm64",
    version: "NetRouter OS 1.4.2",
    uptime: "12d 04:18:22",
    status: "Available",
  },
  {
    identity: "branch-east",
    mac: "AA:BB:CC:DD:EE:31",
    ip: "10.44.0.1",
    architecture: "mips64",
    version: "NetRouter OS 1.4.1",
    uptime: "5d 16:02:11",
    status: "Available",
  },
  {
    identity: "lab-gateway",
    mac: "AA:BB:CC:DD:EE:9A",
    ip: "172.20.8.1",
    architecture: "armv7",
    version: "NetRouter OS 1.3.9",
    uptime: "00:42:09",
    status: "Busy",
  },
];

export const initialMetrics: Metrics = {
  cpu: 14,
  memory: 42,
  wanRx: 125.4,
  wanTx: 38.2,
  lanRx: 117.8,
  lanTx: 35.4,
  secondsOnline: 12 * 86400 + 4 * 3600 + 18 * 60 + 22,
};

export const windowMetadata: Record<
  WindowKind,
  { title: string; defaultPosition: [number, number]; defaultSize: [number, number] }
> = {
  dashboard: { title: "Dashboard", defaultPosition: [18, 16], defaultSize: [510, 470] },
  interfaces: { title: "Interfaces", defaultPosition: [116, 74], defaultSize: [760, 410] },
  wan: { title: "WAN", defaultPosition: [268, 118], defaultSize: [500, 440] },
  lan: { title: "LAN", defaultPosition: [234, 104], defaultSize: [550, 465] },
  dhcp: { title: "DHCP Server", defaultPosition: [196, 88], defaultSize: [590, 455] },
  leases: { title: "DHCP Leases", defaultPosition: [290, 140], defaultSize: [650, 350] },
  traffic: { title: "Traffic Monitor", defaultPosition: [472, 52], defaultSize: [445, 430] },
  system: { title: "System", defaultPosition: [320, 66], defaultSize: [510, 465] },
  firmware: { title: "Firmware Upgrade", defaultPosition: [400, 128], defaultSize: [465, 330] },
  neighbors: { title: "Neighbor Discovery", defaultPosition: [228, 72], defaultSize: [700, 390] },
  terminal: { title: "New Terminal", defaultPosition: [240, 92], defaultSize: [620, 410] },
  files: { title: "Files", defaultPosition: [380, 80], defaultSize: [540, 360] },
  log: { title: "Log", defaultPosition: [345, 110], defaultSize: [620, 360] },
  settings: { title: "Settings", defaultPosition: [410, 102], defaultSize: [430, 350] },
  quickset: { title: "Quick Set", defaultPosition: [74, 34], defaultSize: [485, 505] },
};

export function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const remaining = seconds % 60;
  return `${days}d ${String(hours).padStart(2, "0")}:${String(minutes).padStart(2, "0")}:${String(remaining).padStart(2, "0")}`;
}

export function jitterMetric(value: number, spread: number, lower = 0, upper = 100): number {
  const next = value + (Math.random() - 0.5) * spread;
  return Math.max(lower, Math.min(upper, Number(next.toFixed(1))));
}
