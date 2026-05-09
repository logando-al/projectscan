package render

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"projectscan/internal/model"
)

const (
	reportWidth = 62
)

var (
	lipGoBlue = lipgloss.Color("#00ADD8")
	lipGreen  = lipgloss.Color("2") // Standard ANSI green
	lipWhite  = lipgloss.Color("#FFFFFF")
	lipYellow = lipgloss.Color("#FFFF00")
	lipRed    = lipgloss.Color("#FF0000")
	lipCyan   = lipgloss.Color("#00FFFF")
	lipDim    = lipgloss.Color("#666666")
)

type Style struct {
	Color      bool
	IsSnapshot bool
}

func BuildSummary(rootPath string, projects []model.Project) string {
	return BuildSummaryWithStyle(rootPath, projects, Style{})
}

func BuildSummaryWithStyle(rootPath string, projects []model.Project, style Style) string {
	var b strings.Builder
	WriteSummary(&b, rootPath, projects, style)
	return b.String()
}

func WriteSummary(w io.Writer, rootPath string, projects []model.Project, style Style) {
	totalLanguages, totalFiles := model.SummarizeProjects(projects)

	snapshotStyle := style
	snapshotStyle.IsSnapshot = true
	printSnapshot(w, rootPath, projects, totalLanguages, totalFiles, snapshotStyle)
	fmt.Fprintln(w)
	printLanguageTable(w, totalLanguages, totalFiles, style)
	fmt.Fprintln(w)
	printProjectTable(w, projects, style)
}

func BuildAuditReport(report model.WorkspaceReport, style Style) string {
	var b strings.Builder
	printSnapshot(&b, report.RootPath, report.Projects, report.LanguageSummary, report.TotalFiles, style)
	fmt.Fprintln(&b)
	PrintPortfolioSummary(&b, report, style)
	fmt.Fprintln(&b)
	PrintGitAudit(&b, report.Projects, style)
	fmt.Fprintln(&b)
	PrintReadinessAudit(&b, report.Projects, style)
	fmt.Fprintln(&b)
	PrintPortfolioTable(&b, report.Projects, style)
	return b.String()
}

func BuildDetails(report model.WorkspaceReport, style Style) string {
	var b strings.Builder
	fmt.Fprint(&b, BuildSummaryWithStyle(report.RootPath, report.Projects, style))
	fmt.Fprintln(&b)
	PrintProjectDetails(&b, report.Projects, style)
	return b.String()
}

func PrintPortfolioSummary(w io.Writer, report model.WorkspaceReport, style Style) {
	widths := []int{18, 8, 28}
	WriteTableTop(w, "PORTFOLIO SUMMARY", widths, style)
	WriteTableRow(w, []string{"Label", "Projects", "Meaning"}, widths, style)
	WriteTableSep(w, widths, style)
	WriteTableRow(w, []string{model.LabelProductionReady, fmt.Sprintf("%8d", report.LabelCounts[model.LabelProductionReady]), "ready for portfolio/client"}, widths, style)
	WriteTableRow(w, []string{model.LabelExperiment, fmt.Sprintf("%8d", report.LabelCounts[model.LabelExperiment]), "learning or incomplete"}, widths, style)
	WriteTableRow(w, []string{model.LabelArchived, fmt.Sprintf("%8d", report.LabelCounts[model.LabelArchived]), "retired or superseded"}, widths, style)
	WriteTableBottom(w, widths, style)
}

func PrintGitAudit(w io.Writer, projects []model.Project, style Style) {
	widths := []int{20, 10, 7, 16}
	WriteTableTop(w, "GIT AUDIT", widths, style)
	WriteTableRow(w, []string{"Project", "Branch", "Dirty", "Last Commit"}, widths, style)
	WriteTableSep(w, widths, style)
	for _, project := range model.SortedProjectsByName(projects) {
		dirty := "n/a"
		branch := "-"
		age := "-"
		if project.Git.IsRepo {
			dirty = boolLabel(project.Git.Dirty, "dirty", "clean")
			branch = project.Git.Branch
			if project.Git.LastCommitHash != "" {
				age = fmt.Sprintf("%dd %s", project.Git.DaysSinceCommit, project.Git.LastCommitHash)
			}
		}
		WriteTableRow(w, []string{truncate(project.Name, 20), truncate(branch, 10), dirty, truncate(age, 16)}, widths, style)
	}
	WriteTableBottom(w, widths, style)
}

func PrintReadinessAudit(w io.Writer, projects []model.Project, style Style) {
	widths := []int{20, 5, 5, 5, 5, 5, 5}
	WriteTableTop(w, "READINESS AUDIT", widths, style)
	WriteTableRow(w, []string{"Project", "Read", "Test", "Lic", "Cont", "CI", "Score"}, widths, style)
	WriteTableSep(w, widths, style)
	for _, project := range model.SortedProjectsByName(projects) {
		WriteTableRow(w, []string{
			truncate(project.Name, 20),
			checkMark(project.Readiness.Readme),
			checkMark(project.Readiness.Tests),
			checkMark(project.Readiness.License),
			checkMark(project.Readiness.Container),
			checkMark(project.Readiness.CI || project.Readiness.Deploy),
			fmt.Sprintf("%5d", project.Readiness.Score),
		}, widths, style)
	}
	WriteTableBottom(w, widths, style)
}

func PrintPortfolioTable(w io.Writer, projects []model.Project, style Style) {
	widths := []int{20, 18, 23}
	WriteTableTop(w, "PROJECT PORTFOLIO", widths, style)
	WriteTableRow(w, []string{"Project", "Label", "Main Languages"}, widths, style)
	WriteTableSep(w, widths, style)
	for _, project := range model.SortedProjectsByName(projects) {
		WriteTableRow(w, []string{
			truncate(project.Name, 20),
			truncate(project.Label, 18),
			truncate(strings.Join(project.MainLanguages, ", "), 23),
		}, widths, style)
	}
	WriteTableBottom(w, widths, style)
}

func PrintProjectDetails(w io.Writer, projects []model.Project, style Style) {
	for _, project := range model.SortedProjectsByName(projects) {
		fmt.Fprintf(w, "[%s]\n", project.Name)
		fmt.Fprintf(w, "Path   : %s\n", project.Path)
		fmt.Fprintf(w, "Label  : %s\n", project.Label)
		fmt.Fprintf(w, "Git    : %s\n\n", gitSummary(project.Git))
		printLanguageTable(w, project.Languages, project.TotalFiles, style)
		fmt.Fprintln(w)
	}
}

func ShouldUseColor(file *os.File) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	stat, err := file.Stat()
	if err != nil {
		return false
	}
	return stat.Mode()&os.ModeCharDevice != 0
}

func Colorize(style Style, color lipgloss.TerminalColor, text string) string {
	if !style.Color || text == "" {
		return text
	}
	return lipgloss.NewStyle().Foreground(color).Render(text)
}

func WriteTableTopForTest(w io.Writer, title string, widths []int, style Style) {
	WriteTableTop(w, title, widths, style)
}

func WriteTableRowForTest(w io.Writer, values []string, widths []int, style Style) {
	WriteTableRow(w, values, widths, style)
}

func WriteTableBottomForTest(w io.Writer, widths []int, style Style) {
	WriteTableBottom(w, widths, style)
}

func RenderBarForTest(value int, maxValue int, width int, style Style) string {
	return renderBar(value, maxValue, width, style)
}

func percentageColor(percent int) lipgloss.TerminalColor {
	switch {
	case percent >= 30:
		return lipGreen
	case percent >= 10:
		return lipYellow
	default:
		return lipRed
	}
}

func printSnapshot(w io.Writer, rootPath string, projects []model.Project, languages map[string]int, totalFiles int, style Style) {
	writeBoxTop(w, style)
	title := Colorize(style, lipGoBlue, "PROJECTSCAN SNAPSHOT")
	writeBoxLine(w, model.Center(title, reportWidth), style)
	writeBoxMiddle(w, style)
	writeMetricLine(w, "Root Path", model.DisplayRootPath(rootPath), style)
	writeMetricLine(w, "Projects", fmt.Sprintf("%d", len(projects)), style)
	writeMetricLine(w, "Recognized Files", fmt.Sprintf("%d", totalFiles), style)
	writeMetricLine(w, "Languages", fmt.Sprintf("%d", len(languages)), style)
	writeBoxLine(w, "", style)

	items := model.SortLanguageCounts(languages)
	if len(items) > 5 {
		items = items[:5]
	}
	for _, item := range items {
		writeSnapshotLanguageLine(w, item, totalFiles, style)
	}
	writeBoxBottom(w, style)
}

func writeMetricLine(w io.Writer, label string, value string, style Style) {
	label = model.PadRight(model.FitText(label, 18), 18)
	value = Colorize(style, lipYellow, model.FitText(value, 41))
	label = Colorize(style, lipGoBlue, label)
	writeBoxLine(w, fmt.Sprintf("  %s%s", label, value), style)
}

func writeSnapshotLanguageLine(w io.Writer, item model.LangCount, totalFiles int, style Style) {
	bar := renderBar(item.Count, totalFiles, 20, style)
	label := model.PadRight(model.FitText(item.Lang, 13), 13)
	label = Colorize(style, lipGoBlue, label)
	value := Colorize(style, percentageColor(model.PercentOfTotal(item.Count, totalFiles)), model.PercentLabel(item.Count, totalFiles))
	writeBoxLine(w, fmt.Sprintf("  %s %s  %s", label, bar, value), style)
}

func printLanguageTable(w io.Writer, languages map[string]int, totalFiles int, style Style) {
	items := model.SortLanguageCounts(languages)
	widths := []int{13, 6, 22, 10}
	WriteTableTop(w, "LANGUAGE SUMMARY", widths, style)
	WriteTableRow(w, []string{
		model.FitText("Language", 12),
		model.FitText("Files", 5),
		model.FitText("Usage", 20),
		model.FitText("Share", 5),
	}, widths, style)
	WriteTableSep(w, widths, style)

	for _, item := range items {
		percent := model.PercentOfTotal(item.Count, totalFiles)
		WriteTableRow(w, []string{
			model.FitText(item.Lang, 13),
			Colorize(style, lipYellow, fmt.Sprintf("%6d", item.Count)),
			renderBar(item.Count, totalFiles, 22, style),
			Colorize(style, percentageColor(percent), model.PercentLabel(item.Count, totalFiles)),
		}, widths, style)
	}
	WriteTableBottom(w, widths, style)
}

func printProjectTable(w io.Writer, projects []model.Project, style Style) {
	sortedProjects := model.SortedProjectsByName(projects)
	widths := []int{2, 18, 5, 20, 10}
	WriteTableTop(w, "PROJECTS", widths, style)
	WriteTableRow(w, []string{
		model.FitText("#", 2),
		model.FitText("Project", 18),
		model.FitText("Files", 5),
		model.FitText("Main Languages", 20),
		model.FitText("Health", 10),
	}, widths, style)
	WriteTableSep(w, widths, style)
	for i, project := range sortedProjects {
		langs := strings.Join(model.GetTopLanguages(project.Languages), ", ")

		score := project.Readiness.Score
		scoreColor := lipRed
		scoreLabel := "NOT READY"
		if score >= 80 {
			scoreColor = lipGreen
			scoreLabel = "READY"
		} else if score >= 50 {
			scoreColor = lipYellow
			scoreLabel = "WARN"
		}

		healthBadge := Colorize(style, scoreColor, fmt.Sprintf("%3d/100", score))
		if scoreLabel != "" {
			healthBadge = fmt.Sprintf("%s %s", healthBadge, Colorize(style, scoreColor, scoreLabel))
		}

		WriteTableRow(w, []string{
			Colorize(style, lipYellow, fmt.Sprintf("%02d", i+1)),
			truncate(project.Name, 18),
			Colorize(style, lipYellow, fmt.Sprintf("%5d", project.TotalFiles)),
			truncate(langs, 20),
			healthBadge,
		}, widths, style)
	}
	WriteTableBottom(w, widths, style)
}

func boolLabel(value bool, trueText string, falseText string) string {
	if value {
		return trueText
	}
	return falseText
}

func checkMark(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func gitSummary(git model.GitMetadata) string {
	if !git.IsRepo {
		return "not a git repo"
	}
	return fmt.Sprintf("%s %s", git.Branch, boolLabel(git.Dirty, "dirty", "clean"))
}

func truncate(text string, max int) string {
	return model.FitText(text, max)
}

func writeBoxTop(w io.Writer, style Style) {
	fmt.Fprintln(w, border(style, "┌"+model.Repeat("─", reportWidth)+"┐"))
}

func writeBoxMiddle(w io.Writer, style Style) {
	fmt.Fprintln(w, border(style, "├"+model.Repeat("─", reportWidth)+"┤"))
}

func writeBoxBottom(w io.Writer, style Style) {
	fmt.Fprintln(w, border(style, "└"+model.Repeat("─", reportWidth)+"┘"))
}

func writeBoxLine(w io.Writer, content string, style Style) {
	content = fitDisplay(content, reportWidth)
	padding := model.Repeat(" ", reportWidth-visibleLen(content))
	fmt.Fprintf(w, "%s%s%s%s\n", border(style, "│"), content, padding, border(style, "│"))
}

func WriteTableTop(w io.Writer, title string, widths []int, style Style) {
	tableWidth := tableContentWidth(widths)
	fmt.Fprintln(w, border(style, tableBorder("+", "+", "+", "-", widths)))
	writeSQLBoxLine(w, " "+Colorize(style, lipWhite, model.FitText(title, tableWidth-1)), tableWidth, style)
	WriteTableSep(w, widths, style)
}

func WriteTableRow(w io.Writer, values []string, widths []int, style Style) {
	writeDelimitedRow(w, values, widths, style)
}

func WriteTableSep(w io.Writer, widths []int, style Style) {
	fmt.Fprintln(w, border(style, tableBorder("+", "+", "+", "-", widths)))
}

func WriteTableBottom(w io.Writer, widths []int, style Style) {
	fmt.Fprintln(w, border(style, tableBorder("+", "+", "+", "-", widths)))
}

func writeDelimitedRow(w io.Writer, values []string, widths []int, style Style) {
	fmt.Fprint(w, border(style, "|"))
	for i, value := range values {
		value = fitDisplay(value, widths[i])
		fmt.Fprintf(w, " %s%s %s", value, model.Repeat(" ", widths[i]-visibleLen(value)), border(style, "|"))
	}
	fmt.Fprintln(w)
}

func border(style Style, text string) string {
	c := lipGreen
	if style.IsSnapshot {
		c = lipGoBlue
	}
	return Colorize(style, c, text)
}

func writeSQLBoxLine(w io.Writer, content string, width int, style Style) {
	content = fitDisplay(content, width)
	padding := model.Repeat(" ", width-visibleLen(content))
	fmt.Fprintf(w, "%s%s%s%s\n", border(style, "|"), content, padding, border(style, "|"))
}

func tableContentWidth(widths []int) int {
	width := 0
	for _, col := range widths {
		width += col + 3
	}
	return width - 1
}

func tableBorder(left string, sep string, right string, horizontal string, widths []int) string {
	parts := make([]string, 0, len(widths))
	for _, width := range widths {
		parts = append(parts, model.Repeat(horizontal, width+2))
	}
	return left + strings.Join(parts, sep) + right
}

func renderBar(value int, maxValue int, width int, style Style) string {
	bar := model.MakeBar(value, maxValue, width)
	if !style.Color {
		return bar
	}

	runes := []rune(bar)
	filled := 0
	for filled < len(runes) && runes[filled] == '█' {
		filled++
	}
	filledPart := Colorize(style, percentageColor(model.PercentOfTotal(value, maxValue)), string(runes[:filled]))
	emptyPart := Colorize(style, lipDim, string(runes[filled:]))
	return filledPart + emptyPart
}

func fitDisplay(text string, max int) string {
	if visibleLen(text) <= max {
		return text
	}

	plain := StripANSI(text)
	return model.FitText(plain, max)
}

func visibleLen(text string) int {
	return model.RuneLen(StripANSI(text))
}

func StripANSI(text string) string {
	var b strings.Builder
	inEscape := false

	for i := 0; i < len(text); i++ {
		ch := text[i]
		if inEscape {
			if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
				inEscape = false
			}
			continue
		}
		if ch == 0x1b && i+1 < len(text) && text[i+1] == '[' {
			inEscape = true
			i++
			continue
		}
		b.WriteByte(ch)
	}

	return b.String()
}
