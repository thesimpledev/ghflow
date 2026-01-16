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
	CardRunDetail
)

type Card struct {
	Repo        config.Repo
	Runs        []github.WorkflowRun
	Status      github.RunStatus
	Error       error
	State       CardState
	Width       int
	Height      int
	ScrollPos   int
	RunCursor   int
	DetailRun   *github.WorkflowRun
	DetailJobs  []github.Job
	JobCursor   int
	LoadingJobs bool
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

type JobsFetchedMsg struct {
	CardIndex int
	Jobs      []github.Job
	Error     error
}

func (c Card) SetState(state CardState) Card {
	c.State = state
	if state != CardFocused && state != CardRunDetail {
		c.RunCursor = 0
		c.ScrollPos = 0
		c.DetailRun = nil
		c.DetailJobs = nil
		c.JobCursor = 0
	}
	return c
}

func (c Card) Update(msg tea.Msg) (Card, tea.Cmd) {
	switch msg := msg.(type) {
	case JobsFetchedMsg:
		c.LoadingJobs = false
		c.DetailJobs = msg.Jobs
		return c, nil

	case tea.KeyMsg:
		if c.State == CardRunDetail {
			// In run detail view - navigate jobs
			switch msg.String() {
			case "j", "down":
				if c.JobCursor < len(c.DetailJobs)-1 {
					c.JobCursor++
				}
			case "k", "up":
				if c.JobCursor > 0 {
					c.JobCursor--
				}
			case "esc":
				// Back to run list
				c.State = CardFocused
				c.DetailRun = nil
				c.DetailJobs = nil
				c.JobCursor = 0
			}
			return c, nil
		}

		if c.State == CardFocused {
			// In run list - navigate runs
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
			case "enter":
				if c.RunCursor < len(c.Runs) {
					run := c.Runs[c.RunCursor]
					c.State = CardRunDetail
					c.DetailRun = &run
					c.LoadingJobs = true
					c.JobCursor = 0
					return c, c.fetchJobs(run.ID)
				}
			}
			return c, nil
		}
	}

	return c, nil
}

func (c Card) fetchJobs(runID int64) tea.Cmd {
	owner := c.Repo.Owner
	name := c.Repo.Name
	return func() tea.Msg {
		jobs, err := github.FetchRunJobs(owner, name, runID)
		return JobsFetchedMsg{
			Jobs:  jobs,
			Error: err,
		}
	}
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
	case CardFocused, CardRunDetail:
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

	var content string
	if c.State == CardRunDetail {
		content = c.renderRunDetail()
	} else {
		content = c.renderContent()
	}
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

	// Status line - use a dot indicator instead of [ok] to differentiate from run entries
	statusDot := c.statusDot()
	branch := ""
	if len(c.Runs) > 0 {
		branch = c.Runs[0].HeadBranch
		if len(branch) > 15 {
			branch = branch[:12] + "..."
		}
	}
	branchStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	statusLine := fmt.Sprintf("%s %s", statusDot, branchStyle.Render(branch))
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

func (c Card) statusDot() string {
	switch c.Status {
	case github.StatusSuccess:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("●")
	case github.StatusFailure:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("●")
	case github.StatusInProgress:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("●")
	case github.StatusPending:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("247")).Render("●")
	case github.StatusCancelled:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("●")
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("○")
	}
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

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

func (c Card) renderRunDetail() string {
	var b strings.Builder

	width := c.Width
	if width < 20 {
		width = 20
	}

	if c.DetailRun == nil {
		return "No run selected"
	}

	run := c.DetailRun

	// Header: workflow name and run number
	headerStyle := lipgloss.NewStyle().Bold(true)
	workflowName := run.WorkflowName
	if workflowName == "" {
		workflowName = run.Name
	}
	b.WriteString(headerStyle.Render(fmt.Sprintf("#%d %s", run.RunNumber, workflowName)) + "\n")

	// Status and branch
	statusIcon := runStatusIcon(run.RunStatus())
	branchStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("75"))
	b.WriteString(fmt.Sprintf("%s %s %s\n", statusIcon, branchStyle.Render(run.HeadBranch), formatTimeAgo(run.CreatedAt)))

	// Divider
	dividerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	dividerWidth := width - 4
	if dividerWidth < 1 {
		dividerWidth = 1
	}
	b.WriteString(dividerStyle.Render(strings.Repeat("─", dividerWidth)) + "\n")

	// Jobs header
	jobsHeader := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Jobs:")
	b.WriteString(jobsHeader + "\n")

	if c.LoadingJobs {
		loadStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
		b.WriteString(loadStyle.Render("Loading...") + "\n")
	} else if len(c.DetailJobs) == 0 {
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		b.WriteString(dimStyle.Render("No jobs") + "\n")
	} else {
		for i, job := range c.DetailJobs {
			line := c.renderJobLine(job, i == c.JobCursor)
			b.WriteString(line + "\n")
		}
	}

	// Help line
	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	b.WriteString(helpStyle.Render("esc: back"))

	return b.String()
}

func (c Card) renderJobLine(job github.Job, selected bool) string {
	icon := runStatusIcon(job.JobStatus())

	name := job.Name
	maxNameLen := c.Width - 20
	if maxNameLen < 10 {
		maxNameLen = 10
	}
	if len(name) > maxNameLen {
		name = name[:maxNameLen-3] + "..."
	}

	duration := ""
	if d := job.Duration(); d > 0 {
		duration = formatDuration(d)
	}

	line := fmt.Sprintf("%s %s %s", icon, name, duration)

	if selected {
		return lipgloss.NewStyle().Bold(true).Reverse(true).Render(line)
	}
	return line
}
