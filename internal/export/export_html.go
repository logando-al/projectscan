package export

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	"projectscan/internal/audit"
	"projectscan/internal/model"
)

func buildHTML(result audit.Result) string {
	var b strings.Builder
	fmt.Fprintln(&b, "<!doctype html>")
	fmt.Fprintln(&b, `<html lang="en">`)
	fmt.Fprintln(&b, `<head>`)
	fmt.Fprintln(&b, `<meta charset="utf-8">`)
	fmt.Fprintln(&b, `<meta name="viewport" content="width=device-width, initial-scale=1">`)
	fmt.Fprintf(&b, "<title>Project Scan %s</title>\n", html.EscapeString(result.Title))
	fmt.Fprintln(&b, `<script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.7/dist/chart.umd.min.js"></script>`)
	fmt.Fprintln(&b, `<style>`)
	fmt.Fprintln(&b, htmlCSS())
	fmt.Fprintln(&b, `</style>`)
	fmt.Fprintln(&b, `</head>`)
	fmt.Fprintln(&b, `<body>`)
	fmt.Fprintln(&b, `<div class="shell">`)
	writeHTMLNav(&b, result)
	fmt.Fprintln(&b, `<main>`)
	writeHTMLHero(&b, result)
	writeHTMLPortfolio(&b, result)
	if result.ReportType == model.ReportAudit {
		writeHTMLProjectInventory(&b, result)
		writeHTMLAuditModules(&b, result)
		writeHTMLSafetyFindings(&b, roadmapModule(result, model.ReportSafety))
		writeHTMLLOC(&b, roadmapModule(result, model.ReportLOC))
		writeHTMLExportMap(&b)
		writeHTMLAppendix(&b, result)
	} else {
		writeHTMLModules(&b, result)
	}
	fmt.Fprintln(&b, `<p class="footer">ProjectScan HTML Export · Open Source · MIT License</p>`)
	fmt.Fprintln(&b, `</main></div>`)
	writeChartScript(&b, result)
	fmt.Fprintln(&b, `</body></html>`)
	return b.String()
}

func writeHTMLNav(b *strings.Builder, result audit.Result) {
	fmt.Fprintln(b, `<nav class="toc" aria-label="Report sections">`)
	fmt.Fprintln(b, `<div class="toc-kicker">ProjectScan</div>`)
	fmt.Fprintln(b, `<a href="#snapshot">Snapshot</a>`)
	fmt.Fprintln(b, `<a href="#portfolio">Portfolio</a>`)
	if result.ReportType == model.ReportAudit {
		fmt.Fprintln(b, `<a href="#projects">Projects</a>`)
		fmt.Fprintln(b, `<a href="#audit">Audit Modules</a>`)
		fmt.Fprintln(b, `<a href="#safety">Safety</a>`)
		fmt.Fprintln(b, `<a href="#loc">Lines of Code</a>`)
		fmt.Fprintln(b, `<a href="#export-map">Export Map</a>`)
		fmt.Fprintln(b, `<a href="#appendix">Appendix</a>`)
		fmt.Fprintln(b, `</nav>`)
		return
	}
	for i, module := range result.Modules {
		fmt.Fprintf(b, `<a href="#module-%d">%s</a>`+"\n", i, html.EscapeString(module.Title))
	}
	fmt.Fprintln(b, `</nav>`)
}

func writeHTMLHero(b *strings.Builder, result audit.Result) {
	fmt.Fprintln(b, `<header class="hero" id="snapshot">`)
	fmt.Fprintln(b, `<div class="hero-top"><div>`)
	fmt.Fprintf(b, `<div class="eyebrow">%s export</div>`+"\n", html.EscapeString(result.ReportType))
	fmt.Fprintf(b, `<h1 class="hero-title"><span class="hero-title-project">PROJECT</span> <span>SCAN</span> <span>%s</span></h1>`+"\n", html.EscapeString(strings.ToUpper(result.Title)))
	fmt.Fprintln(b, `<p class="hero-subtitle">A local-first portfolio audit dossier generated from ProjectScan scan data.</p>`)
	fmt.Fprintln(b, `</div></div>`)
	fmt.Fprintln(b, `<div class="hero-meta">`)
	fmt.Fprintf(b, `<div class="meta-cell"><div class="meta-label">Root Path</div><span class="meta-value">%s</span></div>`+"\n", html.EscapeString(result.RootPath))
	fmt.Fprintf(b, `<div class="meta-cell"><div class="meta-label">Report</div><span class="meta-value">%s</span></div>`+"\n", html.EscapeString(result.ReportType))
	fmt.Fprintf(b, `<div class="meta-cell"><div class="meta-label">Projects</div><span class="meta-value">%d</span></div>`+"\n", result.ProjectCount)
	fmt.Fprintf(b, `<div class="meta-cell"><div class="meta-label">Files</div><span class="meta-value">%d</span></div>`+"\n", result.TotalFiles)
	fmt.Fprintln(b, `</div><div class="kpis">`)
	fmt.Fprintf(b, `<div class="kpi"><strong>%d</strong><span>Projects</span></div>`+"\n", result.ProjectCount)
	fmt.Fprintf(b, `<div class="kpi"><strong>%d</strong><span>Files</span></div>`+"\n", result.TotalFiles)
	fmt.Fprintf(b, `<div class="kpi"><strong>%d</strong><span>Languages</span></div>`+"\n", len(result.LanguageSummary))
	fmt.Fprintf(b, `<div class="kpi"><strong>%d</strong><span>Production Ready</span></div>`+"\n", result.LabelCounts[model.LabelProductionReady])
	fmt.Fprintf(b, `<div class="kpi"><strong>%d</strong><span>Experiments</span></div>`+"\n", result.LabelCounts[model.LabelExperiment])
	fmt.Fprintln(b, `</div></header>`)
}

func writeHTMLPortfolio(b *strings.Builder, result audit.Result) {
	fmt.Fprintln(b, `<section id="portfolio"><div class="section-head"><h2>Portfolio Overview</h2><div class="section-note">language share + labels</div></div><div class="section-body grid-2">`)
	fmt.Fprintln(b, `<div class="panel"><div class="panel-title">Language Share</div><div class="chart-wrap"><canvas id="languageShareChart"></canvas></div>`)
	for _, item := range sortedLanguageSummary(result.LanguageSummary, 8) {
		percent := model.PercentOfTotal(item.Count, result.TotalFiles)
		fmt.Fprintf(b, `<div class="bar-row"><div class="bar-label">%s</div><div class="bar-track"><div class="bar-fill" style="width:%d%%"></div></div><div class="bar-value">%d%%</div></div>`+"\n", html.EscapeString(item.Lang), percent, percent)
	}
	fmt.Fprintln(b, `</div>`)
	fmt.Fprintln(b, `<div class="panel"><div class="panel-title">Portfolio Labels</div><div class="chart-wrap compact"><canvas id="portfolioLabelChart"></canvas></div><div class="panel-title">Portfolio Score</div><div class="score-chart-wrap"><canvas id="portfolioScoreChart"></canvas></div><div class="callout">Recommendation: promote projects with README, license, tests, clean git, and deployment hints into production-ready candidates.</div></div>`)
	fmt.Fprintln(b, `</div></section>`)
}

func writeHTMLProjectInventory(b *strings.Builder, result audit.Result) {
	fmt.Fprintln(b, `<section id="projects">`)
	fmt.Fprintln(b, `<div class="section-head"><h2>Project Inventory</h2><div class="section-note">SQL-inspired table for dense comparison</div></div>`)
	fmt.Fprintln(b, `<div class="section-body"><div class="table-wrap"><table><thead><tr>`)
	headers := []string{"Project", "Files", "Main Languages", "Label", "Git", "Readiness", "Path"}
	for _, header := range headers {
		class := ""
		if header == "Files" || header == "Readiness" {
			class = ` class="num"`
		}
		fmt.Fprintf(b, `<th%s>%s</th>`, class, header)
	}
	fmt.Fprintln(b, `</tr></thead><tbody>`)
	for _, project := range model.SortedProjectsByName(result.Projects) {
		fmt.Fprintln(b, `<tr>`)
		fmt.Fprintf(b, `<td>%s</td>`, html.EscapeString(project.Name))
		fmt.Fprintf(b, `<td class="num">%d</td>`, project.TotalFiles)
		fmt.Fprintf(b, `<td>%s</td>`, html.EscapeString(strings.Join(mainLanguages(project), ", ")))
		writeHTMLBadgeCell(b, project.Label, labelBadgeClass(project.Label))
		writeHTMLBadgeCell(b, gitDisplay(project), gitBadgeClass(project))
		fmt.Fprintf(b, `<td class="num">%d</td>`, project.Readiness.Score)
		fmt.Fprintf(b, `<td class="path">%s</td>`, html.EscapeString(project.Path))
		fmt.Fprintln(b, `</tr>`)
	}
	fmt.Fprintln(b, `</tbody></table></div></div></section>`)
}

func writeHTMLAuditModules(b *strings.Builder, result audit.Result) {
	safety := roadmapModule(result, model.ReportSafety)
	readme := roadmapModule(result, model.ReportReadme)
	loc := roadmapModule(result, model.ReportLOC)
	gitHygiene := roadmapModule(result, model.ReportGitHygiene)
	deps := roadmapModule(result, model.ReportDeps)
	readiness := findModule(result.Modules, "readiness")

	cards := []struct {
		title       string
		body        string
		metricLabel string
		metricValue string
	}{
		{"Open Source Safety", "Finds risky files, local machine references, and secret-like assignments without printing secret values.", "findings", safetyMetric(safety)},
		{"README Quality", "Scores documentation against title, description, install, usage, configuration, and license sections.", "avg score", averageModuleScore(readme, "Score")},
		{"Lines of Code", "Counts total, code, blank, and comment lines for recognized source files.", "largest", largestLOCFile(loc)},
		{"Git Hygiene", "Checks repo, branch, remote, dirty state, commit age, and history signals.", "score", averageModuleScore(gitHygiene, "Score")},
		{"Dependency Inventory", "Lists dependency manifests such as go.mod, package.json, Cargo.toml, and lockfiles.", "manifests", sumModuleColumn(deps, "Files")},
		{"Open Source Readiness", "Combines safety, README, license, git hygiene, dependency, and completeness signals.", "decision", readinessDecision(result)},
		{"External Tools", "Reserved for later opt-in integration with scanners such as cloc, trivy, gitleaks, or semgrep.", "status", "planned"},
		{"Completeness", "Base readiness module for README, tests, license, container, CI, deploy, and remote checks.", "base score", averageModuleScore(readiness, "Score")},
	}

	fmt.Fprintln(b, `<section id="audit">`)
	fmt.Fprintln(b, `<div class="section-head"><h2>Audit Modules</h2><div class="section-note">roadmap-aligned export sections</div></div>`)
	fmt.Fprintln(b, `<div class="section-body"><div class="module-grid">`)
	for _, card := range cards {
		fmt.Fprintln(b, `<article class="module-card">`)
		fmt.Fprintf(b, `<h3>%s</h3>`, html.EscapeString(card.title))
		fmt.Fprintf(b, `<p>%s</p>`, html.EscapeString(card.body))
		fmt.Fprintf(b, `<div class="metric-line"><span>%s</span><strong>%s</strong></div>`, html.EscapeString(card.metricLabel), html.EscapeString(card.metricValue))
		fmt.Fprintln(b, `</article>`)
	}
	fmt.Fprintln(b, `</div></div></section>`)
}

func writeHTMLSafetyFindings(b *strings.Builder, module audit.ModuleResult) {
	fmt.Fprintln(b, `<section id="safety">`)
	fmt.Fprintln(b, `<div class="section-head"><h2>Safety Findings</h2><div class="section-note">no secret values, only location and risk type</div></div>`)
	fmt.Fprintln(b, `<div class="section-body"><div class="table-wrap"><table><thead><tr>`)
	for _, header := range []string{"Project", "File", "Line", "Risk", "Severity", "Suggested Fix"} {
		class := ""
		if header == "Line" {
			class = ` class="num"`
		}
		fmt.Fprintf(b, `<th%s>%s</th>`, class, header)
	}
	fmt.Fprintln(b, `</tr></thead><tbody>`)
	for _, row := range module.Rows {
		severity := row["Severity"]
		fmt.Fprintln(b, `<tr>`)
		fmt.Fprintf(b, `<td>%s</td>`, html.EscapeString(row["Project"]))
		fmt.Fprintf(b, `<td class="path">%s</td>`, html.EscapeString(row["File"]))
		fmt.Fprintf(b, `<td class="num">%s</td>`, html.EscapeString(row["Line"]))
		fmt.Fprintf(b, `<td>%s</td>`, html.EscapeString(row["Risk"]))
		writeHTMLBadgeCell(b, strings.ToLower(severity), severityBadgeClass(severity))
		fmt.Fprintf(b, `<td>%s</td>`, html.EscapeString(safetySuggestion(row["Risk"])))
		fmt.Fprintln(b, `</tr>`)
	}
	fmt.Fprintln(b, `</tbody></table></div></div></section>`)
}

func writeHTMLLOC(b *strings.Builder, module audit.ModuleResult) {
	fmt.Fprintln(b, `<section id="loc">`)
	fmt.Fprintln(b, `<div class="section-head"><h2>Lines of Code</h2><div class="section-note">spreadsheet-friendly values with visual scanability</div></div>`)
	fmt.Fprintln(b, `<div class="section-body">`)
	writeHTMLTable(b, module)
	fmt.Fprintln(b, `</div></section>`)
}

func writeHTMLExportMap(b *strings.Builder) {
	rows := [][]string{
		{"terminal", "developer", "fast local scan", "SQL-style tables, no browser-only visual elements"},
		{"html", "portfolio / client", "polished audit dossier", "single file, embedded CSS, Chart.js, no ANSI"},
		{"markdown", "documentation", "README, notes, GitHub issue paste", "plain tables, stable headings"},
		{"json", "automation", "dashboards, scripts, later UI", "full structured result, no presentation data"},
		{"csv", "spreadsheet", "comparisons and sorting", "one module table per export where practical"},
	}
	fmt.Fprintln(b, `<section id="export-map">`)
	fmt.Fprintln(b, `<div class="section-head"><h2>Export Map</h2><div class="section-note">same audit data, different renderers</div></div>`)
	fmt.Fprintln(b, `<div class="section-body"><div class="table-wrap"><table><thead><tr>`)
	for _, header := range []string{"Format", "Audience", "Primary Use", "Renderer Rule"} {
		fmt.Fprintf(b, `<th>%s</th>`, header)
	}
	fmt.Fprintln(b, `</tr></thead><tbody>`)
	for _, row := range rows {
		fmt.Fprintln(b, `<tr>`)
		for _, cell := range row {
			fmt.Fprintf(b, `<td>%s</td>`, html.EscapeString(cell))
		}
		fmt.Fprintln(b, `</tr>`)
	}
	fmt.Fprintln(b, `</tbody></table></div></div></section>`)
}

func writeHTMLAppendix(b *strings.Builder, result audit.Result) {
	fmt.Fprintln(b, `<section id="appendix">`)
	fmt.Fprintln(b, `<div class="section-head"><h2>Appendix Tables</h2><div class="section-note">base audit modules</div></div>`)
	fmt.Fprintln(b, `<div class="section-body appendix-stack">`)
	for _, module := range result.Modules {
		fmt.Fprintf(b, `<div class="appendix-module"><div class="panel-title">%s</div>`, html.EscapeString(module.Title))
		if module.Message != "" {
			fmt.Fprintf(b, `<div class="callout">%s</div>`, html.EscapeString(module.Message))
		} else {
			writeHTMLTable(b, module)
		}
		fmt.Fprintln(b, `</div>`)
	}
	fmt.Fprintln(b, `</div></section>`)
}

func writeHTMLModules(b *strings.Builder, result audit.Result) {
	for i, module := range result.Modules {
		fmt.Fprintf(b, `<section id="module-%d">`, i)
		fmt.Fprintf(b, `<div class="section-head"><h2>%s</h2><div class="section-note">%d rows</div></div>`, html.EscapeString(module.Title), len(module.Rows))
		fmt.Fprintln(b, `<div class="section-body">`)
		if module.Message != "" {
			fmt.Fprintf(b, `<div class="callout">%s</div>`, html.EscapeString(module.Message))
		} else {
			writeHTMLTable(b, module)
		}
		fmt.Fprintln(b, `</div></section>`)
	}
}

func writeHTMLTable(b *strings.Builder, module audit.ModuleResult) {
	if len(module.Columns) == 0 {
		return
	}
	fmt.Fprintln(b, `<div class="table-wrap"><table><thead><tr>`)
	for _, column := range module.Columns {
		class := ""
		if numericColumn(column) {
			class = ` class="num"`
		}
		fmt.Fprintf(b, `<th%s>%s</th>`, class, html.EscapeString(column))
	}
	fmt.Fprintln(b, `</tr></thead><tbody>`)
	for _, row := range module.Rows {
		fmt.Fprintln(b, `<tr>`)
		for _, column := range module.Columns {
			class := ""
			if numericColumn(column) {
				class = ` class="num"`
			} else if column == "File" || column == "Largest File" || column == "Path" || column == "Remote" {
				class = ` class="path"`
			}
			fmt.Fprintf(b, `<td%s>%s</td>`, class, html.EscapeString(row[column]))
		}
		fmt.Fprintln(b, `</tr>`)
	}
	fmt.Fprintln(b, `</tbody></table></div>`)
}

func sortedLanguageSummary(languages map[string]int, limit int) []model.LangCount {
	items := model.SortLanguageCounts(languages)
	if len(items) > limit {
		return items[:limit]
	}
	return items
}

func htmlCSS() string {
	return `:root{--bg:#080d0e;--bg-grid:rgba(0,173,216,.045);--panel:#101718;--panel-2:#131d1e;--panel-3:#182223;--line:rgba(0,173,216,.28);--line-strong:rgba(0,173,216,.62);--text:#e8f2ec;--muted:#95aaa3;--dim:#667871;--go-blue:#00ADD8;--go-blue-bright:#8ee9ff;--go-blue-soft:rgba(0,173,216,.13);--green:#b9d86b;--green-soft:rgba(185,216,107,.12);--cyan:#8bd8d3;--cyan-soft:rgba(139,216,211,.12);--amber:#ffd36a;--amber-soft:rgba(255,211,106,.12);--red:#ff7474;--red-soft:rgba(255,116,116,.12);--shadow:rgba(0,0,0,.34);--mono:ui-monospace,"SFMono-Regular","SF Mono",Menlo,Consolas,monospace;--sans:"Avenir Next","Segoe UI",system-ui,sans-serif}*{box-sizing:border-box}body{margin:0;min-height:100vh;background:linear-gradient(var(--bg-grid) 1px,transparent 1px),linear-gradient(90deg,var(--bg-grid) 1px,transparent 1px),linear-gradient(180deg,#070b0c 0%,#0a1011 48%,#070b0c 100%);background-size:32px 32px,32px 32px,auto;color:var(--text);font-family:var(--sans);line-height:1.55}.shell{width:min(1440px,calc(100% - 48px));margin:0 auto;display:grid;grid-template-columns:230px minmax(0,1fr);gap:32px;padding:32px 0 56px}.toc{position:sticky;top:24px;align-self:start;max-height:calc(100dvh - 48px);overflow:auto;border:1px solid var(--line);background:rgba(16,23,24,.92);box-shadow:0 18px 40px var(--shadow);padding:14px;border-radius:8px}.toc-kicker{color:var(--go-blue);font-family:var(--mono);font-size:11px;letter-spacing:.18em;text-transform:uppercase;padding:6px 8px 12px;border-bottom:1px solid var(--line);margin-bottom:10px}.toc a{display:block;text-decoration:none;color:var(--muted);font-family:var(--mono);font-size:12px;padding:7px 8px;border-left:2px solid transparent;border-radius:5px}.toc a:hover{color:var(--text);background:var(--panel-3);border-left-color:var(--go-blue)}main{min-width:0}.hero,section{border:1px solid var(--line);background:rgba(16,23,24,.94);border-radius:8px;overflow:hidden;box-shadow:0 22px 54px var(--shadow)}section{margin-top:26px;box-shadow:none}.hero{border-color:var(--line-strong);background:linear-gradient(135deg,var(--go-blue-soft),transparent 42%),linear-gradient(180deg,var(--panel) 0%,#0c1213 100%)}.hero-top{display:flex;justify-content:space-between;gap:20px;align-items:flex-start;padding:28px 30px 24px;border-bottom:1px solid var(--line)}.eyebrow{color:var(--go-blue-bright);font-family:var(--mono);font-size:12px;letter-spacing:.18em;text-transform:uppercase;margin-bottom:10px}h1{margin:0;font-family:var(--mono);font-size:clamp(34px,6vw,74px);line-height:.98;text-transform:uppercase}.hero-title-project{color:var(--go-blue)}.hero-subtitle{color:var(--muted);max-width:760px;margin:16px 0 0;font-size:15px}.hero-meta{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));border-bottom:1px solid var(--line)}.meta-cell{padding:18px 22px;border-right:1px solid var(--line)}.meta-cell:last-child{border-right:0}.meta-label{color:var(--dim);font-family:var(--mono);font-size:11px;text-transform:uppercase;letter-spacing:.12em}.meta-value{display:block;color:var(--text);font-family:var(--mono);font-size:15px;margin-top:5px;overflow-wrap:anywhere}.kpis{display:grid;grid-template-columns:repeat(5,minmax(0,1fr));gap:1px;background:var(--line)}.kpi{background:rgba(14,21,22,.98);padding:22px;min-height:124px}.kpi strong{display:block;color:var(--amber);font-family:var(--mono);font-size:30px;line-height:1;margin-bottom:12px}.kpi span{color:var(--muted);font-family:var(--mono);font-size:12px;text-transform:uppercase;letter-spacing:.12em}.section-head{display:flex;justify-content:space-between;gap:16px;align-items:center;background:var(--panel-2);border-bottom:1px solid var(--line);padding:16px 20px}h2{margin:0;color:var(--go-blue);font-family:var(--mono);font-size:15px;letter-spacing:.14em;text-transform:uppercase}.section-note{color:var(--dim);font-family:var(--mono);font-size:12px;text-align:right}.section-body{padding:20px}.grid-2{display:grid;grid-template-columns:minmax(0,1fr) minmax(300px,.62fr);gap:20px}.panel{border:1px solid var(--line);background:var(--panel);border-radius:8px;padding:18px}.panel-title{color:var(--cyan);font-family:var(--mono);font-size:12px;letter-spacing:.14em;text-transform:uppercase;margin-bottom:16px}.chart-wrap,.score-chart-wrap{height:260px;border:1px solid var(--line);background:linear-gradient(180deg,rgba(0,173,216,.055),transparent),#0b1112;border-radius:8px;padding:14px;margin-bottom:16px}.chart-wrap.compact{height:238px}.chart-wrap canvas,.score-chart-wrap canvas{width:100%!important;height:100%!important}.bar-row{display:grid;grid-template-columns:148px minmax(140px,1fr) 54px;gap:14px;align-items:center;margin:12px 0;font-family:var(--mono);font-size:13px}.bar-label{color:var(--text);overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.bar-track{height:14px;border:1px solid rgba(0,173,216,.28);background:#0b1112;overflow:hidden}.bar-fill{height:100%;background:linear-gradient(90deg,var(--go-blue),var(--cyan))}.bar-value{color:var(--amber);text-align:right}.table-wrap{overflow:auto;border:1px solid var(--line);background:#0b1112;border-radius:8px}table{width:100%;min-width:760px;border-collapse:collapse;font-family:var(--mono);font-size:12px;line-height:1.45}th{position:sticky;top:0;z-index:2;color:var(--cyan);background:#11191a;border-bottom:1px solid var(--line-strong);text-align:left;font-weight:600;letter-spacing:.08em;text-transform:uppercase;padding:12px 14px;white-space:nowrap}td{color:var(--text);border-bottom:1px solid rgba(0,173,216,.16);padding:12px 14px;vertical-align:top}tbody tr:nth-child(even) td{background:rgba(139,216,211,.035)}tbody tr:hover td{background:rgba(0,173,216,.075)}.num{text-align:right}.path{color:var(--muted);overflow-wrap:anywhere}.badge{display:inline-flex;align-items:center;gap:6px;border:1px solid currentColor;border-radius:999px;padding:3px 8px;font-family:var(--mono);font-size:11px;white-space:nowrap}.badge:before{content:"";width:6px;height:6px;border-radius:50%;background:currentColor}.badge.good{color:var(--green);background:var(--green-soft)}.badge.warn{color:var(--amber);background:var(--amber-soft)}.badge.bad{color:var(--red);background:var(--red-soft)}.badge.info{color:var(--cyan);background:var(--cyan-soft)}.module-grid{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:14px}.module-card{border:1px solid var(--line);background:var(--panel);border-radius:8px;padding:16px;min-height:154px}.module-card h3{margin:0 0 12px;color:var(--text);font-family:var(--mono);font-size:13px;letter-spacing:0}.module-card p{margin:0 0 16px;color:var(--muted);font-size:13px}.metric-line{display:flex;justify-content:space-between;gap:12px;color:var(--dim);font-family:var(--mono);font-size:11px;border-top:1px solid rgba(0,173,216,.14);padding-top:10px}.metric-line strong{color:var(--amber);font-size:12px}.appendix-stack{display:grid;gap:18px}.appendix-module{border:1px solid var(--line);background:rgba(11,17,18,.72);border-radius:8px;padding:16px}.callout{border-left:3px solid var(--amber);background:var(--amber-soft);padding:14px 16px;color:var(--text);font-family:var(--mono);font-size:12px;border-radius:0 6px 6px 0;margin-top:18px}.footer{color:var(--dim);font-family:var(--mono);font-size:12px;text-align:center;margin:30px 0 0}@media(max-width:1180px){.module-grid{grid-template-columns:repeat(2,minmax(0,1fr))}}@media(max-width:1080px){.shell{width:min(100% - 28px,980px);display:block}.toc{position:sticky;top:0;z-index:20;display:flex;gap:4px;overflow-x:auto;margin-bottom:18px}.toc-kicker{display:none}.toc a{white-space:nowrap;border-left:0;border-bottom:2px solid transparent}.hero-top,.section-head{display:block}.hero-meta,.kpis,.grid-2{grid-template-columns:1fr 1fr}}@media(max-width:720px){.hero-meta,.kpis,.grid-2,.module-grid{grid-template-columns:1fr}.meta-cell{border-right:0;border-bottom:1px solid var(--line)}.bar-row{grid-template-columns:1fr;gap:7px}.bar-value{text-align:left}}`
}

func writeChartScript(b *strings.Builder, result audit.Result) {
	langs := sortedLanguageSummary(result.LanguageSummary, 8)
	labels := make([]string, 0, len(langs))
	values := make([]string, 0, len(langs))
	for _, item := range langs {
		labels = append(labels, jsString(item.Lang))
		values = append(values, fmt.Sprintf("%d", model.PercentOfTotal(item.Count, result.TotalFiles)))
	}
	labelCounts := result.LabelCounts
	fmt.Fprintln(b, `<script>`)
	fmt.Fprintln(b, `(()=>{if(!window.Chart)return;const root=getComputedStyle(document.documentElement);const c=n=>root.getPropertyValue(n).trim();const text=c('--text'),muted=c('--muted'),goBlue=c('--go-blue'),cyan=c('--cyan'),amber=c('--amber'),red=c('--red'),green=c('--green'),line='rgba(0,173,216,.18)';Chart.defaults.color=muted;Chart.defaults.font.family=root.getPropertyValue('--mono').trim();`)
	fmt.Fprintf(b, `const languageLabels=[%s];const languageValues=[%s];`, strings.Join(labels, ","), strings.Join(values, ","))
	fmt.Fprintln(b, `const langCanvas=document.getElementById('languageShareChart');if(langCanvas)new Chart(langCanvas,{type:'bar',data:{labels:languageLabels,datasets:[{label:'Share %',data:languageValues,borderColor:goBlue,backgroundColor:'rgba(0,173,216,.72)',borderWidth:1,borderRadius:2,barThickness:18}]},options:{responsive:true,maintainAspectRatio:false,indexAxis:'y',plugins:{legend:{display:false}},scales:{x:{beginAtZero:true,max:100,grid:{color:line},ticks:{callback:v=>v+'%'}},y:{grid:{display:false},ticks:{color:text}}}}});`)
	fmt.Fprintf(b, `const labelCanvas=document.getElementById('portfolioLabelChart');if(labelCanvas)new Chart(labelCanvas,{type:'doughnut',data:{labels:['Production-ready','Experiment','Archived'],datasets:[{data:[%d,%d,%d],backgroundColor:['rgba(185,216,107,.86)','rgba(0,173,216,.82)','rgba(255,116,116,.80)'],borderColor:'#101718',borderWidth:4,hoverOffset:8}]},options:{responsive:true,maintainAspectRatio:false,cutout:'68%%',plugins:{legend:{position:'bottom',labels:{color:muted,padding:14}}}}});`, labelCounts[model.LabelProductionReady], labelCounts[model.LabelExperiment], labelCounts[model.LabelArchived])
	fmt.Fprintf(b, `const scoreCanvas=document.getElementById('portfolioScoreChart');if(scoreCanvas){const score=%d;new Chart(scoreCanvas,{type:'doughnut',data:{labels:['Ready signals','Review gap','Remaining'],datasets:[{data:[score,Math.max(0,80-score),Math.max(0,100-Math.max(score,80))],backgroundColor:['rgba(0,173,216,.88)','rgba(255,211,106,.84)','rgba(255,255,255,.08)'],borderColor:'#101718',borderWidth:5,hoverOffset:7}]},options:{responsive:true,maintainAspectRatio:false,cutout:'72%%',plugins:{legend:{display:false}}},plugins:[{id:'portfolioScoreCenter',afterDraw(chart){const{ctx,chartArea}=chart,x=(chartArea.left+chartArea.right)/2,y=(chartArea.top+chartArea.bottom)/2;ctx.save();ctx.fillStyle=goBlue;ctx.font='800 54px ui-monospace,SFMono-Regular,Menlo,monospace';ctx.textAlign='center';ctx.textBaseline='middle';ctx.fillText(String(score),x,y-10);ctx.fillStyle=muted;ctx.font='600 13px ui-monospace,SFMono-Regular,Menlo,monospace';ctx.fillText('AUDIT SCORE',x,y+34);ctx.restore();}}]});}})();`, portfolioScore(result))
	fmt.Fprintln(b, `</script>`)
}

func jsString(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `'`, `\'`)
	return "'" + value + "'"
}

func roadmapModule(result audit.Result, reportType string) audit.ModuleResult {
	roadmap, err := audit.Build(reportFromResult(result), reportType)
	if err != nil || len(roadmap.Modules) == 0 {
		return audit.ModuleResult{Title: audit.Title(reportType)}
	}
	return roadmap.Modules[0]
}

func reportFromResult(result audit.Result) model.WorkspaceReport {
	return model.WorkspaceReport{
		RootPath:        result.RootPath,
		Projects:        result.Projects,
		LanguageSummary: result.LanguageSummary,
		TotalFiles:      result.TotalFiles,
		LabelCounts:     result.LabelCounts,
	}
}

func findModule(modules []audit.ModuleResult, name string) audit.ModuleResult {
	for _, module := range modules {
		if module.Name == name {
			return module
		}
	}
	return audit.ModuleResult{}
}

func writeHTMLBadgeCell(b *strings.Builder, label string, class string) {
	fmt.Fprintf(b, `<td><span class="badge %s">%s</span></td>`, html.EscapeString(class), html.EscapeString(label))
}

func mainLanguages(project model.Project) []string {
	if len(project.MainLanguages) > 0 {
		return project.MainLanguages
	}
	return model.GetTopLanguages(project.Languages)
}

func gitDisplay(project model.Project) string {
	if !project.Git.IsRepo {
		return "n/a"
	}
	if project.Git.Dirty {
		return "dirty"
	}
	if project.Git.DaysSinceCommit > 365 {
		return "stale"
	}
	return "clean"
}

func labelBadgeClass(label string) string {
	switch label {
	case model.LabelProductionReady:
		return "good"
	case model.LabelArchived:
		return "bad"
	case model.LabelExperiment:
		return "warn"
	default:
		return "info"
	}
}

func gitBadgeClass(project model.Project) string {
	if !project.Git.IsRepo {
		return "info"
	}
	if project.Git.Dirty || project.Git.DaysSinceCommit > 365 {
		return "warn"
	}
	return "good"
}

func severityBadgeClass(severity string) string {
	switch strings.ToLower(severity) {
	case "high":
		return "bad"
	case "medium", "low":
		return "warn"
	case "pass":
		return "good"
	default:
		return "info"
	}
}

func safetySuggestion(risk string) string {
	switch strings.ToLower(risk) {
	case "no publish-blocking risks detected":
		return "No action required."
	case "environment file":
		return "Remove from export scope and add to ignore rules."
	case "secret-like token":
		return "Replace with a documented test placeholder or load it from a secret manager."
	case "local machine reference":
		return "Move local paths and IPs into configuration examples."
	case "database dump/source":
		return "Review before publishing and keep real dumps outside the repo."
	case "sensitive artifact", "backup artifact":
		return "Remove the artifact from the repo and rotate credentials if needed."
	default:
		return "Review before sharing the export."
	}
}

func safetyMetric(module audit.ModuleResult) string {
	high := 0
	findings := 0
	for _, row := range module.Rows {
		if strings.EqualFold(row["Severity"], "PASS") {
			continue
		}
		findings++
		if strings.EqualFold(row["Severity"], "HIGH") {
			high++
		}
	}
	if high > 0 {
		return fmt.Sprintf("%d high", high)
	}
	return fmt.Sprintf("%d findings", findings)
}

func averageModuleScore(module audit.ModuleResult, column string) string {
	total := 0
	count := 0
	for _, row := range module.Rows {
		value, err := strconv.Atoi(row[column])
		if err != nil {
			continue
		}
		total += value
		count++
	}
	if count == 0 {
		return "0"
	}
	return fmt.Sprintf("%d", total/count)
}

func sumModuleColumn(module audit.ModuleResult, column string) string {
	total := 0
	for _, row := range module.Rows {
		value, err := strconv.Atoi(row[column])
		if err == nil {
			total += value
		}
	}
	return fmt.Sprintf("%d", total)
}

func largestLOCFile(module audit.ModuleResult) string {
	bestFile := "-"
	bestLOC := -1
	for _, row := range module.Rows {
		value, err := strconv.Atoi(row["Total LOC"])
		if err != nil {
			continue
		}
		if value > bestLOC {
			bestLOC = value
			bestFile = row["Largest File"]
		}
	}
	return bestFile
}

func readinessDecision(result audit.Result) string {
	if result.ProjectCount == 0 {
		return "empty"
	}
	ready := result.LabelCounts[model.LabelProductionReady]
	if ready*2 >= result.ProjectCount {
		return "strong"
	}
	if ready > 0 {
		return "review"
	}
	return "early"
}

func portfolioScore(result audit.Result) int {
	if len(result.Projects) == 0 {
		return 0
	}
	total := 0
	for _, project := range result.Projects {
		total += project.Readiness.Score
	}
	return total / len(result.Projects)
}

func numericColumn(column string) bool {
	switch column {
	case "Files", "Line", "Readiness", "Score", "Projects", "Days Since Commit", "Total LOC", "Code LOC", "Blank LOC", "Comment LOC", "Avg/File":
		return true
	default:
		return false
	}
}
