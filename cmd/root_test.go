package cmd

import (
	"fmt"
	"testing"
)

func TestParseArgsDash(t *testing.T) {
	p, err := parseArgs([]string{"-", "web1", "-"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Targets) != 3 {
		t.Errorf("len(Targets) = %d, want 3", len(p.Targets))
	}
	if p.Targets[0] != "-" {
		t.Errorf("Targets[0] = %q, want %q", p.Targets[0], "-")
	}
	if p.Targets[2] != "-" {
		t.Errorf("Targets[2] = %q, want %q", p.Targets[2], "-")
	}
}

func TestParseArgsTooMany(t *testing.T) {
	targets := make([]string, 10)
	for i := range targets {
		targets[i] = fmt.Sprintf("host%d", i)
	}
	_, err := parseArgs(targets)
	if err == nil {
		t.Error("expected error for >9 targets, got nil")
	}
}

func TestParseArgsSave(t *testing.T) {
	p, err := parseArgs([]string{"--save", "prod", "web1", "web2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.SaveGroup != "prod" {
		t.Errorf("SaveGroup = %q, want %q", p.SaveGroup, "prod")
	}
	if len(p.Targets) != 2 {
		t.Errorf("len(Targets) = %d, want 2", len(p.Targets))
	}
	if p.Targets[0] != "web1" || p.Targets[1] != "web2" {
		t.Errorf("Targets = %v, want [web1 web2]", p.Targets)
	}
}

func TestParseArgsGroup(t *testing.T) {
	p, err := parseArgs([]string{"-g", "staging"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Group != "staging" {
		t.Errorf("Group = %q, want %q", p.Group, "staging")
	}
}

func TestParseArgsGroupLong(t *testing.T) {
	p, err := parseArgs([]string{"--group", "prod"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Group != "prod" {
		t.Errorf("Group = %q, want %q", p.Group, "prod")
	}
}

func TestParseArgsListGroups(t *testing.T) {
	p, err := parseArgs([]string{"--list-groups"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.ListGroups {
		t.Error("ListGroups should be true")
	}
}

func TestParseArgsListHosts(t *testing.T) {
	p, err := parseArgs([]string{"--list-hosts"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.ListHosts {
		t.Error("ListHosts should be true")
	}
}

func TestParseArgsUnknownFlag(t *testing.T) {
	_, err := parseArgs([]string{"--unknown"})
	if err == nil {
		t.Error("expected error for unknown flag")
	}
}

func TestParseArgsExactly9(t *testing.T) {
	targets := make([]string, 9)
	for i := range targets {
		targets[i] = fmt.Sprintf("host%d", i)
	}
	p, err := parseArgs(targets)
	if err != nil {
		t.Errorf("unexpected error for exactly 9 targets: %v", err)
	}
	if len(p.Targets) != 9 {
		t.Errorf("len(Targets) = %d, want 9", len(p.Targets))
	}
}

func TestParseArgsMissingGroupName(t *testing.T) {
	_, err := parseArgs([]string{"-g"})
	if err == nil {
		t.Error("expected error when -g has no argument")
	}
}
