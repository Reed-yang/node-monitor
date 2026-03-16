package tui

import (
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/siyuan/node-monitor/internal/model"
	sshpool "github.com/siyuan/node-monitor/internal/ssh"
	"github.com/siyuan/node-monitor/internal/tui/components"
)

// View modes
type ViewMode int

const (
	ViewPanel ViewMode = iota
	ViewCompact
)

// App state
type AppView int

const (
	AppList AppView = iota
	AppDetail
)

// Messages
type tickMsg time.Time
type nodesUpdatedMsg []model.NodeStatus
type detailResultMsg struct {
	node   model.NodeStatus
	system *model.SystemInfo
}

// Sort modes
type SortMode int

const (
	SortName SortMode = iota
	SortUtil
	SortMemory
)

// Model is the Bubble Tea model.
type Model struct {
	// Data
	nodes      []model.NodeStatus
	detailNode *model.NodeStatus
	detailSys  *model.SystemInfo

	// Config
	allHosts      []string
	hosts         []string
	pool          *sshpool.Pool
	interval      time.Duration
	cmdTimeout    int
	debug         bool
	showProcesses bool

	// UI state
	viewMode    ViewMode
	appView     AppView
	selectedIdx int
	sortMode    SortMode
	showHelp    bool
	searchQuery string
	searching   bool
	width       int
	height      int

	// Groups
	groups       map[string][]string
	groupNames   []string
	currentGroup int
}

// NewModel creates a new TUI model.
func NewModel(
	hosts []string,
	pool *sshpool.Pool,
	interval float64,
	cmdTimeout int,
	debug bool,
	showProcesses bool,
	viewMode ViewMode,
	groups map[string][]string,
) Model {
	var groupNames []string
	for name := range groups {
		groupNames = append(groupNames, name)
	}
	sort.Strings(groupNames)

	return Model{
		allHosts:      hosts,
		hosts:         hosts,
		pool:          pool,
		interval:      time.Duration(interval * float64(time.Second)),
		cmdTimeout:    cmdTimeout,
		debug:         debug,
		showProcesses: showProcesses,
		viewMode:      viewMode,
		appView:       AppList,
		selectedIdx:   0,
		currentGroup:  -1,
		groups:        groups,
		groupNames:    groupNames,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.queryNodes(),
		m.tick(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tickMsg:
		return m, tea.Batch(m.queryNodes(), m.tick())

	case nodesUpdatedMsg:
		m.nodes = sortNodes([]model.NodeStatus(msg), m.sortMode)
		if m.selectedIdx >= len(m.nodes) {
			m.selectedIdx = len(m.nodes) - 1
		}
		if m.selectedIdx < 0 {
			m.selectedIdx = 0
		}
		return m, nil

	case detailResultMsg:
		n := msg.node
		m.detailNode = &n
		m.detailSys = msg.system
		return m, nil
	}

	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var view string

	header := components.RenderHeader(m.nodes, m.interval.Seconds(), m.width)

	switch m.appView {
	case AppList:
		var content string
		switch m.viewMode {
		case ViewPanel:
			content = components.RenderPanelView(m.nodes, m.selectedIdx, m.width)
		case ViewCompact:
			content = components.RenderCompactView(m.nodes, m.selectedIdx, m.width, m.showProcesses)
		}

		footer := ""
		if m.searching {
			footer = "\n  Search: " + m.searchQuery + "█"
		}

		view = header + "\n\n" + content + footer

	case AppDetail:
		if m.detailNode != nil {
			view = header + "\n\n" + components.RenderNodeDetail(*m.detailNode, m.detailSys, m.width)
		} else {
			view = header + "\n\n  Loading detail..."
		}
	}

	if m.showHelp {
		return components.RenderHelp(m.width, m.height)
	}

	return view
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.searching {
		switch msg.Type {
		case tea.KeyEscape:
			m.searching = false
			m.searchQuery = ""
			return m, nil
		case tea.KeyEnter:
			m.searching = false
			return m, nil
		case tea.KeyBackspace:
			if len(m.searchQuery) > 0 {
				m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			}
			return m, nil
		default:
			if len(msg.String()) == 1 {
				m.searchQuery += msg.String()
			}
			return m, nil
		}
	}

	switch msg.String() {
	case "q", "ctrl+c":
		if m.appView == AppDetail {
			m.appView = AppList
			m.detailNode = nil
			m.detailSys = nil
			return m, nil
		}
		return m, tea.Quit

	case "esc":
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}
		if m.appView == AppDetail {
			m.appView = AppList
			m.detailNode = nil
			m.detailSys = nil
			return m, nil
		}
		return m, tea.Quit

	case "up", "k":
		if m.appView == AppList && m.selectedIdx > 0 {
			m.selectedIdx--
		}
		return m, nil

	case "down", "j":
		if m.appView == AppList && m.selectedIdx < len(m.nodes)-1 {
			m.selectedIdx++
		}
		return m, nil

	case "enter":
		if m.appView == AppList && len(m.nodes) > 0 && m.selectedIdx < len(m.nodes) {
			node := m.nodes[m.selectedIdx]
			if node.IsOnline() {
				m.appView = AppDetail
				m.detailNode = nil
				m.detailSys = nil
				return m, m.queryDetail(node.Hostname)
			}
		}
		return m, nil

	case "tab":
		if m.viewMode == ViewPanel {
			m.viewMode = ViewCompact
		} else {
			m.viewMode = ViewPanel
		}
		return m, nil

	case "p":
		m.showProcesses = !m.showProcesses
		return m, nil

	case "s":
		m.sortMode = (m.sortMode + 1) % 3
		m.nodes = sortNodes(m.nodes, m.sortMode)
		return m, nil

	case "g":
		if len(m.groupNames) > 0 {
			m.currentGroup++
			if m.currentGroup >= len(m.groupNames) {
				m.currentGroup = -1
			}
			if m.currentGroup >= 0 {
				m.hosts = m.groups[m.groupNames[m.currentGroup]]
			} else {
				m.hosts = m.allHosts
			}
			m.selectedIdx = 0
		}
		return m, nil

	case "/":
		m.searching = true
		m.searchQuery = ""
		return m, nil

	case "?":
		m.showHelp = !m.showHelp
		return m, nil
	}

	return m, nil
}

// Commands

func (m Model) tick() tea.Cmd {
	return tea.Tick(m.interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) queryNodes() tea.Cmd {
	return func() tea.Msg {
		hosts := m.hosts
		if m.searching && m.searchQuery != "" {
			var filtered []string
			for _, h := range hosts {
				if strings.Contains(h, m.searchQuery) {
					filtered = append(filtered, h)
				}
			}
			hosts = filtered
		}
		results := m.pool.QueryAllNodes(hosts, m.cmdTimeout, m.debug, len(hosts))
		return nodesUpdatedMsg(results)
	}
}

func (m Model) queryDetail(hostname string) tea.Cmd {
	return func() tea.Msg {
		node, sys := m.pool.QueryNodeDetail(hostname, m.cmdTimeout, m.debug)
		return detailResultMsg{node: node, system: sys}
	}
}

func sortNodes(nodes []model.NodeStatus, mode SortMode) []model.NodeStatus {
	sorted := make([]model.NodeStatus, len(nodes))
	copy(sorted, nodes)

	switch mode {
	case SortName:
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Hostname < sorted[j].Hostname
		})
	case SortUtil:
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].AvgUtilization() > sorted[j].AvgUtilization()
		})
	case SortMemory:
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].TotalMemoryUsed() > sorted[j].TotalMemoryUsed()
		})
	}

	return sorted
}
