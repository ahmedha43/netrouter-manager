package config

import (
	"path/filepath"
	"testing"
)

func TestConfigLoadSaveDefault(t *testing.T) {
	temp := t.TempDir()
	configPath := filepath.Join(temp, "config.json")

	store := NewStore(configPath)
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("failed to load default config: %v", err)
	}

	if cfg.System.Identity != "NetRouter-Core" {
		t.Errorf("expected identity NetRouter-Core, got %s", cfg.System.Identity)
	}

	if cfg.LAN.Address != "192.168.88.1/24" {
		t.Errorf("expected LAN 192.168.88.1/24, got %s", cfg.LAN.Address)
	}

	// Mutate and save
	cfg.System.Identity = "Office-Router-01"
	if err := store.Save(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Reload from new store instance
	store2 := NewStore(configPath)
	loaded, err := store2.Load()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}

	if loaded.System.Identity != "Office-Router-01" {
		t.Errorf("expected Office-Router-01, got %s", loaded.System.Identity)
	}
}

func TestConfigExportImport(t *testing.T) {
	temp := t.TempDir()
	configPath := filepath.Join(temp, "config.json")

	store := NewStore(configPath)
	_, _ = store.Load()

	exported, err := store.Export()
	if err != nil {
		t.Fatalf("export error: %v", err)
	}

	store2 := NewStore(filepath.Join(temp, "config2.json"))
	if err := store2.Import(exported); err != nil {
		t.Fatalf("import error: %v", err)
	}

	if store2.Get().System.Identity != store.Get().System.Identity {
		t.Errorf("mismatched imported identity")
	}
}
