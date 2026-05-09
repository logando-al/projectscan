package audit

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"projectscan/internal/model"
	"projectscan/internal/scan"
)

const maxAuditFileBytes int64 = 1024 * 1024

type textFileStats struct {
	Files      int
	Total      int
	Code       int
	Blank      int
	Comments   int
	Largest    int
	LargestRel string
}

func safetyModule(projects []model.Project) ModuleResult {
	rows := []map[string]string{}
	for _, project := range model.SortedProjectsByName(projects) {
		projectRows := safetyRows(project)
		if len(projectRows) == 0 {
			projectRows = append(projectRows, map[string]string{
				"Project":  project.Name,
				"File":     "-",
				"Line":     "-",
				"Risk":     "No publish-blocking risks detected",
				"Severity": "PASS",
			})
		}
		rows = append(rows, projectRows...)
	}
	return ModuleResult{
		Name:         "open_source_safety",
		Title:        "Open Source Safety",
		Columns:      []string{"Project", "File", "Line", "Risk", "Severity"},
		Rows:         rows,
		SecretValues: "redacted",
	}
}

func readmeQualityModule(projects []model.Project) ModuleResult {
	rows := []map[string]string{}
	for _, project := range model.SortedProjectsByName(projects) {
		score, found, missing := readmeQuality(project.Path)
		rows = append(rows, map[string]string{
			"Project": project.Name,
			"Score":   fmt.Sprintf("%d", score),
			"Found":   strings.Join(found, ", "),
			"Missing": strings.Join(missing, ", "),
		})
	}
	return ModuleResult{
		Name:    "readme_quality",
		Title:   "README Quality",
		Columns: []string{"Project", "Score", "Found", "Missing"},
		Rows:    rows,
	}
}

func locModule(projects []model.Project) ModuleResult {
	rows := []map[string]string{}
	for _, project := range model.SortedProjectsByName(projects) {
		stats := countProjectLines(project.Path)
		rows = append(rows, map[string]string{
			"Project":      project.Name,
			"Files":        fmt.Sprintf("%d", stats.Files),
			"Total LOC":    fmt.Sprintf("%d", stats.Total),
			"Code LOC":     fmt.Sprintf("%d", stats.Code),
			"Blank LOC":    fmt.Sprintf("%d", stats.Blank),
			"Comment LOC":  fmt.Sprintf("%d", stats.Comments),
			"Avg/File":     fmt.Sprintf("%d", average(stats.Total, stats.Files)),
			"Largest File": fitCell(stats.LargestRel, 32),
		})
	}
	return ModuleResult{
		Name:    "lines_of_code",
		Title:   "Lines of Code",
		Columns: []string{"Project", "Files", "Total LOC", "Code LOC", "Blank LOC", "Comment LOC", "Avg/File", "Largest File"},
		Rows:    rows,
	}
}

func gitHygieneModule(projects []model.Project) ModuleResult {
	rows := []map[string]string{}
	for _, project := range model.SortedProjectsByName(projects) {
		score := 0
		signals := []string{}
		if project.Git.IsRepo {
			score += 20
			signals = append(signals, "repo")
		}
		if project.Git.RemoteURL != "" {
			score += 20
			signals = append(signals, "remote")
		}
		if project.Git.Branch != "" && project.Git.Branch != "master" {
			score += 15
			signals = append(signals, "branch")
		}
		if project.Git.LastCommitHash != "" {
			score += 20
			signals = append(signals, "history")
		}
		if project.Git.IsRepo && !project.Git.Dirty {
			score += 15
			signals = append(signals, "clean")
		}
		if project.Git.IsRepo && project.Git.DaysSinceCommit <= 90 {
			score += 10
			signals = append(signals, "recent")
		}
		rows = append(rows, map[string]string{
			"Project": project.Name,
			"Score":   fmt.Sprintf("%d", score),
			"Branch":  valueOrDash(project.Git.Branch),
			"Dirty":   dirtyLabel(project.Git),
			"Remote":  yesNo(project.Git.RemoteURL != ""),
			"Signals": strings.Join(signals, ", "),
		})
	}
	return ModuleResult{
		Name:    "git_hygiene",
		Title:   "Git Hygiene",
		Columns: []string{"Project", "Score", "Branch", "Dirty", "Remote", "Signals"},
		Rows:    rows,
	}
}

func dependencyModule(projects []model.Project) ModuleResult {
	rows := []map[string]string{}
	for _, project := range model.SortedProjectsByName(projects) {
		files := dependencyFiles(project.Path)
		rows = append(rows, map[string]string{
			"Project": project.Name,
			"Files":   fmt.Sprintf("%d", len(files)),
			"Manifests": func() string {
				if len(files) == 0 {
					return "-"
				}
				return strings.Join(files, ", ")
			}(),
		})
	}
	return ModuleResult{
		Name:    "dependency_inventory",
		Title:   "Dependency Inventory",
		Columns: []string{"Project", "Files", "Manifests"},
		Rows:    rows,
	}
}

func safetyRows(project model.Project) []map[string]string {
	rows := []map[string]string{}
	_ = filepath.WalkDir(project.Path, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if path != project.Path && model.ShouldSkipDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		rel := relativePath(project.Path, path)
		for _, risk := range fileNameRisks(entry.Name(), rel) {
			rows = append(rows, map[string]string{
				"Project":  project.Name,
				"File":     rel,
				"Line":     "-",
				"Risk":     risk.kind,
				"Severity": risk.severity,
			})
		}
		for _, risk := range fileContentRisks(path, rel) {
			rows = append(rows, map[string]string{
				"Project":  project.Name,
				"File":     rel,
				"Line":     fmt.Sprintf("%d", risk.line),
				"Risk":     risk.kind,
				"Severity": risk.severity,
			})
		}
		return nil
	})
	return rows
}

type risk struct {
	kind     string
	severity string
	line     int
}

func fileNameRisks(name string, rel string) []risk {
	lowerName := strings.ToLower(name)
	lowerRel := strings.ToLower(rel)
	risks := []risk{}
	switch lowerName {
	case ".env", ".env.local", ".env.production", ".env.development":
		risks = append(risks, risk{kind: "environment file", severity: "HIGH"})
	}
	switch filepath.Ext(lowerName) {
	case ".pem", ".key", ".p12", ".sqlite", ".db", ".bak", ".backup":
		risks = append(risks, risk{kind: "sensitive artifact", severity: "HIGH"})
	case ".sql":
		risks = append(risks, risk{kind: "database dump/source", severity: "MEDIUM"})
	}
	if strings.Contains(lowerRel, "dump") || strings.Contains(lowerRel, "backup") {
		risks = append(risks, risk{kind: "backup artifact", severity: "MEDIUM"})
	}
	return risks
}

func fileContentRisks(path string, rel string) []risk {
	info, err := os.Stat(path)
	if err != nil || info.Size() > maxAuditFileBytes || !isTextLike(rel) {
		return nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), int(maxAuditFileBytes))
	risks := []risk{}
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.ToLower(scanner.Text())
		if matchesSecretPattern(line) {
			risks = append(risks, risk{kind: "secret-like token", severity: "HIGH", line: lineNo})
		}
		if matchesLocalMachinePattern(line, rel) {
			risks = append(risks, risk{kind: "local machine reference", severity: "LOW", line: lineNo})
		}
	}
	return risks
}

func matchesSecretPattern(line string) bool {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "-----begin rsa private key") || strings.HasPrefix(trimmed, "-----begin openssh private key") {
		return true
	}
	keys := []string{
		"api_key", "apikey", "password", "passwd", "private_key",
		"client_secret", "access_key", "database_url", "jwt_secret", "session_secret",
	}
	if assignmentLike(line) && containsAny(line, "postgres://", "mysql://", "mongodb://", "redis://", "sk-", "ghp_", "github_pat_", "xoxb-", "akia") {
		return true
	}
	if assignmentLike(line) {
		keySide := assignmentKeySide(line)
		for _, key := range keys {
			if strings.Contains(keySide, key) {
				return true
			}
		}
	}
	for _, key := range keys {
		if strings.HasPrefix(trimmed, key+":") {
			return true
		}
	}
	return false
}

func matchesLocalMachinePattern(line string, rel string) bool {
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".go", ".rs", ".ts", ".tsx", ".js", ".jsx", ".py", ".java", ".php", ".rb":
		return false
	}
	patterns := []string{"/users/", `c:\users\`, "/home/", "192.168.", "10.0.", "172.16."}
	for _, pattern := range patterns {
		if strings.Contains(line, pattern) {
			return true
		}
	}
	return false
}

func assignmentLike(line string) bool {
	if strings.Contains(line, "=") {
		return true
	}
	trimmed := strings.TrimSpace(line)
	index := strings.Index(trimmed, ":")
	if index <= 0 {
		return false
	}
	key := trimmed[:index]
	return !strings.ContainsAny(key, " \t\"'()/")
}

func assignmentKeySide(line string) string {
	if index := strings.Index(line, "="); index >= 0 {
		return strings.TrimSpace(line[:index])
	}
	if index := strings.Index(line, ":"); index >= 0 {
		return strings.TrimSpace(line[:index])
	}
	return strings.TrimSpace(line)
}

func readmeQuality(projectPath string) (int, []string, []string) {
	path := findFirst(projectPath, []string{"README.md", "README.mdx", "readme.md", "README"})
	if path == "" {
		return 0, nil, []string{"file", "title", "description", "installation", "usage", "configuration", "license"}
	}
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return 0, nil, []string{"readable file"}
	}
	content := strings.ToLower(string(contentBytes))
	checks := []struct {
		name string
		ok   bool
	}{
		{name: "file", ok: true},
		{name: "title", ok: hasMarkdownHeading(content)},
		{name: "description", ok: len(strings.Fields(content)) >= 8},
		{name: "installation", ok: containsAny(content, "install", "setup", "getting started")},
		{name: "usage", ok: containsAny(content, "usage", "run", "example", "quickstart")},
		{name: "configuration", ok: containsAny(content, "configuration", "config", "environment", ".env")},
		{name: "license", ok: containsAny(content, "license", "mit", "apache", "bsd")},
	}
	found := []string{}
	missing := []string{}
	for _, check := range checks {
		if check.ok {
			found = append(found, check.name)
		} else {
			missing = append(missing, check.name)
		}
	}
	score := len(found) * 100 / len(checks)
	return score, found, missing
}

func countProjectLines(projectPath string) textFileStats {
	stats := textFileStats{}
	_ = filepath.WalkDir(projectPath, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if path != projectPath && model.ShouldSkipDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !isCountableSource(path, entry.Name()) {
			return nil
		}
		fileStats := countFileLines(path)
		stats.Files++
		stats.Total += fileStats.Total
		stats.Code += fileStats.Code
		stats.Blank += fileStats.Blank
		stats.Comments += fileStats.Comments
		if fileStats.Total > stats.Largest {
			stats.Largest = fileStats.Total
			stats.LargestRel = relativePath(projectPath, path)
		}
		return nil
	})
	return stats
}

func countFileLines(path string) textFileStats {
	file, err := os.Open(path)
	if err != nil {
		return textFileStats{}
	}
	defer file.Close()

	stats := textFileStats{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), int(maxAuditFileBytes))
	for scanner.Scan() {
		stats.Total++
		trimmed := strings.TrimSpace(scanner.Text())
		switch {
		case trimmed == "":
			stats.Blank++
		case isCommentLine(path, trimmed):
			stats.Comments++
		default:
			stats.Code++
		}
	}
	return stats
}

func dependencyFiles(projectPath string) []string {
	candidates := []string{
		"go.mod", "go.sum", "package.json", "package-lock.json", "pnpm-lock.yaml", "yarn.lock",
		"Cargo.toml", "Cargo.lock", "requirements.txt", "pyproject.toml", "poetry.lock", "uv.lock",
		"composer.json", "Gemfile", "Gemfile.lock", "pom.xml", "build.gradle", "build.gradle.kts",
	}
	found := []string{}
	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(projectPath, candidate)); err == nil {
			found = append(found, candidate)
		}
	}
	sort.Strings(found)
	return found
}

func isCountableSource(path string, name string) bool {
	_, ok := scan.LanguageForFile(path, name)
	return ok
}

func isTextLike(path string) bool {
	name := filepath.Base(path)
	if _, ok := scan.LanguageForFile(path, name); ok {
		return true
	}
	switch strings.ToLower(name) {
	case ".env", ".env.local", ".env.production", ".env.development", "dockerfile", "containerfile", "makefile":
		return true
	}
	return false
}

func isCommentLine(path string, trimmed string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".py", ".sh", ".bash", ".zsh", ".yaml", ".yml", ".toml":
		return strings.HasPrefix(trimmed, "#")
	case ".sql":
		return strings.HasPrefix(trimmed, "--")
	case ".html", ".htm":
		return strings.HasPrefix(trimmed, "<!--")
	default:
		return strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*")
	}
}

func findFirst(root string, names []string) string {
	for _, name := range names {
		path := filepath.Join(root, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func hasMarkdownHeading(content string) bool {
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "# ") {
			return true
		}
	}
	return false
}

func containsAny(content string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(content, needle) {
			return true
		}
	}
	return false
}

func relativePath(root string, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.Base(path)
	}
	return filepath.ToSlash(rel)
}

func average(total int, count int) int {
	if count <= 0 {
		return 0
	}
	return total / count
}

func valueOrDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func fitCell(value string, width int) string {
	if value == "" {
		return "-"
	}
	return model.FitText(value, width)
}
