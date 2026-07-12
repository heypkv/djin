package gst

import "math"

// validRates are the GST tax rates djin recognises, in percent. Alongside the
// familiar slabs (5/12/18/28) these include the special rates that trip people
// up: 0.1 (notified exports), 0.25 (rough diamonds), 1/1.5/3/6/7.5 (various
// composition and notified goods), and 40 (the new top slab).
var validRates = []float64{0, 0.1, 0.25, 1, 1.5, 3, 5, 6, 7.5, 12, 18, 28, 40}

// ValidRates returns the recognised GST rates in percent.
func ValidRates() []float64 {
	out := make([]float64, len(validRates))
	copy(out, validRates)
	return out
}

// IsValidRate reports whether rate (in percent) is a recognised GST rate.
func IsValidRate(rate float64) bool {
	for _, r := range validRates {
		if math.Abs(r-rate) < 1e-9 {
			return true
		}
	}
	return false
}
