package gstr1

import "fmt"

// ComputeFP returns the GSTR-1 filing period (MMYYYY). For a QRMP quarterly
// filer it is the LAST month of the quarter containing month: Apr-Jun -> 06,
// Jul-Sep -> 09, Oct-Dec -> 12, Jan-Mar -> 03. Financial-year quarters never
// straddle a calendar year at their last month, so the year is unchanged.
func ComputeFP(month, year int, qrmp bool) string {
	m := month
	if qrmp {
		switch {
		case month >= 4 && month <= 6:
			m = 6
		case month >= 7 && month <= 9:
			m = 9
		case month >= 10 && month <= 12:
			m = 12
		default: // Jan, Feb, Mar
			m = 3
		}
	}
	return fmt.Sprintf("%02d%04d", m, year)
}
