package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"projectscan/internal/model"
)

func TestParseOptionsSupportsExportCommandReportFormatAndProject(t *testing.T) {
	opts, err := ParseOptions([]string{"export", "/tmp/code", "--report", "audit", "--format", "json", "--project", "api"}, "/tmp/work")
	if err != nil {
		t.Fatalf("ParseOptions export: %v", err)
	}

	if opts.Mode != model.ModeExport || opts.RootPath != "/tmp/code" || opts.ReportType != model.ReportAudit || opts.ExportFormat != model.ExportJSON || opts.ProjectFilter != "api" {
		t.Fatalf("unexpected export opts: %#v", opts)
	}
}

func TestParseOptionsDefaultsExportCommandToAuditHTML(t *testing.T) {
	opts, err := ParseOptions([]string{"export", "/tmp/code"}, "/tmp/work")
	if err != nil {
		t.Fatalf("ParseOptions default export: %v", err)
	}

	if opts.Mode != model.ModeExport || opts.ReportType != model.ReportAudit || opts.ExportFormat != model.ExportHTML {
		t.Fatalf("expected export command to default to audit html, got %#v", opts)
	}
}

func TestParseOptionsMapsLegacyExportFlags(t *testing.T) {
	opts, err := ParseOptions([]string{"/tmp/code", "--audit", "--json"}, "/tmp/work")
	if err != nil {
		t.Fatalf("ParseOptions legacy audit json: %v", err)
	}
	if opts.Mode != model.ModeExport || opts.ReportType != model.ReportAudit || opts.ExportFormat != model.ExportJSON {
		t.Fatalf("expected audit json export request, got %#v", opts)
	}

	opts, err = ParseOptions([]string{"/tmp/code", "--markdown"}, "/tmp/work")
	if err != nil {
		t.Fatalf("ParseOptions legacy markdown: %v", err)
	}
	if opts.Mode != model.ModeExport || opts.ReportType != model.ReportSummary || opts.ExportFormat != model.ExportMarkdown {
		t.Fatalf("expected summary markdown export request, got %#v", opts)
	}
}

func TestParseOptionsSupportsExplicitTUIMode(t *testing.T) {
	opts, err := ParseOptions([]string{"tui", "/tmp/code"}, "/tmp/work")
	if err != nil {
		t.Fatalf("ParseOptions tui: %v", err)
	}
	if !opts.Interactive || opts.RootPath != "/tmp/code" {
		t.Fatalf("expected explicit tui mode with root path, got %#v", opts)
	}
}

func TestRunWithOptionsWritesExportAndPrintsOnlyPath(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "go.mod"), "module example.com/api\n")
	mustWrite(t, filepath.Join(root, "main.go"), "package main\nfunc main() {}\n")

	var out bytes.Buffer
	err := RunWithOptions(Options{
		RootPath:      root,
		Mode:          model.ModeExport,
		ReportType:    model.ReportAudit,
		ExportFormat:  model.ExportHTML,
		ProjectFilter: "",
	}, &out)
	if err != nil {
		t.Fatalf("RunWithOptions export: %v", err)
	}

	path := strings.TrimSpace(out.String())
	if !strings.HasPrefix(path, filepath.Join(root, "projectscan-exports")) || filepath.Ext(path) != ".html" {
		t.Fatalf("expected printed export path, got %q", path)
	}
	if strings.Contains(path, "<!doctype html>") || strings.Contains(out.String(), "Project Scan Audit") {
		t.Fatalf("expected path only, got %q", out.String())
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected export file at %s: %v", path, err)
	}
}

func mustWrite(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir parent: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
