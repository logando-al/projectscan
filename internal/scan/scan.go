package scan

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"projectscan/internal/config"
	"projectscan/internal/gitmeta"
	"projectscan/internal/model"
	"projectscan/internal/readiness"
	"projectscan/internal/scoring"
)

type Options struct {
	ConfigPath string
}

func Projects(rootPath string) ([]model.Project, error) {
	report, err := Workspace(rootPath, Options{})
	if err != nil {
		return nil, err
	}
	return report.Projects, nil
}

func Workspace(rootPath string, opts Options) (model.WorkspaceReport, error) {
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return model.WorkspaceReport{}, err
	}
	projectConfig, err := config.Load(absRoot, opts.ConfigPath)
	if err != nil {
		return model.WorkspaceReport{}, err
	}
	ignorePatterns, err := config.LoadIgnorePatterns(absRoot, projectConfig.IgnorePatterns)
	if err != nil {
		return model.WorkspaceReport{}, err
	}

	projects, err := projectsWithContext(absRoot, projectConfig, ignorePatterns)
	if err != nil {
		return model.WorkspaceReport{}, err
	}

	report := model.WorkspaceReport{
		RootPath:    absRoot,
		Projects:    projects,
		GeneratedAt: time.Now().UTC(),
	}
	return model.WithComputedTotals(report), nil
}

func projectsWithContext(rootPath string, projectConfig config.ProjectscanConfig, ignorePatterns []string) ([]model.Project, error) {
	if shouldScanRootAsProject(rootPath) {
		project, err := oneProject(rootPath, rootPath, projectConfig, ignorePatterns)
		if err != nil {
			return nil, err
		}
		if project.TotalFiles == 0 {
			return nil, nil
		}
		return []model.Project{project}, nil
	}

	entries, err := os.ReadDir(rootPath)
	if err != nil {
		return nil, err
	}

	var projects []model.Project
	for _, entry := range entries {
		if !entry.IsDir() || shouldSkipPath(rootPath, filepath.Join(rootPath, entry.Name()), entry.Name(), ignorePatterns) {
			continue
		}

		path := filepath.Join(rootPath, entry.Name())
		project, err := oneProject(path, rootPath, projectConfig, ignorePatterns)
		if err != nil {
			return nil, err
		}
		if project.TotalFiles == 0 {
			continue
		}
		projects = append(projects, project)
	}

	if len(projects) > 0 {
		return projects, nil
	}

	project, err := oneProject(rootPath, rootPath, projectConfig, ignorePatterns)
	if err != nil {
		return nil, err
	}
	if project.TotalFiles == 0 {
		return projects, nil
	}

	return []model.Project{project}, nil
}

func shouldScanRootAsProject(rootPath string) bool {
	markers := []string{
		".git",
		"go.mod",
		"package.json",
		"Cargo.toml",
		"pyproject.toml",
		"requirements.txt",
		"uv.lock",
		"composer.json",
		"Gemfile",
		"pom.xml",
		"build.gradle",
		"build.gradle.kts",
		"deno.json",
		"svelte.config.js",
		"svelte.config.ts",
	}
	for _, marker := range markers {
		if _, err := os.Stat(filepath.Join(rootPath, marker)); err == nil {
			return true
		}
	}
	return false
}

func oneProject(path string, rootPath string, projectConfig config.ProjectscanConfig, ignorePatterns []string) (model.Project, error) {
	languages, err := CountLanguagesWithIgnore(path, rootPath, ignorePatterns)
	if err != nil {
		return model.Project{}, err
	}
	rawName := filepath.Base(path)
	project := model.Project{
		Name:          rawName,
		RawName:       rawName,
		Path:          path,
		Languages:     languages,
		TotalFiles:    model.SumCounts(languages),
		Git:           gitmeta.Inspect(path),
		Readiness:     readiness.Inspect(path),
		MainLanguages: model.GetTopLanguages(languages),
	}
	project.Readiness.Remote = project.Git.RemoteURL != ""
	project.Readiness.Score, project.Readiness.Signals = readiness.Score(project.Readiness)
	project.Label, project.ScoreSignals = scoring.ScoreProject(project)
	scoring.ApplyProjectOverride(&project, projectConfig.Projects[rawName])
	return project, nil
}

func CountLanguages(rootPath string) (map[string]int, error) {
	return CountLanguagesWithIgnore(rootPath, rootPath, nil)
}

func CountLanguagesWithIgnore(rootPath string, scanRoot string, ignorePatterns []string) (map[string]int, error) {
	languages := map[string]int{}

	err := filepath.WalkDir(rootPath, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != rootPath && shouldSkipPath(scanRoot, path, entry.Name(), ignorePatterns) {
				return filepath.SkipDir
			}
			return nil
		}
		if shouldSkipPath(scanRoot, path, entry.Name(), ignorePatterns) {
			return nil
		}

		if language, ok := LanguageForFile(path, entry.Name()); ok {
			languages[language]++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return languages, nil
}

func LanguageForFile(path string, name string) (string, bool) {
	switch name {
	case "Dockerfile", "Containerfile":
		return "Docker", true
	case "Makefile":
		return "Make", true
	}

	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return "Go", true
	case ".ts", ".tsx":
		return "TypeScript", true
	case ".svelte":
		return "Svelte", true
	case ".rs":
		return "Rust", true
	case ".py":
		return "Python", true
	case ".sh", ".bash", ".zsh":
		return "Shell", true
	case ".sql":
		return "SQL", true
	case ".yaml", ".yml":
		return "YAML", true
	case ".css", ".scss", ".sass":
		return "CSS", true
	case ".html", ".htm":
		return "HTML", true
	case ".js", ".jsx", ".mjs", ".cjs":
		return "JavaScript", true
	case ".json":
		return "JSON", true
	case ".md", ".mdx":
		return "Markdown", true
	case ".toml":
		return "TOML", true
	case ".java":
		return "Java", true
	case ".php":
		return "PHP", true
	case ".rb":
		return "Ruby", true
	case ".c", ".h":
		return "C", true
	case ".cpp", ".cc", ".cxx", ".hpp":
		return "C++", true
	}

	return "", false
}

func shouldSkipPath(rootPath string, path string, name string, ignorePatterns []string) bool {
	if model.ShouldSkipDir(name) {
		return true
	}
	rel, err := filepath.Rel(rootPath, path)
	if err != nil {
		rel = name
	}
	rel = filepath.ToSlash(rel)
	for _, pattern := range ignorePatterns {
		pattern = strings.TrimSpace(filepath.ToSlash(pattern))
		if pattern == "" || strings.HasPrefix(pattern, "#") {
			continue
		}
		if pattern == name || pattern == rel {
			return true
		}
		if ok, _ := filepath.Match(pattern, name); ok {
			return true
		}
		if ok, _ := filepath.Match(pattern, rel); ok {
			return true
		}
	}
	return false
}
