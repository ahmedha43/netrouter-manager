// Package config manages the declarative unified configuration store for NetRouter OS.
// All system parameters are persisted to /etc/netrouter/config.json with atomic writes.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	DefaultConfigPath = "/etc/netrouter/config.json"
)

type Config struct {
	Version   int             `json:"version"`
	System    SystemConfig    `json:"system"`
	WAN       WANConfig       `json:"wan"`
	LAN       LANConfig       `json:"lan"`
	DHCP      DHCPConfig      `json:"dhcp"`
	Firewall  FirewallConfig  `json:"firewall"`
	WireGuard WireGuardConfig `json:"wireguard"`
}

type SystemConfig struct {
	Identity string `json:"identity"`
	Timezone string `json:"timezone"`
}

type WANConfig struct {
	Mode      string      `json:"mode"` // "dhcp", "static", "pppoe"
	Interface string      `json:"interface"`
	StaticIP  string      `json:"static_ip,omitempty"`
	Netmask   string      `json:"netmask,omitempty"`
	Gateway   string      `json:"gateway,omitempty"`
	DNS       []string    `json:"dns,omitempty"`
	PPPoE     PPPoEConfig `json:"pppoe,omitempty"`
}

type PPPoEConfig struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	ServiceName string `json:"service_name,omitempty"`
	MTU         int    `json:"mtu,omitempty"`
}

type LANConfig struct {
	Interface string `json:"interface"`
	Address   string `json:"address"` // e.g. "192.168.88.1/24"
	MTU       int    `json:"mtu"`
}

type DHCPConfig struct {
	Enabled    bool     `json:"enabled"`
	Interface  string   `json:"interface"`
	SubnetCIDR string   `json:"subnet_cidr"`
	Gateway    string   `json:"gateway"`
	PoolStart  string   `json:"pool_start"`
	PoolEnd    string   `json:"pool_end"`
	LeaseTime  string   `json:"lease_time"`
	DNSServers []string `json:"dns_servers"`
}

type FirewallConfig struct {
	Enabled        bool   `json:"enabled"`
	LANInterface   string `json:"lan_interface"`
	WANInterface   string `json:"wan_interface"`
	ManagementPort uint16 `json:"management_port"`
	NATMasquerade  bool   `json:"nat_masquerade"`
}

type WireGuardConfig struct {
	Enabled    bool             `json:"enabled"`
	Interface  string           `json:"interface"`
	ListenPort int              `json:"listen_port"`
	PrivateKey string           `json:"private_key"`
	Address    string           `json:"address"`
	Peers      []WireGuardPeer  `json:"peers,omitempty"`
}

type WireGuardPeer struct {
	PublicKey  string   `json:"public_key"`
	Endpoint   string   `json:"endpoint,omitempty"`
	AllowedIPs []string `json:"allowed_ips"`
}

type Store struct {
	mu   sync.RWMutex
	path string
	data Config
}

func NewStore(path string) *Store {
	if path == "" {
		path = DefaultConfigPath
	}
	return &Store{
		path: path,
		data: Default(),
	}
}

func Default() Config {
	return Config{
		Version: 1,
		System: SystemConfig{
			Identity: "NetRouter-Core",
			Timezone: "UTC",
		},
		WAN: WANConfig{
			Mode:      "dhcp",
			Interface: "ether1",
			DNS:       []string{"1.1.1.1", "8.8.8.8"},
			PPPoE: PPPoEConfig{
				MTU: 1492,
			},
		},
		LAN: LANConfig{
			Interface: "ether2",
			Address:   "192.168.88.1/24",
			MTU:       1500,
		},
		DHCP: DHCPConfig{
			Enabled:    true,
			Interface:  "ether2",
			SubnetCIDR: "192.168.88.0/24",
			Gateway:    "192.168.88.1",
			PoolStart:  "192.168.88.100",
			PoolEnd:    "192.168.88.200",
			LeaseTime:  "12h",
			DNSServers: []string{"192.168.88.1", "1.1.1.1"},
		},
		Firewall: FirewallConfig{
			Enabled:        true,
			LANInterface:   "ether2",
			WANInterface:   "ether1",
			ManagementPort: 8443,
			NATMasquerade:  true,
		},
		WireGuard: WireGuardConfig{
			Enabled:    false,
			Interface:  "wg0",
			ListenPort: 51820,
			Address:    "10.10.0.1/24",
		},
	}
}

func (s *Store) Load() (Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			// If file does not exist, initialize with default and save
			s.data = Default()
			_ = s.saveAtomicLocked(s.data)
			return s.data, nil
		}
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config json: %w", err)
	}

	s.data = cfg
	return s.data, nil
}

func (s *Store) Get() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data
}

func (s *Store) Save(cfg Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveAtomicLocked(cfg)
}

func (s *Store) saveAtomicLocked(cfg Config) error {
	encoded, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config json: %w", err)
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	tmpFile, err := os.CreateTemp(dir, ".config-*.json.tmp")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName)

	if _, err := tmpFile.Write(encoded); err != nil {
		tmpFile.Close()
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return fmt.Errorf("sync temp config: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp config: %w", err)
	}

	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("atomic rename config: %w", err)
	}

	s.data = cfg
	return nil
}

func (s *Store) Export() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return json.MarshalIndent(s.data, "", "  ")
}

func (s *Store) Import(data []byte) error {
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("invalid config backup json: %w", err)
	}
	return s.Save(cfg)
}
