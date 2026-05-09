package scoring

import (
	"strings"

	"projectscan/internal/config"
	"projectscan/internal/model"
)

func ScoreProject(project model.Project) (string, []string) {
	lowerName := strings.ToLower(project.RawName)
	signals := append([]string{}, project.Readiness.Signals...)
	if strings.Contains(lowerName, "archive") || strings.Contains(lowerName, "archived") || strings.Contains(lowerName, "old") {
		return model.LabelArchived, append(signals, "archive name")
	}
	if project.Git.IsRepo && project.Git.RemoteURL != "" && !project.Git.Dirty && project.Readiness.Score >= 65 {
		return model.LabelProductionReady, append(signals, "production signals")
	}
	return model.LabelExperiment, append(signals, "default")
}

func ApplyProjectOverride(project *model.Project, override config.ProjectOverride) {
	if override.DisplayName != "" {
		project.Name = override.DisplayName
	}
	if override.Label != "" {
		project.Label = override.Label
		project.ScoreSignals = append(project.ScoreSignals, "config override")
	}
	if override.ArchivedReason != "" {
		project.ArchivedReason = override.ArchivedReason
	}
	project.Pinned = override.Pinned
}
