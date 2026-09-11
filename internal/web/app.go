package web

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/springfox/springfox-demos/internal/api"
)

// Stack names the servlet stack an application runs on. The two stacks differ
// in the error document they render and in the Content-Type they attach to a
// string body, so every application declares which one it is.
type Stack int

const (
	// StackServlet is Spring MVC on Tomcat.
	StackServlet Stack = iota
	// StackReactive is Spring WebFlux on Reactor Netty.
	StackReactive
	// StackWAR is Spring MVC deployed as a WAR into a servlet container that
	// Spring Boot did not configure. It differs from StackServlet in two
	// observable ways: text bodies default to ISO-8859-1 rather than UTF-8, and
	// an unmatched request is answered by the container rather than by Spring
	// Boot's BasicErrorController.
	StackWAR
)

// App is an application context plus its dispatcher: the routes, the static
// resource handlers, the view controllers, the CORS registry and the filter
// chain that a Spring Boot application assembles at startup.
type App struct {
	// Name is the module name, used in startup logging.
	Name string
	// Stack selects servlet or reactive semantics.
	Stack Stack
	// ContextPath is server.servlet.context-path; "" when unset.
	ContextPath string
	// Port is the listen port; 8080 unless overridden.
	Port int

	// Models is the document model registry the documentation groups draw on.
	Models *api.Registry
	// Log is the application logger.
	Log *Logger
	// DefaultCharset is the charset the servlet container appends to a text
	// media type when the handler does not choose one.
	DefaultCharset string

	routes    []*Route
	resources []*resourceHandler
	views     []*viewController
	cors      []*corsMapping
	filters   []Filter

	requestSeq  int64
	sealed      bool
	requestUUID func() string
}

// Filter wraps the dispatcher, like a javax.servlet.Filter or a WebFilter.
type Filter func(next http.Handler) http.Handler

type resourceHandler struct {
	pattern  string
	prefix   string
	provider ResourceProvider
}

// ResourceProvider serves a classpath-like resource tree. It replaces
// `addResourceLocations("classpath:/META-INF/resources/webjars/...")`.
type ResourceProvider interface {
	// Open returns the bytes and media type of the named resource.
	Open(name string) (data []byte, mediaType string, ok bool)
}

type viewController struct {
	path     string
	viewName string
}

type corsMapping struct {
	pattern        *pathMatcher
	raw            string
	allowedOrigins []string
	// allowedMethods is CorsRegistration's default when the mapping does not
	// set one, which none of the demos do.
	allowedMethods []string
}

// NewApp creates an application context.
func NewApp(name string, stack Stack) *App {
	// Spring Boot's embedded Tomcat spells the charset in upper case; the
	// Jetty the WAR modules deploy into spells it in lower case.
	charset := "UTF-8"
	if stack == StackWAR {
		charset = "iso-8859-1"
	}
	return &App{
		Name:           name,
		Stack:          stack,
		Port:           8080,
		Models:         api.NewRegistry(),
		Log:            NewLogger(),
		DefaultCharset: charset,
	}
}

// Handle registers a request mapping.
func (a *App) Handle(r Route) *Route {
	route := r
	route.matcher = compilePattern(route.Pattern)
	route.registered = len(a.routes)
	a.routes = append(a.routes, &route)
	return &route
}

// Routes returns the routes in registration order. Documentation generation
// depends on that order, so it is deliberately not the match order the
// dispatcher uses.
func (a *App) Routes() []*Route {
	out := make([]*Route, len(a.routes))
	copy(out, a.routes)
	sort.SliceStable(out, func(i, j int) bool { return out[i].registered < out[j].registered })
	return out
}

// AddResourceHandler mounts a resource tree at an "/x/**" pattern.
func (a *App) AddResourceHandler(pattern string, provider ResourceProvider) {
	prefix := strings.TrimSuffix(pattern, "**")
	a.resources = append(a.resources, &resourceHandler{pattern: pattern, prefix: prefix, provider: provider})
}

// AddViewController maps a path to a view name. Only the "forward:" view names
// the demos use are supported.
func (a *App) AddViewController(path, viewName string) {
	a.views = append(a.views, &viewController{path: path, viewName: viewName})
}

// AddCorsMapping allows the given origins on paths matching the pattern.
func (a *App) AddCorsMapping(pattern string, allowedOrigins ...string) {
	a.cors = append(a.cors, &corsMapping{
		pattern:        compilePattern(pattern),
		raw:            pattern,
		allowedOrigins: allowedOrigins,
		allowedMethods: []string{http.MethodGet, http.MethodHead, http.MethodPost},
	})
}

// AddFilter appends a filter to the chain. Filters run in registration order,
// outermost first.
func (a *App) AddFilter(f Filter) { a.filters = append(a.filters, f) }

// Seal finalises the routing table. Calling it twice is harmless; every entry
// point calls it before serving so that tests and main() behave identically.
func (a *App) Seal() {
	if a.sealed {
		return
	}
	sortRoutes(a.routes)
	sort.SliceStable(a.resources, func(i, j int) bool {
		return len(a.resources[i].prefix) > len(a.resources[j].prefix)
	})
	a.sealed = true
}

// Handler returns the fully wrapped http.Handler for the application.
func (a *App) Handler() http.Handler {
	a.Seal()
	var h http.Handler = http.HandlerFunc(a.dispatch)
	for i := len(a.filters) - 1; i >= 0; i-- {
		h = a.filters[i](h)
	}
	return h
}

// Run starts the server and blocks until SIGINT or SIGTERM, then shuts down
// gracefully. It returns a non-nil error when the listener cannot be bound.
func (a *App) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return a.RunContext(ctx)
}

// RunContext starts the server and blocks until ctx is done, then shuts down
// gracefully. It returns a non-nil error when the listener cannot be bound.
func (a *App) RunContext(ctx context.Context) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", a.Port))
	if err != nil {
		return err
	}
	srv := &http.Server{Handler: a.Handler(), ReadHeaderTimeout: 30 * time.Second}

	a.Log.Infof("Started %s on port %d (context path '%s')", a.Name, a.Port, a.ContextPath)

	errCh := make(chan error, 1)
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		a.Log.Infof("Shutting down %s", a.Name)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
