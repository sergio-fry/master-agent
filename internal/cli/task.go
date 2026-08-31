package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"master-agent/internal/store"
)

func newTaskCmd(opts Options, openStore func() (*store.Store, error)) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage tasks (schedule + command + prompt)",
	}
	cmd.AddCommand(newTaskAddCmd(opts, openStore))
	cmd.AddCommand(newTaskListCmd(opts, openStore))
	cmd.AddCommand(newTaskDisableCmd(opts, openStore))
	cmd.AddCommand(newTaskEnableCmd(opts, openStore))
	return cmd
}

func newTaskAddCmd(opts Options, openStore func() (*store.Store, error)) *cobra.Command {
	var (
		project  string
		name     string
		interval int
		command  string
		prompt   string
	)
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a task to a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			if interval <= 0 {
				return fmt.Errorf("--interval must be a positive number of seconds")
			}
			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()

			proj, err := resolveProject(s, "", project)
			if err != nil {
				return err
			}
			t := &store.Task{
				ProjectID:       proj.ID,
				Name:            name,
				Prompt:          prompt,
				Command:         command,
				IntervalSeconds: interval,
				Enabled:         true,
			}
			if err := s.CreateTask(t); err != nil {
				return err
			}
			fmt.Fprintf(opts.Stdout, "task created: %s (%s) on project %s\n", t.Name, t.ID, proj.Name)
			return nil
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "project name")
	cmd.Flags().StringVar(&name, "name", "", "task name")
	cmd.Flags().IntVar(&interval, "interval", 0, "interval seconds between runs")
	cmd.Flags().StringVar(&command, "command", "", "remote command (shell string or JSON argv)")
	cmd.Flags().StringVar(&prompt, "prompt", "", "instruction for the remote CLI agent")
	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("interval")
	_ = cmd.MarkFlagRequired("command")
	_ = cmd.MarkFlagRequired("prompt")
	return cmd
}

func newTaskListCmd(opts Options, openStore func() (*store.Store, error)) *cobra.Command {
	var project string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()

			projectID := ""
			if project != "" {
				proj, err := resolveProject(s, "", project)
				if err != nil {
					return err
				}
				projectID = proj.ID
			}

			tasks, err := s.ListTasks(projectID)
			if err != nil {
				return err
			}
			w := tabwriter.NewWriter(opts.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tPROJECT_ID\tNAME\tINTERVAL\tENABLED")
			for _, t := range tasks {
				fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%v\n",
					t.ID, t.ProjectID, t.Name, t.IntervalSeconds, t.Enabled)
			}
			return w.Flush()
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "filter by project name")
	return cmd
}

func newTaskDisableCmd(opts Options, openStore func() (*store.Store, error)) *cobra.Command {
	return newTaskSetEnabledCmd(opts, openStore, "disable", false)
}

func newTaskEnableCmd(opts Options, openStore func() (*store.Store, error)) *cobra.Command {
	return newTaskSetEnabledCmd(opts, openStore, "enable", true)
}

func newTaskSetEnabledCmd(opts Options, openStore func() (*store.Store, error), use string, enabled bool) *cobra.Command {
	var (
		project string
		name    string
		id      string
	)
	cmd := &cobra.Command{
		Use:   use,
		Short: fmt.Sprintf("%s a task for scheduling", use),
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" && (project == "" || name == "") {
				return fmt.Errorf("require --id, or both --project and --name")
			}
			if id != "" && (project != "" || name != "") {
				return fmt.Errorf("use either --id or --project/--name, not both")
			}
			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()

			t, err := resolveTask(s, id, project, name)
			if err != nil {
				return err
			}
			t.Enabled = enabled
			if err := s.UpdateTask(t); err != nil {
				return err
			}
			state := "disabled"
			if enabled {
				state = "enabled"
			}
			fmt.Fprintf(opts.Stdout, "task %s: %s (%s)\n", state, t.Name, t.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "project name")
	cmd.Flags().StringVar(&name, "name", "", "task name")
	cmd.Flags().StringVar(&id, "id", "", "task id")
	return cmd
}

func resolveTask(s *store.Store, id, projectName, taskName string) (*store.Task, error) {
	if id != "" {
		t, err := s.GetTask(id)
		if err != nil {
			return nil, fmt.Errorf("task %q: %w", id, err)
		}
		return t, nil
	}
	proj, err := resolveProject(s, "", projectName)
	if err != nil {
		return nil, err
	}
	t, err := s.GetTaskByProjectAndName(proj.ID, taskName)
	if err != nil {
		return nil, fmt.Errorf("task %q on project %q: %w", taskName, projectName, err)
	}
	return t, nil
}
