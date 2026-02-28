package config

import (
	"os"
	"path/filepath"
	"testing"

	ssh_config "github.com/kevinburke/ssh_config"
)

func TestResolveHostFromConfig(t *testing.T) {
	content := `
Host web1
    HostName 192.168.1.10
    User admin
    Port 2222
    IdentityFile ~/.ssh/id_rsa
`
	f, err := os.CreateTemp("", "ssh_config_test_*")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer os.Remove(f.Name())
	f.WriteString(content)
	f.Close()

	// Parse using the library directly for test verification
	file, _ := os.Open(f.Name())
	cfg, err := ssh_config.Decode(file)
	file.Close()
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	hostname, _ := cfg.Get("web1", "HostName")
	if hostname != "192.168.1.10" {
		t.Errorf("HostName = %q, want %q", hostname, "192.168.1.10")
	}

	user, _ := cfg.Get("web1", "User")
	if user != "admin" {
		t.Errorf("User = %q, want %q", user, "admin")
	}

	port, _ := cfg.Get("web1", "Port")
	if port != "2222" {
		t.Errorf("Port = %q, want %q", port, "2222")
	}
}

func TestListHostsFromFile(t *testing.T) {
	content := `
Host myserver
    HostName 10.0.0.1
    User root

Host *.example.com
    User deploy

Host bastion
    HostName bastion.example.com
`
	f, err := os.CreateTemp("", "ssh_config_hosts_*")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer os.Remove(f.Name())
	f.WriteString(content)
	f.Close()

	file, _ := os.Open(f.Name())
	cfg, err := ssh_config.Decode(file)
	file.Close()
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	var nonWildcard []string
	for _, host := range cfg.Hosts {
		for _, pat := range host.Patterns {
			alias := pat.String()
			if alias != "*" && !containsWildcard(alias) {
				nonWildcard = append(nonWildcard, alias)
			}
		}
	}

	if len(nonWildcard) != 2 {
		t.Errorf("expected 2 non-wildcard hosts, got %d: %v", len(nonWildcard), nonWildcard)
	}
}

func TestExpandHome(t *testing.T) {
	home, _ := os.UserHomeDir()
	got := expandHome("~/.ssh/id_rsa")
	want := filepath.Join(home, ".ssh/id_rsa")
	if got != want {
		t.Errorf("expandHome = %q, want %q", got, want)
	}
}

func TestSSHConfigPaths(t *testing.T) {
	paths := sshConfigPaths()
	if len(paths) == 0 {
		t.Error("sshConfigPaths returned no paths")
	}
	for _, p := range paths {
		if p == "" {
			t.Error("sshConfigPaths returned empty path")
		}
	}
}

func containsWildcard(s string) bool {
	for _, c := range s {
		if c == '*' || c == '?' {
			return true
		}
	}
	return false
}
