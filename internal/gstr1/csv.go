package gstr1

import (
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/heypkv/djin/internal/gst"
)

// The importers below read the official GST offline-tool section CSVs (column
// layouts in gst_offline_tool/Section_wise_CSV_files/GSTR1) into portal
// sections. Whether a supply is intra- or inter-state — the CGST/SGST vs IGST
// split — depends on the filer's own state, so importers that carry taxable
// lines take supplierState (the two-digit code of the filer's GSTIN).

// itemsFrom builds rate-wise itm entries from raw (rate, taxable, cess) lines.
func itemsFrom(lines []Line, intra bool) []Itm { return rateItems(lines, intra) }

// ImportB2B reads the b2b,sez,de.csv shape into Table 4A, grouping rows by
// recipient GSTIN and invoice number.
func ImportB2B(r io.Reader, supplierState string) ([]B2BCtin, *Report, error) {
	t, err := readCSV(r)
	if err != nil {
		return nil, nil, err
	}
	rep := &Report{}
	type invAgg struct {
		date, pos, rchrg, invTyp string
		val                      gst.Amount
		intra                    bool
		lines                    []Line
	}
	byCtin := map[string]map[string]*invAgg{}
	var ctinOrder []string
	invOrder := map[string][]string{}

	for n, row := range t.rows {
		ctin := t.col(row, "GSTIN/UIN of Recipient")
		inum := t.col(row, "Invoice Number")
		if ctin == "" || inum == "" {
			continue
		}
		if err := gst.ValidateGSTIN(ctin); err != nil {
			rep.Add("b2b", inum, "ctin", "row %d: %v", n+2, err)
		}
		rate, err := parseRate(t.col(row, "Rate"))
		if err != nil {
			rep.Add("b2b", inum, "rate", "row %d: %v", n+2, err)
		}
		pos := posCode(t.col(row, "Place Of Supply"))
		intra := gst.IsIntraState(supplierState, pos)
		if _, ok := byCtin[ctin]; !ok {
			byCtin[ctin] = map[string]*invAgg{}
			ctinOrder = append(ctinOrder, ctin)
		}
		ia := byCtin[ctin][inum]
		if ia == nil {
			ia = &invAgg{
				date:   parseDate(t.col(row, "Invoice date")),
				pos:    pos,
				rchrg:  yn(t.col(row, "Reverse Charge")),
				invTyp: invTypeCode(t.col(row, "Invoice Type")),
				val:    amt(t.col(row, "Invoice Value")),
				intra:  intra,
			}
			byCtin[ctin][inum] = ia
			invOrder[ctin] = append(invOrder[ctin], inum)
		}
		ia.lines = append(ia.lines, Line{Rate: rate, Taxable: amt(t.col(row, "Taxable Value")), Cess: amt(t.col(row, "Cess Amount"))})
	}

	sort.Strings(ctinOrder)
	var out []B2BCtin
	for _, ctin := range ctinOrder {
		nums := invOrder[ctin]
		sort.Strings(nums)
		row := B2BCtin{Ctin: ctin}
		for _, inum := range nums {
			ia := byCtin[ctin][inum]
			row.Inv = append(row.Inv, B2BInv{
				Inum: inum, Idt: ia.date, Val: ia.val, Pos: ia.pos,
				Rchrg: ia.rchrg, InvTyp: ia.invTyp, Itms: itemsFrom(ia.lines, ia.intra),
			})
		}
		out = append(out, row)
	}
	return out, rep, nil
}

// ImportB2CS reads b2cs.csv into Table 7, consolidating rate-wise by
// (supply type, rate, POS, e-commerce type).
func ImportB2CS(r io.Reader, supplierState string) ([]B2CSRow, *Report, error) {
	t, err := readCSV(r)
	if err != nil {
		return nil, nil, err
	}
	rep := &Report{}
	acc := newB2CSAccum()
	// The accumulator fixes typ to "OE"; track the CSV's Type per key so E-commerce
	// rows keep their label.
	typByKey := map[b2csKey]string{}
	for n, row := range t.rows {
		typ := strings.TrimSpace(t.col(row, "Type"))
		if typ == "" {
			typ = "OE"
		}
		pos := posCode(t.col(row, "Place Of Supply"))
		rate, err := parseRate(t.col(row, "Rate"))
		if err != nil {
			rep.Add("b2cs", strconv.Itoa(n+2), "rate", "%v", err)
		}
		if pos == "" {
			continue
		}
		intra := gst.IsIntraState(supplierState, pos)
		ln := Line{Rate: rate, Taxable: amt(t.col(row, "Taxable Value")), Cess: amt(t.col(row, "Cess Amount"))}
		tax := gst.ComputeTax(ln.Taxable, ln.Rate, intra)
		tax.Cess = ln.Cess
		acc.add(pos, intra, ln, tax)
		splyTy := "INTER"
		if intra {
			splyTy = "INTRA"
		}
		typByKey[b2csKey{splyTy: splyTy, rt: rate, pos: pos, typ: "OE"}] = typ
	}
	rows := acc.rows()
	for i := range rows {
		k := b2csKey{splyTy: rows[i].SplyTy, rt: rows[i].Rt, pos: rows[i].Pos, typ: "OE"}
		if typ, ok := typByKey[k]; ok {
			rows[i].Typ = typ
		}
	}
	return rows, rep, nil
}

// ImportB2CL reads b2cl.csv into Table 5 (large inter-state B2C), grouping by
// POS then invoice. These supplies are inter-state, so tax lands in IGST.
func ImportB2CL(r io.Reader) ([]B2CLPos, *Report, error) {
	t, err := readCSV(r)
	if err != nil {
		return nil, nil, err
	}
	rep := &Report{}
	type invAgg struct {
		date  string
		val   gst.Amount
		lines []Line
	}
	byPos := map[string]map[string]*invAgg{}
	var posOrder []string
	invOrder := map[string][]string{}
	for n, row := range t.rows {
		inum := t.col(row, "Invoice Number")
		if inum == "" {
			continue
		}
		pos := posCode(t.col(row, "Place Of Supply"))
		rate, err := parseRate(t.col(row, "Rate"))
		if err != nil {
			rep.Add("b2cl", inum, "rate", "row %d: %v", n+2, err)
		}
		if _, ok := byPos[pos]; !ok {
			byPos[pos] = map[string]*invAgg{}
			posOrder = append(posOrder, pos)
		}
		ia := byPos[pos][inum]
		if ia == nil {
			ia = &invAgg{date: parseDate(t.col(row, "Invoice date")), val: amt(t.col(row, "Invoice Value"))}
			byPos[pos][inum] = ia
			invOrder[pos] = append(invOrder[pos], inum)
		}
		ia.lines = append(ia.lines, Line{Rate: rate, Taxable: amt(t.col(row, "Taxable Value")), Cess: amt(t.col(row, "Cess Amount"))})
	}
	sort.Strings(posOrder)
	var out []B2CLPos
	for _, pos := range posOrder {
		nums := invOrder[pos]
		sort.Strings(nums)
		row := B2CLPos{Pos: pos}
		for _, inum := range nums {
			ia := byPos[pos][inum]
			row.Inv = append(row.Inv, B2CLInv{Inum: inum, Idt: ia.date, Val: ia.val, Itms: itemsFrom(ia.lines, false)})
		}
		out = append(out, row)
	}
	return out, rep, nil
}

// ImportCDNR reads cdnr.csv into Table 9B (credit/debit notes to registered
// recipients), grouping by recipient GSTIN.
func ImportCDNR(r io.Reader, supplierState string) ([]CDNRCtin, *Report, error) {
	t, err := readCSV(r)
	if err != nil {
		return nil, nil, err
	}
	rep := &Report{}
	byCtin := map[string][]CDNRNote{}
	var order []string
	for n, row := range t.rows {
		ctin := t.col(row, "GSTIN/UIN of Recipient")
		note := t.col(row, "Note Number")
		if ctin == "" || note == "" {
			continue
		}
		if err := gst.ValidateGSTIN(ctin); err != nil {
			rep.Add("cdnr", note, "ctin", "row %d: %v", n+2, err)
		}
		pos := posCode(t.col(row, "Place Of Supply"))
		rate, err := parseRate(t.col(row, "Rate"))
		if err != nil {
			rep.Add("cdnr", note, "rate", "row %d: %v", n+2, err)
		}
		intra := gst.IsIntraState(supplierState, pos)
		lines := []Line{{Rate: rate, Taxable: amt(t.col(row, "Taxable Value")), Cess: amt(t.col(row, "Cess Amount"))}}
		if _, ok := byCtin[ctin]; !ok {
			order = append(order, ctin)
		}
		byCtin[ctin] = append(byCtin[ctin], CDNRNote{
			Ntty:   strings.TrimSpace(t.col(row, "Note Type")),
			Nt_num: note,
			Nt_dt:  parseDate(t.col(row, "Note Date")),
			Pos:    pos,
			Rchrg:  yn(t.col(row, "Reverse Charge")),
			InvTyp: invTypeCode(t.col(row, "Note Supply Type")),
			Val:    amt(t.col(row, "Note Value")),
			Itms:   itemsFrom(lines, intra),
		})
	}
	sort.Strings(order)
	var out []CDNRCtin
	for _, ctin := range order {
		out = append(out, CDNRCtin{Ctin: ctin, Nt: byCtin[ctin]})
	}
	return out, rep, nil
}

// ImportCDNUR reads cdnur.csv into Table 9B for unregistered recipients.
func ImportCDNUR(r io.Reader) ([]CDNURRow, *Report, error) {
	t, err := readCSV(r)
	if err != nil {
		return nil, nil, err
	}
	rep := &Report{}
	var out []CDNURRow
	for n, row := range t.rows {
		note := t.col(row, "Note Number")
		if note == "" {
			continue
		}
		rate, err := parseRate(t.col(row, "Rate"))
		if err != nil {
			rep.Add("cdnur", note, "rate", "row %d: %v", n+2, err)
		}
		urType := strings.TrimSpace(t.col(row, "UR Type"))
		// EXPWP/EXPWOP notes carry no place of supply; B2CL is inter-state.
		lines := []Line{{Rate: rate, Taxable: amt(t.col(row, "Taxable Value")), Cess: amt(t.col(row, "Cess Amount"))}}
		out = append(out, CDNURRow{
			Typ:    urType,
			Ntty:   strings.TrimSpace(t.col(row, "Note Type")),
			Nt_num: note,
			Nt_dt:  parseDate(t.col(row, "Note Date")),
			Pos:    posCode(t.col(row, "Place Of Supply")),
			Val:    amt(t.col(row, "Note Value")),
			Itms:   itemsFrom(lines, false),
		})
	}
	return out, rep, nil
}

// ImportEXP reads exp.csv into Table 6A (exports), grouping by export type.
// WPAY (with payment of IGST) carries IGST; WOPAY carries none.
func ImportEXP(r io.Reader) ([]EXPType, *Report, error) {
	t, err := readCSV(r)
	if err != nil {
		return nil, nil, err
	}
	rep := &Report{}
	type invAgg struct {
		date, port, sbnum, sbdt string
		val                     gst.Amount
		items                   []ExpItm
	}
	byType := map[string]map[string]*invAgg{}
	var typeOrder []string
	invOrder := map[string][]string{}
	for n, row := range t.rows {
		inum := t.col(row, "Invoice Number")
		if inum == "" {
			continue
		}
		expType := strings.ToUpper(strings.TrimSpace(t.col(row, "Export Type")))
		rate, err := parseRate(t.col(row, "Rate"))
		if err != nil {
			rep.Add("exp", inum, "rate", "row %d: %v", n+2, err)
		}
		taxable := amt(t.col(row, "Taxable Value"))
		var iamt gst.Amount
		if expType == "WPAY" {
			iamt = gst.ComputeTax(taxable, rate, false).IGST
		}
		if _, ok := byType[expType]; !ok {
			byType[expType] = map[string]*invAgg{}
			typeOrder = append(typeOrder, expType)
		}
		ia := byType[expType][inum]
		if ia == nil {
			ia = &invAgg{
				date:  parseDate(t.col(row, "Invoice date")),
				port:  strings.TrimSpace(t.col(row, "Port Code")),
				sbnum: strings.TrimSpace(t.col(row, "Shipping Bill Number")),
				sbdt:  parseDate(t.col(row, "Shipping Bill Date")),
				val:   amt(t.col(row, "Invoice Value")),
			}
			byType[expType][inum] = ia
			invOrder[expType] = append(invOrder[expType], inum)
		}
		ia.items = append(ia.items, ExpItm{Txval: taxable, Rt: rate, Iamt: iamt, Csamt: amt(t.col(row, "Cess Amount"))})
	}
	sort.Strings(typeOrder)
	var out []EXPType
	for _, et := range typeOrder {
		nums := invOrder[et]
		sort.Strings(nums)
		sec := EXPType{ExpTyp: et}
		for _, inum := range nums {
			ia := byType[et][inum]
			sec.Inv = append(sec.Inv, EXPInv{
				Inum: inum, Idt: ia.date, Val: ia.val,
				Sbpcode: ia.port, Sbnum: ia.sbnum, Sbdt: ia.sbdt, Itms: ia.items,
			})
		}
		out = append(out, sec)
	}
	return out, rep, nil
}

// exemptSupplyType maps the official exempt-CSV description to a portal
// supply-type code.
var exemptSupplyType = map[string]string{
	"inter-state supplies to registered persons":   "INTRB2B",
	"intra-state supplies to registered persons":   "INTRAB2B",
	"inter-state supplies to unregistered persons": "INTRB2C",
	"intra-state supplies to unregistered persons": "INTRAB2C",
}

// ImportExempt reads exemp.csv into Table 8 (nil-rated/exempt/non-GST).
func ImportExempt(r io.Reader) (*NilSec, *Report, error) {
	t, err := readCSV(r)
	if err != nil {
		return nil, nil, err
	}
	rep := &Report{}
	sec := &NilSec{}
	for n, row := range t.rows {
		desc := strings.TrimSpace(t.col(row, "Description"))
		if desc == "" {
			continue
		}
		sply, ok := exemptSupplyType[strings.ToLower(desc)]
		if !ok {
			rep.Add("nil", strconv.Itoa(n+2), "description", "unrecognised supply type %q", desc)
			continue
		}
		sec.InvType = append(sec.InvType, NilRow{
			SplyTy:   sply,
			NilAmt:   amt(t.col(row, "Nil Rated Supplies")),
			ExptAmt:  amt(t.col(row, "Exempted(other than nil rated/non GST supply)")),
			NgsupAmt: amt(t.col(row, "Non-GST Supplies")),
		})
	}
	if len(sec.InvType) == 0 {
		return nil, rep, nil
	}
	return sec, rep, nil
}

// docNatureCode maps the Table 13 "Nature of Document" label to its code.
var docNatureCode = map[string]int{
	"invoices for outward supply":                           1,
	"invoices for inward supply from unregistered person":   2,
	"revised invoice":                                       3,
	"debit note":                                            4,
	"credit note":                                           5,
	"receipt voucher":                                       6,
	"payment voucher":                                       7,
	"refund voucher":                                        8,
	"delivery challan for job work":                         9,
	"delivery challan for supply on approval":               10,
	"delivery challan in case of liquid gas":                11,
	"delivery challan in cases other than by way of supply": 12,
}

// ImportDocs reads docs.csv into Table 13. Rows with a blank series are
// skipped (the template ships placeholder rows for unused document natures).
func ImportDocs(r io.Reader) (*DocIssue, *Report, error) {
	t, err := readCSV(r)
	if err != nil {
		return nil, nil, err
	}
	rep := &Report{}
	type entry struct {
		from, to      string
		total, cancel int
	}
	byNature := map[int][]entry{}
	var natureOrder []int
	for n, row := range t.rows {
		nature := strings.TrimSpace(t.col(row, "Nature of Document"))
		from := strings.TrimSpace(t.col(row, "Sr. No. From"))
		to := strings.TrimSpace(t.col(row, "Sr. No. To"))
		if nature == "" || from == "" {
			continue
		}
		code, ok := docNatureCode[strings.ToLower(nature)]
		if !ok {
			rep.Add("docs", strconv.Itoa(n+2), "nature", "unrecognised document nature %q", nature)
			continue
		}
		total, _ := strconv.Atoi(strings.TrimSpace(t.col(row, "Total Number")))
		cancel, _ := strconv.Atoi(strings.TrimSpace(t.col(row, "Cancelled")))
		if _, seen := byNature[code]; !seen {
			natureOrder = append(natureOrder, code)
		}
		byNature[code] = append(byNature[code], entry{from: from, to: to, total: total, cancel: cancel})
	}
	if len(natureOrder) == 0 {
		return nil, rep, nil
	}
	sort.Ints(natureOrder)
	di := &DocIssue{}
	for _, code := range natureOrder {
		det := DocDet{DocNum: code}
		for i, e := range byNature[code] {
			det.Docs = append(det.Docs, DocEntry{
				Num: i + 1, From: e.from, To: e.to, Totnum: e.total, Cancel: e.cancel, NetIssue: e.total - e.cancel,
			})
		}
		di.DocDet = append(di.DocDet, det)
	}
	return di, rep, nil
}
