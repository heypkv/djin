package gst

import (
	"fmt"
	"regexp"
	"strings"
)

// gstinChars is the mod-36 alphabet used by the GSTIN check-digit algorithm:
// digits 0-9 then A-Z, valued 0..35.
const gstinChars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

// panRe matches the PAN embedded in a GSTIN (chars 3-12): five letters, four
// digits, one letter.
var panRe = regexp.MustCompile(`^[A-Z]{5}[0-9]{4}[A-Z]$`)

// gstinCheckDigit computes the mod-36 check character over the first 14
// characters of a GSTIN (the GSTN algorithm: alternate weight 2/1 from the
// right, fold each product's quotient and remainder, checksum = 36 - sum%36).
func gstinCheckDigit(first14 string) (byte, error) {
	factor := 2
	sum := 0
	mod := len(gstinChars)
	for i := len(first14) - 1; i >= 0; i-- {
		cp := strings.IndexByte(gstinChars, first14[i])
		if cp < 0 {
			return 0, fmt.Errorf("invalid character %q", first14[i])
		}
		digit := factor * cp
		if factor == 2 {
			factor = 1
		} else {
			factor = 2
		}
		digit = digit/mod + digit%mod
		sum += digit
	}
	check := (mod - (sum % mod)) % mod
	return gstinChars[check], nil
}

// ValidateGSTIN checks a GSTIN's length, alphabet, state code, embedded PAN
// shape, and mod-36 checksum. It returns a descriptive error naming what
// failed, or nil if the GSTIN is well-formed.
func ValidateGSTIN(g string) error {
	g = strings.ToUpper(strings.TrimSpace(g))
	if len(g) != 15 {
		return fmt.Errorf("GSTIN %q: must be 15 characters, got %d", g, len(g))
	}
	for i := 0; i < 15; i++ {
		if strings.IndexByte(gstinChars, g[i]) < 0 {
			return fmt.Errorf("GSTIN %q: invalid character %q at position %d", g, g[i], i+1)
		}
	}
	if _, ok := StateByCode(g[:2]); !ok {
		return fmt.Errorf("GSTIN %q: unknown state code %q", g, g[:2])
	}
	if !panRe.MatchString(g[2:12]) {
		return fmt.Errorf("GSTIN %q: characters 3-12 %q are not a valid PAN", g, g[2:12])
	}
	cd, err := gstinCheckDigit(g[:14])
	if err != nil {
		return fmt.Errorf("GSTIN %q: %w", g, err)
	}
	if g[14] != cd {
		return fmt.Errorf("GSTIN %q: checksum mismatch (expected %c, got %c)", g, cd, g[14])
	}
	return nil
}

// IsValidGSTIN reports whether g is a well-formed GSTIN.
func IsValidGSTIN(g string) bool { return ValidateGSTIN(g) == nil }

// StateCodeOf returns the two-character state code prefix of a GSTIN.
func StateCodeOf(gstin string) string {
	g := strings.TrimSpace(gstin)
	if len(g) < 2 {
		return ""
	}
	return g[:2]
}
