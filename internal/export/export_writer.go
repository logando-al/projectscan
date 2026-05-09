package export

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"projectscan/internal/model"
)

const DefaultDirName = "projectscan-exports"

type WriteResult struct {
	Path       string
	Filename   string
	ReportType string
	Format     string
	Bytes      int
}

func WriteReport(report model.WorkspaceReport, req model.ExportRequest, outputDir string) (WriteResult, error) {
	if req.ReportType == "" {
		req.ReportType = model.ReportAudit
	}
	if req.Format == "" || req.Format == model.ExportTerminal {
		req.Format = model.ExportHTML
	}
	if outputDir == "" {
		outputDir = filepath.Join(report.RootPath, DefaultDirName)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return WriteResult{}, err
	}

	content, err := BuildReport(report, req)
	if err != nil {
		return WriteResult{}, err
	}
	path := uniqueExportPath(outputDir, exportBaseName(report, req), exportExtension(req.Format))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return WriteResult{}, err
	}

	return WriteResult{
		Path:       path,
		Filename:   filepath.Base(path),
		ReportType: req.ReportType,
		Format:     req.Format,
		Bytes:      len([]byte(content)),
	}, nil
}

func exportBaseName(report model.WorkspaceReport, req model.ExportRequest) string {
	generatedAt := report.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	return fmt.Sprintf(
		"projectscan-%s-%s-%s",
		slug(req.ReportType),
		slug(req.Format),
		generatedAt.Format("20060102-150405"),
	)
}

func uniqueExportPath(outputDir string, base string, ext string) string {
	path := filepath.Join(outputDir, base+"."+ext)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	for i := 1; ; i++ {
		candidate := filepath.Join(outputDir, fmt.Sprintf("%s-%02d.%s", base, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

func exportExtension(format string) string {
	switch format {
	case model.ExportMarkdown:
		return "md"
	case model.ExportJSON:
		return "json"
	case model.ExportCSV:
		return "csv"
	default:
		return "html"
	}
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
