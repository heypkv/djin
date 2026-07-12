package gst

import (
	"encoding/json"
	"testing"
)

func TestParseAmount(t *testing.T) {
	cases := []struct {
		in   string
		want Amount
	}{
		{"719.00", 71900},
		{"719", 71900},
		{"862.80", 86280},
		{"99", 9900},
		{"", 0},
		{"4,981", 498100},   // thousands separator (official CSV quirk)
		{"76000.45", 7600045},
		{"-25000", -2500000},
		{" 250000.01 ", 25000001},
		{"1.005", 101}, // third decimal rounds half-up: 1.005 -> 1.01
	}
	for _, c := range cases {
		got, err := ParseAmount(c.in)
		if err != nil {
			t.Errorf("ParseAmount(%q) error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseAmount(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestAmountMarshalJSON(t *testing.T) {
	cases := []struct {
		in   Amount
		want string
	}{
		{71900, "719.00"},
		{6471, "64.71"},
		{84800, "848.00"},
		{0, "0.00"},
		{-42, "-0.42"},
		{5, "0.05"},
	}
	for _, c := range cases {
		b, err := json.Marshal(c.in)
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != c.want {
			t.Errorf("Marshal(%d) = %s, want %s", int64(c.in), b, c.want)
		}
	}
}

func TestComputeTaxIntra(t *testing.T) {
	// Golden invoice line: taxable 719.00 @ 18% intra -> CGST=SGST=64.71.
	tax := ComputeTax(71900, 18, true)
	if tax.CGST != 6471 || tax.SGST != 6471 {
		t.Errorf("719.00@18%% intra = CGST %s SGST %s, want 64.71/64.71", tax.CGST, tax.SGST)
	}
	if tax.CGST != tax.SGST {
		t.Error("intra-state CGST must equal SGST")
	}
	if tax.IGST != 0 {
		t.Errorf("intra-state IGST = %s, want 0", tax.IGST)
	}
	if tax.Total() != 12942 {
		t.Errorf("total tax = %s, want 129.42", tax.Total())
	}

	// 862.80 @ 18% intra -> 155.30/2 = 77.65 each (rounds .2 down).
	tax = ComputeTax(86280, 18, true)
	if tax.CGST != 7765 || tax.SGST != 7765 {
		t.Errorf("862.80@18%% = %s/%s, want 77.65/77.65", tax.CGST, tax.SGST)
	}

	// Subscription line 83.90 @ 18% intra -> 7.55 each.
	tax = ComputeTax(8390, 18, true)
	if tax.CGST != 755 || tax.SGST != 755 {
		t.Errorf("83.90@18%% = %s/%s, want 7.55/7.55", tax.CGST, tax.SGST)
	}
}

func TestComputeTaxInter(t *testing.T) {
	tax := ComputeTax(71900, 18, false)
	if tax.IGST != 12942 {
		t.Errorf("719.00@18%% inter IGST = %s, want 129.42", tax.IGST)
	}
	if tax.CGST != 0 || tax.SGST != 0 {
		t.Error("inter-state must have zero CGST/SGST")
	}
}

func TestComputeTaxFractionalRate(t *testing.T) {
	// 0.25% on 5,887.00 intra -> tax 14.7175 -> 7.36 each (half rate 0.125%).
	tax := ComputeTax(588700, 0.25, true)
	// 588700 * 125 / 100000 = 735.875 -> 736 paise = 7.36
	if tax.CGST != 736 || tax.SGST != 736 {
		t.Errorf("5887.00@0.25%% = %s/%s, want 7.36/7.36", tax.CGST, tax.SGST)
	}
}

func TestRoundToRupee(t *testing.T) {
	// Golden B2B invoice value: 719.00 + 64.71 + 64.71 = 848.42 -> 848.00.
	rounded, roundOff := RoundToRupee(84842)
	if rounded != 84800 {
		t.Errorf("RoundToRupee(848.42) = %s, want 848.00", rounded)
	}
	if roundOff != -42 {
		t.Errorf("round-off = %s, want -0.42", roundOff)
	}
	// .50 rounds away from zero -> up.
	if r, _ := RoundToRupee(84850); r != 84900 {
		t.Errorf("RoundToRupee(848.50) = %s, want 849.00", r)
	}
	if r, _ := RoundToRupee(84849); r != 84800 {
		t.Errorf("RoundToRupee(848.49) = %s, want 848.00", r)
	}
}

func TestIsValidRate(t *testing.T) {
	for _, r := range []float64{0, 0.1, 0.25, 1, 1.5, 3, 5, 6, 7.5, 12, 18, 28, 40} {
		if !IsValidRate(r) {
			t.Errorf("IsValidRate(%v) = false, want true", r)
		}
	}
	for _, r := range []float64{2, 4, 9, 19, 100} {
		if IsValidRate(r) {
			t.Errorf("IsValidRate(%v) = true, want false", r)
		}
	}
}

func TestStateTable(t *testing.T) {
	s, ok := StateByCode("10")
	if !ok || s.Name != "Bihar" {
		t.Errorf("StateByCode(10) = %+v, %v, want Bihar", s, ok)
	}
	if _, ok := StateByCode("00"); ok {
		t.Error("StateByCode(00) should be unknown")
	}
	if !IsIntraState("10", "10") || IsIntraState("10", "27") {
		t.Error("IsIntraState logic wrong")
	}
}
