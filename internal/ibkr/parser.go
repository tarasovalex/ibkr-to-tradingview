package ibkr

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
)

// Row is one line from an IBKR Activity Statement CSV.
type Row struct {
	Section string
	RowType string
	Fields  []string
}

// ParseFile reads all rows from an IBKR activity statement CSV.
func ParseFile(path string) ([]Row, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ParseReader(f)
}

// ParseReader reads all rows from r.
func ParseReader(r io.Reader) ([]Row, error) {
	reader := csv.NewReader(r)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	var rows []Row
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read csv: %w", err)
		}
		if len(record) < 2 {
			continue
		}
		fields := record[2:]
		rows = append(rows, Row{
			Section: record[0],
			RowType: record[1],
			Fields:  fields,
		})
	}
	return rows, nil
}

// Field returns field at index or empty string.
func Field(fields []string, i int) string {
	if i < 0 || i >= len(fields) {
		return ""
	}
	return fields[i]
}

// IsDataRow returns true for actionable Data rows (not SubTotal/Total).
func IsDataRow(rowType string) bool {
	return rowType == RowData
}
