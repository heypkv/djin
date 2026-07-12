package gst

import "testing"

func TestValidateGSTIN_RealVectors(t *testing.T) {
	// Real, in-use GSTINs that must validate (both Bihar, code 10).
	valid := []string{
		"10AAICH1439H1ZZ",
		"10CLZPS0601K1ZY",
	}
	for _, g := range valid {
		if err := ValidateGSTIN(g); err != nil {
			t.Errorf("ValidateGSTIN(%q) = %v, want nil", g, err)
		}
	}
}

func TestValidateGSTIN_CorruptedChecksum(t *testing.T) {
	// Same GSTIN with the check digit flipped Z->Y must fail on checksum.
	if err := ValidateGSTIN("10AAICH1439H1ZY"); err == nil {
		t.Fatal("corrupted check digit accepted, want checksum error")
	}
	// A transposed body character must also fail the checksum.
	if err := ValidateGSTIN("10AAICH1439H1ZZ"[:5] + "X" + "10AAICH1439H1ZZ"[6:]); err == nil {
		t.Fatal("corrupted body accepted, want checksum error")
	}
}

func TestValidateGSTIN_Structural(t *testing.T) {
	cases := map[string]string{
		"too short":          "10AAICH1439H1Z",
		"bad state code":     "00AAICH1439H1Z5",
		"bad PAN shape":      "10AA1CH1439H1ZZ",
		"illegal character":  "10AAICH1439H1Z!",
		"lowercase accepted": "10aaich1439h1zz", // normalised, should PASS
	}
	for name, g := range cases {
		err := ValidateGSTIN(g)
		if name == "lowercase accepted" {
			if err != nil {
				t.Errorf("%s: ValidateGSTIN(%q) = %v, want nil", name, g, err)
			}
			continue
		}
		if err == nil {
			t.Errorf("%s: ValidateGSTIN(%q) = nil, want error", name, g)
		}
	}
}

func TestStateCodeOf(t *testing.T) {
	if got := StateCodeOf("10AAICH1439H1ZZ"); got != "10" {
		t.Errorf("StateCodeOf = %q, want 10", got)
	}
}
