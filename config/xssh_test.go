package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.General.ScrollbackLines != 5000 {
		t.Errorf("ScrollbackLines = %d, want 5000", cfg.General.ScrollbackLines)
	}
	if cfg.General.ReconnectAttempts != 3 {
		t.Errorf("ReconnectAttempts = %d, want 3", cfg.General.ReconnectAttempts)
	}
	if cfg.General.ReconnectInterval != 5*time.Second {
		t.Errorf("ReconnectInterval = %v, want 5s", cfg.General.ReconnectInterval)
	}
	if cfg.General.SSHTimeout != 10*time.Second {
		t.Errorf("SSHTimeout = %v, want 10s", cfg.General.SSHTimeout)
	}
	if cfg.UI.FocusedColor != "#00BFFF" {
		t.Errorf("FocusedColor = %q, want #00BFFF", cfg.UI.FocusedColor)
	}
	if cfg.Groups == nil {
		t.Error("Groups should not be nil")
	}
}

func TestLoadNonExistent(t *testing.T) {
	// Temporarily override HOME to a temp dir with no .xssh directory
	tmp := t.TempDir()
	orig := os.Getenv("HOME")
	t.Setenv("HOME", tmp)
	defer os.Setenv("HOME", orig)

	cfg, err := Load()
	if err != nil {
		t.Errorf("Load on missing file should not error: %v", err)
	}
	// Should return defaults
	if cfg.General.ScrollbackLines != 5000 {
		t.Errorf("default ScrollbackLines = %d, want 5000", cfg.General.ScrollbackLines)
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	cfg := DefaultConfig()
	cfg.General.ScrollbackLines = 9999
	cfg.UI.FocusedColor = "#FFFFFF"
	cfg.Groups["myprod"] = []string{"web1", "web2"}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.General.ScrollbackLines != 9999 {
		t.Errorf("ScrollbackLines = %d, want 9999", loaded.General.ScrollbackLines)
	}
	if loaded.UI.FocusedColor != "#FFFFFF" {
		t.Errorf("FocusedColor = %q, want #FFFFFF", loaded.UI.FocusedColor)
	}
	if len(loaded.Groups["myprod"]) != 2 {
		t.Errorf("Groups[myprod] = %v, want 2 items", loaded.Groups["myprod"])
	}
}

func TestSaveGroupOverwrite(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	if err := SaveGroup("staging", []string{"s1", "s2"}); err != nil {
		t.Fatalf("SaveGroup 1: %v", err)
	}
	if err := SaveGroup("staging", []string{"s3"}); err != nil {
		t.Fatalf("SaveGroup 2: %v", err)
	}

	targets, err := LoadGroup("staging")
	if err != nil {
		t.Fatalf("LoadGroup: %v", err)
	}
	if len(targets) != 1 || targets[0] != "s3" {
		t.Errorf("LoadGroup = %v, want [s3]", targets)
	}
}

func TestLoadGroupNotFound(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	targets, err := LoadGroup("nonexistent")
	if err != nil {
		t.Errorf("LoadGroup missing group should not error: %v", err)
	}
	if targets != nil {
		t.Errorf("LoadGroup missing = %v, want nil", targets)
	}
}

func TestListGroups(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	if err := SaveGroup("prod", []string{"p1"}); err != nil {
		t.Fatalf("SaveGroup: %v", err)
	}
	if err := SaveGroup("dev", []string{"d1", "d2"}); err != nil {
		t.Fatalf("SaveGroup: %v", err)
	}

	groups, err := ListGroups()
	if err != nil {
		t.Fatalf("ListGroups: %v", err)
	}
	if len(groups) != 2 {
		t.Errorf("len(groups) = %d, want 2", len(groups))
	}
}

func TestConfigFilePath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	path, err := configPath()
	if err != nil {
		t.Fatalf("configPath: %v", err)
	}
	want := filepath.Join(tmp, ".xssh", "config.yaml")
	if path != want {
		t.Errorf("configPath = %q, want %q", path, want)
	}
}
