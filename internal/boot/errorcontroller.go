// Package boot supplies the pieces of Spring Boot's auto-configuration whose
// behaviour the demos expose: the servlet stack's BasicErrorController and the
// actuator endpoints that spring-boot-starter-actuator contributes. Both are
// observable — they answer requests and they appear in the generated
// documentation — so they are reproduced rather than dropped.
package boot

import (
	"net/http"

	"github.com/springfox/springfox-demos/internal/api"
	"github.com/springfox/springfox-demos/internal/web"
)

// httpStatusNames is the java.lang.Enum name list of
// org.springframework.http.HttpStatus, in declaration order. Springfox emits it
// as the enum of ModelAndView.status; the OpenAPI mapper sorts it, the Swagger
// 2.0 mapper does not.
var httpStatusNames = []string{
	"CONTINUE", "SWITCHING_PROTOCOLS", "PROCESSING", "CHECKPOINT", "OK", "CREATED",
	"ACCEPTED", "NON_AUTHORITATIVE_INFORMATION", "NO_CONTENT", "RESET_CONTENT",
	"PARTIAL_CONTENT", "MULTI_STATUS", "ALREADY_REPORTED", "IM_USED",
	"MULTIPLE_CHOICES", "MOVED_PERMANENTLY", "FOUND", "MOVED_TEMPORARILY", "SEE_OTHER",
	"NOT_MODIFIED", "USE_PROXY", "TEMPORARY_REDIRECT", "PERMANENT_REDIRECT",
	"BAD_REQUEST", "UNAUTHORIZED", "PAYMENT_REQUIRED", "FORBIDDEN", "NOT_FOUND",
	"METHOD_NOT_ALLOWED", "NOT_ACCEPTABLE", "PROXY_AUTHENTICATION_REQUIRED",
	"REQUEST_TIMEOUT", "CONFLICT", "GONE", "LENGTH_REQUIRED", "PRECONDITION_FAILED",
	"PAYLOAD_TOO_LARGE", "REQUEST_ENTITY_TOO_LARGE", "URI_TOO_LONG",
	"REQUEST_URI_TOO_LONG", "UNSUPPORTED_MEDIA_TYPE", "REQUESTED_RANGE_NOT_SATISFIABLE",
	"EXPECTATION_FAILED", "I_AM_A_TEAPOT", "INSUFFICIENT_SPACE_ON_RESOURCE",
	"METHOD_FAILURE", "DESTINATION_LOCKED", "UNPROCESSABLE_ENTITY", "LOCKED",
	"FAILED_DEPENDENCY", "TOO_EARLY", "UPGRADE_REQUIRED", "PRECONDITION_REQUIRED",
	"TOO_MANY_REQUESTS", "REQUEST_HEADER_FIELDS_TOO_LARGE",
	"UNAVAILABLE_FOR_LEGAL_REASONS", "INTERNAL_SERVER_ERROR", "NOT_IMPLEMENTED",
	"BAD_GATEWAY", "SERVICE_UNAVAILABLE", "GATEWAY_TIMEOUT",
	"HTTP_VERSION_NOT_SUPPORTED", "VARIANT_ALSO_NEGOTIATES", "INSUFFICIENT_STORAGE",
	"LOOP_DETECTED", "BANDWIDTH_LIMIT_EXCEEDED", "NOT_EXTENDED",
	"NETWORK_AUTHENTICATION_REQUIRED",
}

// objectSchema is the untyped `object` Springfox uses for Object and for the
// value type of a Map<String, Object>.
func objectSchema() api.Schema { return api.Schema{Type: api.TypeObject} }

func mapOfObject() api.Schema {
	v := objectSchema()
	return api.Schema{Type: api.TypeObject, AdditionalProperties: &v}
}

// RegisterErrorController mounts Spring Boot's BasicErrorController at /error.
// The servlet stack maps it for every HTTP method; the reactive stack handles
// errors without a controller and therefore registers nothing.
//
// BasicErrorController maps two handlers over the same seven methods: error(),
// returning a Map, and errorHtml(), returning a ModelAndView for text/html.
// Springfox documents only one of them per path and method, and which one it
// picks depends on the order the JVM happens to report the class's declared
// methods in — an order the specification leaves undefined and that differs
// between the original's own modules. htmlFirst pins the choice to the one the
// corresponding Java module was observed to make.
func RegisterErrorController(app *web.App, htmlFirst bool) {
	if app.Stack != web.StackServlet {
		return
	}
	app.Models.Add(api.Model{
		Name: "ModelAndView",
		Properties: []api.Property{
			{Name: "empty", Schema: api.Schema{Type: api.TypeBoolean}},
			{Name: "model", Schema: objectSchema()},
			{Name: "modelMap", Schema: mapOfObject()},
			{Name: "reference", Schema: api.Schema{Type: api.TypeBoolean}},
			{Name: "status", Schema: api.Schema{Type: api.TypeString, Enum: httpStatusNames}},
			{Name: "view", Schema: api.Schema{Ref: "View"}},
			{Name: "viewName", Schema: api.Schema{Type: api.TypeString}},
		},
	})
	app.Models.Add(api.Model{
		Name: "View",
		Properties: []api.Property{
			{Name: "contentType", Schema: api.Schema{Type: api.TypeString}},
		},
	})

	// DELETE alone carries no `consumes`, because BasicErrorController's
	// delete mapping declares no request body.
	methods := []string{
		http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
		http.MethodDelete, http.MethodOptions, http.MethodPatch, http.MethodTrace,
	}
	registerJSON := func(method string) {
		s := mapOfObject()
		consumes := errorConsumes(method)
		app.Handle(web.Route{
			Method:   method,
			Pattern:  "/error",
			Produces: []string{web.MediaTypeAll},
			Handler:  errorHandler,
			Op: api.Operation{
				Name: "error", Tag: "basic-error-controller", Summary: "error",
				Consumes:  consumes,
				Produces:  []string{web.MediaTypeAll},
				Responses: []api.Response{{Code: http.StatusOK, Description: "OK", Schema: &s}},
			},
		})
	}
	registerHTML := func(method string) {
		consumes := errorConsumes(method)
		app.Handle(web.Route{
			Method:         method,
			Pattern:        "/error",
			Produces:       []string{web.MediaTypeTextHTML},
			Accept:         []string{web.MediaTypeTextHTML},
			AcceptExplicit: true,
			Handler:        errorHandler,
			Op: api.Operation{
				Name: "errorHtml", Tag: "basic-error-controller", Summary: "errorHtml",
				Consumes:  consumes,
				Produces:  []string{web.MediaTypeTextHTML},
				Responses: []api.Response{{Code: http.StatusOK, Description: "OK", Schema: &api.Schema{Ref: "ModelAndView"}}},
			},
		})
	}
	for _, method := range methods {
		if htmlFirst {
			registerHTML(method)
			registerJSON(method)
			continue
		}
		registerJSON(method)
		registerHTML(method)
	}
}

// errorConsumes is the `consumes` Springfox reports for a BasicErrorController
// mapping: application/json for every method that can carry a request body.
func errorConsumes(method string) []string {
	if method == http.MethodGet || method == http.MethodDelete {
		return nil
	}
	return []string{web.MediaTypeJSON}
}

// errorHandler renders the error attributes for a direct request to /error, the
// way BasicErrorController does when nothing forwarded to it.
func errorHandler(*web.Request) (web.ResponseEntity, error) {
	return web.ResponseEntity{}, web.ErrNotFound
}
