package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ch55secake/apollo/internal/dashboard"
	"github.com/ch55secake/apollo/internal/prometheus"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Options struct {
	RefreshInterval    time.Duration
	DefaultTimeRange   string
	MaxDataPoints      int
	DashboardSource    string
	DashboardEndpoint  string
	DashboardPath      string
	PrometheusEndpoint string
}

type screen int

const (
	mainMenuScreen screen = iota
	dashboardListScreen
	dashboardDetailScreen
	queryScreen
	connectionScreen
	helpScreen
)

type menuAction int

const (
	browseDashboardsAction menuAction = iota
	loadDashboardAction
	connectionStatusAction
	helpAction
	quitAction
)

type menuItem struct {
	key         string
	title       string
	description string
	action      menuAction
}

var menuItems = []menuItem{
	{key: "1", title: "Browse dashboards", description: "Open the Grafana dashboard catalog", action: browseDashboardsAction},
	{key: "2", title: "Load JSON path", description: "Load a dashboard file or directory", action: loadDashboardAction},
	{key: "3", title: "Connection status", description: "Inspect dashboard and Prometheus links", action: connectionStatusAction},
	{key: "4", title: "Help and shortcuts", description: "Learn how to navigate Apollo", action: helpAction},
	{key: "q", title: "Quit Apollo", description: "Leave the observability console", action: quitAction},
}

type Model struct {
	source  dashboard.Source
	querier prometheus.Querier
	options Options

	screen          screen
	width           int
	height          int
	menuIndex       int
	list            list.Model
	loadInput       textinput.Model
	dashboardScroll viewport.Model
	queryScroll     viewport.Model
	helpScroll      viewport.Model

	listLoading bool
	listError   error
	summaries   []dashboard.DashboardSummary
	loadMode    bool

	selectedSummary dashboard.DashboardSummary
	dashboard       *dashboard.Dashboard
	detailLoading   bool
	detailError     error
	selectedPanel   int
	selectedTarget  int

	healthLoading    bool
	healthChecked    bool
	healthError      error
	healthGeneration uint64

	listGeneration          uint64
	dashboardLoadGeneration uint64
	generation              uint64
	queryResults            map[string]prometheus.Result
	queryErrors             map[string]error
}

type dashboardItem struct {
	summary dashboard.DashboardSummary
}

func (i dashboardItem) Title() string       { return i.summary.Title }
func (i dashboardItem) Description() string { return strings.Join(i.summary.Tags, ", ") }
func (i dashboardItem) FilterValue() string {
	return i.summary.Title + " " + strings.Join(i.summary.Tags, " ")
}

type dashboardsLoadedMsg struct {
	summaries  []dashboard.DashboardSummary
	err        error
	generation uint64
}

type dashboardLoadedMsg struct {
	dashboard  dashboard.Dashboard
	err        error
	generation uint64
}

type healthLoadedMsg struct {
	err        error
	generation uint64
}

type queryResultMsg struct {
	key        string
	generation uint64
	result     prometheus.Result
	err        error
}

type refreshMsg struct {
	generation uint64
}

func New(source dashboard.Source, querier prometheus.Querier, options Options) Model {
	if options.RefreshInterval <= 0 {
		options.RefreshInterval = 30 * time.Second
	}
	if options.DefaultTimeRange == "" {
		options.DefaultTimeRange = "now-6h"
	}
	if options.MaxDataPoints < 2 {
		options.MaxDataPoints = 120
	}

	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(2)
	delegate.SetSpacing(1)
	delegate.Styles.NormalTitle = apolloTheme.ListTitle
	delegate.Styles.NormalDesc = apolloTheme.ListDescription
	delegate.Styles.SelectedTitle = apolloTheme.ListSelectedTitle
	delegate.Styles.SelectedDesc = apolloTheme.ListSelectedDescription
	delegate.Styles.DimmedTitle = apolloTheme.ListDescription
	delegate.Styles.DimmedDesc = apolloTheme.ListDescription

	dashboardList := list.New(nil, delegate, 0, 0)
	dashboardList.Title = "Dashboard catalog"
	dashboardList.Styles.Title = apolloTheme.Section
	dashboardList.Styles.PaginationStyle = apolloTheme.Muted
	dashboardList.Styles.HelpStyle = apolloTheme.Muted
	dashboardList.SetStatusBarItemName("dashboard", "dashboards")

	loadInput := textinput.New()
	loadInput.Prompt = "Path: "
	loadInput.Placeholder = "path to a dashboard JSON file or directory"
	loadInput.CharLimit = 4096

	return Model{
		source:          source,
		querier:         querier,
		options:         options,
		screen:          mainMenuScreen,
		list:            dashboardList,
		loadInput:       loadInput,
		dashboardScroll: viewport.New(0, 0),
		queryScroll:     viewport.New(0, 0),
		helpScroll:      viewport.New(0, 0),
		listLoading:     true,
		healthLoading:   true,
		queryResults:    make(map[string]prometheus.Result),
		queryErrors:     make(map[string]error),
	}
}

func (m Model) Init() tea.Cmd {
	return commandBatch(m.loadDashboardsCmd(), m.checkPrometheusCmd())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typed.Width
		m.height = typed.Height
		m.layoutComponents()
		if m.dashboard != nil {
			m.updateDashboardScroll()
		}
		if m.screen == queryScreen {
			m.queryScroll.SetContent(m.queryContent())
		}
		return m, nil

	case dashboardsLoadedMsg:
		if typed.generation != m.listGeneration {
			return m, nil
		}
		m.listLoading = false
		m.listError = typed.err
		m.summaries = typed.summaries
		items := make([]list.Item, 0, len(typed.summaries))
		for _, summary := range typed.summaries {
			items = append(items, dashboardItem{summary: summary})
		}
		return m, m.list.SetItems(items)

	case dashboardLoadedMsg:
		if typed.generation != m.dashboardLoadGeneration {
			return m, nil
		}
		m.detailLoading = false
		m.detailError = typed.err
		if typed.err != nil {
			return m, nil
		}
		m.dashboard = &typed.dashboard
		m.selectedPanel = 0
		m.selectedTarget = 0
		m.queryResults = make(map[string]prometheus.Result)
		m.queryErrors = make(map[string]error)
		m.generation++
		m.updateDashboardScroll()
		commands := append(m.queryCommands(), m.scheduleRefresh(m.generation))
		return m, commandBatch(commands...)

	case healthLoadedMsg:
		if typed.generation != m.healthGeneration {
			return m, nil
		}
		m.healthLoading = false
		m.healthChecked = true
		m.healthError = typed.err
		return m, nil

	case queryResultMsg:
		if typed.generation != m.generation {
			return m, nil
		}
		if typed.err != nil {
			m.queryErrors[typed.key] = typed.err
		} else {
			m.queryResults[typed.key] = typed.result
			delete(m.queryErrors, typed.key)
		}
		m.updateDashboardScroll()
		if m.screen == queryScreen {
			m.queryScroll.SetContent(m.queryContent())
		}
		return m, nil

	case refreshMsg:
		if typed.generation != m.generation || m.dashboard == nil {
			return m, nil
		}
		return m, m.startQueryRefresh(true)
	}

	switch m.screen {
	case mainMenuScreen:
		return m.updateMenu(msg)
	case dashboardListScreen:
		return m.updateList(msg)
	case dashboardDetailScreen:
		return m.updateDashboard(msg)
	case queryScreen:
		return m.updateQuery(msg)
	case connectionScreen:
		return m.updateConnection(msg)
	case helpScreen:
		return m.updateHelp(msg)
	default:
		return m, nil
	}
}

func (m Model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			m.menuIndex = max(0, m.menuIndex-1)
		case "down", "j":
			m.menuIndex = min(len(menuItems)-1, m.menuIndex+1)
		case "1", "2", "3", "4":
			m.menuIndex = int(key.String()[0] - '1')
			return m.activateMenu()
		case "enter":
			return m.activateMenu()
		}
	}
	return m, nil
}

func (m Model) activateMenu() (tea.Model, tea.Cmd) {
	if m.menuIndex < 0 || m.menuIndex >= len(menuItems) {
		return m, nil
	}
	switch menuItems[m.menuIndex].action {
	case browseDashboardsAction:
		m.screen = dashboardListScreen
		return m, nil
	case loadDashboardAction:
		m.screen = dashboardListScreen
		m.startLoadMode()
		return m, textinput.Blink
	case connectionStatusAction:
		m.screen = connectionScreen
		m.healthGeneration++
		m.healthLoading = true
		return m, m.checkPrometheusCmd()
	case helpAction:
		m.screen = helpScreen
		return m, nil
	case quitAction:
		return m, tea.Quit
	default:
		return m, nil
	}
}

func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.loadMode {
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.loadMode = false
				m.loadInput.Blur()
				m.loadInput.Reset()
				m.layoutComponents()
				return m, nil
			case "enter":
				path := strings.TrimSpace(m.loadInput.Value())
				if path == "" {
					m.listError = fmt.Errorf("dashboard path must not be empty")
					return m, nil
				}
				source, err := dashboard.NewFileSource(path)
				if err != nil {
					m.listError = err
					return m, nil
				}
				m.source = source
				m.listGeneration++
				m.loadMode = false
				m.loadInput.Blur()
				m.loadInput.Reset()
				m.listLoading = true
				m.listError = nil
				m.summaries = nil
				m.list.SetItems(nil)
				m.layoutComponents()
				return m, m.loadDashboardsCmd()
			}
		}
		var cmd tea.Cmd
		m.loadInput, cmd = m.loadInput.Update(msg)
		return m, cmd
	}

	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc", "backspace":
			m.screen = mainMenuScreen
			return m, nil
		case "r":
			if !m.list.SettingFilter() {
				m.listGeneration++
				m.listLoading = true
				m.listError = nil
				return m, m.loadDashboardsCmd()
			}
		case "l":
			if !m.list.SettingFilter() {
				m.startLoadMode()
				return m, textinput.Blink
			}
		case "enter":
			if !m.list.SettingFilter() {
				item, ok := m.list.SelectedItem().(dashboardItem)
				if ok {
					m.selectedSummary = item.summary
					m.dashboardLoadGeneration++
					m.detailLoading = true
					m.detailError = nil
					m.dashboard = nil
					m.screen = dashboardDetailScreen
					return m, m.loadDashboardCmd()
				}
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) updateDashboard(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc", "backspace":
			m.dashboardLoadGeneration++
			m.screen = dashboardListScreen
			m.dashboard = nil
			m.detailLoading = false
			return m, nil
		case "up", "k":
			if m.movePanel(-1) {
				m.updateDashboardScroll()
			}
		case "down", "j":
			if m.movePanel(1) {
				m.updateDashboardScroll()
			}
		case "enter":
			if m.dashboard != nil && m.selectedPanel >= 0 && m.selectedPanel < len(m.dashboard.Panels) {
				panel := m.dashboard.Panels[m.selectedPanel]
				if len(panel.Targets) > 0 {
					m.selectedTarget = min(m.selectedTarget, len(panel.Targets)-1)
					m.screen = queryScreen
					m.queryScroll.SetContent(m.queryContent())
				}
			}
		case "r":
			if m.dashboard != nil {
				return m, m.startQueryRefresh(false)
			}
			if m.selectedSummary.ID != "" {
				m.dashboardLoadGeneration++
				m.detailLoading = true
				m.detailError = nil
				return m, m.loadDashboardCmd()
			}
		}
	}
	var cmd tea.Cmd
	m.dashboardScroll, cmd = m.dashboardScroll.Update(msg)
	return m, cmd
}

func (m Model) updateQuery(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc", "backspace":
			m.screen = dashboardDetailScreen
			return m, nil
		case "left", "h":
			if m.selectedTarget > 0 {
				m.selectedTarget--
				m.queryScroll.SetContent(m.queryContent())
			}
		case "right", "l", "tab":
			if m.dashboard != nil && m.selectedPanel >= 0 && m.selectedPanel < len(m.dashboard.Panels) {
				if m.selectedTarget < len(m.dashboard.Panels[m.selectedPanel].Targets)-1 {
					m.selectedTarget++
					m.queryScroll.SetContent(m.queryContent())
				}
			}
		case "r":
			if m.dashboard != nil {
				return m, m.startQueryRefresh(false)
			}
		}
	}
	var cmd tea.Cmd
	m.queryScroll, cmd = m.queryScroll.Update(msg)
	return m, cmd
}

func (m Model) updateConnection(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc", "backspace":
			m.screen = mainMenuScreen
			return m, nil
		case "r":
			m.healthGeneration++
			m.healthLoading = true
			m.healthError = nil
			m.listGeneration++
			m.listLoading = true
			m.listError = nil
			return m, commandBatch(m.loadDashboardsCmd(), m.checkPrometheusCmd())
		}
	}
	return m, nil
}

func (m Model) updateHelp(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc", "backspace":
			m.screen = mainMenuScreen
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.helpScroll, cmd = m.helpScroll.Update(msg)
	return m, cmd
}

func (m *Model) startLoadMode() {
	m.loadMode = true
	m.listError = nil
	m.loadInput.Reset()
	m.loadInput.Focus()
	m.layoutComponents()
}

func (m *Model) layoutComponents() {
	width := m.bodyContentWidth()
	height := m.bodyContentHeight()
	listHeight := height
	if m.loadMode {
		listHeight = max(1, listHeight-4)
	}
	m.list.SetSize(width, listHeight)
	m.loadInput.Width = width
	m.dashboardScroll.Width = width
	m.dashboardScroll.Height = height
	m.queryScroll.Width = width
	m.queryScroll.Height = height
	m.helpScroll.Width = min(width, 64)
	m.helpScroll.Height = height
	m.helpScroll.SetContent(m.helpContent())
}

func (m Model) loadDashboardsCmd() tea.Cmd {
	source := m.source
	generation := m.listGeneration
	return func() tea.Msg {
		if source == nil {
			return dashboardsLoadedMsg{err: fmt.Errorf("dashboard source is not configured"), generation: generation}
		}
		summaries, err := source.List(context.Background())
		return dashboardsLoadedMsg{summaries: summaries, err: err, generation: generation}
	}
}

func (m Model) loadDashboardCmd() tea.Cmd {
	source := m.source
	id := m.selectedSummary.ID
	generation := m.dashboardLoadGeneration
	return func() tea.Msg {
		if source == nil {
			return dashboardLoadedMsg{err: fmt.Errorf("dashboard source is not configured"), generation: generation}
		}
		loaded, err := source.Get(context.Background(), id)
		return dashboardLoadedMsg{dashboard: loaded, err: err, generation: generation}
	}
}

func (m Model) checkPrometheusCmd() tea.Cmd {
	querier := m.querier
	generation := m.healthGeneration
	return func() tea.Msg {
		if querier == nil {
			return healthLoadedMsg{err: fmt.Errorf("prometheus client is not configured"), generation: generation}
		}
		_, err := querier.Query(context.Background(), prometheus.QueryRequest{
			Expr:    "vector(1)",
			End:     time.Now(),
			Instant: true,
		})
		return healthLoadedMsg{err: err, generation: generation}
	}
}

func (m *Model) startQueryRefresh(schedule bool) tea.Cmd {
	m.generation++
	m.queryResults = make(map[string]prometheus.Result)
	m.queryErrors = make(map[string]error)
	commands := m.queryCommands()
	if schedule {
		commands = append(commands, m.scheduleRefresh(m.generation))
	}
	return commandBatch(commands...)
}

func (m Model) queryCommands() []tea.Cmd {
	if m.dashboard == nil || m.querier == nil {
		return nil
	}
	timeRange := m.dashboard.Time
	if timeRange.From == "" {
		timeRange.From = m.options.DefaultTimeRange
	}
	if timeRange.To == "" {
		timeRange.To = "now"
	}
	start, end, err := timeRange.Resolve(time.Now())
	if err != nil {
		fallback := dashboard.TimeRange{From: m.options.DefaultTimeRange, To: "now"}
		start, end, err = fallback.Resolve(time.Now())
		if err != nil {
			return nil
		}
	}
	commands := make([]tea.Cmd, 0)
	generation := m.generation
	for panelIndex, panel := range m.dashboard.Panels {
		for targetIndex, target := range panel.Targets {
			if strings.TrimSpace(target.Expr) == "" || !target.Datasource.IsPrometheus() {
				continue
			}
			step := end.Sub(start) / time.Duration(m.options.MaxDataPoints)
			if panel.MaxDataPoints > 1 && panel.MaxDataPoints < m.options.MaxDataPoints {
				step = end.Sub(start) / time.Duration(panel.MaxDataPoints)
			}
			if step < time.Second {
				step = time.Second
			}
			request := prometheus.QueryRequest{
				Expr:    dashboard.ExpandQuery(target.Expr, m.dashboard.Variables),
				Start:   start,
				End:     end,
				Step:    step,
				Instant: target.Instant && !target.Range,
			}
			key := queryKey(panelIndex, targetIndex)
			querier := m.querier
			commands = append(commands, func() tea.Msg {
				result, err := querier.Query(context.Background(), request)
				return queryResultMsg{key: key, generation: generation, result: result, err: err}
			})
		}
	}
	return commands
}

func (m Model) scheduleRefresh(generation uint64) tea.Cmd {
	interval := m.options.RefreshInterval
	return tea.Tick(interval, func(time.Time) tea.Msg { return refreshMsg{generation: generation} })
}

func (m Model) dashboardContent() string {
	if m.dashboard == nil {
		return ""
	}
	return renderPanelRows(m)
}

func commandBatch(commands ...tea.Cmd) tea.Cmd {
	active := make([]tea.Cmd, 0, len(commands))
	for _, command := range commands {
		if command != nil {
			active = append(active, command)
		}
	}
	if len(active) == 0 {
		return nil
	}
	return tea.Batch(active...)
}

func queryKey(panelIndex, targetIndex int) string {
	return fmt.Sprintf("panel-%d-target-%d", panelIndex, targetIndex)
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}
