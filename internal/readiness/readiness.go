package readiness

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"projectscan/internal/model"
)

func Inspect(path string) model.ReadinessAudit {
	audit := model.ReadinessAudit{
		Readme:    fileExistsAny(path, "README.md", "readme.md", "README"),
		License:   fileExistsAny(path, "LICENSE", "LICENSE.md", "license.md"),
		Tests:     hasTestFiles(path),
		Container: fileExistsAny(path, "Dockerfile", "Containerfile"),
		CI:        dirExists(filepath.Join(path, ".github", "workflows")) || dirExists(filepath.Join(path, ".gitlab-ci.yml")),
		Deploy:    dirExists(filepath.Join(path, "deploy")) || dirExists(filepath.Join(path, "k8s")) || dirExists(filepath.Join(path, "manifests")),
	}
	audit.Score, audit.Signals = Score(audit)
	return audit
}

func Score(audit model.ReadinessAudit) (int, []string) {
	score := 0
	signals := []string{}
	add := func(ok bool, points int, name string) {
		if ok {
			score += points
			signals = append(signals, name)
		}
	}
	add(audit.Readme, 20, "readme")
	add(audit.Tests, 20, "tests")
	add(audit.License, 10, "license")
	add(audit.Container, 15, "container")
	add(audit.CI, 10, "ci")
	add(audit.Deploy, 15, "deploy")
	add(audit.Remote, 10, "remote")
	return score, signals
}

func fileExistsAny(path string, names ...string) bool {
	for _, name := range names {
		if info, err := os.Stat(filepath.Join(path, name)); err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func hasTestFiles(path string) bool {
	found := false
	_ = filepath.WalkDir(path, func(filePath string, entry fs.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if entry.IsDir() {
			if filePath != path && model.ShouldSkipDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		name := strings.ToLower(entry.Name())
		if strings.Contains(name, "test") || strings.HasSuffix(name, "_spec.ts") || strings.HasSuffix(name, ".spec.ts") {
			found = true
		}
		return nil
	})
	return found
}
