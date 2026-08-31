// Package placeholder substitutes Task.command placeholders before remote execution.
package placeholder

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"master-agent/internal/store"
)

// Documented placeholders from specs/01-data-model.md.
const (
	Prompt      = "prompt"
	ProjectPath = "project_path"
	ProjectName = "project_name"
	TaskName    = "task_name"
	TaskID      = "task_id"
)

var placeholderRE = regexp.MustCompile(`\{\{([a-z0-9_]+)\}\}`)

// Expand replaces {{prompt}}, {{project_path}}, {{project_name}}, {{task_name}},
// and {{task_id}} in command.
//
// If command is a JSON string array, values are substituted literally and the
// result is re-serialized as JSON (safe for argv). Otherwise command is treated
// as a remote shell string and each value is POSIX single-quote escaped.
//
// Unknown placeholders return an error. Empty field values are substituted as
// empty strings (quoted as '' in shell mode).
func Expand(command string, project store.Project, task store.Task) (string, error) {
	vals := map[string]string{
		Prompt:      task.Prompt,
		ProjectPath: project.Path,
		ProjectName: project.Name,
		TaskName:    task.Name,
		TaskID:      task.ID,
	}

	trimmed := strings.TrimSpace(command)
	var argv []string
	if err := json.Unmarshal([]byte(trimmed), &argv); err == nil {
		return expandJSON(argv, vals)
	}
	return replace(command, vals, true)
}

func expandJSON(argv []string, vals map[string]string) (string, error) {
	out := make([]string, len(argv))
	for i, part := range argv {
		expanded, err := replace(part, vals, false)
		if err != nil {
			return "", err
		}
		out[i] = expanded
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func replace(s string, vals map[string]string, shellEscape bool) (string, error) {
	var unknown []string
	result := placeholderRE.ReplaceAllStringFunc(s, func(match string) string {
		key := match[2 : len(match)-2] // strip {{ }}
		val, ok := vals[key]
		if !ok {
			unknown = append(unknown, match)
			return match
		}
		if shellEscape {
			return ShellQuote(val)
		}
		return val
	})
	if len(unknown) > 0 {
		return "", fmt.Errorf("unknown placeholder: %s", unknown[0])
	}
	return result, nil
}

// ShellQuote returns a POSIX single-quoted string safe for remote shells.
func ShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
