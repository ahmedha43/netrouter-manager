// Package protocol owns the stable, transport-neutral NetRouter management API.
// It must stay independent from Linux command execution and desktop rendering.
package protocol

import "encoding/json"

const Version = 1

type Method string

const (
	GetSystemStatus Method = "system.status"
	SetIdentity     Method = "system.identity.set"
	RebootSystem    Method = "system.reboot"
	GetSystemLogs   Method = "system.logs.get"
	ExportConfig    Method = "system.config.export"
	ImportConfig    Method = "system.config.import"
	ListInterfaces  Method = "network.interfaces.list"
	GetTrafficStats Method = "network.traffic.stats"
	SetLinkState    Method = "network.link.set_state"
	AssignAddress   Method = "network.address.assign"
	ReplaceRoute    Method = "network.route.replace_default"
	ApplyDHCPDNS    Method = "services.dhcp_dns.apply"
	ListDHCPLeases  Method = "services.dhcp.leases.list"
	ApplyFirewall   Method = "firewall.apply"
	ApplyPPPoE      Method = "network.wan.pppoe.apply"
	ApplyWireGuard  Method = "services.vpn.wireguard.apply"
	ScanNeighbors   Method = "discovery.neighbors.scan"
)

type Request struct {
	Version int             `json:"version"`
	ID      string          `json:"id"`
	Method  Method          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	Version int             `json:"version"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *APIError       `json:"error,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type SystemStatus struct {
	Identity     string  `json:"identity"`
	Architecture string  `json:"architecture"`
	Kernel       string  `json:"kernel"`
	Uptime       uint64  `json:"uptime_seconds"`
	MemoryTotal  uint64  `json:"memory_total_bytes"`
	MemoryFree   uint64  `json:"memory_free_bytes"`
	Load1        float64 `json:"load_1"`
	DefaultRoute string  `json:"default_route,omitempty"`
}

type Interface struct {
	Name      string   `json:"name"`
	Index     int      `json:"index"`
	MAC       string   `json:"mac"`
	MTU       int      `json:"mtu"`
	Up        bool     `json:"up"`
	Running   bool     `json:"running"`
	Addresses []string `json:"addresses"`
}

type InterfaceTraffic struct {
	Name          string   `json:"name"`
	RxBytes       uint64   `json:"rx_bytes"`
	TxBytes       uint64   `json:"tx_bytes"`
	RxPackets     uint64   `json:"rx_packets"`
	TxPackets     uint64   `json:"tx_packets"`
	RxRateBps     uint64   `json:"rx_rate_bps"`
	TxRateBps     uint64   `json:"tx_rate_bps"`
	HistoryRxRate []uint64 `json:"history_rx_rate,omitempty"`
	HistoryTxRate []uint64 `json:"history_tx_rate,omitempty"`
}

type TrafficStats struct {
	Timestamp  int64              `json:"timestamp"`
	Interfaces []InterfaceTraffic `json:"interfaces"`
}

type DHCPLease struct {
	IPAddress      string `json:"ip_address"`
	MACAddress     string `json:"mac_address"`
	Hostname       string `json:"hostname"`
	ExpirationTime int64  `json:"expiration_time"`
	IsStatic       bool   `json:"is_static"`
}

type SetIdentityParams struct {
	Identity string `json:"identity"`
}

type RebootParams struct {
	Force bool `json:"force"`
}

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Facility  string `json:"facility"`
	Message   string `json:"message"`
}

type SystemLogs struct {
	Entries []LogEntry `json:"entries"`
}

type SetLinkStateParams struct {
	Name string `json:"name"`
	Up   bool   `json:"up"`
}

type AssignAddressParams struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

type ReplaceDefaultRouteParams struct {
	Device  string `json:"device"`
	Gateway string `json:"gateway"`
}

type DHCPDNSParams struct {
	Interface  string   `json:"interface"`
	SubnetCIDR string   `json:"subnet_cidr"`
	Gateway    string   `json:"gateway"`
	PoolStart  string   `json:"pool_start"`
	PoolEnd    string   `json:"pool_end"`
	LeaseTime  string   `json:"lease_time"`
	DNSServers []string `json:"dns_servers"`
}

type FirewallParams struct {
	LANInterface      string `json:"lan_interface"`
	WANInterface      string `json:"wan_interface"`
	ManagementTCPPort uint16 `json:"management_tcp_port"`
}

type ImportConfigParams struct {
	ConfigJSON string `json:"config_json"`
}
