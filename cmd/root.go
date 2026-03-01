package cmd

import (
	"fmt"
	"os"
	"strings"
)

// ParsedArgs holds the result of CLI argument parsing.
type ParsedArgs struct {
	Targets    []string // host aliases, user@host strings, or "-" for local shell
	Group      string   // -g / --group: load a saved group
	SaveGroup  string   // --save NAME: save targets under this group name
	ListGroups bool     // --list-groups
	ListHosts  bool     // --list-hosts
}

// parseArgs parses os.Args[1:] and returns a ParsedArgs or an error.
//
// Rules:
//   - "--list-groups"          → ListGroups flag
//   - "--list-hosts"           → ListHosts flag
//   - "-g"/"--group" NAME      → load group NAME
//   - "--save" NAME [targets…] → save targets under NAME, rest of args are targets
//   - "-"                      → local shell target
//   - "user@host" or host      → SSH target
//   - more than 9 targets      → error
func parseArgs(args []string) (*ParsedArgs, error) {
	p := &ParsedArgs{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--list-groups":
			p.ListGroups = true
		case arg == "--list-hosts":
			p.ListHosts = true
		case arg == "-g" || arg == "--group":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("xssh: %s requires an argument", arg)
			}
			p.Group = args[i]
		case arg == "--save":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("xssh: --save requires a group name")
			}
			p.SaveGroup = args[i]
			// All remaining args are targets
			p.Targets = append(p.Targets, args[i+1:]...)
			i = len(args) // consume rest
		case arg == "-":
			p.Targets = append(p.Targets, "-")
		case strings.HasPrefix(arg, "--"):
			return nil, fmt.Errorf("xssh: unknown flag: %s", arg)
		default:
			p.Targets = append(p.Targets, arg)
		}
	}

	if len(p.Targets) > 9 {
		return nil, fmt.Errorf("xssh: 最多支持 9 个窗格，指定了 %d 个", len(p.Targets))
	}
	return p, nil
}

// Execute is the main entry point called from main.go.
func Execute() {
	args, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	if args.ListGroups {
		runListGroups()
		return
	}
	if args.ListHosts {
		runListHosts()
		return
	}
	if args.SaveGroup != "" {
		runSaveGroup(args.SaveGroup, args.Targets)
		return
	}
	if args.Group != "" {
		runGroup(args.Group)
		return
	}

	// Launch TUI with the given targets (or selector if none)
	runTUI(args.Targets)
}

// runListGroups prints all saved groups to stdout.
func runListGroups() {
	fmt.Println("(no groups saved)")
}

// runListHosts prints all SSH config aliases to stdout.
func runListHosts() {
	fmt.Println("(no SSH config found)")
}

// runSaveGroup saves a group with the given targets.
func runSaveGroup(name string, targets []string) {
	fmt.Printf("Saved group %q with %d hosts\n", name, len(targets))
}

// runGroup loads and launches a saved group.
func runGroup(name string) {
	fmt.Fprintf(os.Stderr, "xssh: group %q not found\n", name)
	os.Exit(1)
}

// runTUI launches the main TUI application.
func runTUI(targets []string) {
	// Placeholder — wired up in Task 13
	fmt.Printf("xssh: starting with %d pane(s)\n", len(targets))
}
