package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/thesimpledev/ghflow/internal/config"
	"github.com/thesimpledev/ghflow/internal/github"
)

const (
	GridCols = 3
	GridRows = 2
)

type GridState int

const (
	GridNavigating GridState = iota
	GridCardFocused
)

type Grid struct {
	Cards       []Card
	Cursor      int
	State       GridState
	Width       int
	Height      int
}

type CardStatusMsg struct {
	Index  int
	Status github.RunStatus
	Runs   []github.WorkflowRun
	Error  error
}

func NewGrid(repos []config.Repo) Grid {
	cards := make([]Card, len(repos))
	for i, repo := range repos {
		cards[i] = NewCard(repo)
	}

	// Set first card as selected
	if len(cards) > 0 {
		cards[0] = cards[0].SetState(CardSelected)
	}

	return Grid{
		Cards:  cards,
		State:  GridNavigating,
		Cursor: 0,
	}
}

func (g Grid) SetSize(width, height int) Grid {
	g.Width = width
	g.Height = height

	// Calculate card dimensions
	cardWidth := width / GridCols
	cardHeight := height / GridRows

	for i := range g.Cards {
		g.Cards[i] = g.Cards[i].SetSize(cardWidth, cardHeight)
	}

	return g
}

func (g Grid) Init() tea.Cmd {
	var cmds []tea.Cmd
	for i, card := range g.Cards {
		cmds = append(cmds, fetchCardStatus(i, card.Repo.Owner, card.Repo.Name))
	}
	return tea.Batch(cmds...)
}

func fetchCardStatus(index int, owner, name string) tea.Cmd {
	return func() tea.Msg {
		runs, err := github.FetchWorkflowRuns(owner, name, 5)
		status := github.StatusUnknown
		if len(runs) > 0 {
			status = runs[0].RunStatus()
		}
		return CardStatusMsg{
			Index:  index,
			Status: status,
			Runs:   runs,
			Error:  err,
		}
	}
}

func (g Grid) Update(msg tea.Msg) (Grid, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case CardStatusMsg:
		if msg.Index < len(g.Cards) {
			g.Cards[msg.Index].Status = msg.Status
			g.Cards[msg.Index].Runs = msg.Runs
			g.Cards[msg.Index].Error = msg.Error
		}
		return g, nil

	case tea.KeyMsg:
		if g.State == GridCardFocused {
			// Forward to focused card
			if g.Cursor < len(g.Cards) {
				var cmd tea.Cmd
				g.Cards[g.Cursor], cmd = g.Cards[g.Cursor].Update(msg)
				cmds = append(cmds, cmd)
			}

			// Handle escape to unfocus
			if msg.String() == "esc" {
				g.State = GridNavigating
				if g.Cursor < len(g.Cards) {
					g.Cards[g.Cursor] = g.Cards[g.Cursor].SetState(CardSelected)
				}
			}
			return g, tea.Batch(cmds...)
		}

		// Grid navigation mode
		switch msg.String() {
		case "h":
			g = g.moveCursor(-1, 0)
		case "l":
			g = g.moveCursor(1, 0)
		case "k":
			g = g.moveCursor(0, -1)
		case "j":
			g = g.moveCursor(0, 1)
		case "enter":
			if g.Cursor < len(g.Cards) {
				g.State = GridCardFocused
				g.Cards[g.Cursor] = g.Cards[g.Cursor].SetState(CardFocused)
			}
		}
	}

	return g, tea.Batch(cmds...)
}

func (g Grid) moveCursor(dx, dy int) Grid {
	if len(g.Cards) == 0 {
		return g
	}

	// Current position
	col := g.Cursor % GridCols
	row := g.Cursor / GridCols

	// Calculate new position
	newCol := col + dx
	newRow := row + dy

	// Clamp to grid bounds
	if newCol < 0 {
		newCol = 0
	}
	if newCol >= GridCols {
		newCol = GridCols - 1
	}
	if newRow < 0 {
		newRow = 0
	}
	maxRow := (len(g.Cards) - 1) / GridCols
	if newRow > maxRow {
		newRow = maxRow
	}

	newCursor := newRow*GridCols + newCol
	if newCursor >= len(g.Cards) {
		// If moving to empty cell, stay in current row but move to last card in that row
		newCursor = len(g.Cards) - 1
	}

	if newCursor != g.Cursor {
		// Update states
		if g.Cursor < len(g.Cards) {
			g.Cards[g.Cursor] = g.Cards[g.Cursor].SetState(CardNormal)
		}
		g.Cursor = newCursor
		if g.Cursor < len(g.Cards) {
			g.Cards[g.Cursor] = g.Cards[g.Cursor].SetState(CardSelected)
		}
	}

	return g
}

func (g Grid) RefreshAll() tea.Cmd {
	var cmds []tea.Cmd
	for i, card := range g.Cards {
		cmds = append(cmds, fetchCardStatus(i, card.Repo.Owner, card.Repo.Name))
	}
	return tea.Batch(cmds...)
}

func (g Grid) SelectedRepo() *config.Repo {
	if g.Cursor < len(g.Cards) {
		return &g.Cards[g.Cursor].Repo
	}
	return nil
}

func (g Grid) View() string {
	if len(g.Cards) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Width(g.Width).
			Height(g.Height).
			Align(lipgloss.Center, lipgloss.Center)
		return emptyStyle.Render("No repositories added.\nType /add to add one.")
	}

	cardWidth := g.Width / GridCols
	cardHeight := g.Height / GridRows

	var rows []string

	for row := 0; row < GridRows; row++ {
		var rowCards []string
		for col := 0; col < GridCols; col++ {
			idx := row*GridCols + col
			if idx < len(g.Cards) {
				card := g.Cards[idx].SetSize(cardWidth, cardHeight)
				rowCards = append(rowCards, card.View())
			} else {
				// Empty cell
				emptyStyle := lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(lipgloss.Color("236")).
					Width(cardWidth - 2).
					Height(cardHeight - 2).
					Align(lipgloss.Center, lipgloss.Center).
					Foreground(lipgloss.Color("241"))
				rowCards = append(rowCards, emptyStyle.Render(""))
			}
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, rowCards...))
	}

	return strings.Join(rows, "\n")
}
