package springfox

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/springfox/springfox-demos/internal/api"
	"github.com/springfox/springfox-demos/internal/web"
)

// newDemoApp builds a small application with two documented routes.
func newDemoApp(t *testing.T) *web.App {
	t.Helper()
	app := web.NewApp("test", web.StackServlet)
	stringSchema := api.Schema{Type: api.TypeString}
	app.Handle(web.Route{
		Method: http.MethodGet, Pattern: "/hello",
		Handler: func(*web.Request) (web.ResponseEntity, error) { return web.OKText("hi"), nil },
		Op: api.Operation{
			Name: "hello", Tag: "greeting-controller", Summary: "hello",
			Produces:  []string{web.MediaTypeAll},
			Responses: []api.Response{{Code: http.StatusOK, Description: "OK", Schema: &stringSchema}},
		},
	})
	app.Handle(web.Route{
		Method: http.MethodGet, Pattern: "/api/pet",
		Handler: func(*web.Request) (web.ResponseEntity, error) { return web.OKEmpty(), nil },
		Op: api.Operation{
			Name: "pets", Tag: "pet-controller", Summary: "pets",
			Produces:  []string{web.MediaTypeAll},
			Responses: []api.Response{{Code: http.StatusOK, Description: "OK"}},
		},
	})
	return app
}

// TestPathSelectorRestrictsAGroup covers the select().paths(...) filter.
func TestPathSelectorRestrictsAGroup(t *testing.T) {
	app := newDemoApp(t)
	p := NewProvider(app)
	p.AddDocket(NewDocket(Swagger2).GroupName("pets").Paths(Regex("/api/.*")))
	p.Build()

	doc, ok := p.Document(Swagger2, "pets")
	if !ok {
		t.Fatal("the pets group was not generated")
	}
	body, err := Marshal(doc)
	if err != nil {
		t.Fatalf("marshalling the document: %v", err)
	}
	var parsed struct {
		Paths map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("parsing the document: %v", err)
	}
	if _, ok := parsed.Paths["/api/pet"]; !ok {
		t.Error("the selected path is missing")
	}
	if _, ok := parsed.Paths["/hello"]; ok {
		t.Error("an unselected path reached the document")
	}
}

// TestOperationIdsAreDeduplicatedAcrossGroups covers Springfox's shared
// operation-name generator: the same handler takes a different id in each
// group it appears in.
func TestOperationIdsAreDeduplicatedAcrossGroups(t *testing.T) {
	app := newDemoApp(t)
	p := NewProvider(app)
	p.AddDocket(NewDocket(Swagger2).GroupName("first"))
	p.AddDocket(NewDocket(Swagger2).GroupName("second"))
	p.AddDocket(NewDocket(Swagger2).GroupName("third"))
	p.Build()

	want := []string{"helloUsingGET", "helloUsingGET_1", "helloUsingGET_2"}
	for i, group := range []string{"first", "second", "third"} {
		doc, ok := p.Document(Swagger2, group)
		if !ok {
			t.Fatalf("the %s group was not generated", group)
		}
		body, err := Marshal(doc)
		if err != nil {
			t.Fatalf("marshalling %s: %v", group, err)
		}
		var parsed struct {
			Paths map[string]map[string]struct {
				OperationID string `json:"operationId"`
			} `json:"paths"`
		}
		if err := json.Unmarshal(body, &parsed); err != nil {
			t.Fatalf("parsing %s: %v", group, err)
		}
		if got := parsed.Paths["/hello"]["get"].OperationID; got != want[i] {
			t.Errorf("%s: operationId = %q, want %q", group, got, want[i])
		}
	}
}

// TestOpenAPIGroupsAreGeneratedFirst covers the ordering that decides which
// group keeps the unsuffixed operation ids.
func TestOpenAPIGroupsAreGeneratedFirst(t *testing.T) {
	app := newDemoApp(t)
	p := NewProvider(app)
	p.AddDocket(NewDocket(Swagger2).GroupName("swagger"))
	p.AddDocket(NewDocket(OAS30).GroupName("openapi"))
	p.Build()

	doc, _ := p.Document(OAS30, "openapi")
	body, _ := Marshal(doc)
	var parsed struct {
		Paths map[string]map[string]struct {
			OperationID string `json:"operationId"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("parsing the document: %v", err)
	}
	if got := parsed.Paths["/hello"]["get"].OperationID; got != "helloUsingGET" {
		t.Errorf("the OpenAPI group's operationId = %q, want the unsuffixed name", got)
	}
}

// TestSwaggerResourcesOrdering covers the resource listing: versions in the
// order they were enabled, groups sorted by name within each version, and no
// ?group= parameter on the group named "default".
func TestSwaggerResourcesOrdering(t *testing.T) {
	app := newDemoApp(t)
	p := NewProvider(app)
	p.EnableSpec(Swagger2)
	p.EnableSpec(OAS30)
	p.AddDocket(NewDocket(Swagger2).GroupName("zeta"))
	p.AddDocket(NewDocket(Swagger2).GroupName("alpha"))
	p.Build()

	resources := p.Resources(true)
	want := []SwaggerResource{
		{Name: "alpha", URL: "/v2/api-docs?group=alpha", SwaggerVersion: "2.0", Location: "/v2/api-docs?group=alpha"},
		{Name: "zeta", URL: "/v2/api-docs?group=zeta", SwaggerVersion: "2.0", Location: "/v2/api-docs?group=zeta"},
		{Name: "alpha", URL: "/v3/api-docs?group=alpha", SwaggerVersion: "3.0.3", Location: "/v3/api-docs?group=alpha"},
		{Name: "zeta", URL: "/v3/api-docs?group=zeta", SwaggerVersion: "3.0.3", Location: "/v3/api-docs?group=zeta"},
	}
	if len(resources) != len(want) {
		t.Fatalf("got %d resources, want %d", len(resources), len(want))
	}
	for i := range want {
		if resources[i] != want[i] {
			t.Errorf("resource %d = %+v, want %+v", i, resources[i], want[i])
		}
	}

	// A group named "default" is advertised without a ?group= parameter.
	def := NewProvider(newDemoApp(t))
	def.EnableSpec(OAS30)
	def.AddDocket(NewDocket(OAS30))
	def.Build()
	if got := def.Resources(true)[0].URL; got != "/v3/api-docs" {
		t.Errorf("the default group's url = %q, want %q", got, "/v3/api-docs")
	}

	// An unpublished listing is an empty array, not null.
	if got := def.Resources(false); got == nil || len(got) != 0 {
		t.Errorf("an unpublished listing = %v, want an empty slice", got)
	}
}

// TestDefaultResponseMessages covers the response messages Springfox adds per
// HTTP method, and the way a declared @ApiResponse overlays them.
func TestDefaultResponseMessages(t *testing.T) {
	cases := []struct {
		method string
		want   []int
	}{
		{http.MethodGet, []int{200, 401, 403, 404}},
		{http.MethodPost, []int{200, 201, 401, 403, 404}},
		{http.MethodPut, []int{200, 201, 401, 403, 404}},
		{http.MethodDelete, []int{200, 204, 401, 403}},
		{http.MethodPatch, []int{200, 204, 401, 403}},
	}
	for _, c := range cases {
		got := mergeResponses(c.method, nil)
		if len(got) != len(c.want) {
			t.Fatalf("%s: got %d responses, want %d", c.method, len(got), len(c.want))
		}
		for i, code := range c.want {
			if got[i].Code != code {
				t.Errorf("%s: response %d = %d, want %d", c.method, i, got[i].Code, code)
			}
		}
	}

	// A declared response replaces the default with the same code, and any
	// other declared code is added; the result stays sorted.
	merged := mergeResponses(http.MethodGet, []api.Response{
		{Code: 404, Description: "Pet not found"},
		{Code: 405, Description: "Invalid input"},
	})
	wantCodes := []int{200, 401, 403, 404, 405}
	for i, code := range wantCodes {
		if merged[i].Code != code {
			t.Fatalf("merged response %d = %d, want %d", i, merged[i].Code, code)
		}
	}
	if merged[3].Description != "Pet not found" {
		t.Errorf("the declared 404 description was lost: %q", merged[3].Description)
	}
}

// TestUIConfigurationDefaults covers the document served at
// /swagger-resources/configuration/ui, whose every field is a Springfox
// default except the base URL.
func TestUIConfigurationDefaults(t *testing.T) {
	body, err := json.Marshal(DefaultUIConfiguration("/documentation"))
	if err != nil {
		t.Fatalf("marshalling the ui configuration: %v", err)
	}
	want := `{"deepLinking":true,"displayOperationId":false,"defaultModelsExpandDepth":1,` +
		`"defaultModelExpandDepth":1,"defaultModelRendering":"example","displayRequestDuration":false,` +
		`"docExpansion":"none","filter":false,"operationsSorter":"alpha","showExtensions":false,` +
		`"showCommonExtensions":false,"tagsSorter":"alpha","validatorUrl":"",` +
		`"supportedSubmitMethods":["get","put","post","delete","options","head","patch","trace"],` +
		`"swaggerBaseUiUrl":"/documentation"}`
	if string(body) != want {
		t.Errorf("ui configuration =\n%s\nwant\n%s", body, want)
	}
}

// TestOrderedObjectPreservesInsertionOrder covers the JSON writer the documents
// depend on, since a Go map would reorder the keys.
func TestOrderedObjectPreservesInsertionOrder(t *testing.T) {
	o := obj().set("swagger", "2.0").set("info", obj().set("title", "t")).set("host", "h")
	body, err := Marshal(o)
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}
	want := `{"swagger":"2.0","info":{"title":"t"},"host":"h"}`
	if string(body) != want {
		t.Errorf("marshal = %s, want %s", body, want)
	}
}

// TestAPIKeySchemeIsRendered covers `new ApiKey(name, keyname, passAs)`. The
// boot-swagger module declares the bean but publishes it from no Docket, so it
// never reaches a document there; the builder is still part of the ported
// configuration API.
func TestAPIKeySchemeIsRendered(t *testing.T) {
	scheme := NewAPIKey("api_key", "api_key", "header")
	if scheme.Type != SchemeAPIKey || scheme.Name != "api_key" ||
		scheme.KeyName != "api_key" || scheme.PassAs != "header" {
		t.Fatalf("NewAPIKey produced %+v", scheme)
	}

	body, err := Marshal(securityDefinitions([]SecurityScheme{scheme}))
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}
	want := `{"api_key":{"type":"apiKey","name":"api_key","in":"header"}}`
	if string(body) != want {
		t.Errorf("swagger 2.0 = %s, want %s", body, want)
	}

	body, err = Marshal(openAPISecuritySchemes([]SecurityScheme{scheme}))
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}
	if string(body) != want {
		t.Errorf("openapi = %s, want %s", body, want)
	}

	// The Swagger 1.2 writer keys it by type name, not by scheme name.
	body, err = Marshal(swagger12Authorizations([]SecurityScheme{scheme}))
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}
	want12 := `{"apiKey":{"keyname":"api_key","name":"api_key","passAs":"header","type":"apiKey"}}`
	if string(body) != want12 {
		t.Errorf("swagger 1.2 = %s, want %s", body, want12)
	}
}

// TestResponseExamplesAreRenderedInMediaTypeOrder covers the @ApiResponse
// example springfox-integration-webmvc declares, and the ordering the two
// specifications give it.
func TestResponseExamplesAreRenderedInMediaTypeOrder(t *testing.T) {
	response := api.Response{
		Code: http.StatusOK, Description: "OK",
		Examples: map[string]string{
			web.MediaTypeXML:  `<gnarf>dragons</gnarf>`,
			web.MediaTypeJSON: `{"gnarf": "dragons"}`,
		},
	}
	body, err := Marshal(swagger2Response(response))
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}
	want := `{"description":"OK","examples":{"application/json":"{\"gnarf\": \"dragons\"}",` +
		`"application/xml":"<gnarf>dragons</gnarf>"}}`
	if string(body) != want {
		t.Errorf("swagger 2.0 =\n%s\nwant\n%s", body, want)
	}

	body, err = Marshal(openAPIResponse(response, nil))
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}
	if !strings.Contains(string(body), `"content":{"application/json":{"example":`) {
		t.Errorf("openapi = %s", body)
	}
}
