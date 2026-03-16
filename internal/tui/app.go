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

// ViewMode kept for API compatibility but only one mode now.
type ViewMode int

const (
	ViewPanel   ViewMode = iota
	ViewCompact          // kept for backward compat, ignored
)

// Bottom panel state
type BottomPanel int

const (
	PanelProcesses BottomPanel = iota
	PanelDetail
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

func (s SortMode) String() string {
	switch s {
	case SortName:
		return "name"
	case SortUtil:
		return "util"
	case SortMemory:
		return "memory"
	}
	return ""
}

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
	viewMode    ViewMode // unused, kept for API
	bottomPanel BottomPanel
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

	// Click regions (rebuilt each render)
	clickRegions []ClickRegion
}

// ClickRegion defines a clickable area.
type ClickRegion struct {
	X1, Y1, X2, Y2 int
	Action          string
	NodeIndex       int
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
		bottomPanel:   PanelProcesses,
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

	case tea.MouseMsg:
		return m.handleMouse(msg)

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

	if m.showHelp {
		return components.RenderHelp(m.width, m.height)
	}

	innerWidth := m.width - 2 // outer frame borders

	// Header
	header := components.RenderHeader(m.nodes, m.interval.Seconds(), innerWidth)

	// Node grid
	nodeGrid := components.RenderNodeGrid(m.nodes, m.selectedIdx, innerWidth)

	// Bottom panel
	var bottomTitle string
	var bottomContent string

	if m.bottomPanel == PanelDetail && m.detailNode != nil {
		bottomTitle = m.detailNode.Hostname + " Detail"
		bottomContent = components.RenderNodeDetail(*m.detailNode, m.detailSys, innerWidth)
	} else if m.showProcesses {
		bottomTitle = "Processes"
		// Calculate available rows for process table
		gridLines := strings.Count(nodeGrid, "\n") + 1
		availRows := m.height - gridLines - 8 // header + divider + borders + footer
		if availRows < 3 {
			availRows = 3
		}
		bottomContent = components.RenderProcessTable(m.nodes, innerWidth, availRows)
	}

	// Assemble body
	var bodyLines []string

	// Add node grid lines
	for _, line := range strings.Split(nodeGrid, "\n") {
		bodyLines = append(bodyLines, " "+line)
	}

	// Divider + bottom panel
	if bottomContent != "" {
		divider := components.RenderDivider(bottomTitle, m.width)
		// Strip the side borders from divider since outer frame adds them
		bodyLines = append(bodyLines, "")
		// We need to handle the divider specially — it replaces a full row in the frame
		_ = divider
		divLine := buildDivider(bottomTitle, innerWidth)
		bodyLines = append(bodyLines, divLine)

		for _, line := range strings.Split(bottomContent, "\n") {
			bodyLines = append(bodyLines, " "+line)
		}
	}

	// Search bar
	if m.searching {
		searchLine := " Search: " + m.searchQuery + "█"
		bodyLines = append(bodyLines, searchLine)
	}

	body := strings.Join(bodyLines, "\n")

	return components.RenderOuterFrame(header, body, m.width, m.height)
}

func buildDivider(title string, width int) string {
	if title == "" {
		return strings.Repeat("─", width)
	}
	prefix := "─┤ " + title + " ├"
	remaining := width - len(prefix)
	if remaining < 0 {
		remaining = 0
	}
	return prefix + strings.Repeat("─", remaining)
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
		if m.bottomPanel == PanelDetail {
			m.bottomPanel = PanelProcesses
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
		if m.bottomPanel == PanelDetail {
			m.bottomPanel = PanelProcesses
			m.detailNode = nil
			m.detailSys = nil
			return m, nil
		}
		return m, tea.Quit

	case "up", "k":
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}
		return m, nil

	case "down", "j":
		if m.selectedIdx < len(m.nodes)-1 {
			m.selectedIdx++
		}
		return m, nil

	case "enter":
		if len(m.nodes) > 0 && m.selectedIdx < len(m.nodes) {
			node := m.nodes[m.selectedIdx]
			if node.IsOnline() {
				m.bottomPanel = PanelDetail
				m.detailNode = nil
				m.detailSys = nil
				return m, m.queryDetail(node.Hostname)
			}
		}
		return m, nil

	case "tab":
		// Could toggle focus between grid and bottom panel
		// For now, no-op
		return m, nil

	case "p":
		m.showProcesses = !m.showProcesses
		if !m.showProcesses {
			m.bottomPanel = PanelProcesses
		}
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

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress {
		return m, nil
	}

	// Calculate which node card was clicked based on grid layout
	// The node grid starts at row 2 (after header border + separator)
	// Each card is approximately 6 rows high (border + 4 content + border)
	if len(m.nodes) == 0 {
		return m, nil
	}

	innerWidth := m.width - 2
	minCardWidth := 40
	numCols := innerWidth / minCardWidth
	if numCols < 1 {
		numCols = 1
	}
	if numCols > len(m.nodes) {
		numCols = len(m.nodes)
	}
	cardWidth := innerWidth / numCols

	// Approximate card height (border top + 3 content lines + border bottom = 5)
	cardHeight := 6
	gridStartY := 2 // after outer frame top border + separator line

	clickX := msg.X - 1 // adjust for outer frame left border
	clickY := msg.Y - gridStartY

	if clickX < 0 || clickY < 0 {
		return m, nil
	}

	col := clickX / cardWidth
	row := clickY / cardHeight

	if col >= numCols {
		col = numCols - 1
	}

	nodeIdx := row*numCols + col
	if nodeIdx >= 0 && nodeIdx < len(m.nodes) {
		m.selectedIdx = nodeIdx
		node := m.nodes[nodeIdx]
		if node.IsOnline() {
			m.bottomPanel = PanelDetail
			m.detailNode = nil
			m.detailSys = nil
			return m, m.queryDetail(node.Hostname)
		}
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
