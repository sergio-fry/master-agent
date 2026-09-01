package store

// Project is a remote workspace: path + SSH target.
type Project struct {
	ID         string
	Name       string
	Path       string
	SSHHost    string
	SSHUser    string
	SSHPort        int
	SSHPrivateKey  string
	SSHHostKey     string // "ssh-ed25519 AAAA..." public key line for known_hosts
	Enabled        bool
	CreatedAt  string
	UpdatedAt  string
}

// Task is schedule + command + prompt within one Project.
type Task struct {
	ID              string
	ProjectID       string
	Name            string
	Prompt          string
	Command         string
	IntervalSeconds int
	Enabled         bool
	LastRunAt       *string
	NextRunAt       *string
	CreatedAt       string
	UpdatedAt       string
}

// Lock holds a project while a run is in progress.
type Lock struct {
	ProjectID  string
	TaskID     string
	RunID      string
	PID        *int
	AcquiredAt string
}

// RunStatus values for Run.Status.
const (
	RunStatusRunning = "running"
	RunStatusSuccess = "success"
	RunStatusError   = "error"
)

// Run is one execution attempt for audit/debug.
type Run struct {
	ID           string
	TaskID       string
	ProjectID    string
	StartedAt    string
	FinishedAt   *string
	ExitCode     *int
	Status       string
	ErrorMessage *string
	LogPath      *string
}
