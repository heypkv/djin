// Package gst holds the GST domain primitives shared across djin: money in
// integer paise with explicit rounding, GST rate tables, state/POS codes, and
// GSTIN checksum validation. It performs no I/O.
package gst

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Amount is a money value in integer paise (1 rupee = 100 paise). Working in
// integers keeps rounding explicit and reproducible — the portal rejects
// returns whose tax does not reconcile to the paise.
type Amount int64

// Rupees returns the amount as a float in rupees (for display only; never for
// further arithmetic).
func (a Amount) Rupees() float64 { return float64(a) / 100 }

// String renders the amount as rupees with two decimals, e.g. "848.00".
func (a Amount) String() string {
	neg := a < 0
	v := int64(a)
	if neg {
		v = -v
	}
	s := fmt.Sprintf("%d.%02d", v/100, v%100)
	if neg {
		s = "-" + s
	}
	return s
}

// MarshalJSON emits the amount as a JSON number with two decimals ("848.00").
// The GST portal accepts rupee amounts to two places; fixed formatting keeps
// the output visually aligned with the portal's own JSON.
func (a Amount) MarshalJSON() ([]byte, error) {
	return []byte(a.String()), nil
}

// ParseAmount parses a rupee string into paise. It tolerates the quirks of the
// official CSV templates: thousands separators ("4,981"), surrounding spaces,
// an empty string (treated as zero), signs, and more than two decimals (which
// are rounded half away from zero).
func ParseAmount(s string) (Amount, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	if s == "" {
		return 0, nil
	}
	neg := false
	switch s[0] {
	case '-':
		neg = true
		s = s[1:]
	case '+':
		s = s[1:]
	}
	intPart, fracPart, _ := strings.Cut(s, ".")
	if intPart == "" {
		intPart = "0"
	}
	rupees, err := strconv.ParseInt(intPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse amount %q: %w", s, err)
	}
	paise := rupees * 100
	if fracPart != "" {
		for _, c := range fracPart {
			if c < '0' || c > '9' {
				return 0, fmt.Errorf("parse amount %q: non-digit in fraction", s)
			}
		}
		// Take the first two fractional digits as paise, rounding on the third.
		frac := fracPart
		var extra byte = '0'
		if len(frac) > 2 {
			extra = frac[2]
			frac = frac[:2]
		}
		for len(frac) < 2 {
			frac += "0"
		}
		p, _ := strconv.ParseInt(frac, 10, 64)
		if extra >= '5' {
			p++
		}
		paise += p
	}
	if neg {
		paise = -paise
	}
	return Amount(paise), nil
}

// divRoundHalfUp divides num by den rounding to nearest, ties away from zero.
func divRoundHalfUp(num, den int64) int64 {
	if den < 0 {
		num, den = -num, -den
	}
	if num >= 0 {
		return (num*2 + den) / (den * 2)
	}
	return -((-num*2 + den) / (den * 2))
}

// RateMilli converts a percentage rate to integer milli-percent so fractional
// GST rates (0.1, 0.25, 1.5, 7.5) stay exact: 18 -> 18000, 0.25 -> 250.
func RateMilli(rate float64) int64 {
	return int64(math.Round(rate * 1000))
}

// Tax holds the split tax on a taxable value.
type Tax struct {
	CGST Amount
	SGST Amount
	IGST Amount
	Cess Amount
}

// Total returns the sum of all tax components.
func (t Tax) Total() Amount { return t.CGST + t.SGST + t.IGST + t.Cess }

// ComputeTax splits tax on a taxable value at the given rate. For intra-state
// supply CGST == SGST, each computed at half the rate and rounded half-up
// independently, guaranteeing they are equal (a portal requirement). For
// inter-state supply the whole tax lands in IGST. Cess must be supplied by the
// caller (it is item-specific, not a percentage of the taxable value here).
func ComputeTax(taxable Amount, rate float64, intra bool) Tax {
	mp := RateMilli(rate)
	if intra {
		half := Amount(divRoundHalfUp(int64(taxable)*mp, 200000))
		return Tax{CGST: half, SGST: half}
	}
	return Tax{IGST: Amount(divRoundHalfUp(int64(taxable)*mp, 100000))}
}

// RoundToRupee rounds paise to the nearest whole rupee (ties away from zero)
// and returns the rounded amount together with the round-off adjustment
// (rounded - original), which is the value that flows into an invoice's
// round-off line.
func RoundToRupee(p Amount) (rounded, roundOff Amount) {
	r := Amount(divRoundHalfUp(int64(p), 100) * 100)
	return r, r - p
}
