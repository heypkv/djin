package gstr1

import (
	"sort"

	"github.com/heypkv/djin/internal/gst"
)

// Build computes a portal-ready GSTR-1 upload from invoice-level input,
// auto-deriving Table 12 (HSN B2B/B2C split) and Table 13 (documents issued),
// and consolidating B2CS rate-wise. It always returns a Report; when the
// report has issues the caller decides whether to proceed.
func Build(in ReturnInput) (*Portal, *Report) {
	rep := &Report{}

	if err := gst.ValidateGSTIN(in.GSTIN); err != nil {
		rep.Add("input", in.GSTIN, "gstin", "%v", err)
	}
	supplierState := gst.StateCodeOf(in.GSTIN)

	version := in.Version
	if version == "" {
		version = DefaultVersion
	}
	hash := in.Hash
	if hash == "" {
		hash = "hash"
	}

	p := &Portal{
		Gstin:   in.GSTIN,
		Fp:      ComputeFP(in.Period.Month, in.Period.Year, in.Period.QRMP),
		Version: version,
		Hash:    hash,
	}

	b2b := newB2BAccum()
	b2cs := newB2CSAccum()
	hsnB2B := newHSNAccum()
	hsnB2C := newHSNAccum()
	docs := newDocAccum()

	for _, inv := range in.Invoices {
		docs.add(inv)

		pos := inv.Buyer.POS
		registered := inv.Buyer.GSTIN != ""
		if pos == "" {
			if registered {
				pos = gst.StateCodeOf(inv.Buyer.GSTIN)
			} else {
				pos = supplierState
			}
		}
		if _, ok := gst.StateByCode(pos); !ok {
			rep.Add("input", inv.Number, "pos", "unknown place-of-supply code %q", pos)
		}
		if registered {
			if err := gst.ValidateGSTIN(inv.Buyer.GSTIN); err != nil {
				rep.Add("b2b", inv.Number, "buyer.gstin", "%v", err)
			}
		}
		intra := gst.IsIntraState(supplierState, pos)

		if len(inv.Lines) == 0 {
			rep.Add("input", inv.Number, "lines", "invoice has no line items")
		}

		// Compute per-line tax once; every aggregate sums these rounded figures.
		var invTotal gst.Amount
		lineTax := make([]gst.Tax, len(inv.Lines))
		for i, ln := range inv.Lines {
			if !gst.IsValidRate(ln.Rate) {
				rep.Add("input", inv.Number, "rate", "unrecognised GST rate %v%%", ln.Rate)
			}
			if ln.HSN == "" {
				rep.Add("input", inv.Number, "hsn", "line %d missing HSN/SAC", i+1)
			}
			tax := gst.ComputeTax(ln.Taxable, ln.Rate, intra)
			tax.Cess = ln.Cess
			lineTax[i] = tax
			invTotal += ln.Taxable + tax.Total()

			hsn := hsnB2C
			if registered {
				hsn = hsnB2B
			}
			hsn.add(ln, tax)
		}
		val, _ := gst.RoundToRupee(invTotal)

		if registered {
			b2b.add(inv, pos, val, intra)
		} else {
			for i, ln := range inv.Lines {
				b2cs.add(pos, intra, ln, lineTax[i])
			}
		}
	}

	p.B2b = b2b.rows()
	p.B2cs = b2cs.rows()
	if h := buildHSN(hsnB2B, hsnB2C); h != nil {
		p.Hsn = h
	}
	if d := docs.section(); d != nil {
		p.DocIssue = d
	}
	return p, rep
}

// ptr returns a pointer to a copy of a (for the optional tax fields).
func ptr(a gst.Amount) *gst.Amount { return &a }

// --- B2B accumulator: ctin -> invoices ---

type b2bAccum struct {
	order   []string
	byCtin  map[string][]Invoice
	posOf   map[string]string // invoice number -> pos
	valOf   map[string]gst.Amount
	intraOf map[string]bool
}

func newB2BAccum() *b2bAccum {
	return &b2bAccum{byCtin: map[string][]Invoice{}, posOf: map[string]string{}, valOf: map[string]gst.Amount{}, intraOf: map[string]bool{}}
}

func (a *b2bAccum) add(inv Invoice, pos string, val gst.Amount, intra bool) {
	ctin := inv.Buyer.GSTIN
	if _, seen := a.byCtin[ctin]; !seen {
		a.order = append(a.order, ctin)
	}
	a.byCtin[ctin] = append(a.byCtin[ctin], inv)
	a.posOf[inv.Number] = pos
	a.valOf[inv.Number] = val
	a.intraOf[inv.Number] = intra
}

func (a *b2bAccum) rows() []B2BCtin {
	if len(a.order) == 0 {
		return nil
	}
	ctins := append([]string(nil), a.order...)
	sort.Strings(ctins)
	out := make([]B2BCtin, 0, len(ctins))
	for _, ctin := range ctins {
		invs := a.byCtin[ctin]
		sort.Slice(invs, func(i, j int) bool { return invs[i].Number < invs[j].Number })
		row := B2BCtin{Ctin: ctin}
		for _, inv := range invs {
			typ := inv.Type
			if typ == "" {
				typ = "R"
			}
			rchrg := "N"
			if inv.Reverse {
				rchrg = "Y"
			}
			row.Inv = append(row.Inv, B2BInv{
				Inum:   inv.Number,
				Idt:    inv.Date,
				Val:    a.valOf[inv.Number],
				Pos:    a.posOf[inv.Number],
				Rchrg:  rchrg,
				InvTyp: typ,
				Itms:   rateItems(inv.Lines, a.intraOf[inv.Number]),
			})
		}
		out = append(out, row)
	}
	return out
}

// rateItems groups an invoice's lines by rate into portal itm entries.
func rateItems(lines []Line, intra bool) []Itm {
	type agg struct {
		txval, cgst, sgst, igst, cess gst.Amount
	}
	byRate := map[float64]*agg{}
	var rates []float64
	for _, ln := range lines {
		a := byRate[ln.Rate]
		if a == nil {
			a = &agg{}
			byRate[ln.Rate] = a
			rates = append(rates, ln.Rate)
		}
		tax := gst.ComputeTax(ln.Taxable, ln.Rate, intra)
		a.txval += ln.Taxable
		a.cgst += tax.CGST
		a.sgst += tax.SGST
		a.igst += tax.IGST
		a.cess += ln.Cess
	}
	sort.Float64s(rates)
	out := make([]Itm, 0, len(rates))
	for i, rt := range rates {
		a := byRate[rt]
		det := ItmDet{Rt: rt, Txval: a.txval, Csamt: a.cess}
		if intra {
			det.Camt = ptr(a.cgst)
			det.Samt = ptr(a.sgst)
		} else {
			det.Iamt = ptr(a.igst)
		}
		out = append(out, Itm{Num: i + 1, ItmDet: det})
	}
	return out
}

// --- B2CS accumulator: (sply_ty, rate, pos, typ) -> totals ---

type b2csKey struct {
	splyTy string
	rt     float64
	pos    string
	typ    string
}

type b2csAccum struct {
	order []b2csKey
	m     map[b2csKey]*b2csTotals
}

type b2csTotals struct {
	txval, cgst, sgst, igst, cess gst.Amount
	intra                         bool
}

func newB2CSAccum() *b2csAccum { return &b2csAccum{m: map[b2csKey]*b2csTotals{}} }

func (a *b2csAccum) add(pos string, intra bool, ln Line, tax gst.Tax) {
	splyTy := "INTER"
	if intra {
		splyTy = "INTRA"
	}
	k := b2csKey{splyTy: splyTy, rt: ln.Rate, pos: pos, typ: "OE"}
	t := a.m[k]
	if t == nil {
		t = &b2csTotals{intra: intra}
		a.m[k] = t
		a.order = append(a.order, k)
	}
	t.txval += ln.Taxable
	t.cgst += tax.CGST
	t.sgst += tax.SGST
	t.igst += tax.IGST
	t.cess += ln.Cess
}

func (a *b2csAccum) rows() []B2CSRow {
	if len(a.order) == 0 {
		return nil
	}
	keys := append([]b2csKey(nil), a.order...)
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].pos != keys[j].pos {
			return keys[i].pos < keys[j].pos
		}
		return keys[i].rt < keys[j].rt
	})
	out := make([]B2CSRow, 0, len(keys))
	for _, k := range keys {
		t := a.m[k]
		row := B2CSRow{SplyTy: k.splyTy, Rt: k.rt, Typ: k.typ, Pos: k.pos, Txval: t.txval, Csamt: t.cess}
		if t.intra {
			row.Camt = ptr(t.cgst)
			row.Samt = ptr(t.sgst)
		} else {
			row.Iamt = ptr(t.igst)
		}
		out = append(out, row)
	}
	return out
}

// --- HSN accumulator (Table 12) ---

type hsnKey struct {
	hsn string
	rt  float64
}

type hsnTotals struct {
	desc                          string
	uqc                           string
	qty                           float64
	txval, iamt, cgst, sgst, cess gst.Amount
}

type hsnAccum struct {
	order []hsnKey
	m     map[hsnKey]*hsnTotals
}

func newHSNAccum() *hsnAccum { return &hsnAccum{m: map[hsnKey]*hsnTotals{}} }

func (a *hsnAccum) add(ln Line, tax gst.Tax) {
	k := hsnKey{hsn: ln.HSN, rt: ln.Rate}
	t := a.m[k]
	if t == nil {
		uqc := ln.UQC
		if uqc == "" {
			uqc = "NA"
		}
		t = &hsnTotals{desc: ln.Description, uqc: uqc}
		a.m[k] = t
		a.order = append(a.order, k)
	}
	t.qty += ln.Qty
	t.txval += ln.Taxable
	t.iamt += tax.IGST
	t.cgst += tax.CGST
	t.sgst += tax.SGST
	t.cess += ln.Cess
}

func (a *hsnAccum) rows() []HSNRow {
	keys := append([]hsnKey(nil), a.order...)
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].hsn != keys[j].hsn {
			return keys[i].hsn < keys[j].hsn
		}
		return keys[i].rt < keys[j].rt
	})
	out := make([]HSNRow, 0, len(keys))
	for i, k := range keys {
		t := a.m[k]
		out = append(out, HSNRow{
			Num: i + 1, HsnSc: k.hsn, Desc: t.desc, Uqc: t.uqc, Qty: t.qty, Rt: k.rt,
			Txval: t.txval, Iamt: t.iamt, Camt: t.cgst, Samt: t.sgst, Csamt: t.cess,
		})
	}
	return out
}

func buildHSN(b2b, b2c *hsnAccum) *HSN {
	hb := b2b.rows()
	hc := b2c.rows()
	if len(hb) == 0 && len(hc) == 0 {
		return nil
	}
	return &HSN{HsnB2b: hb, HsnB2c: hc}
}
