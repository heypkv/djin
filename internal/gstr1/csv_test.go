package gstr1

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/heypkv/djin/internal/gst"
)

func openCSV(t *testing.T, name string) *os.File {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", "csv", name))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func TestImportB2B(t *testing.T) {
	// supplierState 12 (Arunachal) matches the template's recipient rows so a
	// few lines are intra-state; the importer must not panic on the mix.
	rows, rep, err := ImportB2B(openCSV(t, "b2b,sez,de.csv"), "12")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("no B2B ctins parsed")
	}
	// Two distinct recipients in the template.
	if len(rows) != 2 {
		t.Errorf("got %d ctins, want 2", len(rows))
	}
	// Invoice "A/1003" appears twice (two rates) and must consolidate to one
	// invoice with two rate-wise items.
	var found bool
	for _, c := range rows {
		for _, inv := range c.Inv {
			if inv.Inum == "A/1003" {
				found = true
				if len(inv.Itms) != 2 {
					t.Errorf("A/1003: got %d items, want 2 rate lines", len(inv.Itms))
				}
			}
		}
	}
	if !found {
		t.Error("invoice A/1003 not found")
	}
	// The template's recipients are valid GSTINs, so no ctin errors expected.
	for _, is := range rep.Issues {
		if is.Field == "ctin" {
			t.Errorf("unexpected ctin issue: %s", is)
		}
	}
}

func TestImportB2CS(t *testing.T) {
	// supplierState 37: an intra-state row (POS 37) becomes INTRA with CGST/SGST.
	rows, _, err := ImportB2CS(openCSV(t, "b2cs.csv"), "37")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("no B2CS rows parsed")
	}
	for _, r := range rows {
		if r.Pos == "37" && r.SplyTy != "INTRA" {
			t.Errorf("POS 37 with supplier 37 should be INTRA, got %s", r.SplyTy)
		}
		if r.Pos != "37" && r.SplyTy != "INTER" {
			t.Errorf("POS %s should be INTER, got %s", r.Pos, r.SplyTy)
		}
		if r.SplyTy == "INTRA" && r.Camt == nil {
			t.Error("intra row missing camt")
		}
	}
}

func TestImportB2CL(t *testing.T) {
	rows, _, err := ImportB2CL(openCSV(t, "b2cl.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("no B2CL rows")
	}
	// Inter-state: every item must carry IGST, never CGST/SGST.
	for _, pos := range rows {
		for _, inv := range pos.Inv {
			for _, it := range inv.Itms {
				if it.ItmDet.Iamt == nil {
					t.Errorf("%s: B2CL item missing iamt", inv.Inum)
				}
				if it.ItmDet.Camt != nil {
					t.Errorf("%s: B2CL item should not have camt", inv.Inum)
				}
			}
		}
	}
}

func TestImportEXP(t *testing.T) {
	types, _, err := ImportEXP(openCSV(t, "exp.csv"))
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, ty := range types {
		seen[ty.ExpTyp] = true
		for _, inv := range ty.Inv {
			for _, it := range inv.Itms {
				if ty.ExpTyp == "WOPAY" && it.Iamt != 0 {
					t.Errorf("WOPAY invoice %s should carry no IGST, got %s", inv.Inum, it.Iamt)
				}
			}
		}
	}
	if !seen["WPAY"] || !seen["WOPAY"] {
		t.Errorf("expected both WPAY and WOPAY export types, got %v", seen)
	}
	// WPAY invoice 81521 has two rate lines and non-zero IGST.
	var igst gst.Amount
	for _, ty := range types {
		if ty.ExpTyp != "WPAY" {
			continue
		}
		for _, inv := range ty.Inv {
			if inv.Inum == "81521" {
				for _, it := range inv.Itms {
					igst += it.Iamt
				}
			}
		}
	}
	if igst == 0 {
		t.Error("WPAY invoice 81521 should have IGST")
	}
}

func TestImportCDNRandCDNUR(t *testing.T) {
	cdnr, _, err := ImportCDNR(openCSV(t, "cdnr.csv"), "19")
	if err != nil {
		t.Fatal(err)
	}
	if len(cdnr) == 0 {
		t.Fatal("no CDNR notes")
	}
	cdnur, _, err := ImportCDNUR(openCSV(t, "cdnur.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cdnur) == 0 {
		t.Fatal("no CDNUR rows")
	}
	// POS is passed through from the CSV verbatim: note 90011 (blank cell) has
	// no POS, note 90015 (04-Chandigarh) keeps code "04".
	pos := map[string]string{}
	for _, r := range cdnur {
		pos[r.Nt_num] = r.Pos
	}
	if pos["90011"] != "" {
		t.Errorf("note 90011 POS = %q, want empty", pos["90011"])
	}
	if pos["90015"] != "04" {
		t.Errorf("note 90015 POS = %q, want 04", pos["90015"])
	}
}

func TestImportExempt(t *testing.T) {
	sec, rep, err := ImportExempt(openCSV(t, "exemp.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if !rep.OK() {
		t.Fatalf("exempt import issues: %s", rep.Error())
	}
	if sec == nil || len(sec.InvType) != 4 {
		t.Fatalf("expected 4 exempt rows, got %+v", sec)
	}
}

func TestImportDocs(t *testing.T) {
	di, rep, err := ImportDocs(openCSV(t, "docs_clean.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if !rep.OK() {
		t.Fatalf("docs import issues: %s", rep.Error())
	}
	if di == nil {
		t.Fatal("no doc_issue")
	}
	// Nature 1 (outward supply) has two series; nature 4 (debit note) one.
	byNum := map[int]int{}
	for _, d := range di.DocDet {
		byNum[d.DocNum] = len(d.Docs)
	}
	if byNum[1] != 2 {
		t.Errorf("nature 1 should have 2 series, got %d", byNum[1])
	}
	if byNum[4] != 1 {
		t.Errorf("nature 4 should have 1 series, got %d", byNum[4])
	}
	// net_issue = total - cancelled for the first series (51235 - 6123).
	first := di.DocDet[0].Docs[0]
	if first.NetIssue != first.Totnum-first.Cancel {
		t.Errorf("net_issue mismatch: %+v", first)
	}
}
