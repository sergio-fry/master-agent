package runner

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path"
	"strings"

	"master-agent/internal/store"
)

// RemoteDirEntry is one subdirectory in a remote listing.
type RemoteDirEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// RemoteDirListing is a single level of a remote directory tree.
type RemoteDirListing struct {
	Path    string           `json:"path"`
	Parent  string           `json:"parent"`
	Entries []RemoteDirEntry `json:"entries"`
}

// SSHBrowser lists directories on a remote host over SSH.
type SSHBrowser struct {
	SSHPath string

	commandContext func(ctx context.Context, name string, arg ...string) *exec.Cmd
}

// ListDirs returns immediate child directories of dirPath, or the SSH user's home when empty.
func (b *SSHBrowser) ListDirs(ctx context.Context, project store.Project, dirPath string) (RemoteDirListing, error) {
	if err := validateSSHTestProject(project); err != nil {
		return RemoteDirListing{}, err
	}

	keyPath, cleanupKey, err := WriteTempSSHKey(project.SSHPrivateKey)
	if err != nil {
		return RemoteDirListing{}, err
	}
	defer cleanupKey()

	hostKeyLine := strings.TrimSpace(project.SSHHostKey)
	if hostKeyLine == "" {
		tester := &SSHTester{SSHPath: b.SSHPath, commandContext: b.commandContext}
		scanned, err := tester.ScanHostKey(ctx, project)
		if err != nil {
			return RemoteDirListing{}, err
		}
		hostKeyLine = scanned
	}

	khPath, cleanupKH, err := WriteTempKnownHosts(project, hostKeyLine)
	if err != nil {
		return RemoteDirListing{}, err
	}
	defer cleanupKH()

	resolved, err := b.resolvePath(ctx, project, keyPath, khPath, dirPath)
	if err != nil {
		return RemoteDirListing{}, err
	}

	lines, err := b.listChildNames(ctx, project, keyPath, khPath, resolved)
	if err != nil {
		return RemoteDirListing{}, err
	}

	entries := make([]RemoteDirEntry, 0, len(lines))
	for _, name := range lines {
		name = strings.TrimSpace(name)
		if name == "" || name == "." || name == ".." {
			continue
		}
		entries = append(entries, RemoteDirEntry{
			Name: name,
			Path: joinRemotePath(resolved, name),
		})
	}

	parent := parentRemotePath(resolved)
	return RemoteDirListing{
		Path:    resolved,
		Parent:  parent,
		Entries: entries,
	}, nil
}

func (b *SSHBrowser) sshBin() string {
	if b.SSHPath != "" {
		return b.SSHPath
	}
	return "ssh"
}

func (b *SSHBrowser) command(ctx context.Context, name string, arg ...string) *exec.Cmd {
	if b.commandContext != nil {
		return b.commandContext(ctx, name, arg...)
	}
	return exec.CommandContext(ctx, name, arg...)
}

func (b *SSHBrowser) runRemote(ctx context.Context, project store.Project, keyPath, khPath, remote string) (string, error) {
	args, err := buildTestSSHArgs(project, keyPath, khPath, remote)
	if err != nil {
		return "", err
	}
	cmd := b.command(ctx, b.sshBin(), args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if isAuthLike(stderr.String(), err) {
			return "", &SSHTestError{Code: SSHCodeAuthFailed, Message: classifySSHFailure(stderr.String(), err)}
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = "remote directory listing failed"
		}
		return "", &SSHTestError{Code: SSHCodePathNotFound, Message: msg}
	}
	return stdout.String(), nil
}

func (b *SSHBrowser) resolvePath(ctx context.Context, project store.Project, keyPath, khPath, dirPath string) (string, error) {
	dirPath = strings.TrimSpace(dirPath)
	if dirPath == "" {
		out, err := b.runRemote(ctx, project, keyPath, khPath, "bash -lc 'cd ~ && pwd'")
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(out), nil
	}
	quoted := shellSingleQuote(dirPath)
	out, err := b.runRemote(ctx, project, keyPath, khPath,
		fmt.Sprintf("bash -lc 'd=%s; if [ ! -d \"$d\" ]; then exit 2; fi; cd \"$d\" && pwd'", quoted))
	if err != nil {
		return "", &SSHTestError{Code: SSHCodePathNotFound, Message: "remote path does not exist or is not a directory"}
	}
	return strings.TrimSpace(out), nil
}

func (b *SSHBrowser) listChildNames(ctx context.Context, project store.Project, keyPath, khPath, dirPath string) ([]string, error) {
	quoted := shellSingleQuote(dirPath)
	script := fmt.Sprintf(
		"bash -lc 'p=%s; ls -1p -- \"$p\" | while IFS= read -r e; do case \"$e\" in */) printf \"%%s\\n\" \"${e%%/}\";; esac; done'",
		quoted,
	)
	out, err := b.runRemote(ctx, project, keyPath, khPath, script)
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

func joinRemotePath(base, name string) string {
	base = strings.TrimSuffix(base, "/")
	if base == "" {
		return "/" + name
	}
	return base + "/" + name
}

func parentRemotePath(dir string) string {
	dir = strings.TrimSuffix(dir, "/")
	if dir == "" || dir == "/" {
		return ""
	}
	parent := path.Dir(dir)
	if parent == "." {
		return "/"
	}
	return parent
}
