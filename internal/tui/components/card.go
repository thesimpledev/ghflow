package components

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/thesimpledev/ghflow/internal/config"
	"github.com/thesimpledev/ghflow/internal/github"
)

type CardState int

const (
	CardNormal CardState = iota
	CardSelected
	CardFocused
)

type Card struct {
	Repo       config.Repo
	Runs       []github.WorkflowRun
	Status     github.RunStatus
	Error      error
	State      CardState
	Width      int
	Height     int
	ScrollPos  int
	RunCursor  int
}

func NewCard(repo config.Repo) Card {
	return Card{
		Repo:   repo,
		Status: github.StatusUnknown,
		Runs:   []github.WorkflowRun{},
	}
}

func (c Card) SetSize(width, height int) Card {
	c.Width = width
	c.Height = height
	return c
}

func (c Card) SetState(state CardState) Card {
	c.State = state
	if state != CardFocused {
		c.RunCursor = 0
		c.ScrollPos = 0
	}
	return c
}

func (c Card) Update(msg tea.Msg) (Card, tea.Cmd) {
	if c.State != CardFocused {
		return c, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if c.RunCursor < len(c.Runs)-1 {
				c.RunCursor++
				visibleRuns := c.visibleRunCount()
				if c.RunCursor >= c.ScrollPos+visibleRuns {
					c.ScrollPos++
				}
			}
		case "k", "up":
			if c.RunCursor > 0 {
				c.RunCursor--
				if c.RunCursor < c.ScrollPos {
					c.ScrollPos--
				}
			}
		}
	}

	return c, nil
}

func (c Card) visibleRunCount() int {
	// Height minus header (repo name + status + divider) = runs area
	// Header takes about 3 lines
	available := c.Height - 5
	if available < 1 {
		return 1
	}
	return available
}

func (c Card) View() string {
	var borderColor lipgloss.Color
	var borderStyle lipgloss.Border

	switch c.State {
	case CardFocused:
		borderColor = lipgloss.Color("62") // Purple
		borderStyle = lipgloss.ThickBorder()
	case CardSelected:
		borderColor = lipgloss.Color("212") // Pink
		borderStyle = lipgloss.RoundedBorder()
	default:
		borderColor = lipgloss.Color("241") // Gray
		borderStyle = lipgloss.RoundedBorder()
	}

	cardStyle := lipgloss.NewStyle().
		Border(borderStyle).
		BorderForeground(borderColor).
		Width(c.Width - 2).
		Height(c.Height - 2).
		Padding(0, 1)

	content := c.renderContent()
	return cardStyle.Render(content)
}

func (c Card) renderContent() string {
	var b strings.Builder

	// Ensure minimum width
	width := c.Width
	if width < 20 {
		width = 20
	}

	// Repo name
	repoName := fmt.Sprintf("%s/%s", c.Repo.Owner, c.Repo.Name)
	maxNameLen := width - 4
	if maxNameLen < 10 {
		maxNameLen = 10
	}
	if len(repoName) > maxNameLen {
		truncLen := maxNameLen - 3
		if truncLen < 1 {
			truncLen = 1
		}
		repoName = repoName[:truncLen] + "..."
	}
	nameStyle := lipgloss.NewStyle().Bold(true)
	b.WriteString(nameStyle.Render(repoName) + "\n")

	// Status line
	statusIcon := c.statusIcon()
	branch := ""
	if len(c.Runs) > 0 {
		branch = c.Runs[0].HeadBranch
		if len(branch) > 15 {
			branch = branch[:12] + "..."
		}
	}
	statusLine := fmt.Sprintf("%s %s", statusIcon, branch)
	b.WriteString(statusLine + "\n")

	// Divider
	dividerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	dividerWidth := width - 4
	if dividerWidth < 1 {
		dividerWidth = 1
	}
	divider := strings.Repeat("─", dividerWidth)
	b.WriteString(dividerStyle.Render(divider) + "\n")

	// Runs list
	if c.Error != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		b.WriteString(errStyle.Render("Error loading") + "\n")
	} else if len(c.Runs) == 0 {
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		b.WriteString(dimStyle.Render("No runs") + "\n")
	} else {
		visibleCount := c.visibleRunCount()
		endIdx := c.ScrollPos + visibleCount
		if endIdx > len(c.Runs) {
			endIdx = len(c.Runs)
		}

		for i := c.ScrollPos; i < endIdx; i++ {
			run := c.Runs[i]
			line := c.renderRunLine(run, i == c.RunCursor && c.State == CardFocused)
			b.WriteString(line + "\n")
		}

		// Scroll indicator
		if len(c.Runs) > visibleCount {
			indicator := fmt.Sprintf("(%d/%d)", c.RunCursor+1, len(c.Runs))
			indStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
			b.WriteString(indStyle.Render(indicator))
		}
	}

	return b.String()
}

func (c Card) renderRunLine(run github.WorkflowRun, selected bool) string {
	icon := runStatusIcon(run.RunStatus())

	// Workflow name
	name := run.WorkflowName
	if name == "" {
		name = run.Name
	}

	// Branch name
	branch := run.HeadBranch
	if len(branch) > 12 {
		branch = branch[:9] + "..."
	}

	// Truncate workflow name if needed
	// Format: [ok] #123 Name (branch) 2m
	// Reserve space for: icon(4) + space + # + number(~4) + space + name + space + parens + branch + space + time(~4)
	maxNameLen := c.Width - 30 - len(branch)
	if maxNameLen < 6 {
		maxNameLen = 6
	}
	if len(name) > maxNameLen {
		truncLen := maxNameLen - 3
		if truncLen < 1 {
			truncLen = 1
		}
		name = name[:truncLen] + "..."
	}

	timeAgo := formatTimeAgo(run.CreatedAt)

	branchStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	branchStr := branchStyle.Render("(" + branch + ")")

	line := fmt.Sprintf("%s #%d %s %s %s", icon, run.RunNumber, name, branchStr, timeAgo)

	if selected {
		return lipgloss.NewStyle().Bold(true).Reverse(true).Render(line)
	}
	return line
}

func (c Card) statusIcon() string {
	return runStatusIcon(c.Status)
}

func runStatusIcon(status github.RunStatus) string {
	switch status {
	case github.StatusSuccess:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("[ok]")
	case github.StatusFailure:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("[X]")
	case github.StatusInProgress:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("[~]")
	case github.StatusPending:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("247")).Render("[?]")
	case github.StatusCancelled:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("[-]")
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("[.]")
	}
}

func formatTimeAgo(t time.Time) string {
	diff := time.Since(t)

	if diff < time.Minute {
		return "now"
	}
	if diff < time.Hour {
		mins := int(diff.Minutes())
		return fmt.Sprintf("%dm", mins)
	}
	if diff < 24*time.Hour {
		hours := int(diff.Hours())
		return fmt.Sprintf("%dh", hours)
	}
	days := int(diff.Hours() / 24)
	return fmt.Sprintf("%dd", days)
}
