package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ch55secake/apollo/internal/dashboard"
	"github.com/ch55secake/apollo/internal/prometheus"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Options struct {
	RefreshInterval  time.Duration
	DefaultTimeRange string
	MaxDataPoints    int
}

type screen int

const (
	dashboardListScreen screen = iota
	dashboardDetailScreen
	queryScreen
)

type Model struct {
	source  dashboard.Source
	querier prometheus.Querier
	options Options

	screen      screen
	width       int
	height      int
	list        list.Model
	queryScroll viewport.Model

	listLoading bool
	listError   error
	summaries   []dashboard.DashboardSummary

	selectedSummary dashboard.DashboardSummary
	dashboard       *dashboard.Dashboard
	detailLoading   bool
	detailError     error
	selectedPanel   int
	selectedTarget  int

	generation   uint64
	queryResults map[string]prometheus.Result
	queryErrors  map[string]error
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
	summaries []dashboard.DashboardSummary
	err       error
}

type dashboardLoadedMsg struct {
	dashboard dashboard.Dashboard
	err       error
}

type queryResultMsg struct {
	key        string
	generation uint64
	result     prometheus.Result
	err        error
}

type refreshMsg struct{}

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

	dashboardList := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	dashboardList.Title = "Apollo dashboards"
	dashboardList.SetStatusBarItemName("dashboard", "dashboards")

	return Model{
		source:       source,
		querier:      querier,
		options:      options,
		list:         dashboardList,
		queryScroll:  viewport.New(0, 0),
		listLoading:  true,
		queryResults: make(map[string]prometheus.Result),
		queryErrors:  make(map[string]error),
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadDashboardsCmd()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typed.Width
		m.height = typed.Height
		m.list.SetSize(typed.Width, typed.Height)
		m.queryScroll.Width = typed.Width
		m.queryScroll.Height = max(1, typed.Height-4)
		if m.screen == queryScreen {
			m.queryScroll.SetContent(m.queryContent())
		}
		return m, nil

	case dashboardsLoadedMsg:
		m.listLoading = false
		m.listError = typed.err
		m.summaries = typed.summaries
		items := make([]list.Item, 0, len(typed.summaries))
		for _, summary := range typed.summaries {
			items = append(items, dashboardItem{summary: summary})
		}
		return m, m.list.SetItems(items)

	case dashboardLoadedMsg:
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
		return m, tea.Batch(append(m.queryCommands(), m.scheduleRefresh())...)

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
		if m.screen == queryScreen {
			m.queryScroll.SetContent(m.queryContent())
		}
		return m, nil

	case refreshMsg:
		if (m.screen != dashboardDetailScreen && m.screen != queryScreen) || m.dashboard == nil {
			return m, nil
		}
		m.generation++
		m.queryResults = make(map[string]prometheus.Result)
		m.queryErrors = make(map[string]error)
		return m, tea.Batch(append(m.queryCommands(), m.scheduleRefresh())...)
	}

	switch m.screen {
	case dashboardListScreen:
		return m.updateList(msg)
	case dashboardDetailScreen:
		return m.updateDashboard(msg)
	case queryScreen:
		return m.updateQuery(msg)
	default:
		return m, nil
	}
}

func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "r":
			if !m.list.SettingFilter() {
				m.listLoading = true
				m.listError = nil
				return m, m.loadDashboardsCmd()
			}
		case "enter":
			if !m.list.SettingFilter() {
				item, ok := m.list.SelectedItem().(dashboardItem)
				if ok {
					m.selectedSummary = item.summary
					m.detailLoading = true
					m.detailError = nil
					m.dashboard = nil
					m.screen = dashboardDetailScreen
					return m, m.loadDashboardCmd(item.summary.ID)
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
			m.screen = dashboardListScreen
			m.dashboard = nil
			return m, nil
		case "up", "k":
			if m.selectedPanel > 0 {
				m.selectedPanel--
			}
		case "down", "j":
			if m.dashboard != nil && m.selectedPanel < len(m.dashboard.Panels)-1 {
				m.selectedPanel++
			}
		case "enter":
			if m.dashboard != nil && len(m.dashboard.Panels) > 0 {
				panel := m.dashboard.Panels[m.selectedPanel]
				if len(panel.Targets) > 0 {
					m.selectedTarget = min(m.selectedTarget, len(panel.Targets)-1)
					m.screen = queryScreen
					m.queryScroll.SetContent(m.queryContent())
				}
			}
		case "r":
			if m.dashboard != nil {
				m.generation++
				m.queryResults = make(map[string]prometheus.Result)
				m.queryErrors = make(map[string]error)
				return m, tea.Batch(append(m.queryCommands(), m.scheduleRefresh())...)
			}
		}
	}
	return m, nil
}

func (m Model) updateQuery(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc", "backspace":
			m.screen = dashboardDetailScreen
			return m, nil
		case "q":
			return m, tea.Quit
		case "r":
			if m.dashboard != nil {
				m.generation++
				m.queryResults = make(map[string]prometheus.Result)
				m.queryErrors = make(map[string]error)
				return m, tea.Batch(append(m.queryCommands(), m.scheduleRefresh())...)
			}
		}
	}
	var cmd tea.Cmd
	m.queryScroll, cmd = m.queryScroll.Update(msg)
	return m, cmd
}

func (m Model) loadDashboardsCmd() tea.Cmd {
	source := m.source
	return func() tea.Msg {
		if source == nil {
			return dashboardsLoadedMsg{err: fmt.Errorf("dashboard source is not configured")}
		}
		summaries, err := source.List(context.Background())
		return dashboardsLoadedMsg{summaries: summaries, err: err}
	}
}

func (m Model) loadDashboardCmd(id string) tea.Cmd {
	source := m.source
	return func() tea.Msg {
		if source == nil {
			return dashboardLoadedMsg{err: fmt.Errorf("dashboard source is not configured")}
		}
		loaded, err := source.Get(context.Background(), id)
		return dashboardLoadedMsg{dashboard: loaded, err: err}
	}
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

func (m Model) scheduleRefresh() tea.Cmd {
	interval := m.options.RefreshInterval
	return tea.Tick(interval, func(time.Time) tea.Msg { return refreshMsg{} })
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
