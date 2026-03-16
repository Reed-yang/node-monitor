package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/siyuan/node-monitor/internal/config"
	"github.com/siyuan/node-monitor/internal/slurm"
	sshpool "github.com/siyuan/node-monitor/internal/ssh"
	"github.com/siyuan/node-monitor/internal/tui"
	"github.com/siyuan/node-monitor/internal/tui/components"
)

var appVersion string

var (
	flagNodes     string
	flagGroup     string
	flagInterval  float64
	flagWorkers   int
	flagCompact   bool
	flagStatic    bool
	flagProcesses bool
	flagDebug     bool
)

var rootCmd = &cobra.Command{
	Use:   "node-monitor",
	Short: "GPU Cluster Monitor - Monitor GPU resources across Slurm nodes",
	Long: `A beautiful, interactive terminal dashboard for monitoring GPU utilization
and memory usage across Slurm cluster nodes in real-time.

Examples:
  node-monitor                        Auto-detect Slurm nodes
  node-monitor -n visko-1,visko-2     Monitor specific nodes
  node-monitor -c                     Compact table view
  node-monitor -c -p                  Compact + show processes
  node-monitor -s                     Print once and exit
  node-monitor --group train          Monitor a node group`,
	RunE: run,
}

func init() {
	rootCmd.Flags().StringVarP(&flagNodes, "nodes", "n", "", "Comma-separated list of nodes")
	rootCmd.Flags().StringVarP(&flagGroup, "group", "g", "", "Node group from config file")
	rootCmd.Flags().Float64VarP(&flagInterval, "interval", "i", 0, "Refresh interval in seconds")
	rootCmd.Flags().IntVarP(&flagWorkers, "workers", "w", 0, "Max parallel SSH connections")
	rootCmd.Flags().BoolVarP(&flagCompact, "compact", "c", false, "Start in compact view mode")
	rootCmd.Flags().BoolVarP(&flagStatic, "static", "s", false, "Print once and exit (no TUI)")
	rootCmd.Flags().BoolVarP(&flagProcesses, "processes", "p", false, "Show GPU processes")
	rootCmd.Flags().BoolVarP(&flagDebug, "debug", "d", false, "Verbose SSH error output")
}

func run(cmd *cobra.Command, args []string) error {
	cfg := config.Load()

	// CLI flags override config
	if cmd.Flags().Changed("interval") {
		cfg.Interval = flagInterval
	}
	if cmd.Flags().Changed("workers") {
		cfg.Workers = flagWorkers
	}
	if flagCompact {
		cfg.View = "compact"
	}
	if flagProcesses {
		cfg.Processes = true
	}
	if flagDebug {
		cfg.Debug = true
	}
	if flagStatic {
		cfg.Static = true
	}

	// Resolve nodes
	var hosts []string
	if flagNodes != "" {
		hosts = slurm.ParseNodeList(flagNodes)
	} else {
		hosts = cfg.ResolveNodes(flagGroup)
	}

	// If no nodes configured, try Slurm auto-detection
	if len(hosts) == 0 {
		fmt.Println("🔍 Detecting Slurm nodes...")
		detected, err := slurm.DetectNodes()
		if err != nil {
			return fmt.Errorf("no nodes specified and Slurm detection failed: %w\nUse --nodes to specify nodes manually", err)
		}
		hosts = detected
		fmt.Printf("✓ Found %d nodes: %v\n", len(hosts), hosts)
	}

	// Create SSH pool
	pool := sshpool.NewPool(cfg.SSH.ConnectTimeout, cfg.SSH.User, cfg.SSH.IdentityFile)
	defer pool.Close()

	// Static mode: query once and print
	if cfg.Static {
		results := pool.QueryAllNodes(hosts, cfg.SSH.CommandTimeout, cfg.Debug, cfg.Workers)

		termWidth := 120
		if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
			termWidth = w
		}

		header := components.RenderHeader(results, cfg.Interval, termWidth)
		fmt.Println(header)
		fmt.Println()

		viewMode := tui.ViewPanel
		if cfg.View == "compact" {
			viewMode = tui.ViewCompact
		}
		if viewMode == tui.ViewCompact {
			fmt.Println(components.RenderCompactView(results, -1, termWidth, cfg.Processes))
		} else {
			fmt.Println(components.RenderPanelView(results, -1, termWidth))
		}
		return nil
	}

	// Dashboard mode
	viewMode := tui.ViewPanel
	if cfg.View == "compact" {
		viewMode = tui.ViewCompact
	}

	model := tui.NewModel(
		hosts,
		pool,
		cfg.Interval,
		cfg.SSH.CommandTimeout,
		cfg.Debug,
		cfg.Processes,
		viewMode,
		cfg.Groups,
	)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}

func Execute(version string) {
	appVersion = version
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
