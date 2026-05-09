package export

import (
	"fmt"
	"strings"

	"projectscan/internal/audit"
	"projectscan/internal/model"
	"projectscan/internal/render"
)

func BuildReport(report model.WorkspaceReport, req model.ExportRequest) (string, error) {
	if req.ReportType == "" {
		req.ReportType = model.ReportSummary
	}
	if req.Format == "" {
		req.Format = model.ExportTerminal
	}
	if err := model.ValidateReportType(req.ReportType); err != nil {
		return "", err
	}
	if err := model.ValidateExportFormat(req.Format); err != nil {
		return "", err
	}

	filtered, err := filterReport(report, req.ProjectFilter)
	if err != nil {
		return "", err
	}

	auditResult, err := audit.Build(filtered, req.ReportType)
	if err != nil {
		return "", err
	}
	auditResult = displayAuditResult(auditResult)

	switch req.Format {
	case model.ExportTerminal:
		if isStructuredOnlyReport(req.ReportType) {
			return buildMarkdown(auditResult), nil
		}
		return buildTerminal(filtered, req.ReportType), nil
	case model.ExportMarkdown:
		return buildMarkdown(auditResult), nil
	case model.ExportJSON:
		return buildJSONPayload(auditResult)
	case model.ExportCSV:
		return buildCSV(auditResult)
	case model.ExportHTML:
		return buildHTML(auditResult), nil
	default:
		return "", fmt.Errorf("unknown export format %q", req.Format)
	}
}

func BuildJSONReport(report model.WorkspaceReport) (string, error) {
	return buildJSONPayload(displayWorkspaceReport(report))
}

func BuildMarkdownReport(report model.WorkspaceReport) string {
	report = displayWorkspaceReport(report)
	var b strings.Builder
	fmt.Fprintln(&b, "# Projectscan Report")
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "- Root Path: `%s`\n", report.RootPath)
	fmt.Fprintf(&b, "- Projects: %d\n", len(report.Projects))
	fmt.Fprintf(&b, "- Files: %d\n", report.TotalFiles)
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "## Projects")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "| Project | Label | Files | Main Languages | Branch |")
	fmt.Fprintln(&b, "|---|---|---:|---|---|")
	for _, project := range model.SortedProjectsByName(report.Projects) {
		fmt.Fprintf(&b, "| %s | %s | %d | %s | %s |\n",
			project.Name,
			project.Label,
			project.TotalFiles,
			strings.Join(project.MainLanguages, ", "),
			project.Git.Branch,
		)
	}
	fmt.Fprintln(&b)
	return b.String()
}

func displayAuditResult(result audit.Result) audit.Result {
	result.RootPath = model.DisplayRootPath(result.RootPath)
	return result
}

func displayWorkspaceReport(report model.WorkspaceReport) model.WorkspaceReport {
	report.RootPath = model.DisplayRootPath(report.RootPath)
	return report
}

func filterReport(report model.WorkspaceReport, projectFilter string) (model.WorkspaceReport, error) {
	if projectFilter == "" {
		return report, nil
	}

	filter := strings.ToLower(projectFilter)
	for _, project := range report.Projects {
		if strings.ToLower(project.Name) == filter || strings.ToLower(project.RawName) == filter {
			report.Projects = []model.Project{project}
			return model.WithComputedTotals(report), nil
		}
	}
	return report, fmt.Errorf("project %q not found", projectFilter)
}

func buildTerminal(report model.WorkspaceReport, reportType string) string {
	switch reportType {
	case model.ReportSummary:
		return render.BuildSummary(report.RootPath, report.Projects)
	case model.ReportAudit:
		return render.BuildAuditReport(report, render.Style{})
	case model.ReportDetails:
		return render.BuildDetails(report, render.Style{})
	case model.ReportPortfolio:
		var b strings.Builder
		render.PrintPortfolioSummary(&b, report, render.Style{})
		fmt.Fprintln(&b)
		render.PrintPortfolioTable(&b, report.Projects, render.Style{})
		return b.String()
	case model.ReportGit:
		var b strings.Builder
		render.PrintGitAudit(&b, report.Projects, render.Style{})
		return b.String()
	case model.ReportReadiness:
		var b strings.Builder
		render.PrintReadinessAudit(&b, report.Projects, render.Style{})
		return b.String()
	default:
		return ""
	}
}

func isStructuredOnlyReport(reportType string) bool {
	switch reportType {
	case model.ReportSafety, model.ReportReadme, model.ReportLOC, model.ReportGitHygiene, model.ReportDeps, model.ReportOpenSourceReadiness, model.ReportExternalTools:
		return true
	default:
		return false
	}
}
