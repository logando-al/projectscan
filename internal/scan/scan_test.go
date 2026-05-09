package scan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceScansManifestRootAsSingleProject(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "go.mod"), "module example\n")
	mustWrite(t, filepath.Join(root, "main.go"), "package main\n")
	mustMkdir(t, filepath.Join(root, "internal", "render"))
	mustWrite(t, filepath.Join(root, "internal", "render", "render.go"), "package render\n")

	report, err := Workspace(root, Options{})
	if err != nil {
		t.Fatalf("Workspace: %v", err)
	}
	if len(report.Projects) != 1 {
		t.Fatalf("expected root to scan as one project, got %#v", report.Projects)
	}
	project := report.Projects[0]
	if project.Name != filepath.Base(root) {
		t.Fatalf("expected root project name %q, got %q", filepath.Base(root), project.Name)
	}
	if project.Path != root {
		t.Fatalf("expected root project path %q, got %q", root, project.Path)
	}
	if project.TotalFiles != 2 || report.TotalFiles != 2 {
		t.Fatalf("expected root and internal Go files to be counted, got project=%d report=%d langs=%#v", project.TotalFiles, report.TotalFiles, project.Languages)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func mustWrite(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
