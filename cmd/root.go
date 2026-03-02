package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/xssh/xssh/app"
	"github.com/xssh/xssh/config"
	"github.com/xssh/xssh/selector"
)

// ParsedArgs holds the result of CLI argument parsing.
type ParsedArgs struct {
	Targets    []string // host aliases, user@host strings, or "-" for local shell
	Group      string   // -g / --group: load a saved group
	SaveGroup  string   // --save NAME: save targets under this group name
	ListGroups bool     // --list-groups
	ListHosts  bool     // --list-hosts
	BorderMode string   // --borders: "shared" (default) or "full"
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
	p := &ParsedArgs{BorderMode: "shared"}
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
		case arg == "--borders":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("xssh: --borders requires an argument (shared or full)")
			}
			switch args[i] {
			case "shared", "full":
				p.BorderMode = args[i]
			default:
				return nil, fmt.Errorf("xssh: --borders must be 'shared' or 'full', got %q", args[i])
			}
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

	// Parse border mode
	borderMode := app.BorderShared
	if args.BorderMode == "full" {
		borderMode = app.BorderFull
	}

	if args.Group != "" {
		runGroup(args.Group, borderMode)
		return
	}

	// Launch TUI with the given targets (or selector if none)
	runTUI(args.Targets, borderMode)
}

// runListGroups prints all saved groups to stdout.
func runListGroups() {
	groups, err := config.ListGroups()
	if err != nil {
		fmt.Fprintf(os.Stderr, "xssh: list-groups: %v\n", err)
		os.Exit(1)
	}
	if len(groups) == 0 {
		fmt.Println("(no groups saved — use --save NAME hosts… to create one)")
		return
	}
	for name, targets := range groups {
		fmt.Printf("%-20s %s\n", name, strings.Join(targets, " "))
	}
}

// runListHosts prints all SSH config aliases to stdout.
func runListHosts() {
	hosts, err := config.ListHosts()
	if err != nil || len(hosts) == 0 {
		fmt.Println("(no SSH config hosts found)")
		return
	}
	for _, h := range hosts {
		line := h.Alias
		if h.HostName != h.Alias {
			line += "\t→ " + h.HostName
		}
		if h.User != "" {
			line += "\t(" + h.User + ")"
		}
		fmt.Println(line)
	}
}

// runSaveGroup saves a group with the given targets.
func runSaveGroup(name string, targets []string) {
	if len(targets) == 0 {
		fmt.Fprintf(os.Stderr, "xssh: --save %q: no targets specified\n", name)
		os.Exit(1)
	}
	if err := config.SaveGroup(name, targets); err != nil {
		fmt.Fprintf(os.Stderr, "xssh: save group %q: %v\n", name, err)
		os.Exit(1)
	}
	fmt.Printf("Saved group %q with %d hosts\n", name, len(targets))
}

// runGroup loads a saved group and launches the TUI.
func runGroup(name string, borderMode app.BorderMode) {
	targets, err := config.LoadGroup(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "xssh: load group %q: %v\n", name, err)
		os.Exit(1)
	}
	if len(targets) == 0 {
		fmt.Fprintf(os.Stderr, "xssh: group %q not found (use --save to create groups)\n", name)
		os.Exit(1)
	}
	runTUI(targets, borderMode)
}

// runTUI launches the main TUI application. If targets is empty, the
// interactive host selector runs first.
func runTUI(targets []string, borderMode app.BorderMode) {
	if len(targets) == 0 {
		chosen, err := selector.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "xssh: selector error: %v\n", err)
			os.Exit(1)
		}
		if len(chosen) == 0 {
			// User cancelled
			return
		}
		targets = chosen
	}
	if err := app.Run(targets, borderMode); err != nil {
		fmt.Fprintf(os.Stderr, "xssh: %v\n", err)
		os.Exit(1)
	}
}

