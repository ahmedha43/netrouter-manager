// Package protocol owns the stable, transport-neutral NetRouter management API.
// It must stay independent from Linux command execution and desktop rendering.
package protocol

import "encoding/json"

const Version = 1

type Method string

const (
	GetSystemStatus Method = "system.status"
	ListInterfaces  Method = "network.interfaces.list"
	SetLinkState    Method = "network.link.set_state"
	AssignAddress   Method = "network.address.assign"
	ReplaceRoute    Method = "network.route.replace_default"
	ApplyDHCPDNS    Method = "services.dhcp_dns.apply"
	ApplyFirewall   Method = "firewall.apply"
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
