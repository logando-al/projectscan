package export

import (
	"bytes"
	"encoding/csv"

	"projectscan/internal/audit"
)

func buildCSV(result audit.Result) (string, error) {
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	if result.Message != "" {
		writeCSV(w, []string{"report_type", "message", "secret_values"})
		writeCSV(w, []string{result.ReportType, result.Message, result.SecretValues})
		w.Flush()
		return b.String(), w.Error()
	}
	for moduleIndex, module := range result.Modules {
		if len(module.Columns) == 0 {
			continue
		}
		if moduleIndex > 0 {
			writeCSV(w, []string{})
		}
		writeCSV(w, module.Columns)
		for _, row := range module.Rows {
			values := make([]string, 0, len(module.Columns))
			for _, column := range module.Columns {
				values = append(values, row[column])
			}
			writeCSV(w, values)
		}
	}
	w.Flush()
	return b.String(), w.Error()
}

func writeCSV(w *csv.Writer, row []string) {
	_ = w.Write(row)
}
