package components

import (
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/thesimpledev/ghflow/internal/config"
	"github.com/thesimpledev/ghflow/internal/repo"
)

type CommandType int

const (
	CmdUnknown CommandType = iota
	CmdAdd
	CmdRemove
	CmdRefresh
	CmdQuit
	CmdSave
	CmdLoad
	CmdNew
)

type Command struct {
	Type CommandType
	Arg  string
}

type CommandInput struct {
	Input           string
	Focused         bool
	Suggestions     []string
	SuggestionIdx   int
	ShowSuggestions bool
	Width           int
	repos           []config.Repo // For /remove completion
	LastPath        string        // Remember last browsed path
}

// Messages
type ExecuteCommandMsg struct {
	Cmd Command
}

func NewCommandInput(repos []config.Repo) CommandInput {
	return CommandInput{
		repos: repos,
	}
}

func (c CommandInput) SetSize(width int) CommandInput {
	c.Width = width
	return c
}

func (c CommandInput) SetRepos(repos []config.Repo) CommandInput {
	c.repos = repos
	return c
}

func (c CommandInput) SetLastPath(path string) CommandInput {
	c.LastPath = filepath.Dir(path) // Store the parent directory
	return c
}

func (c CommandInput) SetFocused(focused bool) CommandInput {
	c.Focused = focused
	if focused {
		c.Input = "/"
		c.updateSuggestions()
	} else {
		c.Input = ""
		c.Suggestions = nil
		c.ShowSuggestions = false
	}
	return c
}

func (c CommandInput) Update(msg tea.Msg) (CommandInput, tea.Cmd) {
	if !c.Focused {
		return c, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			c.Focused = false
			c.Input = ""
			c.Suggestions = nil
			c.ShowSuggestions = false
			return c, nil

		case "enter":
			if c.ShowSuggestions && len(c.Suggestions) > 0 {
				// Accept suggestion
				c = c.acceptSuggestion()
				return c, nil
			}
			// Execute command
			cmd := c.parseCommand()
			c.Focused = false
			c.Input = ""
			c.Suggestions = nil
			c.ShowSuggestions = false
			return c, func() tea.Msg { return ExecuteCommandMsg{Cmd: cmd} }

		case "tab":
			if len(c.Suggestions) > 0 {
				// Complete the current suggestion
				suggestion := c.Suggestions[c.SuggestionIdx]

				// Handle hint placeholders like <path> or <repo>
				if strings.Contains(suggestion, " <") {
					// Strip the hint, keep command with space
					idx := strings.Index(suggestion, " <")
					c.Input = suggestion[:idx+1] // Include the space
					c.updateSuggestions()
					return c, nil
				}

				isRepo := strings.HasSuffix(suggestion, " [repo]")
				isAddPath := strings.HasPrefix(suggestion, "/add ")
				// Strip [repo] marker
				suggestion = strings.TrimSuffix(suggestion, " [repo]")

				if isRepo {
					// It's a repo, just complete it
					c.Input = suggestion
				} else if isAddPath && !isRepo {
					// It's a directory path, add / to continue browsing
					c.Input = suggestion + "/"
				} else {
					// Other completions (like /load profile), no trailing slash
					c.Input = suggestion
				}
				c.updateSuggestions()
			}
			return c, nil

		case "shift+tab", "up":
			if len(c.Suggestions) > 0 {
				c.ShowSuggestions = true
				c.SuggestionIdx--
				if c.SuggestionIdx < 0 {
					c.SuggestionIdx = len(c.Suggestions) - 1
				}
			}
			return c, nil

		case "down":
			if len(c.Suggestions) > 0 {
				c.ShowSuggestions = true
				c.SuggestionIdx = (c.SuggestionIdx + 1) % len(c.Suggestions)
			}
			return c, nil

		case "backspace":
			if len(c.Input) > 0 {
				c.Input = c.Input[:len(c.Input)-1]
				c.updateSuggestions()
			}
			return c, nil

		default:
			if len(msg.String()) == 1 {
				c.Input += msg.String()
				c.updateSuggestions()
			}
			return c, nil
		}
	}

	return c, nil
}

func (c *CommandInput) updateSuggestions() {
	c.Suggestions = nil
	c.SuggestionIdx = 0
	c.ShowSuggestions = false

	input := strings.TrimPrefix(c.Input, "/")
	parts := strings.SplitN(input, " ", 2)
	cmd := parts[0]
	arg := ""
	if len(parts) > 1 {
		arg = parts[1]
	}

	type cmdInfo struct {
		name string
		hint string
	}
	commands := []cmdInfo{
		{"add", "<path>"},
		{"remove", "<repo>"},
		{"save", "<name>"},
		{"load", "<profile>"},
		{"new", ""},
		{"refresh", ""},
		{"quit", ""},
		{"q", ""},
	}

	if arg == "" && !strings.Contains(input, " ") {
		// Complete command name with hints
		for _, command := range commands {
			if strings.HasPrefix(command.name, cmd) {
				suggestion := "/" + command.name
				if command.hint != "" {
					suggestion += " " + command.hint
				}
				c.Suggestions = append(c.Suggestions, suggestion)
			}
		}
	} else {
		// Complete argument
		switch cmd {
		case "add":
			c.Suggestions = c.completePath(arg)
		case "remove":
			c.Suggestions = c.completeRepo(arg)
		case "load":
			c.Suggestions = c.completeProfile(arg)
		}
	}
}

func (c *CommandInput) completePath(partial string) []string {
	var suggestions []string

	// Start from last path if available, otherwise current directory
	dir := "."
	if c.LastPath != "" {
		dir = c.LastPath
	}
	prefix := ""

	if partial != "" {
		// If path ends with /, list that directory's contents
		if strings.HasSuffix(partial, "/") {
			dir = partial
			prefix = ""
		} else {
			// Otherwise, we're filtering by prefix in the parent dir
			dir = filepath.Dir(partial)
			prefix = filepath.Base(partial)
		}
	}

	// Clean up the directory path
	dir = filepath.Clean(dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return suggestions
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if prefix != "" && !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}

		fullPath := filepath.Join(dir, entry.Name())

		// Check if it's a git repo
		isRepo := repo.IsGitRepo(fullPath)
		suggestion := "/add " + fullPath
		if isRepo {
			suggestion += " [repo]"
		}
		suggestions = append(suggestions, suggestion)

		if len(suggestions) >= 10 {
			break
		}
	}

	return suggestions
}

func (c *CommandInput) completeRepo(partial string) []string {
	var suggestions []string

	for _, r := range c.repos {
		name := r.Owner + "/" + r.Name
		if partial == "" || strings.Contains(strings.ToLower(name), strings.ToLower(partial)) {
			suggestions = append(suggestions, "/remove "+name)
		}
		if len(suggestions) >= 10 {
			break
		}
	}

	return suggestions
}

func (c *CommandInput) completeProfile(partial string) []string {
	var suggestions []string

	profiles, err := config.ListProfiles()
	if err != nil {
		return suggestions
	}

	for _, profile := range profiles {
		if partial == "" || strings.HasPrefix(strings.ToLower(profile), strings.ToLower(partial)) {
			suggestions = append(suggestions, "/load "+profile)
		}
		if len(suggestions) >= 10 {
			break
		}
	}

	return suggestions
}

func (c CommandInput) acceptSuggestion() CommandInput {
	if c.SuggestionIdx < len(c.Suggestions) {
		suggestion := c.Suggestions[c.SuggestionIdx]
		// Remove [repo] marker if present
		suggestion = strings.TrimSuffix(suggestion, " [repo]")
		c.Input = suggestion
		c.ShowSuggestions = false
		c.updateSuggestions()
	}
	return c
}

func (c CommandInput) parseCommand() Command {
	input := strings.TrimPrefix(c.Input, "/")
	parts := strings.SplitN(input, " ", 2)
	cmd := parts[0]
	arg := ""
	if len(parts) > 1 {
		arg = strings.TrimSuffix(parts[1], " [repo]")
	}

	switch cmd {
	case "add":
		return Command{Type: CmdAdd, Arg: arg}
	case "remove":
		return Command{Type: CmdRemove, Arg: arg}
	case "save":
		return Command{Type: CmdSave, Arg: arg}
	case "load":
		return Command{Type: CmdLoad, Arg: arg}
	case "new":
		return Command{Type: CmdNew}
	case "refresh":
		return Command{Type: CmdRefresh}
	case "quit", "q":
		return Command{Type: CmdQuit}
	default:
		return Command{Type: CmdUnknown}
	}
}

func (c CommandInput) View() string {
	var b strings.Builder

	// Input line
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Width(c.Width - 2).
		Padding(0, 1)

	prompt := "> "
	if c.Focused {
		promptStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("62"))
		inputLine := promptStyle.Render(prompt) + c.Input + "█"
		b.WriteString(inputStyle.Render(inputLine))
	} else {
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		inputLine := dimStyle.Render(prompt + "Type / to enter command...")
		b.WriteString(inputStyle.Render(inputLine))
	}

	// Suggestions
	if c.Focused && len(c.Suggestions) > 0 {
		b.WriteString("\n")
		sugStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			PaddingLeft(2)

		for i, sug := range c.Suggestions {
			if i >= 5 {
				b.WriteString(sugStyle.Render("  ..."))
				break
			}
			prefix := "  "
			style := sugStyle
			if c.ShowSuggestions && i == c.SuggestionIdx {
				prefix = "> "
				style = lipgloss.NewStyle().
					Foreground(lipgloss.Color("212")).
					Bold(true).
					PaddingLeft(2)
			}
			// Style special markers
			display := sug
			hintColor := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
			repoColor := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))

			if strings.HasSuffix(sug, " [repo]") {
				base := strings.TrimSuffix(sug, " [repo]")
				display = base + repoColor.Render(" [repo]")
			} else if strings.Contains(sug, " <") {
				// Style <path> or <repo> hints
				idx := strings.Index(sug, " <")
				base := sug[:idx]
				hint := sug[idx:]
				display = base + hintColor.Render(hint)
			}
			b.WriteString(style.Render(prefix+display) + "\n")
		}
	} else if c.Focused {
		// Show available commands hint
		b.WriteString("\n")
		hintStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			PaddingLeft(2)
		b.WriteString(hintStyle.Render("  /add  /remove  /refresh  /quit"))
	}

	return b.String()
}
