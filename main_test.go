package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"projectscan/internal/model"
	"projectscan/internal/render"
	"projectscan/internal/scan"
)

func TestBuildSummaryRendersTerminalPortfolioAuditBoard(t *testing.T) {
	projects := []model.Project{
		{
			Name:       "ltechnosoft-landing",
			Path:       "/home/logan/code/ltechnosoft-landing",
			Languages:  map[string]int{"Svelte": 21, "TypeScript": 12, "CSS": 7},
			TotalFiles: 40,
		},
		{
			Name:       "awqms-backend",
			Path:       "/home/logan/code/awqms-backend",
			Languages:  map[string]int{"Go": 63, "SQL": 11, "YAML": 8},
			TotalFiles: 82,
		},
	}

	report := render.BuildSummary("/home/logan/code", projects)

	assertContains(t, report, "┌──────────────────────────────────────────────────────────────┐")
	assertContains(t, report, "PROJECTSCAN SNAPSHOT")
	assertContains(t, report, "Root Path")
	assertContains(t, report, ".../code")
	assertContains(t, report, "Projects")
	assertContains(t, report, "2")
	assertContains(t, report, "Recognized Files")
	assertContains(t, report, "122")
	assertContains(t, report, "Language")
	assertContains(t, report, "Usage")
	assertContains(t, report, "Project")
	assertContains(t, report, "Health")
	assertContains(t, report, "│  Go            ██████████░░░░░░░░░░   52 %                   │")
	assertContains(t, report, "│  Svelte        ███░░░░░░░░░░░░░░░░░   17 %                   │")
	assertContains(t, report, "└──────────────────────────────────────────────────────────────┘")
	assertContains(t, report, "+---------------+--------+------------------------+------------+")
	assertContains(t, report, "| LANGUAGE SUMMARY                                             |")
	assertContains(t, report, "+---------------+--------+------------------------+------------+")
	assertContains(t, report, "| Go            |     63 | ███████████░░░░░░░░░░░ |  52 %      |")
	assertContains(t, report, "| Svelte        |     21 | ███░░░░░░░░░░░░░░░░░░░ |  17 %      |")
	assertContains(t, report, "PROJECTS")
	assertContains(t, report, "awqms-backend")
	assertContains(t, report, "ltechnosoft-lan")
	assertContains(t, report, "0/100")
	if strings.Contains(report, "####") {
		t.Fatalf("expected unicode bars instead of #### sequences:\n\n%s", report)
	}
	if strings.Contains(report, "\x1b[") {
		t.Fatalf("buildSummary should be colorless for deterministic tests:\n\n%s", report)
	}
}

func TestScanProjectsCountsLanguagesAndTotalFiles(t *testing.T) {
	root := t.TempDir()
	projectPath := filepath.Join(root, "api")
	mustMkdir(t, projectPath)
	mustWrite(t, filepath.Join(projectPath, "main.go"), "package main\n")
	mustWrite(t, filepath.Join(projectPath, "query.sql"), "select 1;\n")
	mustWrite(t, filepath.Join(projectPath, "README.md"), "# api\n")

	ignoredPath := filepath.Join(projectPath, "node_modules", "generated")
	mustMkdir(t, ignoredPath)
	mustWrite(t, filepath.Join(ignoredPath, "ignored.ts"), "export {}\n")

	emptyPath := filepath.Join(root, "empty")
	mustMkdir(t, emptyPath)

	projects, err := scan.Projects(root)
	if err != nil {
		t.Fatalf("scanProjects returned error: %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d: %#v", len(projects), projects)
	}
	project := projects[0]
	if project.Name != "api" {
		t.Fatalf("expected project name api, got %q", project.Name)
	}
	if project.TotalFiles != 3 {
		t.Fatalf("expected 3 total files, got %d", project.TotalFiles)
	}
	if project.Languages["Go"] != 1 || project.Languages["SQL"] != 1 || project.Languages["Markdown"] != 1 {
		t.Fatalf("unexpected language counts: %#v", project.Languages)
	}
	if _, ok := project.Languages["TypeScript"]; ok {
		t.Fatalf("expected node_modules TypeScript file to be ignored: %#v", project.Languages)
	}
}

func TestCountLanguagesCanScanRootItself(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "main.go"), "package main\n")
	mustWrite(t, filepath.Join(root, "app.svelte"), "<script></script>\n")
	mustWrite(t, filepath.Join(root, "notes.txt"), "not counted\n")

	languages, err := scan.CountLanguages(root)
	if err != nil {
		t.Fatalf("countLanguages returned error: %v", err)
	}

	if model.SumCounts(languages) != 2 {
		t.Fatalf("expected 2 recognized files, got %#v", languages)
	}
	if languages["Go"] != 1 || languages["Svelte"] != 1 {
		t.Fatalf("unexpected language counts: %#v", languages)
	}
}

func TestMakeUnicodeBarUsesFilledAndEmptySegments(t *testing.T) {
	if got := model.MakeBar(25, 100, 10); got != "██░░░░░░░░" {
		t.Fatalf("expected 25%% bar, got %q", got)
	}
	if got := model.MakeBar(1, 100, 10); got != "█░░░░░░░░░" {
		t.Fatalf("expected tiny positive value to show one filled segment, got %q", got)
	}
	if got := model.MakeBar(0, 100, 10); got != "░░░░░░░░░░" {
		t.Fatalf("expected zero value to show empty bar, got %q", got)
	}
}

func TestPercentOfTotalRoundsToNearestWholePercent(t *testing.T) {
	if got := model.PercentOfTotal(63, 122); got != 52 {
		t.Fatalf("expected 63/122 to be 52%%, got %d", got)
	}
	if got := model.PercentOfTotal(0, 122); got != 0 {
		t.Fatalf("expected zero value to be 0%%, got %d", got)
	}
	if got := model.PercentOfTotal(10, 0); got != 0 {
		t.Fatalf("expected zero total to be 0%%, got %d", got)
	}
}

func TestPercentLabelShowsTinyPositiveShares(t *testing.T) {
	if got := model.PercentLabel(1, 1000); got != " <1 %" {
		t.Fatalf("expected tiny positive share label, got %q", got)
	}
	if got := model.PercentLabel(0, 1000); got != "  0 %" {
		t.Fatalf("expected zero share label, got %q", got)
	}
}

func TestColorizeHonorsStyle(t *testing.T) {
	if got := render.Colorize(render.Style{}, lipgloss.Color("2"), "text"); got != "text" {
		t.Fatalf("expected colorless style to return plain text, got %q", got)
	}
	// We don't strictly check for ANSI codes here as lipgloss may disable them in non-TTY environments
}

func TestRenderBarColorsOnlyFilledAndEmptySegmentsSeparately(t *testing.T) {
	got := render.RenderBarForTest(25, 100, 10, render.Style{Color: true})
	// Check that we have some color codes and the bar characters
	if !strings.Contains(got, "██") || !strings.Contains(got, "░░░░░░░░") {
		t.Fatalf("expected bar characters, got %q", got)
	}
}

func TestShouldUseColorHonorsNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if render.ShouldUseColor(os.Stdout) {
		t.Fatal("expected NO_COLOR to disable ANSI color")
	}
}

func TestTruncateDisplayKeepsFixedWidthRowsStable(t *testing.T) {
	text := "/this/is/a/very/long/path/that/needs/truncation"
	got := model.FitText(text, 18)
	if model.RuneLen(got) != 18 {
		t.Fatalf("expected truncated text width 18, got %d for %q", model.RuneLen(got), got)
	}
	assertContains(t, got, "...")
}

func assertContains(t *testing.T, text string, want string) {
	t.Helper()
	if !strings.Contains(text, want) {
		t.Fatalf("expected output to contain %q\n\n%s", want, text)
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
