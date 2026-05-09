package export

import (
	"fmt"
	"strings"

	"projectscan/internal/audit"
)

func buildMarkdown(result audit.Result) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Projectscan %s Report\n\n", result.Title)
	fmt.Fprintf(&b, "- Root Path: `%s`\n", result.RootPath)
	fmt.Fprintf(&b, "- Projects: %d\n", result.ProjectCount)
	fmt.Fprintf(&b, "- Files: %d\n\n", result.TotalFiles)
	if result.Message != "" {
		fmt.Fprintln(&b, result.Message)
		fmt.Fprintln(&b)
	}
	for _, module := range result.Modules {
		writeMarkdownModule(&b, module)
	}
	return b.String()
}

func writeMarkdownModule(b *strings.Builder, module audit.ModuleResult) {
	fmt.Fprintf(b, "## %s\n\n", module.Title)
	if module.Message != "" {
		fmt.Fprintln(b, module.Message)
		fmt.Fprintln(b)
		return
	}
	if len(module.Columns) == 0 {
		return
	}
	fmt.Fprintf(b, "| %s |\n", strings.Join(module.Columns, " | "))
	align := make([]string, len(module.Columns))
	for i := range align {
		align[i] = "---"
	}
	fmt.Fprintf(b, "|%s|\n", strings.Join(align, "|"))
	for _, row := range module.Rows {
		values := make([]string, 0, len(module.Columns))
		for _, column := range module.Columns {
			values = append(values, row[column])
		}
		fmt.Fprintf(b, "| %s |\n", strings.Join(values, " | "))
	}
	fmt.Fprintln(b)
}
