package audit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"projectscan/internal/model"
)

func TestBuildAuditProducesReusableModuleResults(t *testing.T) {
	report := sampleReport()

	result, err := Build(report, model.ReportAudit)
	if err != nil {
		t.Fatalf("Build audit: %v", err)
	}

	if result.ReportType != model.ReportAudit || result.Title != "Audit" {
		t.Fatalf("unexpected audit result header: %#v", result)
	}
	if len(result.Modules) != 4 {
		t.Fatalf("expected audit to compose four modules, got %#v", result.Modules)
	}
	if result.Modules[1].Name != "git" || result.Modules[1].Rows[0]["Dirty"] != "clean" {
		t.Fatalf("expected structured git module row, got %#v", result.Modules[1])
	}
	if result.Modules[2].Name != "readiness" || result.Modules[2].Rows[0]["Score"] != "75" {
		t.Fatalf("expected structured readiness module row, got %#v", result.Modules[2])
	}
}

func TestBuildSafetyAuditReportsRiskWithoutSecretValues(t *testing.T) {
	root := t.TempDir()
	projectPath := filepath.Join(root, "api")
	mustMkdir(t, projectPath)
	mustWrite(t, filepath.Join(projectPath, "main.go"), "package main\n")
	mustWrite(t, filepath.Join(projectPath, "types.go"), "package main\n\ntype Result struct { SecretValues string }\n")
	mustWrite(t, filepath.Join(projectPath, ".env"), "DATABASE_URL=postgres://logan:secret@localhost/app\n")
	mustWrite(t, filepath.Join(projectPath, "config.toml"), "api_key = \"sk-test-secret\"\n")
	report := sampleReportWithProjectPath(root, projectPath)

	result, err := Build(report, model.ReportSafety)
	if err != nil {
		t.Fatalf("Build safety: %v", err)
	}

	if result.Title != "Open-source Safety" {
		t.Fatalf("expected friendly safety title, got %q", result.Title)
	}
	if result.SecretValues != "redacted" || result.Modules[0].SecretValues != "redacted" {
		t.Fatalf("expected safety audit to redact secret values, got %#v", result)
	}
	if len(result.Modules[0].Rows) < 2 {
		t.Fatalf("expected safety findings, got %#v", result.Modules[0])
	}
	rendered := stringifyRows(result.Modules[0].Rows)
	if strings.Contains(rendered, "postgres://logan:secret") || strings.Contains(rendered, "sk-test-secret") {
		t.Fatalf("safety audit leaked secret values:\n%s", rendered)
	}
	if !strings.Contains(rendered, ".env") || !strings.Contains(rendered, "secret-like token") {
		t.Fatalf("expected redacted safety risks, got:\n%s", rendered)
	}
	if strings.Contains(rendered, "types.go") {
		t.Fatalf("safety audit should not flag source field names as secrets:\n%s", rendered)
	}
}

func TestBuildReadmeQualityAuditScoresSections(t *testing.T) {
	root := t.TempDir()
	projectPath := filepath.Join(root, "api")
	mustMkdir(t, projectPath)
	mustWrite(t, filepath.Join(projectPath, "README.md"), "# API\n\nA useful API.\n\n## Installation\n\n## Usage\n")
	report := sampleReportWithProjectPath(root, projectPath)

	result, err := Build(report, model.ReportReadme)
	if err != nil {
		t.Fatalf("Build readme: %v", err)
	}

	row := result.Modules[0].Rows[0]
	if row["Project"] != "api" || row["Score"] == "0" {
		t.Fatalf("expected README quality row with score, got %#v", row)
	}
	if !strings.Contains(row["Found"], "title") || !strings.Contains(row["Missing"], "license") {
		t.Fatalf("expected found and missing README sections, got %#v", row)
	}
}

func TestBuildLOCAuditCountsCodeBlankAndComments(t *testing.T) {
	root := t.TempDir()
	projectPath := filepath.Join(root, "api")
	mustMkdir(t, projectPath)
	mustWrite(t, filepath.Join(projectPath, "main.go"), "package main\n\n// comment\nfunc main() {}\n")
	report := sampleReportWithProjectPath(root, projectPath)

	result, err := Build(report, model.ReportLOC)
	if err != nil {
		t.Fatalf("Build loc: %v", err)
	}

	row := result.Modules[0].Rows[0]
	if row["Project"] != "api" || row["Files"] != "1" || row["Total LOC"] != "4" || row["Blank LOC"] != "1" || row["Comment LOC"] != "1" {
		t.Fatalf("expected LOC counts, got %#v", row)
	}
}

func sampleReport() model.WorkspaceReport {
	projects := []model.Project{
		{
			Name:          "api",
			RawName:       "api",
			Path:          "/tmp/code/api",
			Languages:     map[string]int{"Go": 10},
			TotalFiles:    10,
			Label:         model.LabelProductionReady,
			MainLanguages: []string{"Go"},
			Git: model.GitMetadata{
				IsRepo:          true,
				Branch:          "main",
				RemoteURL:       "git@example.com:logan/api.git",
				LastCommitHash:  "abc1234",
				DaysSinceCommit: 2,
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

func sampleReportWithProjectPath(root string, projectPath string) model.WorkspaceReport {
	projects := []model.Project{{
		Name:          filepath.Base(projectPath),
		RawName:       filepath.Base(projectPath),
		Path:          projectPath,
		Languages:     map[string]int{"Go": 1},
		TotalFiles:    1,
		Label:         model.LabelExperiment,
		MainLanguages: []string{"Go"},
	}}
	languages, totalFiles := model.SummarizeProjects(projects)
	return model.WorkspaceReport{
		RootPath:        root,
		Projects:        projects,
		LanguageSummary: languages,
		TotalFiles:      totalFiles,
		LabelCounts:     model.LabelCounts(projects),
		GeneratedAt:     time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC),
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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir parent %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func stringifyRows(rows []map[string]string) string {
	var b strings.Builder
	for _, row := range rows {
		for key, value := range row {
			b.WriteString(key)
			b.WriteString("=")
			b.WriteString(value)
			b.WriteString("\n")
		}
	}
	return b.String()
}
