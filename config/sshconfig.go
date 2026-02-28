package config

import (
	"os"
	"path/filepath"
	"strings"

	ssh_config "github.com/kevinburke/ssh_config"
)

// HostEntry holds connection details for a single SSH host alias.
type HostEntry struct {
	Alias        string
	HostName     string
	User         string
	Port         string
	IdentityFile string
	ProxyJump    string
	ProxyCommand string
	ForwardAgent bool
}

// ListHosts returns all non-wildcard Host aliases from ~/.ssh/config.
func ListHosts() ([]HostEntry, error) {
	paths := sshConfigPaths()
	var entries []HostEntry
	seen := make(map[string]bool)

	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		cfg, err := ssh_config.Decode(f)
		f.Close()
		if err != nil {
			continue
		}
		for _, host := range cfg.Hosts {
			for _, pat := range host.Patterns {
				alias := pat.String()
				// Skip wildcards and already-seen aliases
				if strings.ContainsAny(alias, "*?") || seen[alias] {
					continue
				}
				seen[alias] = true
				entry, err := Resolve(alias)
				if err == nil {
					entries = append(entries, *entry)
				}
			}
		}
	}
	return entries, nil
}

// Resolve looks up full connection details for the given alias.
func Resolve(alias string) (*HostEntry, error) {
	hostname := ssh_config.Get(alias, "HostName")
	if hostname == "" {
		hostname = alias
	}
	user := ssh_config.Get(alias, "User")
	port := ssh_config.Get(alias, "Port")
	if port == "" {
		port = "22"
	}
	identFiles := ssh_config.GetAll(alias, "IdentityFile")
	var identFile string
	if len(identFiles) > 0 {
		identFile = expandHome(identFiles[0])
	}
	proxyJump := ssh_config.Get(alias, "ProxyJump")
	proxyCmd := ssh_config.Get(alias, "ProxyCommand")
	fa := strings.EqualFold(ssh_config.Get(alias, "ForwardAgent"), "yes")

	return &HostEntry{
		Alias:        alias,
		HostName:     hostname,
		User:         user,
		Port:         port,
		IdentityFile: identFile,
		ProxyJump:    proxyJump,
		ProxyCommand: proxyCmd,
		ForwardAgent: fa,
	}, nil
}

// IsKnownAlias returns true if alias appears as a non-wildcard host in any ssh config.
func IsKnownAlias(name string) bool {
	entries, err := ListHosts()
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.Alias == name {
			return true
		}
	}
	return false
}

// sshConfigPaths returns candidate ssh config file paths in priority order.
func sshConfigPaths() []string {
	var paths []string
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		paths = append(paths, filepath.Join(xdg, "ssh", "config"))
	}
	home, _ := os.UserHomeDir()
	paths = append(paths, filepath.Join(home, ".ssh", "config"))
	paths = append(paths, "/etc/ssh/ssh_config")
	return paths
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
