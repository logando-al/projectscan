package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"projectscan/internal/model"
)

func TestAppModelRendersProjectScanMenu(t *testing.T) {
	app := NewAppModel(sampleReport(), 38*time.Millisecond, nil)
	updatedApp, _ := app.Update(tea.WindowSizeMsg{Width: 200, Height: 40})
	app = updatedApp.(AppModel)

	view := app.View()
	assertContains(t, view, "██████")
	assertContains(t, view, "Scan Folder")
	assertContains(t, view, "Audit Projects")
	assertContains(t, view, "⚡")
	assertContains(t, view, "ProjectScan loaded 1 project")
	if strings.Contains(view, "z   z   z") {
		t.Fatalf("home screen should not render sleep marks:\n%s", view)
	}
	if strings.Contains(view, "Root      /tmp/code") {
		t.Fatalf("home screen should stay close to the reference menu, got stats panel:\n%s", view)
	}
}

func TestAppModelDisplaysCompactRootPath(t *testing.T) {
	app := NewAppModel(sampleReport(), 38*time.Millisecond, nil)
	updatedApp, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = updatedApp.(AppModel)

	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	app = updated.(AppModel)

	view := app.View()
	assertContains(t, view, ".../code")
	if strings.Contains(view, "  /tmp/code\n") {
		t.Fatalf("expected compact root path in recent scans view:\n%s", view)
	}
}

func TestAppModelHotkeysOpenScreensAndQuit(t *testing.T) {
	app := NewAppModel(sampleReport(), 38*time.Millisecond, nil)
	updatedApp, _ := app.Update(tea.WindowSizeMsg{Width: 200, Height: 40})
	app = updatedApp.(AppModel)

	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	app = updated.(AppModel)
	if app.Screen != ScreenGit {
		t.Fatalf("expected git hotkey to open git screen, got %#v", app.Screen)
	}
	assertContains(t, app.View(), "GIT METADATA")

	updated, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	app = updated.(AppModel)
	if !app.Quitting {
		t.Fatalf("expected q to mark model quitting")
	}
	if cmd == nil {
		t.Fatalf("expected q to return tea.Quit command")
	}
}

func TestAppModelAuditScreenShowsRoadmapModules(t *testing.T) {
	app := NewAppModel(sampleReport(), 38*time.Millisecond, nil)
	updatedApp, _ := app.Update(tea.WindowSizeMsg{Width: 200, Height: 40})
	app = updatedApp.(AppModel)

	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	app = updated.(AppModel)

	if app.Screen != ScreenAudit {
		t.Fatalf("expected audit hotkey to open audit screen, got %#v", app.Screen)
	}
	view := app.View()
	t.Logf("View:\n%s", view)
	assertContains(t, view, "Open Source Safety")
	assertContains(t, view, "README Quality")
	assertContains(t, view, "Lines of Code")
	assertContains(t, view, "Dependency Inventory")
	assertContains(t, view, "External Tool Integration")
}

func TestAppModelAuditScreenSwitchesRoadmapModule(t *testing.T) {
	app := NewAppModel(sampleReport(), 38*time.Millisecond, nil)
	updatedApp, _ := app.Update(tea.WindowSizeMsg{Width: 200, Height: 40})
	app = updatedApp.(AppModel)

	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	app = updated.(AppModel)

	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	app = updated.(AppModel)

	t.Logf("Screen: %s, Focus: %s, AuditCursor: %d", app.Screen, app.Focus, app.AuditCursor)

	if app.AuditCursor != 2 {
		t.Fatalf("expected audit cursor to move with hotkey '3', got %#v", app.AuditCursor)
	}
	assertContains(t, app.View(), "README")
	assertContains(t, app.View(), "SCORE")
}

func TestAppModelTabFocusToggles(t *testing.T) {
	app := NewAppModel(sampleReport(), 38*time.Millisecond, nil)
	updatedApp, _ := app.Update(tea.WindowSizeMsg{Width: 200, Height: 40})
	app = updatedApp.(AppModel)

	// Initially in Sidebar focus when opening a screen
	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	app = updated.(AppModel)
	if app.Focus != FocusViewport {
		t.Fatalf("expected initial focus to be viewport when opening screen, got %v", app.Focus)
	}

	// Press Tab to focus Sidebar
	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = updated.(AppModel)
	if app.Focus != FocusSidebar {
		t.Fatalf("expected Tab to switch focus to sidebar, got %v", app.Focus)
	}

	// Press Tab to focus Viewport again
	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = updated.(AppModel)
	if app.Focus != FocusViewport {
		t.Fatalf("expected Tab to switch focus back to viewport, got %v", app.Focus)
	}
}

func TestAppModelScanFolderInputUpdatesRoot(t *testing.T) {
	app := NewAppModel(sampleReport(), 38*time.Millisecond, nil)
	updatedApp, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = updatedApp.(AppModel)

	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	app = updated.(AppModel)
	if app.Screen != ScreenScanFolder || !app.InputActive {
		t.Fatalf("expected scan folder input screen, got %#v", app)
	}

	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/tmp/code")})
	app = updated.(AppModel)
	if app.InputValue != "/tmp/code/tmp/code" {
		t.Fatalf("expected typed path, got %q", app.InputValue)
	}
}

func TestAppModelExportsScreenSelectsFormatAndWritesFile(t *testing.T) {
	root := t.TempDir()
	app := NewAppModel(sampleReportAtRoot(root), 38*time.Millisecond, nil)
	updatedApp, _ := app.Update(tea.WindowSizeMsg{Width: 200, Height: 40})
	app = updatedApp.(AppModel)

	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	app = updated.(AppModel)
	if app.Screen != ScreenExports {
		t.Fatalf("expected exports screen, got %s", app.Screen)
	}
	assertContains(t, app.View(), "REPORT")
	assertContains(t, app.View(), "FORMAT")

	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	app = updated.(AppModel)
	if app.ExportFormat != model.ExportMarkdown {
		t.Fatalf("expected markdown format, got %q", app.ExportFormat)
	}

	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = updated.(AppModel)
	if app.ExportPath == "" || filepath.Ext(app.ExportPath) != ".md" {
		t.Fatalf("expected markdown export path, got %#v", app)
	}
	if _, err := os.Stat(app.ExportPath); err != nil {
		t.Fatalf("expected export file: %v", err)
	}
	assertContains(t, app.View(), "Saved")
}

func sampleReport() model.WorkspaceReport {
	project := model.Project{
		Name:          "api",
		Path:          "/tmp/code/api",
		Languages:     map[string]int{"Go": 3},
		TotalFiles:    3,
		Label:         model.LabelExperiment,
		MainLanguages: []string{"Go"},
		Git:           model.GitMetadata{IsRepo: true, Branch: "main"},
		Readiness:     model.ReadinessAudit{Readme: true, Tests: true, Score: 40},
	}
	return model.WithComputedTotals(model.WorkspaceReport{
		RootPath:    "/tmp/code",
		Projects:    []model.Project{project},
		GeneratedAt: time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC),
	})
}

func sampleReportAtRoot(root string) model.WorkspaceReport {
	report := sampleReport()
	report.RootPath = root
	report.GeneratedAt = time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	return report
}

func assertContains(t *testing.T, text string, want string) {
	t.Helper()
	if !strings.Contains(text, want) {
		t.Fatalf("expected output to contain %q\n\n%s", want, text)
	}
}
