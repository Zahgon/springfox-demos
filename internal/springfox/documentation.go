package springfox

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/springfox/springfox-demos/internal/api"
	"github.com/springfox/springfox-demos/internal/web"
)

// documentedOperation is one operation resolved for one documentation group.
type documentedOperation struct {
	path   string
	method string
	op     api.Operation
	id     string
}

// group is a Docket together with the operations it selected.
type group struct {
	docket *Docket
	// ops are the documented operations, one per path and method.
	ops []documentedOperation
	// candidates are every path-selected operation before collapsing duplicate
	// path/method pairs. Springfox harvests models from all of them, so a model
	// reachable only from a collapsed handler still reaches the document.
	candidates []documentedOperation
}

// nameGenerator reproduces Springfox's OperationNameGenerator: an operation is
// named "<methodName>Using<HTTP_METHOD>", and a name already handed out gets an
// "_N" suffix. The generator is shared across every documentation group in an
// application, so the same handler appears under different operation ids in
// different groups — which is exactly what the original emits.
type nameGenerator struct {
	used map[string]int
}

func newNameGenerator() *nameGenerator { return &nameGenerator{used: map[string]int{}} }

func (g *nameGenerator) next(methodName, httpMethod string) string {
	base := methodName + "Using" + strings.ToUpper(httpMethod)
	n, seen := g.used[base]
	g.used[base] = n + 1
	if !seen {
		return base
	}
	return fmt.Sprintf("%s_%d", base, n)
}

// Provider owns an application's documentation groups and renders them on
// demand. It is the Go equivalent of Springfox's DocumentationCache plus the
// per-specification document mappers.
type Provider struct {
	app *web.App
	// Host is the authority reported in a Swagger 2.0 document and in the
	// OpenAPI `servers` block.
	Host string
	// mounted lists the specification versions whose api-docs endpoint is
	// served, in mount order.
	mounted []DocumentationType
	// advertised lists the versions /swagger-resources reports, in the order it
	// reports them. springfox-boot-starter serves both v2 and v3 but advertises
	// only the OpenAPI one, so the two lists are configured separately.
	advertised []DocumentationType
	// UseModelV3 follows springfox.documentation.swagger.v2.use-model-v3. When
	// it is false the Swagger 2.0 document is built by the legacy model mapper,
	// whose only visible difference is that it leaves a property's enum values
	// in declaration order instead of sorting them.
	UseModelV3 bool

	dockets []*Docket
	groups  map[string]*group
	// order preserves the sequence groups were generated in.
	order []string
}

// NewProvider creates a documentation provider for an application.
func NewProvider(app *web.App) *Provider {
	return &Provider{app: app, Host: "localhost:8080", groups: map[string]*group{}, UseModelV3: true}
}

// AddDocket registers a documentation group.
func (p *Provider) AddDocket(d *Docket) { p.dockets = append(p.dockets, d) }

// EnableSpec mounts a specification version's api-docs endpoint and advertises
// it in /swagger-resources.
func (p *Provider) EnableSpec(t DocumentationType) {
	p.MountSpec(t)
	p.advertised = appendSpec(p.advertised, t)
}

// MountSpec serves a specification version's api-docs endpoint without
// advertising it in /swagger-resources.
func (p *Provider) MountSpec(t DocumentationType) {
	p.mounted = appendSpec(p.mounted, t)
}

func appendSpec(specs []DocumentationType, t DocumentationType) []DocumentationType {
	for _, s := range specs {
		if s == t {
			return specs
		}
	}
	return append(specs, t)
}

// Build resolves every group. It must run after all routes are registered,
// because operation-id assignment depends on the complete route table.
//
// Groups are resolved OpenAPI-first and then in registration order within a
// specification version. That ordering is not documented by Springfox; it was
// recovered from the original's output, where the OAS_30 group holds the
// unsuffixed operation ids and the SWAGGER_2 groups take the "_1"/"_2" suffixes
// in declaration order.
func (p *Provider) Build() {
	p.app.Seal()
	p.groups = map[string]*group{}
	p.order = nil

	ordered := make([]*Docket, len(p.dockets))
	copy(ordered, p.dockets)
	sort.SliceStable(ordered, func(i, j int) bool {
		return specRank(ordered[i].Type()) < specRank(ordered[j].Type())
	})

	names := newNameGenerator()
	for _, d := range ordered {
		candidates, collapsed := p.selectOperations(d)
		g := &group{docket: d, candidates: candidates}
		// Operation ids are handed out in the order the handlers were
		// registered, then the operations are re-ordered for rendering.
		for _, op := range collapsed {
			op.id = names.next(op.op.Name, op.method)
			g.ops = append(g.ops, op)
		}
		sort.SliceStable(g.ops, func(i, j int) bool {
			if g.ops[i].path != g.ops[j].path {
				return g.ops[i].path < g.ops[j].path
			}
			return methodRank(g.ops[i].method) < methodRank(g.ops[j].method)
		})
		p.groups[d.Group()] = g
		p.order = append(p.order, d.Group())
	}
}

func specRank(t DocumentationType) int {
	switch t {
	case OAS30:
		return 0
	case Swagger2:
		return 1
	default:
		return 2
	}
}

// selectOperations applies the Docket's path selector to the application's
// routes and collapses routes that share a path and method — Springfox keeps a
// single operation per path/method pair even when Spring maps several handlers
// there under different request-parameter conditions.
func (p *Provider) selectOperations(d *Docket) (candidates, collapsed []documentedOperation) {
	seen := map[string]bool{}
	for _, r := range p.app.Routes() {
		if r.Op.Name == "" || r.Op.Hidden {
			continue
		}
		path := p.documentedPath(r.Pattern)
		if !d.paths(path) {
			continue
		}
		dop := documentedOperation{path: path, method: r.Method, op: r.Op}
		candidates = append(candidates, dop)
		key := r.Method + " " + path
		if seen[key] {
			continue
		}
		seen[key] = true
		collapsed = append(collapsed, dop)
	}
	return candidates, collapsed
}

// documentedPath is the path Springfox reports: the servlet context path
// followed by the mapping pattern.
func (p *Provider) documentedPath(pattern string) string {
	if p.app.ContextPath == "" {
		return pattern
	}
	return p.app.ContextPath + pattern
}

// methodRank orders the methods within a path the way the Swagger 2.0 mapper
// emits them.
func methodRank(m string) int { return rankIn(swagger2MethodOrder, m) }

// oasMethodRank orders the methods within a path item the way the OpenAPI
// mapper emits them, which is the field order of the OpenAPI PathItem object.
func oasMethodRank(m string) int { return rankIn(oasMethodOrder, m) }

var (
	swagger2MethodOrder = []string{
		http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
		http.MethodDelete, http.MethodOptions, http.MethodPatch, http.MethodTrace,
	}
	oasMethodOrder = []string{
		http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete,
		http.MethodOptions, http.MethodHead, http.MethodPatch, http.MethodTrace,
	}
)

func rankIn(order []string, m string) int {
	for i, v := range order {
		if v == m {
			return i
		}
	}
	return len(order)
}

// orderedByPath groups operations by their documented path, preserving path
// order, and orders the methods inside each path with the given comparator.
func orderedByPath(ops []documentedOperation, rank func(string) int) [][]documentedOperation {
	var paths []string
	byPath := map[string][]documentedOperation{}
	for _, op := range ops {
		if _, ok := byPath[op.path]; !ok {
			paths = append(paths, op.path)
		}
		byPath[op.path] = append(byPath[op.path], op)
	}
	out := make([][]documentedOperation, 0, len(paths))
	for _, p := range paths {
		group := byPath[p]
		sort.SliceStable(group, func(i, j int) bool { return rank(group[i].method) < rank(group[j].method) })
		out = append(out, group)
	}
	return out
}

// Document renders the named group under the given specification version,
// reporting false when the group is unknown.
func (p *Provider) Document(spec DocumentationType, groupName string) (any, bool) {
	if groupName == "" {
		groupName = "default"
	}
	g, ok := p.groups[groupName]
	if !ok {
		return nil, false
	}
	switch spec {
	case OAS30:
		return p.renderOpenAPI(g), true
	case Swagger12:
		return p.renderSwagger12(g), true
	default:
		return p.renderSwagger2(g), true
	}
}

// uriTemplate renders a path as an RFC 6570 template with the operation's query
// parameters appended, which is what .enableUrlTemplating(true) produces.
func uriTemplate(path string, op api.Operation, ignoreAnnotated bool) string {
	var qs []string
	for _, prm := range op.Params {
		if prm.In != api.InQuery {
			continue
		}
		if ignoreAnnotated && prm.Ignored {
			continue
		}
		qs = append(qs, prm.Name)
	}
	if len(qs) == 0 {
		return path
	}
	return path + "{?" + strings.Join(qs, ",") + "}"
}

// visibleParams drops the parameters annotated @ApiIgnore. Springfox honours
// that annotation in every document, whatever a Docket's ignoredParameterTypes
// says.
func visibleParams(op api.Operation, _ bool) []api.Parameter {
	out := make([]api.Parameter, 0, len(op.Params))
	for _, prm := range op.Params {
		if prm.Ignored {
			continue
		}
		out = append(out, prm)
	}
	return out
}

// tagsOf collects the distinct tags of a group's operations, sorted by name,
// each with the humanised description Springfox derives from the tag.
func tagsOf(g *group) []*object {
	descriptions := map[string]string{}
	var names []string
	for _, op := range g.ops {
		name := op.op.ListedTag
		if name == "" {
			// An operation whose tag was overridden by @ApiOperation(tags = …)
			// sets ListedTag explicitly; otherwise the operation's own tag is
			// the controller's.
			name = op.op.Tag
		}
		if name == "" {
			continue
		}
		if _, seen := descriptions[name]; seen {
			continue
		}
		description := op.op.ListedTagDescription
		if description == "" {
			description = humanise(name)
		}
		descriptions[name] = description
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]*object, 0, len(names))
	for _, n := range names {
		out = append(out, obj().set("name", n).set("description", descriptions[n]))
	}
	return out
}

// humanise turns "hello-world-controller" into "Hello World Controller", the
// tag description Springfox synthesises from a controller's class name.
func humanise(tag string) string {
	parts := strings.Split(tag, "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}
