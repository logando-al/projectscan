package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"projectscan/internal/audit"
	"projectscan/internal/export"
	"projectscan/internal/model"
	"projectscan/internal/render"
	"projectscan/internal/scan"
)

type Screen string

const (
	ScreenHome        Screen = "home"
	ScreenScanFolder  Screen = "scan-folder"
	ScreenOpenProject Screen = "open-project"
	ScreenAudit       Screen = "audit"
	ScreenPortfolio   Screen = "portfolio"
	ScreenGit         Screen = "git"
	ScreenExports     Screen = "exports"
	ScreenSettings    Screen = "settings"
	ScreenRecentScans Screen = "recent-scans"
	ScreenSummary     Screen = "summary"
)

type Focus string

const (
	FocusSidebar  Focus = "sidebar"
	FocusViewport Focus = "viewport"
)

type MenuItem struct {
	Icon   string
	Label  string
	Hotkey string
	Screen Screen
}

type AuditMenuItem struct {
	Icon       string
	Label      string
	Hotkey     string
	ReportType string
}

type ExportReportItem struct {
	Label      string
	ReportType string
}

type AppModel struct {
	Report        model.WorkspaceReport
	LoadTime      time.Duration
	LoadError     error
	Menu          []MenuItem
	Cursor        int
	AuditCursor   int
	ExportCursor  int
	ExportFormat  string
	ExportPath    string
	ExportMessage string
	ExportError   string
	ExportBytes   int
	Screen        Screen
	InputActive   bool
	InputValue    string
	Quitting      bool
	Width         int
	Height        int
	Viewport      viewport.Model
	Ready         bool
	Focus         Focus
}

var (
	dashboardWidth = 112
	screenStyle    = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#DDE7FF")).
			Padding(0, 0)
	logoProjectStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00ADD8")).
				Bold(true)
	logoScanStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F3FBFF")).
			Bold(true)
	menuIconStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00ADD8"))
	menuLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#B9D4FF"))
	menuHotkeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF956D"))
	menuCursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ADD8")).Bold(true)
	panelStyle      = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C8D7F2"))
	statusPulseStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#D77CFF"))
	statusTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00ADD8"))
	statusValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#E29CFF"))
)

func NewAppModel(report model.WorkspaceReport, loadTime time.Duration, loadError error) AppModel {
	return AppModel{
		Report:    report,
		LoadTime:  loadTime,
		LoadError: loadError,
		Menu: []MenuItem{
			{Icon: "󰉋", Label: "Scan Folder", Hotkey: "f", Screen: ScreenScanFolder},
			{Icon: "", Label: "Open Project", Hotkey: "o", Screen: ScreenOpenProject},
			{Icon: "󰱒", Label: "Audit Projects", Hotkey: "a", Screen: ScreenAudit},
			{Icon: "󰓎", Label: "Portfolio Score", Hotkey: "p", Screen: ScreenPortfolio},
			{Icon: "", Label: "Git Metadata", Hotkey: "g", Screen: ScreenGit},
			{Icon: "󰈧", Label: "Exports", Hotkey: "e", Screen: ScreenExports},
			{Icon: "", Label: "Settings", Hotkey: "s", Screen: ScreenSettings},
			{Icon: "󰋚", Label: "Recent Scans", Hotkey: "r", Screen: ScreenRecentScans},
			{Icon: "󰍃", Label: "Quit", Hotkey: "q", Screen: ScreenHome},
		},
		Screen:       ScreenHome,
		InputValue:   report.RootPath,
		Focus:        FocusSidebar,
		ExportCursor: 1,
		ExportFormat: model.ExportHTML,
	}
}

var auditMenu = []AuditMenuItem{
	{Icon: "󰱒", Label: "Completeness", Hotkey: "1", ReportType: model.ReportReadiness},
	{Icon: "󰒃", Label: "Open Source Safety", Hotkey: "2", ReportType: model.ReportSafety},
	{Icon: "󰈙", Label: "README Quality", Hotkey: "3", ReportType: model.ReportReadme},
	{Icon: "󰉻", Label: "Lines of Code", Hotkey: "4", ReportType: model.ReportLOC},
	{Icon: "", Label: "Git Hygiene", Hotkey: "5", ReportType: model.ReportGitHygiene},
	{Icon: "󰏗", Label: "Dependency Inventory", Hotkey: "6", ReportType: model.ReportDeps},
	{Icon: "󰐱", Label: "Open Source Readiness", Hotkey: "7", ReportType: model.ReportOpenSourceReadiness},
	{Icon: "󰒋", Label: "External Tool Integration", Hotkey: "8", ReportType: model.ReportExternalTools},
}

var exportReports = []ExportReportItem{
	{Label: "Summary", ReportType: model.ReportSummary},
	{Label: "Full Audit", ReportType: model.ReportAudit},
	{Label: "Details", ReportType: model.ReportDetails},
	{Label: "Portfolio", ReportType: model.ReportPortfolio},
	{Label: "Git Metadata", ReportType: model.ReportGit},
	{Label: "Readiness", ReportType: model.ReportReadiness},
	{Label: "Open Source Safety", ReportType: model.ReportSafety},
	{Label: "README Quality", ReportType: model.ReportReadme},
	{Label: "Lines of Code", ReportType: model.ReportLOC},
	{Label: "Git Hygiene", ReportType: model.ReportGitHygiene},
	{Label: "Dependency Inventory", ReportType: model.ReportDeps},
	{Label: "Open Source Readiness", ReportType: model.ReportOpenSourceReadiness},
	{Label: "External Tool Integration", ReportType: model.ReportExternalTools},
}

func RunApp(rootPath string) error {
	start := time.Now()
	report, err := scan.Workspace(rootPath, scan.Options{})
	app := NewAppModel(report, time.Since(start), err)
	program := tea.NewProgram(app, tea.WithAltScreen())
	_, runErr := program.Run()
	if runErr != nil {
		return runErr
	}
	return nil
}

func (m AppModel) Init() tea.Cmd {
	return nil
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.InputActive {
			newModel, cmd := m.updateInput(msg)
			m = newModel.(AppModel)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		} else {
			if msg.Type == tea.KeyTab && m.Screen != ScreenHome {
				if m.Focus == FocusSidebar {
					m.Focus = FocusViewport
				} else {
					m.Focus = FocusSidebar
				}
				return m, nil
			}

			switch msg.Type {
			case tea.KeyCtrlC, tea.KeyEsc:
				if m.Screen != ScreenHome {
					m.Screen = ScreenHome
					m.Focus = FocusSidebar
					m.Viewport.GotoTop()
					return m, nil
				}
				m.Quitting = true
				return m, tea.Quit
			}

			if m.Screen == ScreenHome || m.Focus == FocusSidebar {
				switch msg.Type {
				case tea.KeyUp, tea.KeyDown:
					if msg.Type == tea.KeyUp && m.Cursor > 0 {
						m.Cursor--
					} else if msg.Type == tea.KeyDown && m.Cursor < len(m.Menu)-1 {
						m.Cursor++
					}
				case tea.KeyEnter:
					newModel, cmd := m.activate(m.Menu[m.Cursor])
					m = newModel.(AppModel)
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
					if m.Screen != ScreenHome {
						m.Focus = FocusViewport
					}
				case tea.KeyRunes:
					newModel, cmd := m.handleHotkey(string(msg.Runes))
					m = newModel.(AppModel)
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
					if m.Screen != ScreenHome {
						m.Focus = FocusViewport
					}
				}
			} else {
				if m.Screen == ScreenAudit && msg.Type == tea.KeyRunes {
					newModel, cmd := m.updateAudit(msg)
					m = newModel.(AppModel)
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
				} else if m.Screen == ScreenExports {
					newModel, cmd := m.updateExports(msg)
					m = newModel.(AppModel)
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
				} else if msg.Type == tea.KeyRunes {
					newModel, cmd := m.handleHotkey(string(msg.Runes))
					m = newModel.(AppModel)
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
				}
			}
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

		_, rightWidth := calculateWidths(m)
		vpHeight := m.Height - 6
		if vpHeight < 10 {
			vpHeight = 10
		}

		if !m.Ready {
			m.Viewport = viewport.New(rightWidth, vpHeight)
			m.Ready = true
		} else {
			m.Viewport.Width = rightWidth
			m.Viewport.Height = vpHeight
		}
	}

	if m.Ready {
		var contentBuilder strings.Builder
		if m.Screen != ScreenHome {
			writePanel(&contentBuilder, m)
			m.Viewport.SetContent(contentBuilder.String())
		}

		forward := true
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			isUpDown := keyMsg.Type == tea.KeyUp || keyMsg.Type == tea.KeyDown || keyMsg.String() == "j" || keyMsg.String() == "k"
			if (m.Focus == FocusSidebar || m.InputActive) && isUpDown {
				forward = false
			}
			if m.Screen == ScreenExports && m.Focus == FocusViewport {
				key := keyMsg.String()
				if keyMsg.Type == tea.KeyUp || keyMsg.Type == tea.KeyDown || keyMsg.Type == tea.KeyEnter || key == "h" || key == "m" || key == "j" || key == "c" || key == "k" {
					forward = false
				}
			}
		}
		if forward && (m.Focus == FocusViewport || m.Screen == ScreenHome) && !m.InputActive {
			var cmd tea.Cmd
			m.Viewport, cmd = m.Viewport.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m AppModel) updateAudit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyRunes {
		key := strings.ToLower(string(msg.Runes))
		if key == "q" {
			m.Quitting = true
			return m, tea.Quit
		}
		for i, item := range auditMenu {
			if item.Hotkey == key {
				m.AuditCursor = i
				m.Viewport.GotoTop()
				return m, nil
			}
		}
		return m.handleHotkey(key)
	}
	return m, nil
}

func (m AppModel) updateExports(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		if m.ExportCursor > 0 {
			m.ExportCursor--
			m.Viewport.GotoTop()
		}
	case tea.KeyDown:
		if m.ExportCursor < len(exportReports)-1 {
			m.ExportCursor++
			m.Viewport.GotoTop()
		}
	case tea.KeyEnter:
		return m.writeSelectedExport()
	case tea.KeyRunes:
		key := strings.ToLower(string(msg.Runes))
		switch key {
		case "q":
			m.Quitting = true
			return m, tea.Quit
		case "k":
			if m.ExportCursor > 0 {
				m.ExportCursor--
			}
		case "h":
			m.ExportFormat = model.ExportHTML
		case "m":
			m.ExportFormat = model.ExportMarkdown
		case "j":
			m.ExportFormat = model.ExportJSON
		case "c":
			m.ExportFormat = model.ExportCSV
		default:
			return m.handleHotkey(key)
		}
		m.ExportPath = ""
		m.ExportError = ""
		m.ExportMessage = ""
		m.Viewport.GotoTop()
	}
	return m, nil
}

func (m AppModel) writeSelectedExport() (tea.Model, tea.Cmd) {
	selected := exportReports[m.ExportCursor]
	result, err := export.WriteReport(m.Report, model.ExportRequest{
		ReportType: selected.ReportType,
		Format:     m.ExportFormat,
	}, "")
	if err != nil {
		m.ExportError = err.Error()
		m.ExportMessage = ""
		m.ExportPath = ""
		return m, nil
	}
	m.ExportPath = result.Path
	m.ExportBytes = result.Bytes
	m.ExportError = ""
	m.ExportMessage = fmt.Sprintf("Saved %s %s export (%d bytes)", result.ReportType, result.Format, result.Bytes)
	return m, nil
}

func (m AppModel) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.InputActive = false
		m.Screen = ScreenHome
	case tea.KeyBackspace, tea.KeyDelete:
		runes := []rune(m.InputValue)
		if len(runes) > 0 {
			m.InputValue = string(runes[:len(runes)-1])
		}
	case tea.KeyEnter:
		start := time.Now()
		report, err := scan.Workspace(m.InputValue, scan.Options{})
		m.Report = report
		m.LoadTime = time.Since(start)
		m.LoadError = err
		m.InputActive = false
		m.Screen = ScreenSummary
	case tea.KeyRunes:
		m.InputValue += string(msg.Runes)
	}
	return m, nil
}

func (m AppModel) handleHotkey(key string) (tea.Model, tea.Cmd) {
	key = strings.ToLower(key)
	if key == "q" {
		m.Quitting = true
		return m, tea.Quit
	}
	for i, item := range m.Menu {
		if item.Hotkey == key {
			m.Cursor = i
			return m.activate(item)
		}
	}
	return m, nil
}

func (m AppModel) activate(item MenuItem) (tea.Model, tea.Cmd) {
	if item.Hotkey == "q" {
		m.Quitting = true
		return m, tea.Quit
	}
	m.Screen = item.Screen
	m.InputActive = item.Screen == ScreenScanFolder
	if m.InputActive && m.InputValue == "" {
		m.InputValue = m.Report.RootPath
	}
	m.Viewport.GotoTop()
	return m, nil
}

func (m AppModel) View() string {
	if m.Quitting {
		return ""
	}

	leftWidth, rightWidth := calculateWidths(m)

	if m.Screen == ScreenHome {
		var b strings.Builder
		writeLogo(&b)
		fmt.Fprintln(&b)
		var menuBox strings.Builder
		writeMenu(&menuBox, m, 40)

		var nextActions strings.Builder
		writeNextActions(&nextActions)

		dashboard := lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().PaddingRight(5).Render(menuBox.String()),
			nextActions.String(),
		)

		fmt.Fprintln(&b, dashboard)
		fmt.Fprintln(&b)
		writeStatus(&b, m, false)

		placed := lipgloss.Place(
			viewWidth(m),
			viewHeight(m),
			lipgloss.Center,
			lipgloss.Center,
			strings.TrimRight(b.String(), "\n"),
		)
		return screenStyle.Width(viewWidth(m)).Height(viewHeight(m)).Render(placed)
	}

	var leftBuilder strings.Builder
	writeMenu(&leftBuilder, m, leftWidth)
	fmt.Fprintln(&leftBuilder)
	writeStatus(&leftBuilder, m, true)

	leftCol := lipgloss.NewStyle().
		Width(leftWidth).
		Render(leftBuilder.String())

	title := fmt.Sprintf(" Home > %s ", screenTitle(m.Screen))
	headerBg := lipgloss.Color("#444444")
	if m.Focus == FocusViewport {
		headerBg = lipgloss.Color("#00ADD8")
	}

	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1a1a1a")).
		Background(headerBg).
		Padding(0, 1).
		Bold(true).
		Render(title)

	headerBar := lipgloss.NewStyle().
		Width(rightWidth).
		Render(header)

	rightCol := lipgloss.JoinVertical(lipgloss.Left,
		headerBar,
		m.Viewport.View(),
	)

	borderColor := lipgloss.Color("#333333")
	if m.Focus == FocusViewport {
		borderColor = lipgloss.Color("#00ADD8")
	}

	rightColBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(rightWidth + 2).
		Height(m.Height - 4).
		Render(rightCol)

	layout := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, "  ", rightColBox)

	return lipgloss.NewStyle().Padding(1, 1).Render(layout)
}

func calculateWidths(m AppModel) (int, int) {
	leftWidth := m.Width / 4
	if leftWidth < 28 {
		leftWidth = 28
	}
	if leftWidth > 40 {
		leftWidth = 40
	}
	rightWidth := m.Width - leftWidth - 6
	if rightWidth < 10 {
		rightWidth = 10
	}
	return leftWidth, rightWidth
}

func screenTitle(screen Screen) string {
	switch screen {
	case ScreenScanFolder:
		return "Scan Folder"
	case ScreenOpenProject:
		return "Open Project"
	case ScreenAudit:
		return "Audit Projects"
	case ScreenPortfolio:
		return "Portfolio Score"
	case ScreenGit:
		return "Git Metadata"
	case ScreenExports:
		return "Exports"
	case ScreenSettings:
		return "Settings"
	case ScreenRecentScans:
		return "Recent Scans"
	case ScreenSummary:
		return "Workspace Summary"
	}
	return "Dashboard"
}

func writeLogo(b *strings.Builder) {
	projectLines := []string{
		"██████╗ ██████╗  ██████╗      ██╗███████╗ ██████╗████████╗",
		"██╔══██╗██╔══██╗██╔═══██╗     ██║██╔════╝██╔════╝╚══██╔══╝",
		"██████╔╝██████╔╝██║   ██║     ██║█████╗  ██║        ██║   ",
		"██╔═══╝ ██╔══██╗██║   ██║██   ██║██╔══╝  ██║        ██║   ",
		"██║     ██║  ██║╚██████╔╝╚█████╔╝███████╗╚██████╗   ██║   ",
		"╚═╝     ╚═╝  ╚═╝ ╚═════╝  ╚════╝ ╚══════╝ ╚═════╝   ╚═╝   ",
	}
	scanLines := []string{
		"███████╗ ██████╗ █████╗ ███╗   ██╗",
		"██╔════╝██╔════╝██╔══██╗████╗  ██║",
		"███████╗██║     ███████║██╔██╗ ██║",
		"╚════██║██║     ██╔══██║██║╚██╗██║",
		"███████║╚██████╗██║  ██║██║ ╚████║",
		"╚══════╝ ╚═════╝╚═╝  ╚═╝╚═╝  ╚═══╝",
	}

	for i := 0; i < len(projectLines); i++ {
		p := projectLines[i]
		s := scanLines[i]

		projectStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00ADD8")).Bold(true) // Go blue
		scanStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true)    // White

		line := projectStyle.Render(p) + "  " + scanStyle.Render(s)

		fmt.Fprintf(b, "%s\n", lipgloss.NewStyle().Width(dashboardWidth).Align(lipgloss.Center).Render(line))
	}
}

func writeMenu(b *strings.Builder, m AppModel, width int) {
	for i, item := range m.Menu {
		prefix := "  "
		itemStyle := lipgloss.NewStyle()
		isHome := m.Screen == ScreenHome

		if i == m.Cursor && (isHome || m.Focus == FocusSidebar) {
			prefix = menuCursorStyle.Render("▶ ")
			itemStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00ADD8"))
		}

		icon := menuIconStyle.Render(item.Icon)
		label := itemStyle.Render(item.Label)
		hotkey := menuHotkeyStyle.Render(item.Hotkey)
		if i == m.Cursor && (isHome || m.Focus == FocusSidebar) {
			hotkey = menuCursorStyle.Render(item.Hotkey)
		}

		if m.Screen == ScreenHome {
			spacing := 30 - displayWidth(item.Label)
			if spacing < 1 {
				spacing = 1
			}
			row := fmt.Sprintf("%s%s  %s%s %s", prefix, icon, label, repeatSpaces(spacing), hotkey)
			fmt.Fprintln(b, centerLine(row, width-70))
		} else {
			leftPart := fmt.Sprintf("  %s%s  %s", prefix, icon, label)
			rightPart := hotkey

			leftDisplayWidth := displayWidth(leftPart)
			rightDisplayWidth := displayWidth(rightPart)

			spacing := width - 2 - leftDisplayWidth - rightDisplayWidth
			if spacing < 1 {
				spacing = 1
			}

			row := leftPart + repeatSpaces(spacing) + rightPart
			fmt.Fprintln(b, row)
		}
	}
}

func writePanel(b *strings.Builder, m AppModel) {
	switch m.Screen {
	case ScreenHome:
		writeHomePanel(b, m)
	case ScreenScanFolder:
		fmt.Fprintln(b, panelStyle.Render("  SCAN FOLDER"))
		fmt.Fprintf(b, "  %s %s_\n", panelStyle.Render("Path:"), panelStyle.Render(m.InputValue))
		fmt.Fprintln(b, panelStyle.Render("  Enter scans this folder. Esc returns to menu."))
	case ScreenOpenProject:
		fmt.Fprintln(b, panelStyle.Render("  OPEN PROJECT"))
		for _, project := range model.SortedProjectsByName(m.Report.Projects) {
			fmt.Fprintf(b, "  %-24s %4d files  %s\n", project.Name, project.TotalFiles, strings.Join(project.MainLanguages, ", "))
		}
	case ScreenAudit:
		writeAuditPanel(b, m)
	case ScreenPortfolio:
		writePortfolioPanel(b, m)
	case ScreenGit:
		fmt.Fprintln(b, panelStyle.Render("  GIT METADATA"))
		var tableBuilder strings.Builder
		render.PrintGitAudit(&tableBuilder, m.Report.Projects, render.Style{Color: true})
		fmt.Fprint(b, indentBlock(tableBuilder.String(), "  "))
	case ScreenExports:
		writeExportsPanel(b, m)
	case ScreenSettings:
		fmt.Fprintln(b, panelStyle.Render("  SETTINGS"))
		fmt.Fprintln(b, panelStyle.Render("  CLI mode : projectscan <root> [--audit|--details|--json|--markdown]"))
		fmt.Fprintln(b, panelStyle.Render("  TUI mode : projectscan"))
		fmt.Fprintln(b, panelStyle.Render("  Color    : auto, disabled by NO_COLOR or --no-color"))
	case ScreenRecentScans:
		fmt.Fprintln(b, panelStyle.Render("  RECENT SCANS"))
		fmt.Fprintf(b, "  %s\n", model.DisplayRootPath(m.Report.RootPath))
	case ScreenSummary:
		fmt.Fprint(b, indentBlock(render.BuildSummaryWithStyle(m.Report.RootPath, m.Report.Projects, render.Style{Color: true}), "  "))
	}
}

func writeAuditPanel(b *strings.Builder, m AppModel) {
	var menuBuilder strings.Builder
	for i, item := range auditMenu {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
		if i == m.AuditCursor {
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ADD8")).Bold(true).Underline(true)
		}

		label := style.Render(item.Label)
		hotkey := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF956D")).Render(item.Hotkey)

		menuItem := fmt.Sprintf("%s %s", label, hotkey)

		if i > 0 {
			menuBuilder.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Render(" | "))
		}
		menuBuilder.WriteString(menuItem)
	}

	menuRendered := lipgloss.NewStyle().Width(m.Viewport.Width - 4).Render(menuBuilder.String())
	fmt.Fprintln(b, "  "+menuRendered)
	fmt.Fprintln(b)

	sepWidth := m.Viewport.Width - 4 - 12
	if sepWidth < 1 {
		sepWidth = 1
	}
	separator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00ADD8")).
		Render(strings.Repeat("─", 4) + " OUTPUT " + strings.Repeat("─", sepWidth))
	fmt.Fprintln(b, "  "+separator)
	fmt.Fprintln(b)

	selected := auditMenu[m.AuditCursor]
	result, err := audit.Build(m.Report, selected.ReportType)
	if err != nil {
		fmt.Fprintf(b, "  %s\n", err)
		return
	}
	fmt.Fprintf(b, "  %s\n", panelStyle.Render(strings.ToUpper(result.Title)))
	for _, module := range result.Modules {
		writeAuditModulePreview(b, module)
	}
}

func writeAuditModulePreview(b *strings.Builder, module audit.ModuleResult) {
	if module.Message != "" {
		fmt.Fprintf(b, "  %s\n", module.Message)
		return
	}
	if len(module.Columns) == 0 {
		return
	}

	title := module.Title
	if title == "" {
		title = "RESULTS"
	}

	limit := len(module.Rows)
	if limit > 8 {
		limit = 8
	}
	for i := 0; i < limit; i++ {
		row := module.Rows[i]

		cardTitle := title
		if len(module.Columns) > 0 {
			if firstColVal := row[module.Columns[0]]; firstColVal != "" {
				cardTitle = firstColVal
			}
		}

		var cardContent strings.Builder
		for j, column := range module.Columns {
			if j == 0 {
				continue
			}
			val := row[column]
			fmt.Fprintf(&cardContent, "%-20s : %s\n", strings.ToUpper(column), val)
		}

		cardBody := lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Foreground(lipgloss.Color("#00ADD8")).Bold(true).Render(cardTitle),
			"",
			strings.TrimRight(cardContent.String(), "\n"),
		)

		card := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#00ADD8")).
			Padding(0, 1).
			MarginLeft(2).
			Render(cardBody)

		fmt.Fprintln(b, card)
		fmt.Fprintln(b)
	}

	if len(module.Rows) > limit {
		fmt.Fprintf(b, "  ... %d more rows\n", len(module.Rows)-limit)
	}
}

func writeHomePanel(b *strings.Builder, m AppModel) {
	if m.LoadError != nil {
		fmt.Fprintf(b, "  Scan error: %v\n", m.LoadError)
		return
	}
	fmt.Fprintf(b, "  Root      %s\n", model.DisplayRootPath(m.Report.RootPath))
	fmt.Fprintf(b, "  Projects  %d\n", len(m.Report.Projects))
	fmt.Fprintf(b, "  Files     %d\n", m.Report.TotalFiles)
	fmt.Fprintf(b, "  Languages %d\n", len(m.Report.LanguageSummary))
}

func writePortfolioPanel(b *strings.Builder, m AppModel) {
	var tableBuilder strings.Builder
	render.PrintPortfolioSummary(&tableBuilder, m.Report, render.Style{Color: true})
	fmt.Fprint(b, indentBlock(tableBuilder.String(), "  "))
}

func writeNextActions(b *strings.Builder) {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#333333")).
		Padding(0, 1).
		Width(30)

	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true).
		Render(" NEXT ACTIONS ")

	actions := []struct {
		key  string
		desc string
	}{
		{"a", "Run Audit"},
		{"p", "View Portfolio Score"},
		{"e", "Export Reports"},
		{"?", "Help"},
	}

	var content strings.Builder
	fmt.Fprintln(&content, title)
	fmt.Fprintln(&content)
	for _, action := range actions {
		fmt.Fprintf(&content, " %s  %s\n",
			menuHotkeyStyle.Render(action.key),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#CCCCCC")).Render(action.desc),
		)
	}

	fmt.Fprint(b, style.Render(content.String()))
}

func writeExportsPanel(b *strings.Builder, m AppModel) {
	fmt.Fprintln(b, "  EXPORT REPORTS")
	fmt.Fprintln(b)
	fmt.Fprintln(b, "  REPORT")
	for i, item := range exportReports {
		prefix := "  "
		if i == m.ExportCursor {
			prefix = menuCursorStyle.Render("▶ ")
		}
		fmt.Fprintf(b, "%s%-28s %s\n", prefix, item.Label, panelStyle.Render(item.ReportType))
	}
	fmt.Fprintln(b)
	fmt.Fprintln(b, "  FORMAT")
	for _, item := range []struct {
		key    string
		format string
		label  string
	}{
		{"h", model.ExportHTML, "HTML dossier"},
		{"m", model.ExportMarkdown, "Markdown"},
		{"j", model.ExportJSON, "JSON"},
		{"c", model.ExportCSV, "CSV"},
	} {
		marker := " "
		if m.ExportFormat == item.format {
			marker = "●"
		}
		fmt.Fprintf(b, "  %s %s  %-14s %s\n", marker, menuHotkeyStyle.Render(item.key), item.format, item.label)
	}
	fmt.Fprintln(b)
	selected := exportReports[m.ExportCursor]
	fmt.Fprintf(b, "  Command: projectscan export %s --report %s --format %s\n", shellPath(m.Report.RootPath), selected.ReportType, m.ExportFormat)
	fmt.Fprintln(b, "  Enter: write export to projectscan-exports/")
	if m.ExportMessage != "" {
		fmt.Fprintf(b, "\n  Saved: %s\n", m.ExportPath)
		fmt.Fprintf(b, "  %s\n", m.ExportMessage)
	}
	if m.ExportError != "" {
		fmt.Fprintf(b, "\n  Error: %s\n", m.ExportError)
	}
}

func writeStatus(b *strings.Builder, m AppModel, compact bool) {
	if m.LoadError != nil {
		if compact {
			fmt.Fprintf(b, "  %s %s\n", statusPulseStyle.Render("⌁"), statusTextStyle.Render("Failed"))
		} else {
			fmt.Fprintf(b, "  %s %s\n", statusPulseStyle.Render("⌁"), statusTextStyle.Render(fmt.Sprintf("ProjectScan failed: %v", m.LoadError)))
		}
		return
	}

	count := len(m.Report.Projects)
	projectLabel := "projects"
	if count == 1 {
		projectLabel = "project"
	}

	if compact {
		fmt.Fprintf(b, "  %s  %s %s / %s\n",
			statusPulseStyle.Render("⚡"),
			statusValueStyle.Render(fmt.Sprintf("%d %s", count, projectLabel)),
			statusTextStyle.Render("loaded"),
			statusValueStyle.Render(fmt.Sprintf("%.1fms", float64(m.LoadTime.Microseconds())/1000)),
		)
	} else {
		fmt.Fprintf(b, "  %s  %s %s %s %s\n",
			statusPulseStyle.Render("⚡"),
			statusTextStyle.Render("ProjectScan loaded"),
			statusValueStyle.Render(fmt.Sprintf("%d %s", count, projectLabel)),
			statusTextStyle.Render("in"),
			statusValueStyle.Render(fmt.Sprintf("%.1fms", float64(m.LoadTime.Microseconds())/1000)),
		)
	}
}

func centerLine(text string, width int) string {
	visible := lipgloss.Width(text)
	if visible >= width {
		return text
	}
	left := (width - visible) / 2
	return repeatSpaces(left) + text
}

func repeatSpaces(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.Repeat(" ", count)
}

func displayWidth(text string) int {
	return lipgloss.Width(text)
}

func viewWidth(m AppModel) int {
	if m.Width > 0 {
		if m.Width < dashboardWidth+4 {
			return dashboardWidth + 4
		}
		return m.Width
	}
	return 132
}

func viewHeight(m AppModel) int {
	if m.Height > 0 {
		return m.Height
	}
	return 34
}

func mustExport(report model.WorkspaceReport, reportType string, format string) string {
	output, err := export.BuildReport(report, model.ExportRequest{ReportType: reportType, Format: format})
	if err != nil {
		return err.Error()
	}
	return output
}

func indentBlock(text string, indent string) string {
	var b strings.Builder
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		fmt.Fprintf(&b, "%s%s\n", indent, line)
	}
	return b.String()
}

func shellPath(path string) string {
	if strings.ContainsAny(path, " \t\n'\"") {
		return fmt.Sprintf("%q", path)
	}
	return path
}
