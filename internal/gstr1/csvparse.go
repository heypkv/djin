package gstr1

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/heypkv/djin/internal/gst"
)

// csvTable is a header-indexed view of a CSV, so importers address columns by
// their official template name rather than a brittle positional index.
type csvTable struct {
	header map[string]int
	rows   [][]string
}

// readCSV parses r (tolerating ragged rows and a UTF-8 BOM) into a csvTable.
func readCSV(r io.Reader) (*csvTable, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	cr.TrimLeadingSpace = true
	records, err := cr.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("empty CSV")
	}
	header := map[string]int{}
	for i, h := range records[0] {
		h = strings.TrimPrefix(h, "\ufeff")
		header[normalizeHeader(h)] = i
	}
	return &csvTable{header: header, rows: records[1:]}, nil
}

func normalizeHeader(h string) string {
	return strings.ToLower(strings.Join(strings.Fields(h), " "))
}

// col returns the cell for the named column in row, or "" if absent.
func (t *csvTable) col(row []string, name string) string {
	i, ok := t.header[normalizeHeader(name)]
	if !ok || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

// posCode extracts the two-digit state code from a Place-of-Supply cell such as
// "37-Andhra Pradesh".
func posCode(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if i := strings.IndexByte(s, '-'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

// parseRate parses a percentage rate ("18", "0.25", "5.00").
func parseRate(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	return strconv.ParseFloat(s, 64)
}

// parseDate reformats the official CSV date ("14-Jul-17", "3-Mar-20") to the
// portal's DD-MM-YYYY. Unparseable values pass through unchanged.
func parseDate(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	months := map[string]string{
		"jan": "01", "feb": "02", "mar": "03", "apr": "04", "may": "05", "jun": "06",
		"jul": "07", "aug": "08", "sep": "09", "oct": "10", "nov": "11", "dec": "12",
	}
	parts := strings.Split(s, "-")
	if len(parts) != 3 {
		return s
	}
	d, err := strconv.Atoi(parts[0])
	if err != nil {
		return s
	}
	mm, ok := months[strings.ToLower(parts[1])]
	if !ok {
		return s
	}
	y := parts[2]
	if len(y) == 2 {
		y = "20" + y
	}
	return fmt.Sprintf("%02d-%s-%s", d, mm, y)
}

// yn returns "Y" or "N" from a reverse-charge cell.
func yn(s string) string {
	if strings.EqualFold(strings.TrimSpace(s), "Y") {
		return "Y"
	}
	return "N"
}

// invTypeCode maps the official "Invoice Type" label to the portal inv_typ code.
func invTypeCode(label string) string {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "sez supplies with payment":
		return "SEWP"
	case "sez supplies without payment":
		return "SEWOP"
	case "deemed exp", "deemed export":
		return "DE"
	default:
		return "R"
	}
}

// amt is ParseAmount with the error folded into a report entry by the caller.
func amt(s string) gst.Amount {
	a, _ := gst.ParseAmount(s)
	return a
}
