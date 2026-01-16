package views

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/thesimpledev/ghflow/internal/config"
	"github.com/thesimpledev/ghflow/internal/repo"
	"github.com/thesimpledev/ghflow/internal/tui/components"
)

type InputMode int

const (
	ModeGrid InputMode = iota
	ModeCommand
)

type DashboardModel struct {
	config      *config.Config
	grid        components.Grid
	commandInput components.CommandInput
	mode        InputMode
	width       int
	height      int
	err         error
}

// Messages
type RefreshMsg struct{}
type RepoAddedMsg struct {
	Repo config.Repo
}
type RepoRemovedMsg struct {
	Owner string
	Name  string
}

func NewDashboardModel(cfg *config.Config) DashboardModel {
	return DashboardModel{
		config:       cfg,
		grid:         components.NewGrid(cfg.Repos),
		commandInput: components.NewCommandInput(cfg.Repos),
		mode:         ModeGrid,
	}
}

func (m DashboardModel) SetSize(width, height int) DashboardModel {
	m.width = width
	m.height = height

	// Reserve space for title and command input
	titleHeight := 2
	commandHeight := 4

	gridHeight := height - titleHeight - commandHeight
	if gridHeight < 6 {
		gridHeight = 6
	}

	m.grid = m.grid.SetSize(width, gridHeight)
	m.commandInput = m.commandInput.SetSize(width)

	return m
}

func (m DashboardModel) Init() tea.Cmd {
	return m.grid.Init()
}

func (m DashboardModel) Update(msg tea.Msg) (DashboardModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m = m.SetSize(msg.Width, msg.Height)

	case tea.KeyMsg:
		// Global keys
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.mode == ModeGrid && m.grid.State == components.GridNavigating {
				return m, tea.Quit
			}
		case "/":
			if m.mode == ModeGrid && m.grid.State == components.GridNavigating {
				m.mode = ModeCommand
				m.commandInput = m.commandInput.SetFocused(true)
				return m, nil
			}
		case "esc":
			if m.mode == ModeCommand {
				m.mode = ModeGrid
				m.commandInput = m.commandInput.SetFocused(false)
				return m, nil
			}
		}

	case components.CardStatusMsg:
		var cmd tea.Cmd
		m.grid, cmd = m.grid.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case components.ExecuteCommandMsg:
		return m.handleCommand(msg.Cmd)

	case RefreshMsg:
		cmds = append(cmds, m.grid.RefreshAll())
		return m, tea.Batch(cmds...)
	}

	// Route to focused component
	switch m.mode {
	case ModeCommand:
		var cmd tea.Cmd
		m.commandInput, cmd = m.commandInput.Update(msg)
		cmds = append(cmds, cmd)
		// Check if command input unfocused itself
		if !m.commandInput.Focused {
			m.mode = ModeGrid
		}
	case ModeGrid:
		var cmd tea.Cmd
		m.grid, cmd = m.grid.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m DashboardModel) handleCommand(cmd components.Command) (DashboardModel, tea.Cmd) {
	switch cmd.Type {
	case components.CmdQuit:
		return m, tea.Quit

	case components.CmdRefresh:
		m.mode = ModeGrid
		return m, m.grid.RefreshAll()

	case components.CmdAdd:
		if cmd.Arg != "" {
			info, err := repo.GetRepoInfo(cmd.Arg)
			if err != nil || info == nil {
				m.err = err
				m.mode = ModeGrid
				return m, nil
			}

			newRepo := config.Repo{
				Path:  info.Path,
				Owner: info.Owner,
				Name:  info.Name,
			}
			m.config.AddRepo(newRepo)
			m.config.Save()

			// Rebuild grid with new repo
			m.grid = components.NewGrid(m.config.Repos)
			m.grid = m.grid.SetSize(m.width, m.height-6)
			m.commandInput = m.commandInput.SetRepos(m.config.Repos)
			m.commandInput = m.commandInput.SetLastPath(info.Path) // Remember for next /add
			m.mode = ModeGrid
			return m, m.grid.Init()
		}
		m.mode = ModeGrid
		return m, nil

	case components.CmdRemove:
		if cmd.Arg != "" {
			// Parse owner/name from arg
			parts := splitOwnerName(cmd.Arg)
			if len(parts) == 2 {
				m.config.RemoveRepo(parts[0], parts[1])
				m.config.Save()

				// Rebuild grid without removed repo
				m.grid = components.NewGrid(m.config.Repos)
				m.grid = m.grid.SetSize(m.width, m.height-6)
				m.commandInput = m.commandInput.SetRepos(m.config.Repos)
			}
		}
		m.mode = ModeGrid
		return m, nil

	case components.CmdSave:
		if cmd.Arg != "" {
			m.config.SaveProfile(cmd.Arg)
		}
		m.mode = ModeGrid
		return m, nil

	case components.CmdLoad:
		if cmd.Arg != "" {
			loadedCfg, err := config.LoadProfile(cmd.Arg)
			if err == nil && loadedCfg != nil {
				// Replace current config with loaded profile
				m.config.Repos = loadedCfg.Repos
				m.config.Save() // Save as current config too

				// Rebuild grid with loaded repos
				m.grid = components.NewGrid(m.config.Repos)
				m.grid = m.grid.SetSize(m.width, m.height-6)
				m.commandInput = m.commandInput.SetRepos(m.config.Repos)
				m.mode = ModeGrid
				return m, m.grid.Init()
			}
		}
		m.mode = ModeGrid
		return m, nil

	case components.CmdNew:
		// Clear all repos and start fresh
		m.config.Repos = []config.Repo{}
		m.config.Save()

		// Rebuild empty grid
		m.grid = components.NewGrid(m.config.Repos)
		m.grid = m.grid.SetSize(m.width, m.height-6)
		m.commandInput = m.commandInput.SetRepos(m.config.Repos)
		m.mode = ModeGrid
		return m, nil

	default:
		m.mode = ModeGrid
		return m, nil
	}
}

func splitOwnerName(s string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return nil
}

func (m DashboardModel) View() string {
	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62")).
		MarginBottom(1)
	title := titleStyle.Render("ghflow - GitHub Workflow Dashboard")

	// Grid
	gridView := m.grid.View()

	// Command input
	cmdView := m.commandInput.View()

	// Help line when not in command mode
	helpLine := ""
	if m.mode == ModeGrid && m.grid.State == components.GridNavigating {
		helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		helpLine = helpStyle.Render("h/j/k/l: navigate | enter: focus | /: command | q: quit")
	} else if m.mode == ModeGrid && m.grid.State == components.GridCardFocused {
		helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		helpLine = helpStyle.Render("j/k: scroll runs | esc: unfocus | q: quit")
	}

	return title + "\n" + gridView + "\n" + cmdView + "\n" + helpLine
}

// TickCmd returns a command that triggers periodic refresh
func TickCmd() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return RefreshMsg{}
	})
}
