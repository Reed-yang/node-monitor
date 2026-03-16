package cmd

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/siyuan/node-monitor/internal/config"
	"github.com/siyuan/node-monitor/internal/model"
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
  node-monitor -s                     Print once and exit
  node-monitor --group train          Monitor a node group`,
	RunE: run,
}

func init() {
	rootCmd.Flags().StringVarP(&flagNodes, "nodes", "n", "", "Comma-separated list of nodes")
	rootCmd.Flags().StringVarP(&flagGroup, "group", "g", "", "Node group from config file")
	rootCmd.Flags().Float64VarP(&flagInterval, "interval", "i", 0, "Refresh interval in seconds")
	rootCmd.Flags().IntVarP(&flagWorkers, "workers", "w", 0, "Max parallel SSH connections")
	rootCmd.Flags().BoolVarP(&flagStatic, "static", "s", false, "Print once and exit (no TUI)")
	rootCmd.Flags().BoolVarP(&flagProcesses, "processes", "p", false, "Toggle processes off (default: on)")
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
	if flagDebug {
		cfg.Debug = true
	}
	if flagStatic {
		cfg.Static = true
	}
	// Processes default to true now
	showProcs := true
	if cmd.Flags().Changed("processes") {
		showProcs = !flagProcesses // -p toggles off
	}

	// Resolve nodes
	var hosts []string
	if flagNodes != "" {
		hosts = slurm.ParseNodeList(flagNodes)
	} else {
		hosts = cfg.ResolveNodes(flagGroup)
	}

	if len(hosts) == 0 {
		fmt.Println("🔍 Detecting Slurm nodes...")
		detected, err := slurm.DetectNodes()
		if err != nil {
			return fmt.Errorf("no nodes specified and Slurm detection failed: %w\nUse --nodes to specify nodes manually", err)
		}
		hosts = detected
		fmt.Printf("✓ Found %d nodes: %v\n", len(hosts), hosts)
	}

	pool := sshpool.NewPool(cfg.SSH.ConnectTimeout, cfg.SSH.User, cfg.SSH.IdentityFile)
	defer pool.Close()

	// Static mode
	if cfg.Static {
		results := pool.QueryAllNodes(hosts, cfg.SSH.CommandTimeout, cfg.Debug, cfg.Workers)
		termWidth := 120
		if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
			termWidth = w
		}
		renderStatic(results, cfg.Interval, termWidth, showProcs)
		return nil
	}

	// Dashboard mode with mouse support
	m := tui.NewModel(
		hosts,
		pool,
		cfg.Interval,
		cfg.SSH.CommandTimeout,
		cfg.Debug,
		showProcs,
		tui.ViewPanel,
		cfg.Groups,
	)

	p := tea.NewProgram(m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}

func renderStatic(results []model.NodeStatus, interval float64, width int, showProcs bool) {
	// Header line
	header := components.RenderHeader(results, interval, width)
	fmt.Println(header)
	fmt.Println()

	// Node summaries (one line each)
	for _, node := range results {
		if !node.IsOnline() {
			errMsg := "Offline"
			if node.Error != nil {
				errMsg = *node.Error
			}
			name := lipgloss.NewStyle().Foreground(components.ColorRed).Render("✗ " + node.Hostname)
			fmt.Printf(" %-16s  %s\n", name,
				lipgloss.NewStyle().Foreground(components.ColorDim).Render("⚠ "+errMsg))
			continue
		}

		icon := components.NodeStatusIcon(node.AvgUtilization())
		utilBar := components.RenderGradientBar(node.AvgUtilization(), 20, components.UtilGradient)
		utilPct := lipgloss.NewStyle().Bold(true).Foreground(components.UtilColor(node.AvgUtilization())).
			Render(fmt.Sprintf("%3.0f%%", node.AvgUtilization()))
		heatmap := components.RenderGPUHeatmap(node.GPUs)
		memStr := lipgloss.NewStyle().Foreground(components.MemColor(
			float64(node.TotalMemoryUsed())/float64(node.TotalMemory())*100)).
			Render(fmt.Sprintf("%s/%s", model.FormatMemory(node.TotalMemoryUsed()), model.FormatMemory(node.TotalMemory())))

		name := lipgloss.NewStyle().Bold(true).Render(icon + " " + node.Hostname)
		fmt.Printf(" %-16s  %s %s  %s  %s  %s\n",
			name, utilBar, utilPct, memStr, heatmap, node.GPUModelSummary())
	}

	// Processes
	if showProcs {
		fmt.Println()

		// Check if any processes
		hasProcs := false
		for _, n := range results {
			if len(n.AllProcesses()) > 0 {
				hasProcs = true
				break
			}
		}

		if hasProcs {
			fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(components.ColorFg).Render("Processes:"))
			fmt.Println(components.RenderProcessTable(results, width, 0))
		}
	}
}

// renderNodeUsers returns comma-separated active users.
func renderNodeUsers(node model.NodeStatus) string {
	users := node.ActiveUsers()
	if len(users) == 0 {
		return ""
	}
	return strings.Join(users, ",")
}

func Execute(version string) {
	appVersion = version
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
