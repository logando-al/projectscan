package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"projectscan/internal/cli"
	"projectscan/internal/export"
	"projectscan/internal/gitmeta"
	"projectscan/internal/model"
	"projectscan/internal/render"
	"projectscan/internal/scan"
	"projectscan/internal/tui"
)

func TestParseCLIOptionsSupportsModesAndNoArgLauncher(t *testing.T) {
	opts, err := cli.ParseOptions([]string{}, "/tmp/work")
	if err != nil {
		t.Fatalf("parseCLIOptions no args: %v", err)
	}
	if !opts.Interactive || opts.RootPath != "/tmp/work" || opts.Mode != model.ModeSummary {
		t.Fatalf("unexpected no-arg opts: %#v", opts)
	}

	opts, err = cli.ParseOptions([]string{"/tmp/code", "--audit", "--no-color"}, "/tmp/work")
	if err != nil {
		t.Fatalf("parseCLIOptions audit: %v", err)
	}
	if opts.Interactive || opts.RootPath != "/tmp/code" || opts.Mode != model.ModeAudit || !opts.NoColor {
		t.Fatalf("unexpected audit opts: %#v", opts)
	}

	opts, err = cli.ParseOptions([]string{"--json", "--config", "/tmp/projectscan.toml", "/tmp/code"}, "/tmp/work")
	if err != nil {
		t.Fatalf("parseCLIOptions json: %v", err)
	}
	if opts.Mode != model.ModeExport || opts.ExportFormat != model.ExportJSON || opts.ReportType != model.ReportSummary || opts.ConfigPath != "/tmp/projectscan.toml" || opts.RootPath != "/tmp/code" {
		t.Fatalf("unexpected json opts: %#v", opts)
	}
}

func TestParseCLIOptionsSupportsExportCommandReportFormatAndProject(t *testing.T) {
	opts, err := cli.ParseOptions([]string{"export", "/tmp/code", "--report", "audit", "--format", "json", "--project", "api"}, "/tmp/work")
	if err != nil {
		t.Fatalf("parseCLIOptions export: %v", err)
	}
	if opts.Mode != model.ModeExport || opts.RootPath != "/tmp/code" || opts.ReportType != model.ReportAudit || opts.ExportFormat != model.ExportJSON || opts.ProjectFilter != "api" {
		t.Fatalf("unexpected export opts: %#v", opts)
	}

	opts, err = cli.ParseOptions([]string{"/tmp/code", "--audit", "--json"}, "/tmp/work")
	if err != nil {
		t.Fatalf("parseCLIOptions legacy audit json: %v", err)
	}
	if opts.Mode != model.ModeExport || opts.ReportType != model.ReportAudit || opts.ExportFormat != model.ExportJSON {
		t.Fatalf("expected legacy audit json to map into export request, got %#v", opts)
	}

	opts, err = cli.ParseOptions([]string{"/tmp/code", "--markdown"}, "/tmp/work")
	if err != nil {
		t.Fatalf("parseCLIOptions legacy markdown: %v", err)
	}
	if opts.Mode != model.ModeExport || opts.ReportType != model.ReportSummary || opts.ExportFormat != model.ExportMarkdown {
		t.Fatalf("expected legacy markdown to export summary markdown, got %#v", opts)
	}
}

func TestScanWorkspaceLoadsConfigIgnoreGitMetadataAndScoring(t *testing.T) {
	root := t.TempDir()
	apiPath := filepath.Join(root, "api")
	mustMkdir(t, apiPath)
	mustWrite(t, filepath.Join(apiPath, "main.go"), "package main\n")
	mustWrite(t, filepath.Join(apiPath, "main_test.go"), "package main\n")
	mustWrite(t, filepath.Join(apiPath, "README.md"), "# API\n")
	mustWrite(t, filepath.Join(apiPath, "LICENSE"), "MIT\n")
	mustWrite(t, filepath.Join(apiPath, "Dockerfile"), "FROM scratch\n")
	mustMkdir(t, filepath.Join(apiPath, ".github", "workflows"))
	mustWrite(t, filepath.Join(apiPath, ".github", "workflows", "ci.yml"), "name: ci\n")
	initGitRepo(t, apiPath)

	ignoredPath := filepath.Join(root, "ignored-app")
	mustMkdir(t, ignoredPath)
	mustWrite(t, filepath.Join(ignoredPath, "main.go"), "package main\n")

	mustWrite(t, filepath.Join(root, ".projectscanignore"), "ignored-app\n")
	mustWrite(t, filepath.Join(root, ".projectscan.toml"), `
[projects.api]
label = "archived"
display_name = "API Service"
archived_reason = "replaced by v2"
pinned = true
`)

	report, err := scan.Workspace(root, scan.Options{})
	if err != nil {
		t.Fatalf("scanWorkspace: %v", err)
	}
	if len(report.Projects) != 1 {
		t.Fatalf("expected ignored project to be excluded, got %#v", report.Projects)
	}

	project := report.Projects[0]
	if project.Name != "API Service" || project.RawName != "api" {
		t.Fatalf("expected config display name and raw name, got %#v", project)
	}
	if project.Label != model.LabelArchived || project.ArchivedReason != "replaced by v2" || !project.Pinned {
		t.Fatalf("expected config scoring override, got %#v", project)
	}
	if !project.Git.IsRepo || project.Git.Branch == "" || project.Git.LastCommitHash == "" {
		t.Fatalf("expected git metadata, got %#v", project.Git)
	}
	if !project.Readiness.Readme || !project.Readiness.Tests || !project.Readiness.License || !project.Readiness.Container || !project.Readiness.CI {
		t.Fatalf("expected readiness signals, got %#v", project.Readiness)
	}
	if report.LabelCounts[model.LabelArchived] != 1 {
		t.Fatalf("expected archived label count, got %#v", report.LabelCounts)
	}
}

func TestProductionReadyScoringUsesLocalSignals(t *testing.T) {
	root := t.TempDir()
	appPath := filepath.Join(root, "ship-ready")
	mustMkdir(t, appPath)
	mustWrite(t, filepath.Join(appPath, "main.go"), "package main\n")
	mustWrite(t, filepath.Join(appPath, "main_test.go"), "package main\n")
	mustWrite(t, filepath.Join(appPath, "README.md"), "# Ready\n")
	mustWrite(t, filepath.Join(appPath, "LICENSE"), "MIT\n")
	mustWrite(t, filepath.Join(appPath, "Dockerfile"), "FROM scratch\n")
	mustMkdir(t, filepath.Join(appPath, "deploy"))
	mustWrite(t, filepath.Join(appPath, "deploy", "app.yaml"), "apiVersion: v1\n")
	initGitRepo(t, appPath)

	report, err := scan.Workspace(root, scan.Options{})
	if err != nil {
		t.Fatalf("scanWorkspace: %v", err)
	}
	if got := report.Projects[0].Label; got != model.LabelProductionReady {
		t.Fatalf("expected production-ready, got %q with project %#v", got, report.Projects[0])
	}
}

func TestJSONAndMarkdownExportsAreColorlessAndStructured(t *testing.T) {
	report := model.WorkspaceReport{
		RootPath: "/tmp/code",
		Projects: []model.Project{
			{
				Name:       "api",
				Path:       "/tmp/code/api",
				Languages:  map[string]int{"Go": 2},
				TotalFiles: 2,
				Label:      model.LabelExperiment,
				Git:        model.GitMetadata{IsRepo: true, Branch: "main"},
				Readiness:  model.ReadinessAudit{Readme: true},
			},
		},
		LabelCounts: map[string]int{model.LabelExperiment: 1},
		GeneratedAt: time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC),
	}

	jsonOutput, err := export.BuildJSONReport(report)
	if err != nil {
		t.Fatalf("buildJSONReport: %v", err)
	}
	if containsANSI(jsonOutput) {
		t.Fatalf("json output should be colorless: %q", jsonOutput)
	}
	var decoded model.WorkspaceReport
	if err := json.Unmarshal([]byte(jsonOutput), &decoded); err != nil {
		t.Fatalf("json should decode: %v", err)
	}
	if decoded.Projects[0].Label != model.LabelExperiment {
		t.Fatalf("expected label in json, got %#v", decoded.Projects[0])
	}
	if decoded.RootPath != ".../code" {
		t.Fatalf("expected compact root path in json, got %q", decoded.RootPath)
	}

	markdown := export.BuildMarkdownReport(report)
	assertContains(t, markdown, "# Projectscan Report")
	assertContains(t, markdown, "- Root Path: `.../code`")
	assertContains(t, markdown, "| api | experiment |")
	if containsANSI(markdown) {
		t.Fatalf("markdown output should be colorless: %q", markdown)
	}
}

func TestSelectableReportExportsSupportCSVMarkdownAndHTML(t *testing.T) {
	report := sampleExportReport()

	csvOutput, err := export.BuildReport(report, model.ExportRequest{ReportType: model.ReportGit, Format: model.ExportCSV})
	if err != nil {
		t.Fatalf("buildReportExport git csv: %v", err)
	}
	assertContains(t, csvOutput, "Project,Branch,Dirty,Remote,Last Commit")
	assertContains(t, csvOutput, "api,main,clean")
	if containsANSI(csvOutput) {
		t.Fatalf("csv output should be colorless: %q", csvOutput)
	}

	markdown, err := export.BuildReport(report, model.ExportRequest{ReportType: model.ReportReadiness, Format: model.ExportMarkdown})
	if err != nil {
		t.Fatalf("buildReportExport readiness markdown: %v", err)
	}
	assertContains(t, markdown, "## Readiness Audit")
	assertContains(t, markdown, "| Project | README | Tests | License | Container | CI | Deploy | Remote | Score |")
	assertContains(t, markdown, "| api | yes | yes | no | yes | yes | no | yes | 75 |")
	if containsANSI(markdown) {
		t.Fatalf("markdown output should be colorless: %q", markdown)
	}

	html, err := export.BuildReport(report, model.ExportRequest{ReportType: model.ReportPortfolio, Format: model.ExportHTML})
	if err != nil {
		t.Fatalf("buildReportExport portfolio html: %v", err)
	}
	assertContains(t, html, "<!doctype html>")
	assertContains(t, html, "Project Portfolio")
	assertContains(t, html, "<td>api</td>")
	if containsANSI(html) {
		t.Fatalf("html output should be colorless: %q", html)
	}
}

func TestProjectFilterLimitsExports(t *testing.T) {
	report := sampleExportReport()

	output, err := export.BuildReport(report, model.ExportRequest{ReportType: model.ReportAudit, Format: model.ExportJSON, ProjectFilter: "api"})
	if err != nil {
		t.Fatalf("buildReportExport filtered json: %v", err)
	}
	assertContains(t, output, `"report_type": "audit"`)
	assertContains(t, output, `"name": "api"`)
	if strings.Contains(output, `"name": "web"`) {
		t.Fatalf("expected project filter to exclude web project:\n%s", output)
	}
}

func TestRoadmapReportsExportRealAuditData(t *testing.T) {
	report := sampleExportReport()

	output, err := export.BuildReport(report, model.ExportRequest{ReportType: model.ReportLOC, Format: model.ExportMarkdown})
	if err != nil {
		t.Fatalf("buildReportExport loc: %v", err)
	}
	assertContains(t, output, "Lines of Code")
	assertContains(t, output, "Total LOC")

	jsonOutput, err := export.BuildReport(report, model.ExportRequest{ReportType: model.ReportSafety, Format: model.ExportJSON})
	if err != nil {
		t.Fatalf("buildReportExport safety: %v", err)
	}
	assertContains(t, jsonOutput, `"report_type": "safety"`)
	assertContains(t, jsonOutput, `"secret_values": "redacted"`)
	if strings.Contains(jsonOutput, "planned roadmap audit module") {
		t.Fatalf("expected implemented safety audit, got placeholder:\n%s", jsonOutput)
	}
}

func TestTUIAppModelStateTransitions(t *testing.T) {
	app := tui.NewAppModel(sampleExportReport(), 38*time.Millisecond, nil)
	updatedApp, _ := app.Update(tea.WindowSizeMsg{Width: 200, Height: 40})
	app = updatedApp.(tui.AppModel)
	if app.Report.RootPath != "/tmp/code" || app.Screen != tui.ScreenHome {
		t.Fatalf("unexpected TUI defaults: %#v", app)
	}

	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = updated.(tui.AppModel)
	if app.Cursor != 1 {
		t.Fatalf("expected cursor to move, got %#v", app)
	}

	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	app = updated.(tui.AppModel)
	if app.Screen != tui.ScreenAudit {
		t.Fatalf("expected audit hotkey to open audit screen: %#v", app)
	}
	if !stringsContains(app.View(), "Open Source Safety") {
		t.Fatalf("expected TUI audit view to mention roadmap modules")
	}
}

func TestSQLTableTitleAndBordersUseColumnWidth(t *testing.T) {
	var b strings.Builder
	widths := []int{5, 8}
	render.WriteTableTopForTest(&b, "TINY", widths, render.Style{})
	render.WriteTableRowForTest(&b, []string{"A", "B"}, widths, render.Style{})
	render.WriteTableBottomForTest(&b, widths, render.Style{})

	lines := strings.Split(strings.TrimSpace(b.String()), "\n")
	if len(lines) != 5 {
		t.Fatalf("expected 5 table lines, got %d:\n%s", len(lines), b.String())
	}
	width := len(lines[0])
	for _, line := range lines {
		if len(line) != width {
			t.Fatalf("expected SQL table lines to share width %d, got %d for %q\n%s", width, len(line), line, b.String())
		}
	}
	if lines[0] != "+-------+----------+" {
		t.Fatalf("expected top border to follow column widths, got %q", lines[0])
	}
	if lines[1] != "| TINY             |" {
		t.Fatalf("expected title row to follow table width, got %q", lines[1])
	}
}

func initGitRepo(t *testing.T, path string) {
	t.Helper()
	runGit(t, path, "init", "-b", "main")
	runGit(t, path, "config", "user.email", "logan@example.com")
	runGit(t, path, "config", "user.name", "Logan")
	runGit(t, path, "remote", "add", "origin", "git@example.com:logan/example.git")
	runGit(t, path, "add", ".")
	runGit(t, path, "commit", "-m", "initial commit")
}

func runGit(t *testing.T, path string, args ...string) {
	t.Helper()
	output, err := gitmeta.RunCommand(path, "git", args...)
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}

func containsANSI(text string) bool {
	return strings.Contains(text, "\x1b[")
}

func stringsContains(text string, want string) bool {
	return strings.Contains(text, want)
}

func sampleExportReport() model.WorkspaceReport {
	projects := []model.Project{
		{
			Name:          "api",
			Path:          "/tmp/code/api",
			Languages:     map[string]int{"Go": 10, "SQL": 2},
			TotalFiles:    12,
			Label:         model.LabelProductionReady,
			MainLanguages: []string{"Go", "SQL"},
			Git: model.GitMetadata{
				IsRepo:          true,
				Branch:          "main",
				Dirty:           false,
				RemoteURL:       "git@example.com:logan/api.git",
				LastCommitHash:  "abc1234",
				DaysSinceCommit: 3,
			},
			Readiness: model.ReadinessAudit{
				Readme:    true,
				Tests:     true,
				License:   false,
				Container: true,
				CI:        true,
				Deploy:    false,
				Remote:    true,
				Score:     75,
			},
		},
		{
			Name:          "web",
			Path:          "/tmp/code/web",
			Languages:     map[string]int{"TypeScript": 8},
			TotalFiles:    8,
			Label:         model.LabelExperiment,
			MainLanguages: []string{"TypeScript"},
			Git:           model.GitMetadata{},
			Readiness:     model.ReadinessAudit{Readme: true, Score: 20},
		},
	}
	languages, totalFiles := model.SummarizeProjects(projects)
	return model.WorkspaceReport{
		RootPath:        "/tmp/code",
		Projects:        projects,
		LanguageSummary: languages,
		TotalFiles:      totalFiles,
		LabelCounts: map[string]int{
			model.LabelProductionReady: 1,
			model.LabelExperiment:      1,
			model.LabelArchived:        0,
		},
		GeneratedAt: time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC),
	}
}
