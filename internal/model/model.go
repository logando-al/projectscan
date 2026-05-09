package model

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Project is one scanned local project and its recognized source-language counts.
type Project struct {
	Name           string            `json:"name"`
	RawName        string            `json:"raw_name,omitempty"`
	Path           string            `json:"path"`
	Languages      map[string]int    `json:"languages"`
	TotalFiles     int               `json:"total_files"`
	Git            GitMetadata       `json:"git"`
	Readiness      ReadinessAudit    `json:"readiness"`
	Label          string            `json:"label"`
	ArchivedReason string            `json:"archived_reason,omitempty"`
	Pinned         bool              `json:"pinned,omitempty"`
	ScoreSignals   []string          `json:"score_signals,omitempty"`
	MainLanguages  []string          `json:"main_languages,omitempty"`
	Extras         map[string]string `json:"extras,omitempty"`
}

type WorkspaceReport struct {
	RootPath        string         `json:"root_path"`
	Projects        []Project      `json:"projects"`
	LanguageSummary map[string]int `json:"language_summary"`
	TotalFiles      int            `json:"total_files"`
	LabelCounts     map[string]int `json:"label_counts"`
	GeneratedAt     time.Time      `json:"generated_at"`
}

type GitMetadata struct {
	IsRepo            bool   `json:"is_repo"`
	Branch            string `json:"branch,omitempty"`
	Dirty             bool   `json:"dirty"`
	RemoteURL         string `json:"remote_url,omitempty"`
	LastCommitHash    string `json:"last_commit_hash,omitempty"`
	LastCommitMessage string `json:"last_commit_message,omitempty"`
	LastCommitDate    string `json:"last_commit_date,omitempty"`
	DaysSinceCommit   int    `json:"days_since_commit,omitempty"`
}

type ReadinessAudit struct {
	Readme    bool     `json:"readme"`
	License   bool     `json:"license"`
	Tests     bool     `json:"tests"`
	Container bool     `json:"container"`
	CI        bool     `json:"ci"`
	Deploy    bool     `json:"deploy"`
	Remote    bool     `json:"remote"`
	Score     int      `json:"score"`
	Signals   []string `json:"signals,omitempty"`
}

type LangCount struct {
	Lang  string
	Count int
}

const (
	ModeSummary  = "summary"
	ModeAudit    = "audit"
	ModeDetails  = "details"
	ModeJSON     = "json"
	ModeMarkdown = "markdown"
	ModeExport   = "export"

	LabelProductionReady = "production-ready"
	LabelExperiment      = "experiment"
	LabelArchived        = "archived"
)

const (
	ReportSummary             = "summary"
	ReportAudit               = "audit"
	ReportDetails             = "details"
	ReportPortfolio           = "portfolio"
	ReportGit                 = "git"
	ReportReadiness           = "readiness"
	ReportSafety              = "safety"
	ReportReadme              = "readme"
	ReportLOC                 = "loc"
	ReportGitHygiene          = "git-hygiene"
	ReportDeps                = "deps"
	ReportOpenSourceReadiness = "open-source-readiness"
	ReportExternalTools       = "external-tools"

	ExportTerminal = "terminal"
	ExportMarkdown = "markdown"
	ExportJSON     = "json"
	ExportCSV      = "csv"
	ExportHTML     = "html"
)

type ExportRequest struct {
	ReportType    string `json:"report_type"`
	Format        string `json:"format"`
	ProjectFilter string `json:"project,omitempty"`
}

func ValidateReportType(reportType string) error {
	switch reportType {
	case ReportSummary, ReportAudit, ReportDetails, ReportPortfolio, ReportGit, ReportReadiness,
		ReportSafety, ReportReadme, ReportLOC, ReportGitHygiene, ReportDeps, ReportOpenSourceReadiness, ReportExternalTools:
		return nil
	default:
		return fmt.Errorf("unknown report type %q", reportType)
	}
}

func ValidateExportFormat(format string) error {
	switch format {
	case ExportTerminal, ExportMarkdown, ExportJSON, ExportCSV, ExportHTML:
		return nil
	default:
		return fmt.Errorf("unknown export format %q", format)
	}
}

func SumCounts(m map[string]int) int {
	total := 0
	for _, count := range m {
		total += count
	}
	return total
}

func SummarizeProjects(projects []Project) (map[string]int, int) {
	totalLanguages := map[string]int{}
	totalFiles := 0
	for _, project := range projects {
		totalFiles += project.TotalFiles
		for lang, count := range project.Languages {
			totalLanguages[lang] += count
		}
	}
	return totalLanguages, totalFiles
}

func LabelCounts(projects []Project) map[string]int {
	counts := map[string]int{
		LabelProductionReady: 0,
		LabelExperiment:      0,
		LabelArchived:        0,
	}
	for _, project := range projects {
		counts[project.Label]++
	}
	return counts
}

func WithComputedTotals(report WorkspaceReport) WorkspaceReport {
	report.LanguageSummary, report.TotalFiles = SummarizeProjects(report.Projects)
	report.LabelCounts = LabelCounts(report.Projects)
	return report
}

// DisplayRootPath keeps user-facing reports compact while scan data keeps full paths.
func DisplayRootPath(path string) string {
	if path == "" {
		return "..."
	}
	base := filepath.Base(filepath.Clean(path))
	if base == "." {
		return "..."
	}
	return ".../" + base
}

func SortedProjectsByName(projects []Project) []Project {
	sortedProjects := append([]Project(nil), projects...)
	sort.Slice(sortedProjects, func(i, j int) bool {
		return sortedProjects[i].Name < sortedProjects[j].Name
	})
	return sortedProjects
}

func GetTopLanguages(languages map[string]int) []string {
	items := SortLanguageCounts(languages)
	if len(items) > 3 {
		items = items[:3]
	}

	top := make([]string, 0, len(items))
	for _, item := range items {
		top = append(top, item.Lang)
	}
	return top
}

func SortLanguageCounts(languages map[string]int) []LangCount {
	var items []LangCount
	for lang, count := range languages {
		items = append(items, LangCount{
			Lang:  lang,
			Count: count,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Lang < items[j].Lang
		}
		return items[i].Count > items[j].Count
	})
	return items
}

func MakeBar(value int, maxValue int, width int) string {
	if maxValue <= 0 {
		maxValue = 1
	}

	filled := value * width / maxValue
	if value > 0 && filled == 0 {
		filled = 1
	}
	if filled > width {
		filled = width
	}
	return Repeat("█", filled) + Repeat("░", width-filled)
}

func PercentOfTotal(value int, total int) int {
	if total <= 0 || value <= 0 {
		return 0
	}
	return (value*100 + total/2) / total
}

func PercentLabel(value int, total int) string {
	percent := PercentOfTotal(value, total)
	if value > 0 && percent == 0 {
		return " <1 %"
	}
	return fmt.Sprintf("%3d %%", percent)
}

func Repeat(char string, count int) string {
	if count <= 0 {
		return ""
	}
	return strings.Repeat(char, count)
}

func Center(text string, width int) string {
	if RuneLen(text) >= width {
		return FitText(text, width)
	}
	left := (width - RuneLen(text)) / 2
	right := width - RuneLen(text) - left
	return Repeat(" ", left) + text + Repeat(" ", right)
}

func FitText(text string, max int) string {
	if RuneLen(text) <= max {
		return text
	}
	if max <= 3 {
		return string([]rune(text)[:max])
	}
	return string([]rune(text)[:max-3]) + "..."
}

func PadRight(text string, width int) string {
	return text + Repeat(" ", width-RuneLen(text))
}

func RuneLen(text string) int {
	return len([]rune(text))
}

func ShouldSkipDir(name string) bool {
	ignored := map[string]bool{
		".git":         true,
		".idea":        true,
		".next":        true,
		".nuxt":        true,
		".svelte-kit":  true,
		".turbo":       true,
		".venv":        true,
		".vscode":      true,
		"__pycache__":  true,
		"build":        true,
		"coverage":     true,
		"dist":         true,
		"node_modules": true,
		"target":       true,
		"vendor":       true,
		"venv":         true,
	}
	return ignored[name]
}
