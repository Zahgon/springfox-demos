package springfox

import (
	"regexp"
	"strings"
)

// PathSelector is the Go form of java.util.function.Predicate<String> as used
// by Docket.select().paths(...).
type PathSelector func(path string) bool

// Or combines selectors disjunctively, like Predicate.or.
func (p PathSelector) Or(other PathSelector) PathSelector {
	return func(path string) bool { return p(path) || other(path) }
}

// And combines selectors conjunctively, like Predicate.and.
func (p PathSelector) And(other PathSelector) PathSelector {
	return func(path string) bool { return p(path) && other(path) }
}

// Negate inverts a selector, like Predicate.negate.
func (p PathSelector) Negate() PathSelector {
	return func(path string) bool { return !p(path) }
}

// Regex is PathSelectors.regex. Springfox implements it with
// String.matches(pattern), which anchors the pattern to the whole path — a
// substring match is not enough.
func Regex(pattern string) PathSelector {
	re := regexp.MustCompile("^(?:" + pattern + ")$")
	return func(path string) bool { return re.MatchString(path) }
}

// Ant is PathSelectors.ant, matching an Ant-style path pattern where "*" spans
// one path segment and "**" spans any number.
func Ant(pattern string) PathSelector {
	re := regexp.MustCompile("^" + antToRegex(pattern) + "$")
	return func(path string) bool { return re.MatchString(path) }
}

func antToRegex(pattern string) string {
	var b strings.Builder
	for i := 0; i < len(pattern); i++ {
		switch {
		case strings.HasPrefix(pattern[i:], "**"):
			b.WriteString(".*")
			i++
		case pattern[i] == '*':
			b.WriteString("[^/]*")
		case pattern[i] == '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(string(pattern[i])))
		}
	}
	return b.String()
}

// Any is PathSelectors.any.
func Any() PathSelector { return func(string) bool { return true } }

// None is PathSelectors.none.
func None() PathSelector { return func(string) bool { return false } }

// Contains is the plain `input -> input.contains(s)` lambda boot-swagger's
// user-api Docket uses in place of a PathSelectors helper.
func Contains(s string) PathSelector {
	return func(path string) bool { return strings.Contains(path, s) }
}
