package gstr1

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestGoldenGSTR1 reconstructs the invoices behind a real, accepted GSTR-1
// upload and asserts djin's emitter reproduces it. The comparison is semantic:
// both documents are parsed into generic maps (numbers as float64) and
// deep-compared, so key ordering and equivalent numeric formatting (848 vs
// 848.00) do not matter.
func TestGoldenGSTR1(t *testing.T) {
	in := loadInput(t, filepath.Join("testdata", "golden_input.json"))

	portal, rep := Build(in)
	if !rep.OK() {
		t.Fatalf("Build reported issues:\n%s", rep.Error())
	}

	got := toNormalized(t, portal)
	want := loadNormalized(t, filepath.Join("testdata", "golden_GSTR1_10AAICH1439H1ZZ_062026.json"))

	if !reflect.DeepEqual(got, want) {
		gj, _ := json.MarshalIndent(got, "", "  ")
		wj, _ := json.MarshalIndent(want, "", "  ")
		t.Fatalf("emitted upload does not match golden.\n--- got ---\n%s\n--- want ---\n%s", gj, wj)
	}
}

func TestGoldenFP(t *testing.T) {
	// QRMP quarterly filer: any month in Apr-Jun -> fp 062026.
	if got := ComputeFP(4, 2026, true); got != "062026" {
		t.Errorf("ComputeFP(Apr,2026,QRMP) = %q, want 062026", got)
	}
	if got := ComputeFP(6, 2026, true); got != "062026" {
		t.Errorf("ComputeFP(Jun,2026,QRMP) = %q, want 062026", got)
	}
	// Monthly filer keeps its own month.
	if got := ComputeFP(4, 2026, false); got != "042026" {
		t.Errorf("ComputeFP(Apr,2026,monthly) = %q, want 042026", got)
	}
	// Jan-Mar quarter -> 03.
	if got := ComputeFP(1, 2027, true); got != "032027" {
		t.Errorf("ComputeFP(Jan,2027,QRMP) = %q, want 032027", got)
	}
}

func loadInput(t *testing.T, path string) ReturnInput {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var in ReturnInput
	if err := json.Unmarshal(b, &in); err != nil {
		t.Fatalf("parse input %s: %v", path, err)
	}
	return in
}

// toNormalized marshals v then re-parses it into a generic value so numbers
// become float64 and object keys become unordered maps.
func toNormalized(t *testing.T, v any) any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func loadNormalized(t *testing.T, path string) any {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("parse golden %s: %v", path, err)
	}
	return out
}
