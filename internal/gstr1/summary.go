package gstr1

import (
	"fmt"
	"strings"

	"github.com/heypkv/djin/internal/gst"
)

// Summary is a pre-upload reconciliation view mirroring the portal's "Generate
// Summary" tiles: per-section taxable and tax totals for eyeballing before you
// commit the upload.
type Summary struct {
	GSTIN string
	FP    string
	Tiles []Tile
}

// Tile is one section's totals.
type Tile struct {
	Section string
	Count   int // documents/rows in the section
	Taxable gst.Amount
	Tax     gst.Amount // CGST+SGST+IGST
	Cess    gst.Amount
}

// Summarize computes reconciliation tiles from an emitted portal document.
func Summarize(p *Portal) Summary {
	s := Summary{GSTIN: p.Gstin, FP: p.Fp}

	if len(p.B2b) > 0 {
		t := Tile{Section: "B2B (4A)"}
		for _, c := range p.B2b {
			for _, inv := range c.Inv {
				t.Count++
				for _, it := range inv.Itms {
					addItm(&t, it.ItmDet)
				}
			}
		}
		s.Tiles = append(s.Tiles, t)
	}
	if len(p.B2cs) > 0 {
		t := Tile{Section: "B2CS (7)", Count: len(p.B2cs)}
		for _, r := range p.B2cs {
			t.Taxable += r.Txval
			t.Tax += deref(r.Camt) + deref(r.Samt) + deref(r.Iamt)
			t.Cess += r.Csamt
		}
		s.Tiles = append(s.Tiles, t)
	}
	if p.Hsn != nil {
		t := Tile{Section: "HSN (12)"}
		for _, h := range append(append([]HSNRow{}, p.Hsn.HsnB2b...), p.Hsn.HsnB2c...) {
			t.Count++
			t.Taxable += h.Txval
			t.Tax += h.Camt + h.Samt + h.Iamt
			t.Cess += h.Csamt
		}
		s.Tiles = append(s.Tiles, t)
	}
	if p.DocIssue != nil {
		t := Tile{Section: "Docs (13)"}
		for _, d := range p.DocIssue.DocDet {
			for _, e := range d.Docs {
				t.Count += e.NetIssue
			}
		}
		s.Tiles = append(s.Tiles, t)
	}
	return s
}

func addItm(t *Tile, d ItmDet) {
	t.Taxable += d.Txval
	t.Tax += deref(d.Camt) + deref(d.Samt) + deref(d.Iamt)
	t.Cess += d.Csamt
}

func deref(a *gst.Amount) gst.Amount {
	if a == nil {
		return 0
	}
	return *a
}

// String renders the summary as an aligned text block.
func (s Summary) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "GSTR-1 summary  gstin=%s  fp=%s\n", s.GSTIN, s.FP)
	fmt.Fprintf(&b, "%-12s %6s %14s %14s %12s\n", "section", "count", "taxable", "tax", "cess")
	for _, t := range s.Tiles {
		fmt.Fprintf(&b, "%-12s %6d %14s %14s %12s\n", t.Section, t.Count, t.Taxable, t.Tax, t.Cess)
	}
	return b.String()
}
