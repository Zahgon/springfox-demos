package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	demoweb "github.com/springfox/springfox-demos/boot-swagger/web"
	"github.com/springfox/springfox-demos/internal/web"
)

func newTestMvc(t *testing.T) *web.MockMvc {
	t.Helper()
	app, err := NewApplication()
	if err != nil {
		t.Fatalf("the application context failed to start: %v", err)
	}
	return web.NewMockMvc(app)
}

// TestContextLoads gives boot-swagger the construction guarantee the boot
// modules' contextLoads tests give. The Java module had no test of its own.
func TestContextLoads(t *testing.T) {
	if _, err := NewApplication(); err != nil {
		t.Fatalf("the application context failed to start: %v", err)
	}
}

// TestContextPathApplies covers server.servlet.contextPath=/springfox.
func TestContextPathApplies(t *testing.T) {
	mvc := newTestMvc(t)
	if res := mvc.Perform(web.Get("/springfox/category/Resource?someEnum=ONE")); res.Status != http.StatusOK {
		t.Errorf("under the context path, status = %d, want %d", res.Status, http.StatusOK)
	}
	if res := mvc.Perform(web.Get("/category/Resource?someEnum=ONE")); res.Status != http.StatusNotFound {
		t.Errorf("without the context path, status = %d, want %d", res.Status, http.StatusNotFound)
	}
}

// TestCategoryController covers every mapping of CategoryController.
func TestCategoryController(t *testing.T) {
	mvc := newTestMvc(t)
	cases := []struct {
		name        string
		build       func() *web.RequestBuilder
		wantStatus  int
		wantBody    string
		wantContent string
	}{
		{
			name:        "search returns the enum constant's own name",
			build:       func() *web.RequestBuilder { return web.Get("/springfox/category/Resource?someEnum=TWO") },
			wantStatus:  http.StatusOK,
			wantBody:    "TWO",
			wantContent: web.CharsetTextPlain,
		},
		{
			name:       "search rejects a value that names no constant",
			build:      func() *web.RequestBuilder { return web.Get("/springfox/category/Resource?someEnum=BAD") },
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "search requires someEnum",
			build:      func() *web.RequestBuilder { return web.Get("/springfox/category/Resource") },
			wantStatus: http.StatusBadRequest,
		},
		{
			name:        "map returns a freshly empty map",
			build:       func() *web.RequestBuilder { return web.Get("/springfox/category/map") },
			wantStatus:  http.StatusOK,
			wantBody:    "{}",
			wantContent: web.MediaTypeJSON,
		},
		{
			name: "someOperation returns an empty 200",
			build: func() *web.RequestBuilder {
				return web.Post("/springfox/category/7").ContentType(web.MediaTypeJSON).Content("42")
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "ignoredParam returns an empty 200",
			build:      func() *web.RequestBuilder { return web.Post("/springfox/category/7/9") },
			wantStatus: http.StatusOK,
		},
		{
			name:       "map with request parameters returns an empty 200",
			build:      func() *web.RequestBuilder { return web.Post("/springfox/category/x/map?a=1&b=2") },
			wantStatus: http.StatusOK,
		},
		{
			name:       "categories returns an empty 200",
			build:      func() *web.RequestBuilder { return web.Post("/springfox/categories?categories=A&categories=B") },
			wantStatus: http.StatusOK,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := mvc.Perform(c.build())
			if res.Status != c.wantStatus {
				t.Fatalf("status = %d, want %d", res.Status, c.wantStatus)
			}
			if c.wantBody != "" && res.Body != c.wantBody {
				t.Errorf("body = %q, want %q", res.Body, c.wantBody)
			}
			if c.wantContent != "" && res.ContentType != c.wantContent {
				t.Errorf("Content-Type = %q, want %q", res.ContentType, c.wantContent)
			}
		})
	}
}

// TestHomeControllerIsNotMapped pins the deliberate non-mapping: HomeController
// has no @RequestMapping, so its view name is never reachable over HTTP.
func TestHomeControllerIsNotMapped(t *testing.T) {
	mvc := newTestMvc(t)
	for _, path := range []string{"/springfox/", "/springfox/home", "/springfox/index"} {
		if res := mvc.Perform(web.Get(path)); res.Status != http.StatusNotFound {
			t.Errorf("%s: status = %d, want %d", path, res.Status, http.StatusNotFound)
		}
	}
	if got := demoweb.NewHomeController().Home(); got != "redirect:/swagger-ui/index.html" {
		t.Errorf("Home() = %q, want %q", got, "redirect:/swagger-ui/index.html")
	}
}

// TestSwaggerResourcesListsEveryGroupUnderEveryVersion covers the resource
// listing: five groups repeated once per enabled specification version, the
// versions in the order the @Enable… annotations turned them on and the groups
// sorted by name within each version.
func TestSwaggerResourcesListsEveryGroupUnderEveryVersion(t *testing.T) {
	res := newTestMvc(t).Perform(web.Get("/springfox/swagger-resources"))
	if res.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Status, http.StatusOK)
	}
	var resources []struct {
		Name           string `json:"name"`
		URL            string `json:"url"`
		SwaggerVersion string `json:"swaggerVersion"`
		Location       string `json:"location"`
	}
	if err := json.Unmarshal([]byte(res.Body), &resources); err != nil {
		t.Fatalf("parsing the resource listing: %v", err)
	}
	if len(resources) != 15 {
		t.Fatalf("got %d resources, want 15", len(resources))
	}
	wantGroups := []string{"category-api", "full-petstore-api", "multipart-api", "open-api-pet-store", "user-api"}
	wantVersions := []string{"1.2", "2.0", "3.0.3"}
	wantPaths := []string{"/api-docs", "/v2/api-docs", "/v3/api-docs"}
	for v, version := range wantVersions {
		for g, group := range wantGroups {
			r := resources[v*len(wantGroups)+g]
			if r.Name != group || r.SwaggerVersion != version {
				t.Fatalf("resource %d = %s/%s, want %s/%s", v*len(wantGroups)+g, r.Name, r.SwaggerVersion, group, version)
			}
			want := wantPaths[v] + "?group=" + group
			if r.URL != want || r.Location != want {
				t.Errorf("resource %s/%s url = %s, want %s", group, version, r.URL, want)
			}
		}
	}
}

// TestApiKeyBeanIsDeclaredButUnpublished covers the apiKey() bean. The original
// declares it and then hands it to no Docket, so it reaches no document; the
// bean and its values are still part of the ported configuration.
func TestApiKeyBeanIsDeclaredButUnpublished(t *testing.T) {
	scheme := apiKey()
	if scheme.Name != "api_key" || scheme.KeyName != "api_key" || scheme.PassAs != "header" {
		t.Errorf("apiKey() = %+v", scheme)
	}

	mvc := newTestMvc(t)
	for _, group := range []string{"full-petstore-api", "category-api", "multipart-api", "user-api"} {
		res := mvc.Perform(web.Get("/springfox/v2/api-docs?group=" + group))
		if strings.Contains(res.Body, `"api_key":{"type":"apiKey"`) {
			t.Errorf("%s publishes an apiKey security definition, but no Docket declares it", group)
		}
	}
	// The pet operations still name api_key in their own security block, from
	// @Authorization on the handler rather than from the Docket.
	petstore := mvc.Perform(web.Get("/springfox/v2/api-docs?group=full-petstore-api"))
	if !strings.Contains(petstore.Body, `{"api_key":[]}`) {
		t.Error("getPetById lost its api_key authorization requirement")
	}
}

// TestSwagger12ResourceListing covers @EnableSwagger, whose 1.2 writer emits a
// resource listing rather than an API declaration: one entry per controller
// tag at /<group>/<tag>, keys in alphabetical order, `contact` flattened to the
// contact's name and written even when empty, and authorizations keyed by the
// scheme's 1.2 type name rather than by its own name.
func TestSwagger12ResourceListing(t *testing.T) {
	mvc := newTestMvc(t)

	category := mvc.Perform(web.Get("/springfox/api-docs?group=category-api"))
	if category.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", category.Status, http.StatusOK)
	}
	wantCategory := `{"apiVersion":"2.0","apis":[{"description":"Category Controller",` +
		`"path":"/category-api/category-controller","position":0}],"authorizations":{},` +
		`"info":{"contact":"springfox","description":"` + apiDescription + `",` +
		`"license":"Apache License Version 2.0",` +
		`"licenseUrl":"https://github.com/springfox/springfox/blob/master/LICENSE",` +
		`"termsOfServiceUrl":"http://springfox.io","title":"Springfox petstore API"},` +
		`"swaggerVersion":"1.2"}`
	if category.Body != wantCategory {
		t.Errorf("category-api =\n%s\nwant\n%s", category.Body, wantCategory)
	}

	// The OAuth scheme is keyed "oauth2", not "petstore_auth".
	petstore := mvc.Perform(web.Get("/springfox/api-docs?group=full-petstore-api"))
	wantAuth := `"authorizations":{"oauth2":{"grantTypes":{"implicit":{"loginEndpoint":` +
		`{"url":"http://petstore.swagger.io/api/oauth/dialog"},"type":"implicit"}},` +
		`"name":"petstore_auth","scopes":[{"description":"modify pets in your account",` +
		`"scope":"write:pets"},{"description":"read your pets","scope":"read:pets"}],` +
		`"type":"oauth2"}}`
	if !strings.Contains(petstore.Body, wantAuth) {
		t.Errorf("full-petstore-api authorizations are wrong:\n%s", petstore.Body)
	}
	// BasicAuth is keyed "basicAuth".
	userAPI := mvc.Perform(web.Get("/springfox/api-docs?group=user-api"))
	if !strings.Contains(userAPI.Body, `"authorizations":{"basicAuth":{"name":"test","type":"basicAuth"}}`) {
		t.Errorf("user-api authorizations are wrong:\n%s", userAPI.Body)
	}
	// A group whose Docket publishes no scheme gets an empty object, and its
	// empty contact name is still written.
	openAPI := mvc.Perform(web.Get("/springfox/api-docs?group=open-api-pet-store"))
	if !strings.Contains(openAPI.Body, `"authorizations":{},"info":{"contact":"",`) {
		t.Errorf("open-api-pet-store info is wrong:\n%s", openAPI.Body)
	}
	// An unknown group is a bodyless 404, as at the other api-docs endpoints.
	if unknown := mvc.Perform(web.Get("/springfox/api-docs?group=nope")); unknown.Status != http.StatusNotFound || unknown.Body != "" {
		t.Errorf("an unknown group gave %d %q", unknown.Status, unknown.Body)
	}
}

// TestSecurityConfigurationIsPublished covers the securityInfo bean.
func TestSecurityConfigurationIsPublished(t *testing.T) {
	res := newTestMvc(t).Perform(web.Get("/springfox/swagger-resources/configuration/security"))
	want := `{"clientId":"abc","clientSecret":"123","realm":"pets","appName":"petstore","scopeSeparator":","}`
	if res.Body != want {
		t.Errorf("security configuration = %s, want %s", res.Body, want)
	}
}

// TestUnknownDocumentationGroupIs404 covers the bodyless 404 the api-docs
// controller answers with when no group matches.
func TestUnknownDocumentationGroupIs404(t *testing.T) {
	res := newTestMvc(t).Perform(web.Get("/springfox/v2/api-docs"))
	if res.Status != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", res.Status, http.StatusNotFound)
	}
	if res.Body != "" {
		t.Errorf("body = %q, want it empty", res.Body)
	}
	if res.ContentType != "" {
		t.Errorf("Content-Type = %q, want none", res.ContentType)
	}
}

// TestCategoryApiUsesUriTemplates covers .enableUrlTemplating(true): the
// category-api group's paths are RFC 6570 templates with their query parameters
// appended, and the @ApiIgnore parameter is dropped.
func TestCategoryApiUsesUriTemplates(t *testing.T) {
	res := newTestMvc(t).Perform(web.Get("/springfox/v2/api-docs?group=category-api"))
	for _, want := range []string{
		`"/springfox/category/Resource{?someEnum}"`,
		`"/springfox/categories{?categories}"`,
		`"/springfox/category/{id}/map{?test}"`,
	} {
		if !strings.Contains(res.Body, want) {
			t.Errorf("the document does not contain the template %s", want)
		}
	}
	// The @ApiIgnore path variable of /category/{id}/{userId} is dropped, so
	// that operation documents only `id`. The body parameter also named userId,
	// on /category/{id}, is not annotated and survives.
	var doc struct {
		Paths map[string]map[string]struct {
			Parameters []struct{ Name, In string } `json:"parameters"`
		} `json:"paths"`
	}
	if err := json.Unmarshal([]byte(res.Body), &doc); err != nil {
		t.Fatalf("parsing the document: %v", err)
	}
	ignored := doc.Paths["/springfox/category/{id}/{userId}"]["post"]
	if len(ignored.Parameters) != 1 || ignored.Parameters[0].Name != "id" {
		t.Errorf("ignoredParam documents %+v, want only the id path variable", ignored.Parameters)
	}
	someOperation := doc.Paths["/springfox/category/{id}"]["post"]
	if len(someOperation.Parameters) != 2 {
		t.Errorf("someOperation documents %+v, want id and the userId body", someOperation.Parameters)
	}
}

// TestApiInfoIsCarriedByTheFourSwagger2Groups covers the shared ApiInfo, whose
// description is assembled from five concatenated literals in the original.
func TestApiInfoIsCarriedByTheFourSwagger2Groups(t *testing.T) {
	mvc := newTestMvc(t)
	for _, group := range []string{"full-petstore-api", "category-api", "multipart-api", "user-api"} {
		res := mvc.Perform(web.Get("/springfox/v2/api-docs?group=" + group))
		var doc struct {
			Info struct {
				Title          string `json:"title"`
				Description    string `json:"description"`
				Version        string `json:"version"`
				TermsOfService string `json:"termsOfService"`
				Contact        struct{ Name string }
				License        struct{ Name, URL string }
			} `json:"info"`
		}
		if err := json.Unmarshal([]byte(res.Body), &doc); err != nil {
			t.Fatalf("%s: parsing the document: %v", group, err)
		}
		if doc.Info.Title != "Springfox petstore API" {
			t.Errorf("%s: title = %q", group, doc.Info.Title)
		}
		if doc.Info.Description != apiDescription {
			t.Errorf("%s: description does not match the original literal", group)
		}
		if doc.Info.Version != "2.0" || doc.Info.TermsOfService != "http://springfox.io" {
			t.Errorf("%s: version/terms = %q/%q", group, doc.Info.Version, doc.Info.TermsOfService)
		}
		if doc.Info.Contact.Name != "springfox" {
			t.Errorf("%s: contact = %q", group, doc.Info.Contact.Name)
		}
		if doc.Info.License.Name != "Apache License Version 2.0" {
			t.Errorf("%s: license = %q", group, doc.Info.License.Name)
		}
	}
	// openApiPetStore declares no ApiInfo, so it keeps the default.
	res := mvc.Perform(web.Get("/springfox/v3/api-docs?group=open-api-pet-store"))
	if !strings.Contains(res.Body, `"title":"Api Documentation"`) {
		t.Error("the open-api-pet-store group did not fall back to the default ApiInfo")
	}
}

// TestOAuthSecurityDefinitions covers the oauth bean and its security context.
func TestOAuthSecurityDefinitions(t *testing.T) {
	res := newTestMvc(t).Perform(web.Get("/springfox/v2/api-docs?group=full-petstore-api"))
	for _, want := range []string{
		`"petstore_auth":{"type":"oauth2","authorizationUrl":"http://petstore.swagger.io/api/oauth/dialog","flow":"implicit"`,
		`"write:pets":"modify pets in your account"`,
		`"read:pets":"read your pets"`,
	} {
		if !strings.Contains(res.Body, want) {
			t.Errorf("the document does not contain %s", want)
		}
	}
	userAPI := newTestMvc(t).Perform(web.Get("/springfox/v2/api-docs?group=user-api"))
	if !strings.Contains(userAPI.Body, `"securityDefinitions":{"test":{"type":"basic"}}`) {
		t.Error("the user-api group does not publish the BasicAuth scheme")
	}
}

// TestFileUpload covers the multipart mapping.
func TestFileUpload(t *testing.T) {
	body := strings.Join([]string{
		"--BOUNDARY",
		`Content-Disposition: form-data; name="description"`,
		"",
		"a description",
		"--BOUNDARY",
		`Content-Disposition: form-data; name="file"; filename="pet.txt"`,
		"Content-Type: text/plain",
		"",
		"contents",
		"--BOUNDARY--",
		"",
	}, "\r\n")
	res := newTestMvc(t).Perform(web.Post("/springfox/upload").
		ContentType(web.MediaTypeMultipart + "; boundary=BOUNDARY").
		Content(body))
	if res.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Status, http.StatusOK)
	}
	if res.Body != "" {
		t.Errorf("body = %q, want it empty", res.Body)
	}
}

// TestCorsPreflight covers the two CORS mappings the configurer registers.
func TestCorsPreflight(t *testing.T) {
	res := newTestMvc(t).Perform(web.RequestFor(http.MethodOptions, "/springfox/api/pet").
		Header("Origin", "http://editor.swagger.io").
		Header("Access-Control-Request-Method", "GET"))
	if res.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Status, http.StatusOK)
	}
	want := map[string]string{
		"Access-Control-Allow-Origin":  "http://editor.swagger.io",
		"Access-Control-Allow-Methods": "GET,HEAD,POST",
		"Access-Control-Max-Age":       "1800",
		"Allow":                        "GET, HEAD, POST, PUT, DELETE, OPTIONS, PATCH",
	}
	for header, value := range want {
		if got := res.Headers.Get(header); got != value {
			t.Errorf("%s = %q, want %q", header, got, value)
		}
	}
	// An origin the mapping does not allow gets no CORS headers.
	other := newTestMvc(t).Perform(web.RequestFor(http.MethodOptions, "/springfox/api/pet").
		Header("Origin", "http://example.com").
		Header("Access-Control-Request-Method", "GET"))
	if other.Headers.Get("Access-Control-Allow-Origin") != "" {
		t.Error("a disallowed origin was echoed back")
	}
}

// TestSwaggerUIIsMountedAtTheBaseURL covers the resource handler and the view
// controller the configurer registers.
func TestSwaggerUIIsMountedAtTheBaseURL(t *testing.T) {
	mvc := newTestMvc(t)
	res := mvc.Perform(web.Get("/springfox/swagger-ui/"))
	if res.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Status, http.StatusOK)
	}
	if !strings.Contains(res.Body, "<title>Swagger UI</title>") {
		t.Error("the forwarded view is not the swagger-ui index page")
	}
	if res.ContentType != "text/html;charset=UTF-8" {
		t.Errorf("Content-Type = %q, want %q", res.ContentType, "text/html;charset=UTF-8")
	}
}
