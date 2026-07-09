package github

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"time"
)

// GitHub owner and repo names only contain letters, digits, hyphens,
// underscores, and dots. Anything else could alter the API path we
// build below (e.g. a "/" or "?" smuggled in via a git remote URL).
var validNamePart = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

func checkOwnerRepo(owner, repo string) error {
	if !validNamePart.MatchString(owner) || !validNamePart.MatchString(repo) {
		return fmt.Errorf("invalid owner/repo name: %s/%s", owner, repo)
	}
	return nil
}

type RunStatus string

const (
	StatusSuccess    RunStatus = "success"
	StatusFailure    RunStatus = "failure"
	StatusInProgress RunStatus = "in_progress"
	StatusPending    RunStatus = "pending"
	StatusCancelled  RunStatus = "cancelled"
	StatusUnknown    RunStatus = "unknown"
)

type WorkflowRun struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	HeadBranch   string    `json:"head_branch"`
	Status       string    `json:"status"`
	Conclusion   string    `json:"conclusion"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	HTMLURL      string    `json:"html_url"`
	RunNumber    int       `json:"run_number"`
	WorkflowName string    `json:"workflow_name"`
}

type workflowRunsResponse struct {
	TotalCount   int           `json:"total_count"`
	WorkflowRuns []WorkflowRun `json:"workflow_runs"`
}

func (r *WorkflowRun) RunStatus() RunStatus {
	if r.Status == "completed" {
		switch r.Conclusion {
		case "success":
			return StatusSuccess
		case "failure":
			return StatusFailure
		case "cancelled":
			return StatusCancelled
		default:
			return StatusUnknown
		}
	}
	if r.Status == "in_progress" || r.Status == "queued" {
		return StatusInProgress
	}
	if r.Status == "pending" || r.Status == "waiting" {
		return StatusPending
	}
	return StatusUnknown
}

func FetchWorkflowRuns(owner, repo string, limit int) ([]WorkflowRun, error) {
	if err := checkOwnerRepo(owner, repo); err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("repos/%s/%s/actions/runs?per_page=%d", owner, repo, limit)

	cmd := exec.Command("gh", "api", endpoint) // #nosec G204 -- fixed binary, owner/repo validated by checkOwnerRepo, no shell involved
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh api failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("gh api failed: %w", err)
	}

	var response workflowRunsResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	for i := range response.WorkflowRuns {
		if response.WorkflowRuns[i].WorkflowName == "" {
			response.WorkflowRuns[i].WorkflowName = response.WorkflowRuns[i].Name
		}
	}

	return response.WorkflowRuns, nil
}

func GetLatestRunStatus(owner, repo string) (RunStatus, *WorkflowRun, error) {
	runs, err := FetchWorkflowRuns(owner, repo, 1)
	if err != nil {
		return StatusUnknown, nil, err
	}

	if len(runs) == 0 {
		return StatusUnknown, nil, nil
	}

	run := &runs[0]
	return run.RunStatus(), run, nil
}

type Job struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Conclusion  string    `json:"conclusion"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
}

type jobsResponse struct {
	TotalCount int   `json:"total_count"`
	Jobs       []Job `json:"jobs"`
}

func (j *Job) JobStatus() RunStatus {
	if j.Status == "completed" {
		switch j.Conclusion {
		case "success":
			return StatusSuccess
		case "failure":
			return StatusFailure
		case "cancelled":
			return StatusCancelled
		default:
			return StatusUnknown
		}
	}
	if j.Status == "in_progress" || j.Status == "queued" {
		return StatusInProgress
	}
	if j.Status == "pending" || j.Status == "waiting" {
		return StatusPending
	}
	return StatusUnknown
}

func (j *Job) Duration() time.Duration {
	if j.CompletedAt.IsZero() || j.StartedAt.IsZero() {
		return 0
	}
	return j.CompletedAt.Sub(j.StartedAt)
}

func FetchRunJobs(owner, repo string, runID int64) ([]Job, error) {
	if err := checkOwnerRepo(owner, repo); err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("repos/%s/%s/actions/runs/%d/jobs", owner, repo, runID)

	cmd := exec.Command("gh", "api", endpoint) // #nosec G204 -- fixed binary, owner/repo validated by checkOwnerRepo, no shell involved
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh api failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("gh api failed: %w", err)
	}

	var response jobsResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return response.Jobs, nil
}

func IsGHInstalled() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}

func IsAuthenticated() bool {
	cmd := exec.Command("gh", "auth", "status")
	return cmd.Run() == nil
}
