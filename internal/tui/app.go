package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thesimpledev/ghflow/internal/config"
	"github.com/thesimpledev/ghflow/internal/tui/views"
)

type App struct {
	config    *config.Config
	dashboard views.DashboardModel
	width     int
	height    int
}

type TickMsg time.Time

func NewApp(cfg *config.Config) App {
	return App{
		config:    cfg,
		dashboard: views.NewDashboardModel(cfg),
	}
}

func (a App) Init() tea.Cmd {
	return tea.Batch(
		a.dashboard.Init(),
		tickCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.dashboard = a.dashboard.SetSize(msg.Width, msg.Height)

	case TickMsg:
		cmds = append(cmds, tickCmd())
		var cmd tea.Cmd
		a.dashboard, cmd = a.dashboard.Update(views.RefreshMsg{})
		cmds = append(cmds, cmd)
		return a, tea.Batch(cmds...)
	}

	var cmd tea.Cmd
	a.dashboard, cmd = a.dashboard.Update(msg)
	cmds = append(cmds, cmd)

	return a, tea.Batch(cmds...)
}

func (a App) View() string {
	return a.dashboard.View()
}
