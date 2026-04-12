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
	viewMode     ViewMode // unused, kept for API
	bottomPanel  BottomPanel
	selectedIdx  int
	sortMode     SortMode
	showHelp     bool
	searchQuery  string
	searching    bool
	width        int
	height       int
	expanded     bool              // cards show inline processes
	displayNames map[string]string // hostname -> short display name

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

	displayNames := components.ComputeDisplayNames(hosts)

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
		expanded:      true,
		displayNames:  displayNames,
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

	innerWidth := m.width - 2

	header := components.RenderHeader(m.nodes, m.interval.Seconds(), innerWidth)
	nodeGrid := components.RenderNodeGrid(m.nodes, m.selectedIdx, innerWidth, m.displayNames, m.expanded)

	var bottomTitle string
	var bottomContent string

	if m.bottomPanel == PanelDetail && m.detailNode != nil {
		bottomTitle = m.detailNode.Hostname + " Detail"
		bottomContent = components.RenderNodeDetail(*m.detailNode, m.detailSys, innerWidth)
	}

	var bodyLines []string

	for _, line := range strings.Split(nodeGrid, "\n") {
		bodyLines = append(bodyLines, " "+line)
	}

	if bottomContent != "" {
		bodyLines = append(bodyLines, "")
		divLine := components.RenderDivider(bottomTitle, m.width)
		bodyLines = append(bodyLines, divLine)
		for _, line := range strings.Split(bottomContent, "\n") {
			bodyLines = append(bodyLines, " "+line)
		}
	}

	if m.searching {
		searchLine := " Search: " + m.searchQuery + "█"
		bodyLines = append(bodyLines, searchLine)
	}

	body := strings.Join(bodyLines, "\n")

	return components.RenderOuterFrame(header, body, m.width, m.height)
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
		m.expanded = !m.expanded
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
			m.displayNames = components.ComputeDisplayNames(m.hosts)
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

	gridStartY := 2

	clickX := msg.X - 1
	clickY := msg.Y - gridStartY

	if clickX < 0 || clickY < 0 {
		return m, nil
	}

	col := clickX / cardWidth
	if col >= numCols {
		col = numCols - 1
	}

	// Walk rows to find which row was clicked
	nodeIdx := -1
	currentY := 0
	for rowStart := 0; rowStart < len(m.nodes); rowStart += numCols {
		rowEnd := rowStart + numCols
		if rowEnd > len(m.nodes) {
			rowEnd = len(m.nodes)
		}

		maxHeight := 6
		if m.expanded {
			for i := rowStart; i < rowEnd; i++ {
				procCount := len(m.nodes[i].ActiveUsers())
				h := 6 + procCount
				if h > maxHeight {
					maxHeight = h
				}
			}
		}

		if clickY >= currentY && clickY < currentY+maxHeight {
			idx := rowStart + col
			if idx < rowEnd {
				nodeIdx = idx
			}
			break
		}
		currentY += maxHeight
	}

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
