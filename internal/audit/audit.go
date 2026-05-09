package audit

import (
	"fmt"
	"strings"

	"projectscan/internal/model"
)

type Result struct {
	ReportType      string          `json:"report_type"`
	Title           string          `json:"title"`
	RootPath        string          `json:"root_path"`
	ProjectCount    int             `json:"project_count"`
	TotalFiles      int             `json:"total_files"`
	LanguageSummary map[string]int  `json:"language_summary,omitempty"`
	LabelCounts     map[string]int  `json:"label_counts,omitempty"`
	Projects        []model.Project `json:"projects,omitempty"`
	Modules         []ModuleResult  `json:"modules,omitempty"`
	Message         string          `json:"message,omitempty"`
	SecretValues    string          `json:"secret_values,omitempty"`
}

type ModuleResult struct {
	Name         string              `json:"name"`
	Title        string              `json:"title"`
	Columns      []string            `json:"columns,omitempty"`
	Rows         []map[string]string `json:"rows,omitempty"`
	Counts       map[string]int      `json:"counts,omitempty"`
	Message      string              `json:"message,omitempty"`
	SecretValues string              `json:"secret_values,omitempty"`
}

func Build(report model.WorkspaceReport, reportType string) (Result, error) {
	if reportType == "" {
		reportType = model.ReportSummary
	}
	if err := model.ValidateReportType(reportType); err != nil {
		return Result{}, err
	}

	result := Result{
		ReportType:      reportType,
		Title:           Title(reportType),
		RootPath:        report.RootPath,
		ProjectCount:    len(report.Projects),
		TotalFiles:      report.TotalFiles,
		LanguageSummary: report.LanguageSummary,
		LabelCounts:     report.LabelCounts,
		Projects:        report.Projects,
	}

	if IsFutureReport(reportType) {
		result.Projects = nil
		result.LanguageSummary = nil
		result.LabelCounts = nil
		result.Message = fmt.Sprintf("%s is a planned roadmap audit module. Export wiring is ready; scanner implementation will arrive in a later phase.", Title(reportType))
		result.SecretValues = "redacted"
		result.Modules = []ModuleResult{{
			Name:         reportType,
			Title:        Title(reportType),
			Message:      result.Message,
			SecretValues: result.SecretValues,
		}}
		return result, nil
	}

	switch reportType {
	case model.ReportSummary:
		result.Modules = []ModuleResult{projectModule(report.Projects)}
	case model.ReportAudit:
		result.Modules = []ModuleResult{
			portfolioSummaryModule(report.LabelCounts),
			gitModule(report.Projects),
			readinessModule(report.Projects),
			projectModuleWithTitle(report.Projects, "Project Portfolio"),
		}
	case model.ReportDetails:
		result.Modules = []ModuleResult{projectModule(report.Projects), languageDetailsModule(report.Projects)}
	case model.ReportPortfolio:
		result.Modules = []ModuleResult{portfolioSummaryModule(report.LabelCounts), projectModuleWithTitle(report.Projects, "Project Portfolio")}
	case model.ReportGit:
		result.Modules = []ModuleResult{gitModule(report.Projects)}
	case model.ReportReadiness:
		result.Modules = []ModuleResult{readinessModule(report.Projects)}
	case model.ReportSafety:
		result.SecretValues = "redacted"
		result.Modules = []ModuleResult{safetyModule(report.Projects)}
	case model.ReportReadme:
		result.Modules = []ModuleResult{readmeQualityModule(report.Projects)}
	case model.ReportLOC:
		result.Modules = []ModuleResult{locModule(report.Projects)}
	case model.ReportGitHygiene:
		result.Modules = []ModuleResult{gitHygieneModule(report.Projects)}
	case model.ReportDeps:
		result.Modules = []ModuleResult{dependencyModule(report.Projects)}
	case model.ReportOpenSourceReadiness:
		result.SecretValues = "redacted"
		result.Modules = []ModuleResult{
			readinessModule(report.Projects),
			readmeQualityModule(report.Projects),
			safetyModule(report.Projects),
			gitHygieneModule(report.Projects),
			dependencyModule(report.Projects),
		}
	default:
		return Result{}, fmt.Errorf("unknown report type %q", reportType)
	}
	return result, nil
}

func IsFutureReport(reportType string) bool {
	switch reportType {
	case model.ReportExternalTools:
		return true
	default:
		return false
	}
}

func Title(reportType string) string {
	switch reportType {
	case model.ReportLOC:
		return "Lines of Code"
	case model.ReportDeps:
		return "Dependency Inventory"
	case model.ReportSafety:
		return "Open-source Safety"
	case model.ReportReadme:
		return "README Quality"
	case model.ReportGitHygiene:
		return "Git Hygiene"
	case model.ReportOpenSourceReadiness:
		return "Open-source Readiness"
	case model.ReportExternalTools:
		return "External Tool Integration"
	default:
		return strings.Title(strings.ReplaceAll(reportType, "-", " "))
	}
}

func portfolioSummaryModule(labelCounts map[string]int) ModuleResult {
	return ModuleResult{
		Name:    "portfolio_summary",
		Title:   "Portfolio Summary",
		Columns: []string{"Label", "Projects", "Meaning"},
		Rows: []map[string]string{
			{"Label": model.LabelProductionReady, "Projects": fmt.Sprintf("%d", labelCounts[model.LabelProductionReady]), "Meaning": "ready for portfolio/client"},
			{"Label": model.LabelExperiment, "Projects": fmt.Sprintf("%d", labelCounts[model.LabelExperiment]), "Meaning": "learning or incomplete"},
			{"Label": model.LabelArchived, "Projects": fmt.Sprintf("%d", labelCounts[model.LabelArchived]), "Meaning": "retired or superseded"},
		},
		Counts: labelCounts,
	}
}

func projectModule(projects []model.Project) ModuleResult {
	return projectModuleWithTitle(projects, "Projects")
}

func projectModuleWithTitle(projects []model.Project, title string) ModuleResult {
	rows := []map[string]string{}
	for _, project := range model.SortedProjectsByName(projects) {
		rows = append(rows, map[string]string{
			"Project":        project.Name,
			"Label":          project.Label,
			"Files":          fmt.Sprintf("%d", project.TotalFiles),
			"Main Languages": strings.Join(project.MainLanguages, ", "),
			"Branch":         project.Git.Branch,
		})
	}
	return ModuleResult{
		Name:    "projects",
		Title:   title,
		Columns: []string{"Project", "Label", "Files", "Main Languages", "Branch"},
		Rows:    rows,
	}
}

func gitModule(projects []model.Project) ModuleResult {
	rows := []map[string]string{}
	for _, project := range model.SortedProjectsByName(projects) {
		branch := "-"
		dirty := "n/a"
		remote := "-"
		lastCommit := "-"
		days := "0"
		if project.Git.IsRepo {
			branch = project.Git.Branch
			dirty = dirtyLabel(project.Git)
			remote = project.Git.RemoteURL
			lastCommit = project.Git.LastCommitHash
			days = fmt.Sprintf("%d", project.Git.DaysSinceCommit)
		}
		rows = append(rows, map[string]string{
			"Project":           project.Name,
			"Branch":            branch,
			"Dirty":             dirty,
			"Remote":            remote,
			"Last Commit":       lastCommit,
			"Days Since Commit": days,
		})
	}
	return ModuleResult{
		Name:    "git",
		Title:   "Git Audit",
		Columns: []string{"Project", "Branch", "Dirty", "Remote", "Last Commit", "Days Since Commit"},
		Rows:    rows,
	}
}

func readinessModule(projects []model.Project) ModuleResult {
	rows := []map[string]string{}
	for _, project := range model.SortedProjectsByName(projects) {
		rows = append(rows, map[string]string{
			"Project":   project.Name,
			"README":    yesNo(project.Readiness.Readme),
			"Tests":     yesNo(project.Readiness.Tests),
			"License":   yesNo(project.Readiness.License),
			"Container": yesNo(project.Readiness.Container),
			"CI":        yesNo(project.Readiness.CI),
			"Deploy":    yesNo(project.Readiness.Deploy),
			"Remote":    yesNo(project.Readiness.Remote),
			"Score":     fmt.Sprintf("%d", project.Readiness.Score),
		})
	}
	return ModuleResult{
		Name:    "readiness",
		Title:   "Readiness Audit",
		Columns: []string{"Project", "README", "Tests", "License", "Container", "CI", "Deploy", "Remote", "Score"},
		Rows:    rows,
	}
}

func languageDetailsModule(projects []model.Project) ModuleResult {
	rows := []map[string]string{}
	for _, project := range model.SortedProjectsByName(projects) {
		for _, item := range model.SortLanguageCounts(project.Languages) {
			rows = append(rows, map[string]string{
				"Project":  project.Name,
				"Language": item.Lang,
				"Files":    fmt.Sprintf("%d", item.Count),
			})
		}
	}
	return ModuleResult{
		Name:    "language_details",
		Title:   "Language Details",
		Columns: []string{"Project", "Language", "Files"},
		Rows:    rows,
	}
}

func dirtyLabel(git model.GitMetadata) string {
	if !git.IsRepo {
		return "n/a"
	}
	if git.Dirty {
		return "dirty"
	}
	return "clean"
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}
