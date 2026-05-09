package export

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"projectscan/internal/model"
)

func TestBuildReportExportsSelectableFormats(t *testing.T) {
	report := sampleReport()

	csvOutput, err := BuildReport(report, model.ExportRequest{ReportType: model.ReportGit, Format: model.ExportCSV})
	if err != nil {
		t.Fatalf("BuildReport git csv: %v", err)
	}
	assertContains(t, csvOutput, "Project,Branch,Dirty,Remote,Last Commit")
	assertContains(t, csvOutput, "api,main,clean")

	markdown, err := BuildReport(report, model.ExportRequest{ReportType: model.ReportReadiness, Format: model.ExportMarkdown})
	if err != nil {
		t.Fatalf("BuildReport readiness markdown: %v", err)
	}
	assertContains(t, markdown, "## Readiness Audit")
	assertContains(t, markdown, "| api | yes | yes | no | yes | yes | no | yes | 75 |")

	html, err := BuildReport(report, model.ExportRequest{ReportType: model.ReportPortfolio, Format: model.ExportHTML})
	if err != nil {
		t.Fatalf("BuildReport portfolio html: %v", err)
	}
	assertContains(t, html, "<!doctype html>")
	assertContains(t, html, "cdn.jsdelivr.net/npm/chart.js")
	assertContains(t, html, "#00ADD8")
	assertContains(t, html, "Project Portfolio")
	assertContains(t, html, "<td>api</td>")
	assertContains(t, html, ".../code")
}

func TestWriteReportCreatesExportFolderAndWritesSelectedFormat(t *testing.T) {
	root := t.TempDir()
	report := sampleReport()
	report.RootPath = root

	result, err := WriteReport(report, model.ExportRequest{ReportType: model.ReportAudit, Format: model.ExportHTML}, "")
	if err != nil {
		t.Fatalf("WriteReport audit html: %v", err)
	}

	if filepath.Dir(result.Path) != filepath.Join(root, DefaultDirName) {
		t.Fatalf("expected export in default dir, got %s", result.Path)
	}
	if filepath.Ext(result.Path) != ".html" {
		t.Fatalf("expected html extension, got %s", result.Path)
	}
	if result.Bytes == 0 {
		t.Fatalf("expected byte count, got %#v", result)
	}
	content, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatalf("read written export: %v", err)
	}
	assertContains(t, string(content), "Project Scan Audit")
	assertContains(t, string(content), "chart.umd.min.js")
}

func TestHTMLAuditExportFollowsDesignSections(t *testing.T) {
	output, err := BuildReport(sampleReport(), model.ExportRequest{ReportType: model.ReportAudit, Format: model.ExportHTML})
	if err != nil {
		t.Fatalf("BuildReport audit html: %v", err)
	}

	assertContains(t, output, "Project Inventory")
	assertContains(t, output, "Audit Modules")
	assertContains(t, output, "Safety Findings")
	assertContains(t, output, "Lines of Code")
	assertContains(t, output, "Export Map")
	assertContains(t, output, "module-grid")
	assertContains(t, output, "module-card")
	assertContains(t, output, "portfolioScoreChart")
	assertContains(t, output, "cdn.jsdelivr.net/npm/chart.js")
	assertContains(t, output, "#00ADD8")
	assertContains(t, output, ".../code")
	assertContains(t, output, "/tmp/code/api")
	assertContains(t, output, "Suggested Fix")
}

func TestHTMLHeroOmitsImplementationBadge(t *testing.T) {
	output, err := BuildReport(sampleReport(), model.ExportRequest{ReportType: model.ReportAudit, Format: model.ExportHTML})
	if err != nil {
		t.Fatalf("BuildReport audit html: %v", err)
	}

	assertNotContains(t, output, "format: html")
	assertNotContains(t, output, "primary: #00ADD8")
	assertNotContains(t, output, "renderer: Chart.js")
	assertNotContains(t, output, "export-badge")
}

func TestHTMLHeroTitleStylesProjectWordOnly(t *testing.T) {
	output, err := BuildReport(sampleReport(), model.ExportRequest{ReportType: model.ReportAudit, Format: model.ExportHTML})
	if err != nil {
		t.Fatalf("BuildReport audit html: %v", err)
	}

	assertContains(t, output, `<h1 class="hero-title"><span class="hero-title-project">PROJECT</span> <span>SCAN</span> <span>AUDIT</span></h1>`)
	assertContains(t, output, `.hero-title-project{color:var(--go-blue)}`)
}

func TestExportsUseCompactRootPath(t *testing.T) {
	report := sampleReport()

	markdown, err := BuildReport(report, model.ExportRequest{ReportType: model.ReportAudit, Format: model.ExportMarkdown})
	if err != nil {
		t.Fatalf("BuildReport markdown: %v", err)
	}
	assertContains(t, markdown, "- Root Path: `.../code`")
	if strings.Contains(markdown, "- Root Path: `/tmp/code`") {
		t.Fatalf("expected compact root path in markdown export:\n%s", markdown)
	}

	jsonOutput, err := BuildReport(report, model.ExportRequest{ReportType: model.ReportAudit, Format: model.ExportJSON})
	if err != nil {
		t.Fatalf("BuildReport json: %v", err)
	}
	assertContains(t, jsonOutput, `"root_path": ".../code"`)
}

func TestWriteReportHandlesFilenameCollisions(t *testing.T) {
	root := t.TempDir()
	report := sampleReport()
	report.RootPath = root

	first, err := WriteReport(report, model.ExportRequest{ReportType: model.ReportGit, Format: model.ExportCSV}, "")
	if err != nil {
		t.Fatalf("WriteReport first: %v", err)
	}
	second, err := WriteReport(report, model.ExportRequest{ReportType: model.ReportGit, Format: model.ExportCSV}, "")
	if err != nil {
		t.Fatalf("WriteReport second: %v", err)
	}

	if first.Path == second.Path {
		t.Fatalf("expected collision-safe unique paths, got %s", first.Path)
	}
	if !strings.Contains(filepath.Base(second.Path), "-01.csv") {
		t.Fatalf("expected -01 collision suffix, got %s", second.Path)
	}
}

func TestBuildReportFiltersProjectsAndKeepsExportsColorless(t *testing.T) {
	output, err := BuildReport(sampleReport(), model.ExportRequest{ReportType: model.ReportAudit, Format: model.ExportJSON, ProjectFilter: "api"})
	if err != nil {
		t.Fatalf("BuildReport filtered json: %v", err)
	}

	assertContains(t, output, `"report_type": "audit"`)
	assertContains(t, output, `"name": "api"`)
	if strings.Contains(output, `"name": "web"`) {
		t.Fatalf("expected project filter to exclude web project:\n%s", output)
	}
	if strings.Contains(output, "\x1b[") {
		t.Fatalf("export output should not contain ANSI color: %q", output)
	}
}

func TestBuildReportFutureModuleRedactsSecrets(t *testing.T) {
	output, err := BuildReport(sampleReport(), model.ExportRequest{ReportType: model.ReportSafety, Format: model.ExportJSON})
	if err != nil {
		t.Fatalf("BuildReport safety json: %v", err)
	}

	assertContains(t, output, `"report_type": "safety"`)
	assertContains(t, output, `"secret_values": "redacted"`)
	if strings.Contains(output, "postgres://") {
		t.Fatalf("safety export leaked secret-like value: %s", output)
	}
}

func sampleReport() model.WorkspaceReport {
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
				RemoteURL:       "git@example.com:logan/api.git",
				LastCommitHash:  "abc1234",
				DaysSinceCommit: 3,
			},
			Readiness: model.ReadinessAudit{
				Readme:    true,
				Tests:     true,
				Container: true,
				CI:        true,
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
			Readiness:     model.ReadinessAudit{Readme: true, Score: 20},
		},
	}
	languages, totalFiles := model.SummarizeProjects(projects)
	return model.WorkspaceReport{
		RootPath:        "/tmp/code",
		Projects:        projects,
		LanguageSummary: languages,
		TotalFiles:      totalFiles,
		LabelCounts:     model.LabelCounts(projects),
		GeneratedAt:     time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC),
	}
}

func assertContains(t *testing.T, text string, want string) {
	t.Helper()
	if !strings.Contains(text, want) {
		t.Fatalf("expected output to contain %q\n\n%s", want, text)
	}
}

func assertNotContains(t *testing.T, text string, unwanted string) {
	t.Helper()
	if strings.Contains(text, unwanted) {
		t.Fatalf("expected output not to contain %q\n\n%s", unwanted, text)
	}
}
