package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"master-agent/internal/store"
)

func newProjectCmd(opts Options, openStore func() (*store.Store, error)) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects (remote path + SSH target)",
	}
	cmd.AddCommand(newProjectAddCmd(opts, openStore))
	cmd.AddCommand(newProjectListCmd(opts, openStore))
	cmd.AddCommand(newProjectDisableCmd(opts, openStore))
	cmd.AddCommand(newProjectEnableCmd(opts, openStore))
	return cmd
}

func newProjectAddCmd(opts Options, openStore func() (*store.Store, error)) *cobra.Command {
	var (
		name       string
		path       string
		sshHost    string
		sshUser    string
		sshKey     string
		sshPort    int
	)
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()

			keyBytes, err := os.ReadFile(sshKey)
			if err != nil {
				return fmt.Errorf("read ssh key %q: %w", sshKey, err)
			}
			keyMaterial := string(keyBytes)
			if err := store.ValidateSSHPrivateKey(keyMaterial); err != nil {
				return err
			}

			p := &store.Project{
				Name:          name,
				Path:          path,
				SSHHost:       sshHost,
				SSHUser:       sshUser,
				SSHPort:       sshPort,
				SSHPrivateKey: keyMaterial,
				Enabled:       true,
			}
			if err := s.CreateProject(p); err != nil {
				return err
			}
			fmt.Fprintf(opts.Stdout, "project created: %s (%s)\n", p.Name, p.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "project name")
	cmd.Flags().StringVar(&path, "path", "", "absolute path on the worker")
	cmd.Flags().StringVar(&sshHost, "ssh-host", "", "SSH host or alias")
	cmd.Flags().StringVar(&sshUser, "ssh-user", "", "SSH user")
	cmd.Flags().StringVar(&sshKey, "ssh-key", "", "path to private key file (read and stored inline in SQLite)")
	cmd.Flags().IntVar(&sshPort, "ssh-port", 22, "SSH port")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("path")
	_ = cmd.MarkFlagRequired("ssh-host")
	_ = cmd.MarkFlagRequired("ssh-user")
	_ = cmd.MarkFlagRequired("ssh-key")
	return cmd
}

func newProjectListCmd(opts Options, openStore func() (*store.Store, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()

			projects, err := s.ListProjects()
			if err != nil {
				return err
			}
			w := tabwriter.NewWriter(opts.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tPATH\tSSH\tENABLED")
			for _, p := range projects {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s@%s:%d\t%v\n",
					p.ID, p.Name, p.Path, p.SSHUser, p.SSHHost, p.SSHPort, p.Enabled)
			}
			return w.Flush()
		},
	}
}

func newProjectDisableCmd(opts Options, openStore func() (*store.Store, error)) *cobra.Command {
	return newProjectSetEnabledCmd(opts, openStore, "disable", false)
}

func newProjectEnableCmd(opts Options, openStore func() (*store.Store, error)) *cobra.Command {
	return newProjectSetEnabledCmd(opts, openStore, "enable", true)
}

func newProjectSetEnabledCmd(opts Options, openStore func() (*store.Store, error), use string, enabled bool) *cobra.Command {
	var (
		name string
		id   string
	)
	cmd := &cobra.Command{
		Use:   use,
		Short: fmt.Sprintf("%s a project for scheduling", use),
		RunE: func(cmd *cobra.Command, args []string) error {
			if (name == "") == (id == "") {
				return fmt.Errorf("exactly one of --name or --id is required")
			}
			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()

			p, err := resolveProject(s, id, name)
			if err != nil {
				return err
			}
			p.Enabled = enabled
			if err := s.UpdateProject(p); err != nil {
				return err
			}
			state := "disabled"
			if enabled {
				state = "enabled"
			}
			fmt.Fprintf(opts.Stdout, "project %s: %s (%s)\n", state, p.Name, p.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "project name")
	cmd.Flags().StringVar(&id, "id", "", "project id")
	return cmd
}

func resolveProject(s *store.Store, id, name string) (*store.Project, error) {
	if id != "" {
		p, err := s.GetProject(id)
		if err != nil {
			return nil, fmt.Errorf("project %q: %w", id, err)
		}
		return p, nil
	}
	p, err := s.GetProjectByName(name)
	if err != nil {
		return nil, fmt.Errorf("project %q: %w", name, err)
	}
	return p, nil
}
