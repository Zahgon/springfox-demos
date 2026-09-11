// Package javalang reproduces the java.lang behaviour the ported code depends
// on. These are not utilities: the original's output goes through Java's own
// formatting, so anything a response body or a generated document exposes has
// to be reproduced rather than approximated with Go's defaults.
package javalang

import (
	"math"
	"strconv"
	"strings"
)

// The spellings java.lang.Double.toString gives the non-finite values. Go's
// strconv writes "NaN", "+Inf" and "-Inf" instead.
const (
	NaNString              = "NaN"
	PositiveInfinityString = "Infinity"
	NegativeInfinityString = "-Infinity"
)

// DoubleToString reproduces java.lang.Double.toString.
//
// Go's strconv and Java's Double.toString agree on which digits round-trip but
// not on how to lay them out: Java always keeps at least one digit after the
// decimal point, switches to scientific notation outside [1e-3, 1e7), spells
// the exponent marker "E" with no "+" sign, and prints the non-finite values as
// "NaN", "Infinity" and "-Infinity". boot-webmvc's /hello/double writes the
// value with Java string concatenation, so this layout is part of that
// endpoint's response body.
func DoubleToString(d float64) string {
	switch {
	case math.IsNaN(d):
		return NaNString
	case math.IsInf(d, 1):
		return PositiveInfinityString
	case math.IsInf(d, -1):
		return NegativeInfinityString
	}
	sign := ""
	if math.Signbit(d) {
		sign = "-"
	}
	a := math.Abs(d)
	if a == 0 {
		return sign + "0.0"
	}

	// strconv gives the shortest round-tripping digits plus a decimal exponent.
	mant, expStr, _ := strings.Cut(strconv.FormatFloat(a, 'e', -1, 64), "e")
	exp, err := strconv.Atoi(expStr)
	if err != nil {
		return sign + strconv.FormatFloat(a, 'g', -1, 64)
	}
	digits := strings.Replace(mant, ".", "", 1)

	if a >= 1e-3 && a < 1e7 {
		return sign + plainDecimal(digits, exp)
	}
	frac := digits[1:]
	if frac == "" {
		frac = "0"
	}
	return sign + digits[:1] + "." + frac + "E" + strconv.Itoa(exp)
}

// plainDecimal lays out digits[0].digits[1:] x 10^exp without an exponent,
// always emitting at least one fractional digit.
func plainDecimal(digits string, exp int) string {
	if exp < 0 {
		return "0." + strings.Repeat("0", -exp-1) + digits
	}
	intLen := exp + 1
	if intLen >= len(digits) {
		return digits + strings.Repeat("0", intLen-len(digits)) + ".0"
	}
	return digits[:intLen] + "." + digits[intLen:]
}

// LongToString reproduces java.lang.Long.toString, which agrees with strconv
// for every value; it exists so call sites read the same way as the original's
// string concatenation.
func LongToString(v int64) string { return strconv.FormatInt(v, 10) }

// Double is a Java `double` for serialisation purposes. Jackson writes a double
// with Double.toString, so 10 is written as "10.0" where Go's encoding/json
// would write "10".
type Double float64

// MarshalJSON writes the value the way Jackson writes a Java double.
func (d Double) MarshalJSON() ([]byte, error) { return []byte(DoubleToString(float64(d))), nil }
