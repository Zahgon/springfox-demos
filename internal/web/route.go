package web

import (
	"regexp"
	"sort"
	"strings"

	"github.com/springfox/springfox-demos/internal/api"
)

// HandlerFunc is the Go form of a Spring handler method.
type HandlerFunc func(*Request) (ResponseEntity, error)

// Route is one request mapping: the matching conditions Spring derives from
// @RequestMapping, the handler, and the documentation metadata Springfox used
// to derive from annotations and reflection.
type Route struct {
	Method   string
	Pattern  string
	Consumes []string
	Produces []string
	// Params are `params = "x=TX"` conditions. All must hold for the route to
	// match.
	Params []ParamCondition
	// Accept restricts the route to requests whose Accept header admits one of
	// these media types, reproducing RequestPredicates.accept.
	Accept []string
	// AcceptExplicit narrows Accept so that a wildcard Accept header does not
	// match. Spring's content negotiation uses it to prefer a JSON handler over
	// an HTML one when the client expresses no preference.
	AcceptExplicit bool
	Handler        HandlerFunc
	// Op is the operation's documentation. A zero Op means the route is not
	// documented at all.
	Op api.Operation

	matcher *pathMatcher
	// registered is the order the route was added in. Springfox hands out
	// operation ids in that order, which the match order must not disturb.
	registered int
}

// ParamCondition is a single `name=value` request-parameter condition.
type ParamCondition struct {
	Name  string
	Value string
}

// pathMatcher compiles a URI template such as "/category/{id}/map".
type pathMatcher struct {
	re       *regexp.Regexp
	varNames []string
	// literals counts the non-variable path segments, used to order routes the
	// way Spring's AntPatternComparator does: more specific patterns first.
	literals int
	vars     int
}

var templateVar = regexp.MustCompile(`\{([^/{}]+)\}`)

func compilePattern(pattern string) *pathMatcher {
	m := &pathMatcher{}
	var b strings.Builder
	b.WriteString("^")
	for _, seg := range strings.Split(strings.TrimPrefix(pattern, "/"), "/") {
		b.WriteString("/")
		if loc := templateVar.FindStringSubmatch(seg); loc != nil && loc[0] == seg {
			m.varNames = append(m.varNames, loc[1])
			m.vars++
			// "{*path}" is WebFlux's trailing-segment variable.
			if strings.HasPrefix(loc[1], "*") {
				m.varNames[len(m.varNames)-1] = strings.TrimPrefix(loc[1], "*")
				b.WriteString(`(.*)`)
				continue
			}
			b.WriteString(`([^/]+)`)
			continue
		}
		if seg == "**" {
			m.vars++
			b.WriteString(`.*`)
			continue
		}
		m.literals++
		b.WriteString(regexp.QuoteMeta(seg))
	}
	b.WriteString("$")
	m.re = regexp.MustCompile(b.String())
	return m
}

func (m *pathMatcher) match(path string) (map[string]string, bool) {
	g := m.re.FindStringSubmatch(path)
	if g == nil {
		return nil, false
	}
	if len(m.varNames) == 0 {
		return nil, true
	}
	vars := make(map[string]string, len(m.varNames))
	for i, name := range m.varNames {
		vars[name] = g[i+1]
	}
	return vars, true
}

// sortRoutes orders routes most-specific-first: more literal segments win, then
// fewer template variables, then more request-parameter conditions, then the
// longer pattern. Registration order breaks any remaining tie so that document
// generation stays deterministic.
func sortRoutes(routes []*Route) {
	sort.SliceStable(routes, func(i, j int) bool {
		a, b := routes[i], routes[j]
		if a.matcher.literals != b.matcher.literals {
			return a.matcher.literals > b.matcher.literals
		}
		if a.matcher.vars != b.matcher.vars {
			return a.matcher.vars < b.matcher.vars
		}
		if len(a.Params) != len(b.Params) {
			return len(a.Params) > len(b.Params)
		}
		if len(a.Pattern) != len(b.Pattern) {
			return len(a.Pattern) > len(b.Pattern)
		}
		return a.registered < b.registered
	})
}

func (r *Route) consumesMatch(contentType string) bool {
	if len(r.Consumes) == 0 {
		return true
	}
	for _, c := range r.Consumes {
		if c == MediaTypeAll || c == contentType {
			return true
		}
		if strings.HasSuffix(c, "/*") && strings.HasPrefix(contentType, strings.TrimSuffix(c, "*")) {
			return true
		}
	}
	return false
}

func (r *Route) paramsMatch(req *Request) bool {
	for _, p := range r.Params {
		v, ok := req.Query(p.Name)
		if !ok || v != p.Value {
			return false
		}
	}
	return true
}

func (r *Route) acceptMatch(req *Request) bool {
	if len(r.Accept) == 0 {
		return true
	}
	for _, a := range r.Accept {
		if r.AcceptExplicit {
			if req.AcceptsExplicitly(a) {
				return true
			}
			continue
		}
		if req.Accepts(a) {
			return true
		}
	}
	return false
}
