package placeholder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"master-agent/internal/store"
)

func sampleProject() store.Project {
	return store.Project{
		ID:   "proj-1",
		Name: "my-app",
		Path: "/home/dev/my-app",
	}
}

func sampleTask() store.Task {
	return store.Task{
		ID:      "task-1",
		Name:    "drain",
		Prompt:  "Do the work",
		Command: "unused",
	}
}

func TestExpandAllPlaceholdersShell(t *testing.T) {
	cmd := `cursor agent --workspace {{project_path}} -p {{prompt}} # {{project_name}}/{{task_name}}/{{task_id}}`
	got, err := Expand(cmd, sampleProject(), sampleTask())
	require.NoError(t, err)
	assert.Equal(t,
		`cursor agent --workspace '/home/dev/my-app' -p 'Do the work' # 'my-app'/'drain'/'task-1'`,
		got,
	)
}

func TestExpandJSONArgvLiteral(t *testing.T) {
	cmd := `["cursor", "agent", "--workspace", "{{project_path}}", "-p", "{{prompt}}"]`
	got, err := Expand(cmd, sampleProject(), sampleTask())
	require.NoError(t, err)
	assert.JSONEq(t,
		`["cursor","agent","--workspace","/home/dev/my-app","-p","Do the work"]`,
		got,
	)
}

func TestExpandShellEscapesMetacharacters(t *testing.T) {
	project := sampleProject()
	project.Path = `/tmp/dir with spaces`
	task := sampleTask()
	task.Prompt = `say "hi"; rm -rf /`

	got, err := Expand(`run -p {{prompt}} --cwd {{project_path}}`, project, task)
	require.NoError(t, err)
	assert.Equal(t, `run -p 'say "hi"; rm -rf /' --cwd '/tmp/dir with spaces'`, got)
}

func TestExpandShellEscapesSingleQuotes(t *testing.T) {
	task := sampleTask()
	task.Prompt = `it's fine`

	got, err := Expand(`echo {{prompt}}`, sampleProject(), task)
	require.NoError(t, err)
	assert.Equal(t, `echo 'it'\''s fine'`, got)
}

func TestExpandUnknownPlaceholderError(t *testing.T) {
	_, err := Expand(`cmd {{unknown_thing}}`, sampleProject(), sampleTask())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "{{unknown_thing}}")
}

func TestExpandEmptyValues(t *testing.T) {
	project := store.Project{Name: "", Path: ""}
	task := store.Task{ID: "", Name: "", Prompt: ""}

	got, err := Expand(`p={{prompt}} path={{project_path}}`, project, task)
	require.NoError(t, err)
	assert.Equal(t, `p='' path=''`, got)

	gotJSON, err := Expand(`["{{prompt}}", "{{task_id}}"]`, project, task)
	require.NoError(t, err)
	assert.JSONEq(t, `["",""]`, gotJSON)
}

func TestExpandNoPlaceholdersUnchanged(t *testing.T) {
	cmd := `echo hello && touch /tmp/flag`
	got, err := Expand(cmd, sampleProject(), sampleTask())
	require.NoError(t, err)
	assert.Equal(t, cmd, got)
}

func TestExpandTableDriven(t *testing.T) {
	project := sampleProject()
	task := sampleTask()

	tests := []struct {
		name    string
		command string
		project store.Project
		task    store.Task
		want    string
		jsonEq  bool
		wantErr string
	}{
		{
			name:    "prompt only",
			command: `agent -p {{prompt}}`,
			project: project,
			task:    task,
			want:    `agent -p 'Do the work'`,
		},
		{
			name:    "all fields",
			command: `{{project_name}} {{project_path}} {{task_name}} {{task_id}} {{prompt}}`,
			project: project,
			task:    task,
			want:    `'my-app' '/home/dev/my-app' 'drain' 'task-1' 'Do the work'`,
		},
		{
			name:    "newline in prompt",
			command: `p={{prompt}}`,
			project: project,
			task: store.Task{
				ID: "t", Name: "n", Prompt: "line1\nline2",
			},
			want: `p='line1` + "\n" + `line2'`,
		},
		{
			name:    "dollar and backticks",
			command: `p={{prompt}}`,
			project: project,
			task: store.Task{
				ID: "t", Name: "n", Prompt: "`id`; echo $HOME",
			},
			want: `p='` + "`id`; echo $HOME" + `'`,
		},
		{
			name:    "unknown placeholder",
			command: `x={{foo}}`,
			project: project,
			task:    task,
			wantErr: "{{foo}}",
		},
		{
			name:    "json with unknown",
			command: `["{{prompt}}", "{{nope}}"]`,
			project: project,
			task:    task,
			wantErr: "{{nope}}",
		},
		{
			name:    "shell-looking bracket is not json",
			command: `[start] {{task_name}}`,
			project: project,
			task:    task,
			want:    `[start] 'drain'`,
		},
		{
			name:    "json preserves non-placeholder args",
			command: `["echo", "static", "{{task_id}}"]`,
			project: project,
			task:    task,
			want:    `["echo","static","task-1"]`,
			jsonEq:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Expand(tt.command, tt.project, tt.task)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			if tt.jsonEq {
				assert.JSONEq(t, tt.want, got)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestShellQuote(t *testing.T) {
	assert.Equal(t, `''`, ShellQuote(""))
	assert.Equal(t, `'abc'`, ShellQuote("abc"))
	assert.Equal(t, `'a'\''b'`, ShellQuote("a'b"))
}
