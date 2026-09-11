package javalang

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"testing"
)

// knownJDK8Divergences are the inputs on which OpenJDK 8/11's legacy
// FloatingDecimal emits more significant digits than the shortest decimal that
// round-trips to the same double. They are instances of JDK-4511638, "Double
// .toString(double) sometimes produces inaccurate results", which OpenJDK fixed
// in JDK 19 by switching to the shortest form — the form this package produces.
//
// The map records the value the pinned Java 8 baseline emitted, so that the
// differential test below still guards every other value and would fail if a
// *new* divergence appeared.
var knownJDK8Divergences = map[string]string{
	// Subnormals: the legacy algorithm keeps a second significant digit.
	"5e-324":   "4.9E-324",
	"5e-323":   "4.9E-323",
	"6e-323":   "5.9E-323",
	"7e-323":   "6.9E-323",
	"8e-323":   "7.9E-323",
	"9e-323":   "8.9E-323",
	"1.6e-322": "1.58E-322",
	// The classic case: 1e23 is the shortest decimal that reads back to this
	// double, but the legacy algorithm generates digits from the exact value
	// and never rounds up across the power of ten.
	"1e+23":  "9.999999999999999E22",
	"-1e+23": "-9.999999999999999E22",
	// Large integers where the legacy algorithm emits the exact value rather
	// than the shortest representation of it.
	"7.161075118865759e+16":  "7.1610751188657592E16",
	"-2.58586360363498e+17":  "-2.58586360363497984E17",
	"-4.667206712619586e+17": "-4.6672067126195859E17",
}

// TestDoubleToStringAgainstJava is a differential test against the original.
//
// testdata/double_tostring_java.tsv was captured by querying the unmodified
// Java boot-webmvc application's /hello/double endpoint — whose body is
// "Value " + Double.toString(count) — once for each of 639 values covering the
// subnormal range, powers of ten across the whole exponent range, the
// decimal/scientific switchover, and randomly generated bit patterns. Each line
// is the query value and the string the JDK produced.
func TestDoubleToStringAgainstJava(t *testing.T) {
	f, err := os.Open("testdata/double_tostring_java.tsv")
	if err != nil {
		t.Fatalf("opening the captured corpus: %v", err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	checked, diverged := 0, 0
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		input, java, ok := strings.Cut(line, "\t")
		if !ok {
			t.Fatalf("malformed corpus line: %q", line)
		}
		v, err := strconv.ParseFloat(input, 64)
		if err != nil {
			t.Fatalf("the corpus value %q does not parse: %v", input, err)
		}
		checked++
		got := DoubleToString(v)
		if got == java {
			if _, listed := knownJDK8Divergences[input]; listed {
				t.Errorf("%s is listed as a JDK 8 divergence but now agrees (%s); remove it from the list", input, got)
			}
			continue
		}
		want, listed := knownJDK8Divergences[input]
		if !listed {
			t.Errorf("DoubleToString(%s): java = %s, go = %s", input, java, got)
			continue
		}
		if want != java {
			t.Errorf("%s: the corpus now says %s, but the recorded JDK 8 divergence was %s", input, java, want)
		}
		// The two strings must at least denote the same double.
		back, err := strconv.ParseFloat(got, 64)
		if err != nil || back != v {
			t.Errorf("DoubleToString(%s) = %s, which does not read back to the same double", input, got)
		}
		diverged++
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("reading the corpus: %v", err)
	}
	if checked < 600 {
		t.Fatalf("only %d values were checked; the corpus looks truncated", checked)
	}
	if diverged != len(knownJDK8Divergences) {
		t.Errorf("hit %d of the %d recorded JDK 8 divergences; the corpus and the list are out of step",
			diverged, len(knownJDK8Divergences))
	}
	t.Logf("%d of %d values are byte-identical to the Java baseline; %d differ, all of them recorded JDK-4511638 cases",
		checked-diverged, checked, diverged)
}
