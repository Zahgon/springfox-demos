package boot

import (
	"net/http"
	"strings"
	"testing"

	"github.com/springfox/springfox-demos/internal/web"
)

// TestErrorControllerIsServletOnly covers the asymmetry between the two stacks:
// Spring Boot maps BasicErrorController on the servlet stack only.
func TestErrorControllerIsServletOnly(t *testing.T) {
	servlet := web.NewApp("servlet", web.StackServlet)
	RegisterErrorController(servlet, false)
	if len(servlet.Routes()) == 0 {
		t.Error("the servlet stack registered no error controller")
	}

	reactive := web.NewApp("reactive", web.StackReactive)
	RegisterErrorController(reactive, false)
	if got := len(reactive.Routes()); got != 0 {
		t.Errorf("the reactive stack registered %d error routes, want none", got)
	}
}

// TestErrorControllerMapsEveryMethod covers the seven methods
// BasicErrorController's @RequestMapping covers, and the two handlers it maps
// over each of them.
func TestErrorControllerMapsEveryMethod(t *testing.T) {
	app := web.NewApp("servlet", web.StackServlet)
	RegisterErrorController(app, false)

	methods := map[string]bool{}
	names := map[string]int{}
	for _, r := range app.Routes() {
		methods[r.Method] = true
		names[r.Op.Name]++
	}
	for _, want := range []string{
		http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
		http.MethodDelete, http.MethodOptions, http.MethodPatch, http.MethodTrace,
	} {
		if !methods[want] {
			t.Errorf("%s is not mapped on /error", want)
		}
	}
	if names["error"] != 8 || names["errorHtml"] != 8 {
		t.Errorf("mapped error=%d errorHtml=%d, want 8 of each", names["error"], names["errorHtml"])
	}
}

// TestErrorControllerRegistrationOrder covers htmlFirst, which decides which of
// the two handlers a documentation group keeps when it collapses the pair.
func TestErrorControllerRegistrationOrder(t *testing.T) {
	jsonFirst := web.NewApp("a", web.StackServlet)
	RegisterErrorController(jsonFirst, false)
	if got := jsonFirst.Routes()[0].Op.Name; got != "error" {
		t.Errorf("with htmlFirst false, the first route is %q, want %q", got, "error")
	}

	htmlFirst := web.NewApp("b", web.StackServlet)
	RegisterErrorController(htmlFirst, true)
	if got := htmlFirst.Routes()[0].Op.Name; got != "errorHtml" {
		t.Errorf("with htmlFirst true, the first route is %q, want %q", got, "errorHtml")
	}
}

// TestErrorControllerConsumes covers the `consumes` Springfox reports: every
// method that can carry a body except GET and DELETE.
func TestErrorControllerConsumes(t *testing.T) {
	cases := map[string]bool{
		http.MethodGet: false, http.MethodDelete: false,
		http.MethodHead: true, http.MethodPost: true, http.MethodPut: true,
		http.MethodOptions: true, http.MethodPatch: true, http.MethodTrace: true,
	}
	for method, want := range cases {
		got := errorConsumes(method) != nil
		if got != want {
			t.Errorf("errorConsumes(%s) present = %v, want %v", method, got, want)
		}
	}
}

// TestModelAndViewEnumIsTheHttpStatusList covers the enum Springfox derives
// from org.springframework.http.HttpStatus.
func TestModelAndViewEnumIsTheHttpStatusList(t *testing.T) {
	app := web.NewApp("servlet", web.StackServlet)
	RegisterErrorController(app, false)

	model, ok := app.Models.Get("ModelAndView")
	if !ok {
		t.Fatal("the ModelAndView model was not registered")
	}
	var status []string
	for _, p := range model.Properties {
		if p.Name == "status" {
			status = p.Schema.Enum
		}
	}
	if len(status) != 68 {
		t.Errorf("the HttpStatus enum has %d values, want 68", len(status))
	}
	// The list is in declaration order, which the legacy v2 model mapper keeps.
	if status[0] != "CONTINUE" || status[4] != "OK" {
		t.Errorf("the enum does not start in declaration order: %v", status[:5])
	}
}

// TestActuatorEndpoints covers the three web endpoints
// spring-boot-starter-actuator contributes and their vendor media type.
func TestActuatorEndpoints(t *testing.T) {
	app := web.NewApp("servlet", web.StackServlet)
	RegisterActuator(app, "http://localhost:8080")
	mvc := web.NewMockMvc(app)

	health := mvc.Perform(web.Get("/actuator/health"))
	if health.Body != `{"status":"UP"}` {
		t.Errorf("health = %s", health.Body)
	}
	if health.ContentType != actuatorContentType {
		t.Errorf("health Content-Type = %q, want %q", health.ContentType, actuatorContentType)
	}
	if info := mvc.Perform(web.Get("/actuator/info")); info.Body != "{}" {
		t.Errorf("info = %s", info.Body)
	}
	if nested := mvc.Perform(web.Get("/actuator/health/db")); nested.Status != http.StatusOK {
		t.Errorf("the health wildcard route gave %d", nested.Status)
	}
}

// TestActuatorLinkOrderPerStack covers the different order the two stacks emit
// the links index in.
func TestActuatorLinkOrderPerStack(t *testing.T) {
	servlet := web.NewApp("servlet", web.StackServlet)
	RegisterActuator(servlet, "http://localhost:8080")
	got := web.NewMockMvc(servlet).Perform(web.Get("/actuator")).Body
	if !strings.Contains(got, `"self":{`) ||
		strings.Index(got, `"health":{`) > strings.Index(got, `"health-path":{`) {
		t.Errorf("the servlet link order is wrong: %s", got)
	}

	reactive := web.NewApp("reactive", web.StackReactive)
	RegisterActuator(reactive, "http://localhost:8080")
	got = web.NewMockMvc(reactive).Perform(web.Get("/actuator")).Body
	if strings.Index(got, `"health-path":{`) > strings.Index(got, `"health":{`) {
		t.Errorf("the reactive link order is wrong: %s", got)
	}
}

// TestActuatorTagsPerStack covers the different tags Springfox derives for the
// actuator handlers on the two stacks, and the reactive reply model it cannot
// unwrap.
func TestActuatorTagsPerStack(t *testing.T) {
	servlet := web.NewApp("servlet", web.StackServlet)
	RegisterActuator(servlet, "")
	reactive := web.NewApp("reactive", web.StackReactive)
	RegisterActuator(reactive, "")

	tagsOf := func(app *web.App) map[string]bool {
		out := map[string]bool{}
		for _, r := range app.Routes() {
			out[r.Op.Tag] = true
		}
		return out
	}
	if !tagsOf(servlet)["web-mvc-links-handler"] || !tagsOf(servlet)["operation-handler"] {
		t.Errorf("the servlet tags are %v", tagsOf(servlet))
	}
	if !tagsOf(reactive)["web-flux-links-handler"] || !tagsOf(reactive)["read-operation-handler"] {
		t.Errorf("the reactive tags are %v", tagsOf(reactive))
	}

	for _, r := range reactive.Routes() {
		if r.Op.Name != "handle" {
			continue
		}
		if r.Op.Responses[0].Schema.Ref != ReactivePublisherModel {
			t.Errorf("the reactive reply schema is %+v, want a %s reference",
				r.Op.Responses[0].Schema, ReactivePublisherModel)
		}
	}
}

// TestErrorPageIsNotDirectlyRenderable covers the /error mapping itself: the
// controller exists so that Springfox documents it and so that a forwarded
// error resolves, but a client that requests it directly gets a 404 rather
// than a rendered page.
func TestErrorPageIsNotDirectlyRenderable(t *testing.T) {
	app := web.NewApp("servlet", web.StackServlet)
	RegisterErrorController(app, false)
	res := web.NewMockMvc(app).Perform(web.Get("/error"))
	if res.Status != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", res.Status, http.StatusNotFound)
	}
}
