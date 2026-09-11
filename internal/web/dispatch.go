package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

// dispatch is the DispatcherServlet / DispatcherHandler: it strips the context
// path, answers CORS preflights, then resolves the request against the view
// controllers, the resource handlers and the request mappings, in that order.
func (a *App) dispatch(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if a.ContextPath != "" {
		switch {
		case path == a.ContextPath:
			path = "/"
		case strings.HasPrefix(path, a.ContextPath+"/"):
			path = strings.TrimPrefix(path, a.ContextPath)
		default:
			a.writeError(w, r, r.URL.Path, ErrNotFound, true)
			return
		}
	}

	req := &Request{Raw: r, Path: path, query: r.URL.Query()}

	if a.handlePreflight(w, req) {
		return
	}
	a.applyCorsResponseHeaders(w, req)

	if a.serveView(w, req) {
		return
	}
	if a.serveResource(w, req) {
		return
	}
	a.serveRoutes(w, req)
}

// serveRoutes performs Spring's two-stage match: first by path, then by method,
// consumes, request-parameter conditions and Accept. The status reported when a
// stage fails mirrors the exception the DispatcherServlet would have raised.
func (a *App) serveRoutes(w http.ResponseWriter, req *Request) {
	var pathMatched []*Route
	for _, route := range a.routes {
		if _, ok := route.matcher.match(req.Path); ok {
			pathMatched = append(pathMatched, route)
		}
	}
	if len(pathMatched) == 0 {
		a.writeError(w, req.Raw, req.Raw.URL.Path, ErrNotFound, true)
		return
	}

	var methodMatched []*Route
	for _, route := range pathMatched {
		if route.Method == req.Method() {
			methodMatched = append(methodMatched, route)
		}
	}
	if len(methodMatched) == 0 {
		a.writeError(w, req.Raw, req.Raw.URL.Path, ErrNotAllowed, false)
		return
	}

	var paramMatched []*Route
	for _, route := range methodMatched {
		if route.paramsMatch(req) {
			paramMatched = append(paramMatched, route)
		}
	}
	if len(paramMatched) == 0 {
		// UnsatisfiedServletRequestParameterException is a 400.
		a.writeError(w, req.Raw, req.Raw.URL.Path, ErrBadRequest, false)
		return
	}

	contentType := req.ContentType()
	var consumeMatched []*Route
	for _, route := range paramMatched {
		if route.consumesMatch(contentType) {
			consumeMatched = append(consumeMatched, route)
		}
	}
	if len(consumeMatched) == 0 {
		a.writeError(w, req.Raw, req.Raw.URL.Path, ErrUnsupported, false)
		return
	}

	var chosen *Route
	for _, route := range consumeMatched {
		if route.acceptMatch(req) {
			chosen = route
			break
		}
	}
	if chosen == nil {
		// A router-function predicate that does not match leaves nothing to
		// handle the request at all, which surfaces as a 404.
		a.writeError(w, req.Raw, req.Raw.URL.Path, ErrNotFound, true)
		return
	}

	req.PathVars, _ = chosen.matcher.match(req.Path)
	entity, err := chosen.Handler(req)
	if err != nil {
		a.writeError(w, req.Raw, req.Raw.URL.Path, err, false)
		return
	}
	a.writeEntity(w, chosen, entity)
}

func (a *App) writeEntity(w http.ResponseWriter, route *Route, entity ResponseEntity) {
	body, contentType, err := entity.render(route.Produces, a.DefaultCharset)
	if err != nil {
		a.writeError(w, nil, "", ErrInternalError, false)
		return
	}
	for k, vs := range entity.Headers {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	status := entity.Status
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	if len(body) > 0 {
		_, _ = w.Write(body)
	}
}

// serveView resolves the view controllers registered by addViewController.
// Only "forward:" view names occur in these demos; a forward re-enters the
// dispatcher at the target path.
func (a *App) serveView(w http.ResponseWriter, req *Request) bool {
	for _, v := range a.views {
		if v.path != req.Path {
			continue
		}
		target := strings.TrimPrefix(v.viewName, "forward:")
		fwd := &Request{Raw: req.Raw, Path: target, query: req.query}
		if a.serveResource(w, fwd) {
			return true
		}
		a.writeError(w, req.Raw, req.Raw.URL.Path, ErrNotFound, true)
		return true
	}
	return false
}

func (a *App) serveResource(w http.ResponseWriter, req *Request) bool {
	for _, h := range a.resources {
		if !strings.HasPrefix(req.Path, h.prefix) {
			continue
		}
		name := strings.TrimPrefix(req.Path, h.prefix)
		data, mediaType, ok := h.provider.Open(name)
		if !ok {
			continue
		}
		if strings.HasSuffix(mediaType, ";charset=") {
			mediaType += a.DefaultCharset
		}
		w.Header().Set("Content-Type", mediaType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return true
	}
	return false
}

// handlePreflight answers a CORS preflight request, reproducing the headers
// Spring's DefaultCorsProcessor writes: the echoed origin, the methods actually
// mapped for the path, the default 1800-second max age, and the Allow header
// the servlet container adds for OPTIONS.
func (a *App) handlePreflight(w http.ResponseWriter, req *Request) bool {
	if req.Method() != http.MethodOptions {
		return false
	}
	origin := req.Raw.Header.Get("Origin")
	requested := req.Raw.Header.Get("Access-Control-Request-Method")
	if origin == "" || requested == "" {
		return false
	}
	m := a.corsFor(req.Path)
	if m == nil || !originAllowed(m, origin) {
		return false
	}
	if len(a.methodsFor(req.Path)) == 0 {
		return false
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", strings.Join(m.allowedMethods, ","))
	w.Header().Set("Access-Control-Max-Age", "1800")
	w.Header().Set("Allow", strings.Join(servletAllowedMethods, ", "))
	w.WriteHeader(http.StatusOK)
	return true
}

func (a *App) applyCorsResponseHeaders(w http.ResponseWriter, req *Request) {
	origin := req.Raw.Header.Get("Origin")
	if origin == "" {
		return
	}
	m := a.corsFor(req.Path)
	if m == nil || !originAllowed(m, origin) {
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
}

func (a *App) corsFor(path string) *corsMapping {
	for _, m := range a.cors {
		if _, ok := m.pattern.match(path); ok {
			return m
		}
	}
	return nil
}

func originAllowed(m *corsMapping, origin string) bool {
	for _, o := range m.allowedOrigins {
		if o == "*" || o == origin {
			return true
		}
	}
	return false
}

// methodsFor lists the methods mapped for a path, plus the HEAD that Spring
// derives from a GET mapping, in the canonical order Spring reports them.
func (a *App) methodsFor(path string) []string {
	seen := map[string]bool{}
	for _, route := range a.routes {
		if _, ok := route.matcher.match(path); ok {
			seen[route.Method] = true
		}
	}
	if seen[http.MethodGet] {
		seen[http.MethodHead] = true
	}
	order := []string{http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
		http.MethodDelete, http.MethodOptions, http.MethodPatch}
	var out []string
	for _, m := range order {
		if seen[m] {
			out = append(out, m)
		}
	}
	return out
}

// servletAllowedMethods is the Allow header the servlet container writes for an
// OPTIONS request: every method its dispatcher can handle, independent of what
// the path actually maps.
var servletAllowedMethods = []string{
	http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
	http.MethodDelete, http.MethodOptions, http.MethodPatch,
}

// writeError renders the error document. Spring MVC and Spring WebFlux disagree
// on both the key order and the treatment of `message`, so each stack gets its
// own renderer.
func (a *App) writeError(w http.ResponseWriter, r *http.Request, path string, err error, noHandler bool) {
	he, ok := AsHTTPError(err)
	if !ok {
		he = ErrInternalError
	}
	if path == "" && r != nil {
		path = r.URL.Path
	}
	timestamp := nowTimestamp()

	if a.Stack == StackWAR {
		// The servlet container answers an unmatched request itself; Spring
		// Boot's error controller is not on the classpath.
		w.Header().Set("Content-Type", MediaTypeTextPlain+";charset="+a.DefaultCharset)
		w.WriteHeader(he.Status)
		_, _ = w.Write([]byte(he.Reason))
		return
	}

	var payload []byte
	switch a.Stack {
	case StackReactive:
		message := any("")
		if noHandler {
			message = nil
		}
		payload = mustMarshalOrdered([]kv{
			{"timestamp", timestamp},
			{"path", path},
			{"status", he.Status},
			{"error", he.Reason},
			{"message", message},
			{"requestId", a.nextRequestID()},
		})
	default:
		payload = mustMarshalOrdered([]kv{
			{"timestamp", timestamp},
			{"status", he.Status},
			{"error", he.Reason},
			{"message", ""},
			{"path", path},
		})
	}
	w.Header().Set("Content-Type", MediaTypeJSON)
	w.WriteHeader(he.Status)
	_, _ = w.Write(payload)
}

// nextRequestID reproduces Reactor Netty's connection identifier: eight hex
// digits, a dash, and a monotonically increasing request counter.
func (a *App) nextRequestID() string {
	n := atomic.AddInt64(&a.requestSeq, 1)
	if a.requestUUID != nil {
		return fmt.Sprintf("%s-%d", a.requestUUID(), n)
	}
	return fmt.Sprintf("%08x-%d", randomHex32(), n)
}

type kv struct {
	k string
	v any
}

// mustMarshalOrdered serialises key/value pairs preserving their order, which
// encoding/json cannot do for a map.
func mustMarshalOrdered(pairs []kv) []byte {
	var b strings.Builder
	b.WriteString("{")
	for i, p := range pairs {
		if i > 0 {
			b.WriteString(",")
		}
		key, _ := json.Marshal(p.k)
		val, _ := json.Marshal(p.v)
		b.Write(key)
		b.WriteString(":")
		b.Write(val)
	}
	b.WriteString("}")
	return []byte(b.String())
}

// nowTimestamp renders the instant the way Spring Boot's DefaultErrorAttributes
// serialises java.util.Date under Jackson's default configuration.
func nowTimestamp() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05.000+00:00")
}
