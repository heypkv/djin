package gstr1

import "github.com/heypkv/djin/internal/gst"

// ReturnInput is the simple invoice-level input djin builds a return from. It
// is the format for people who never touched the official tool; the CSV
// importers also lower into it.
type ReturnInput struct {
	GSTIN    string    `json:"gstin"`
	Period   Period    `json:"period"`
	Version  string    `json:"version,omitempty"` // defaults to DefaultVersion
	Hash     string    `json:"hash,omitempty"`    // defaults to "hash"
	Invoices []Invoice `json:"invoices"`
}

// Period identifies the tax period. For a QRMP (quarterly) filer the filing
// period fp is the LAST month of the quarter — a real footgun the portal does
// not warn about.
type Period struct {
	Month int  `json:"month"` // 1-12, any month in the period
	Year  int  `json:"year"`  // calendar year of that month
	QRMP  bool `json:"qrmp"`  // quarterly filer
}

// Invoice is one outward-supply document with its line items.
type Invoice struct {
	Number    string `json:"number"`
	Date      string `json:"date"` // DD-MM-YYYY
	Buyer     Buyer  `json:"buyer"`
	Type      string `json:"type,omitempty"`      // inv_typ; defaults to "R"
	Reverse   bool   `json:"reverse,omitempty"`   // reverse charge
	Cancelled bool   `json:"cancelled,omitempty"` // for Table 13 counts
	// DocNature is the Table 13 nature-of-document code (1 = invoice for
	// outward supply, the default when zero).
	DocNature int    `json:"doc_nature,omitempty"`
	Lines     []Line `json:"lines"`
}

// Buyer is the recipient. An empty GSTIN means an unregistered (B2C) buyer.
type Buyer struct {
	GSTIN string `json:"gstin,omitempty"`
	Name  string `json:"name,omitempty"`
	POS   string `json:"pos,omitempty"` // place-of-supply state code; derived from GSTIN if empty
}

// Line is a rate-wise taxable line. Taxable and Cess are rupee amounts in the
// input JSON, stored as paise.
type Line struct {
	HSN         string     `json:"hsn"` // HSN or SAC code
	Description string     `json:"description,omitempty"`
	UQC         string     `json:"uqc,omitempty"` // unit; "NA" for services
	Qty         float64    `json:"qty,omitempty"`
	Rate        float64    `json:"rate"`
	Taxable     gst.Amount `json:"taxable"`
	Cess        gst.Amount `json:"cess,omitempty"`
}
