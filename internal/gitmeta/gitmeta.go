package gitmeta

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"projectscan/internal/model"
)

func Inspect(path string) model.GitMetadata {
	if _, err := os.Stat(filepath.Join(path, ".git")); err != nil {
		return model.GitMetadata{}
	}
	meta := model.GitMetadata{IsRepo: true}
	if out, err := RunCommand(path, "git", "branch", "--show-current"); err == nil {
		meta.Branch = strings.TrimSpace(out)
	}
	if out, err := RunCommand(path, "git", "status", "--porcelain"); err == nil {
		meta.Dirty = strings.TrimSpace(out) != ""
	}
	if out, err := RunCommand(path, "git", "remote", "get-url", "origin"); err == nil {
		meta.RemoteURL = strings.TrimSpace(out)
	}
	if out, err := RunCommand(path, "git", "log", "-1", "--format=%H%x00%s%x00%cI"); err == nil {
		parts := strings.Split(strings.TrimSpace(out), "\x00")
		if len(parts) >= 3 {
			meta.LastCommitHash = shortHash(parts[0])
			meta.LastCommitMessage = parts[1]
			meta.LastCommitDate = parts[2]
			if commitTime, err := time.Parse(time.RFC3339, parts[2]); err == nil {
				meta.DaysSinceCommit = int(time.Since(commitTime).Hours() / 24)
			}
		}
	}
	return meta
}

func RunCommand(dir string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func shortHash(hash string) string {
	if len(hash) <= 7 {
		return hash
	}
	return hash[:7]
}
