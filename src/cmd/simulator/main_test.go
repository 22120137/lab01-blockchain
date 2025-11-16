package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// duplicate config struct
type testConfig struct {
	NumNodes           int     `json:"NumNodes"`
	BlocksToFinalize   int     `json:"BlocksToFinalize"`
	Seed               int64   `json:"Seed"`
	MaxTicks           int     `json:"MaxTicks"`
	LatencyMin         int     `json:"LatencyMin"`
	LatencyMax         int     `json:"LatencyMax"`
	DropRate           float64 `json:"DropRate"`
	DuplicateRate      float64 `json:"DuplicateRate"`
	MaxQueuePerTick    int     `json:"MaxQueue"`
	MaxOutboundPerTick int     `json:"MaxOutboundPerTick"`
	BlockDurationTicks int     `json:"BlockDurationTicks"`
	DeterministicLogs  bool    `json:"DeterministicLogs"`
}

// findProjectRoot walks up from cwd until it finds go.mod or reaches root
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		} // reached filesystem root
		dir = parent
	}
	return "", os.ErrNotExist
}

func TestLoadConfigFile(t *testing.T) {
	root, err := findProjectRoot()
	if err != nil {
		t.Fatalf("cannot find project root (go.mod): %v", err)
	}
	configPath := filepath.Join(root, "config", "scenario1.json")
	if _, err := os.Stat(configPath); err != nil {
		alt := filepath.Join(root, "..", "config", "scenario1.json")
		if _, err2 := os.Stat(alt); err2 == nil {
			configPath = alt
		} else {
			t.Fatalf("config file not found (checked %s and %s)", configPath, alt)
		}
	}

	b, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed read config: %v", err)
	}
	var cfg testConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		t.Fatalf("failed unmarshal config: %v", err)
	}

	// Basic sanity checks
	if cfg.NumNodes <= 0 {
		t.Fatalf("NumNodes must be >0, got %d", cfg.NumNodes)
	}
	if cfg.MaxTicks <= 0 {
		t.Fatalf("MaxTicks must be >0, got %d", cfg.MaxTicks)
	}
}
