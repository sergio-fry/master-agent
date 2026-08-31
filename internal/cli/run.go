package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"master-agent/internal/store"
)

func newRunCmd(opts Options, openStore func() (*store.Store, error)) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Inspect run history",
	}
	cmd.AddCommand(newRunListCmd(opts, openStore))
	return cmd
}

func newRunListCmd(opts Options, openStore func() (*store.Store, error)) *cobra.Command {
	var (
		project string
		task    string
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List runs for a project (optionally filtered by task)",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()

			proj, err := resolveProject(s, "", project)
			if err != nil {
				return err
			}

			taskID := ""
			if task != "" {
				t, err := resolveTask(s, "", project, task)
				if err != nil {
					return err
				}
				taskID = t.ID
			}

			runs, err := s.ListRuns(proj.ID, taskID)
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(opts.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tTASK_ID\tSTATUS\tEXIT\tSTARTED\tFINISHED\tERROR")
			for _, r := range runs {
				exit := ""
				if r.ExitCode != nil {
					exit = fmt.Sprintf("%d", *r.ExitCode)
				}
				finished := ""
				if r.FinishedAt != nil {
					finished = *r.FinishedAt
				}
				errMsg := ""
				if r.ErrorMessage != nil {
					errMsg = *r.ErrorMessage
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					r.ID, r.TaskID, r.Status, exit, r.StartedAt, finished, errMsg)
			}
			return w.Flush()
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "project name")
	cmd.Flags().StringVar(&task, "task", "", "optional task name filter")
	_ = cmd.MarkFlagRequired("project")
	return cmd
}
