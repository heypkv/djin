package gst

import "strings"

// State is a GST state/UT with its two-digit code (also the Place of Supply
// code) and name.
type State struct {
	Code string
	Name string
}

// states is the GST state/UT code table (Place of Supply codes). Codes are the
// leading two digits of every GSTIN issued in that jurisdiction.
var states = []State{
	{"01", "Jammu & Kashmir"},
	{"02", "Himachal Pradesh"},
	{"03", "Punjab"},
	{"04", "Chandigarh"},
	{"05", "Uttarakhand"},
	{"06", "Haryana"},
	{"07", "Delhi"},
	{"08", "Rajasthan"},
	{"09", "Uttar Pradesh"},
	{"10", "Bihar"},
	{"11", "Sikkim"},
	{"12", "Arunachal Pradesh"},
	{"13", "Nagaland"},
	{"14", "Manipur"},
	{"15", "Mizoram"},
	{"16", "Tripura"},
	{"17", "Meghalaya"},
	{"18", "Assam"},
	{"19", "West Bengal"},
	{"20", "Jharkhand"},
	{"21", "Odisha"},
	{"22", "Chhattisgarh"},
	{"23", "Madhya Pradesh"},
	{"24", "Gujarat"},
	{"25", "Daman & Diu"},
	{"26", "Dadra & Nagar Haveli and Daman & Diu"},
	{"27", "Maharashtra"},
	{"28", "Andhra Pradesh (before division)"},
	{"29", "Karnataka"},
	{"30", "Goa"},
	{"31", "Lakshadweep"},
	{"32", "Kerala"},
	{"33", "Tamil Nadu"},
	{"34", "Puducherry"},
	{"35", "Andaman & Nicobar Islands"},
	{"36", "Telangana"},
	{"37", "Andhra Pradesh"},
	{"38", "Ladakh"},
	{"97", "Other Territory"},
	{"99", "Centre Jurisdiction"},
}

var stateByCode = func() map[string]State {
	m := make(map[string]State, len(states))
	for _, s := range states {
		m[s.Code] = s
	}
	return m
}()

// StateByCode returns the state for a two-digit code (ok=false if unknown).
func StateByCode(code string) (State, bool) {
	s, ok := stateByCode[strings.TrimSpace(code)]
	return s, ok
}

// States returns the full state/POS code table.
func States() []State { return states }

// IsIntraState reports whether a supply from supplierState to placeOfSupply is
// intra-state (same state code) — the CGST+SGST case, versus IGST.
func IsIntraState(supplierState, placeOfSupply string) bool {
	return strings.TrimSpace(supplierState) == strings.TrimSpace(placeOfSupply)
}
