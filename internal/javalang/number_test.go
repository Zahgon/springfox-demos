package javalang

import (
	"math"
	"testing"
)

// TestDoubleToString covers java.lang.Double.toString, which the original got
// from the JDK and boot-webmvc's /hello/double exposes verbatim in its body.
func TestDoubleToString(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		// Decimal notation applies on [1e-3, 1e7), always with a fractional digit.
		{0, "0.0"},
		{1, "1.0"},
		{10, "10.0"},
		{3.5, "3.5"},
		{0.001, "0.001"},
		{0.0015, "0.0015"},
		{1234567, "1234567.0"},
		{9999999, "9999999.0"},
		{123.456, "123.456"},
		// Outside that band the layout is scientific, with "E" and no "+".
		{1e7, "1.0E7"},
		{1e-4, "1.0E-4"},
		{0.0001, "1.0E-4"},
		{1e21, "1.0E21"},
		{1.5e-9, "1.5E-9"},
		{12345678, "1.2345678E7"},
		// Signs and the non-finite values.
		{-0.0, "0.0"},
		{-3.5, "-3.5"},
		{-1e-4, "-1.0E-4"},
		{math.MaxFloat64, "1.7976931348623157E308"},
		// OpenJDK 8 prints "4.9E-324" here; see knownJDK8Divergences in
		// double_differential_test.go.
		{math.SmallestNonzeroFloat64, "5.0E-324"},
	}
	for _, c := range cases {
		if got := DoubleToString(c.in); got != c.want {
			t.Errorf("DoubleToString(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestDoubleToStringNegativeZero covers the negative zero Go's == cannot
// distinguish from positive zero.
func TestDoubleToStringNegativeZero(t *testing.T) {
	if got := DoubleToString(math.Copysign(0, -1)); got != "-0.0" {
		t.Errorf("DoubleToString(-0.0) = %q, want %q", got, "-0.0")
	}
}

// TestDoubleToStringNonFinite covers NaN and the infinities, whose Java
// spellings differ from Go's.
func TestDoubleToStringNonFinite(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{math.NaN(), "NaN"},
		{math.Inf(1), "Infinity"},
		{math.Inf(-1), "-Infinity"},
	}
	for _, c := range cases {
		if got := DoubleToString(c.in); got != c.want {
			t.Errorf("DoubleToString(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestDoubleMarshalsLikeJackson covers the JSON form, which Jackson writes with
// Double.toString and Go's encoding/json would otherwise write as an integer.
func TestDoubleMarshalsLikeJackson(t *testing.T) {
	cases := []struct {
		in   Double
		want string
	}{
		{Double(10), "10.0"},
		{Double(3.5), "3.5"},
		{Double(0), "0.0"},
	}
	for _, c := range cases {
		got, err := c.in.MarshalJSON()
		if err != nil {
			t.Fatalf("marshalling %v: %v", float64(c.in), err)
		}
		if string(got) != c.want {
			t.Errorf("marshal(%v) = %s, want %s", float64(c.in), got, c.want)
		}
	}
}

// TestLongToString covers java.lang.Long.toString.
func TestLongToString(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0"},
		{-1, "-1"},
		{math.MaxInt64, "9223372036854775807"},
		{math.MinInt64, "-9223372036854775808"},
	}
	for _, c := range cases {
		if got := LongToString(c.in); got != c.want {
			t.Errorf("LongToString(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}
