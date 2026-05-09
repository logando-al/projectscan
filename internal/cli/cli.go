package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"projectscan/internal/export"
	"projectscan/internal/model"
	"projectscan/internal/render"
	"projectscan/internal/scan"
	"projectscan/internal/tui"
)

type Options struct {
	RootPath      string
	Mode          string
	ConfigPath    string
	NoColor       bool
	Interactive   bool
	ReportType    string
	ExportFormat  string
	ProjectFilter string
}

func Run(args []string, cwd string, w io.Writer) error {
	opts, err := ParseOptions(args, cwd)
	if err != nil {
		return err
	}
	if opts.Interactive {
		return tui.RunApp(opts.RootPath)
	}
	return RunWithOptions(opts, w)
}

func ParseOptions(args []string, cwd string) (Options, error) {
	opts := Options{
		RootPath:     cwd,
		Mode:         model.ModeSummary,
		ReportType:   model.ReportSummary,
		ExportFormat: model.ExportTerminal,
	}
	if len(args) == 0 {
		opts.Interactive = true
		return opts, nil
	}

	tuiCommand := args[0] == "tui"
	if tuiCommand {
		opts.Interactive = true
		args = args[1:]
	}

	exportCommand := len(args) > 0 && args[0] == model.ModeExport
	if len(args) > 0 && exportCommand {
		opts.Mode = model.ModeExport
		args = args[1:]
	}

	flags := flag.NewFlagSet("projectscan", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	audit := flags.Bool("audit", false, "show full audit")
	details := flags.Bool("details", false, "show details")
	jsonMode := flags.Bool("json", false, "write json")
	markdownMode := flags.Bool("markdown", false, "write markdown")
	flags.StringVar(&opts.ConfigPath, "config", "", "config path")
	flags.BoolVar(&opts.NoColor, "no-color", false, "disable color")
	flags.StringVar(&opts.ReportType, "report", opts.ReportType, "report type")
	flags.StringVar(&opts.ExportFormat, "format", opts.ExportFormat, "export format")
	flags.StringVar(&opts.ExportFormat, "export", opts.ExportFormat, "export format")
	flags.StringVar(&opts.ProjectFilter, "project", "", "project name")

	normalized, rootPath := normalizeArgs(args)
	if err := flags.Parse(normalized); err != nil {
		return opts, err
	}
	reportSet := flagWasSet(flags, "report")
	formatSet := flagWasSet(flags, "format") || flagWasSet(flags, "export")
	if rootPath == "" && flags.NArg() > 0 {
		rootPath = flags.Arg(0)
	}
	if rootPath != "" {
		absRoot, err := filepath.Abs(rootPath)
		if err != nil {
			return opts, err
		}
		opts.RootPath = absRoot
	}

	terminalModes := 0
	if *audit {
		opts.Mode = model.ModeAudit
		opts.ReportType = model.ReportAudit
		terminalModes++
	}
	if *details {
		opts.Mode = model.ModeDetails
		opts.ReportType = model.ReportDetails
		terminalModes++
	}
	if terminalModes > 1 {
		return opts, errors.New("choose only one terminal mode flag")
	}
	if *jsonMode {
		opts.Mode = model.ModeExport
		opts.ExportFormat = model.ExportJSON
	}
	if *markdownMode {
		opts.Mode = model.ModeExport
		opts.ExportFormat = model.ExportMarkdown
	}
	if exportCommand || opts.ExportFormat != model.ExportTerminal {
		opts.Mode = model.ModeExport
	}
	if exportCommand {
		if !reportSet {
			opts.ReportType = model.ReportAudit
		}
		if !formatSet {
			opts.ExportFormat = model.ExportHTML
		}
	}
	if opts.Mode == model.ModeExport && opts.ReportType == "" {
		opts.ReportType = model.ReportSummary
	}
	if err := model.ValidateReportType(opts.ReportType); err != nil {
		return opts, err
	}
	if err := model.ValidateExportFormat(opts.ExportFormat); err != nil {
		return opts, err
	}
	return opts, nil
}

func flagWasSet(flags *flag.FlagSet, name string) bool {
	wasSet := false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == name {
			wasSet = true
		}
	})
	return wasSet
}

func normalizeArgs(args []string) ([]string, string) {
	normalized := []string{}
	rootPath := ""
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			normalized = append(normalized, arg)
			if flagTakesValue(arg) && i+1 < len(args) {
				i++
				normalized = append(normalized, args[i])
			}
			continue
		}
		if rootPath == "" {
			rootPath = arg
			continue
		}
		normalized = append(normalized, arg)
	}
	return normalized, rootPath
}

func flagTakesValue(arg string) bool {
	switch arg {
	case "--config", "--report", "--format", "--export", "--project":
		return true
	default:
		return false
	}
}

func RunWithOptions(opts Options, w io.Writer) error {
	report, err := scan.Workspace(opts.RootPath, scan.Options{ConfigPath: opts.ConfigPath})
	if err != nil {
		return err
	}
	style := render.Style{
		Color: !opts.NoColor && render.ShouldUseColor(os.Stdout),
	}

	switch opts.Mode {
	case model.ModeSummary:
		fmt.Fprint(w, render.BuildSummaryWithStyle(report.RootPath, report.Projects, style))
	case model.ModeAudit:
		fmt.Fprint(w, render.BuildAuditReport(report, style))
	case model.ModeDetails:
		fmt.Fprint(w, render.BuildDetails(report, style))
	case model.ModeJSON:
		result, err := export.WriteReport(report, model.ExportRequest{ReportType: model.ReportSummary, Format: model.ExportJSON}, "")
		if err != nil {
			return err
		}
		fmt.Fprintln(w, result.Path)
	case model.ModeMarkdown:
		result, err := export.WriteReport(report, model.ExportRequest{ReportType: model.ReportSummary, Format: model.ExportMarkdown}, "")
		if err != nil {
			return err
		}
		fmt.Fprintln(w, result.Path)
	case model.ModeExport:
		result, err := export.WriteReport(report, model.ExportRequest{
			ReportType:    opts.ReportType,
			Format:        opts.ExportFormat,
			ProjectFilter: opts.ProjectFilter,
		}, "")
		if err != nil {
			return err
		}
		fmt.Fprintln(w, result.Path)
	default:
		return fmt.Errorf("unknown mode %q", opts.Mode)
	}
	return nil
}
