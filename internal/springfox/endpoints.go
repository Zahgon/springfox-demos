package springfox

import (
	"net/http"
	"sort"
	"strings"

	"github.com/springfox/springfox-demos/internal/web"
)

// SwaggerResource is one entry of the /swagger-resources listing.
type SwaggerResource struct {
	Name           string `json:"name"`
	URL            string `json:"url"`
	SwaggerVersion string `json:"swaggerVersion"`
	Location       string `json:"location"`
}

// UIConfiguration is springfox.documentation.swagger.web.UiConfiguration, served
// verbatim at /swagger-resources/configuration/ui. Every value is Springfox's
// default; only swaggerBaseUiUrl varies, following the configured base URL.
type UIConfiguration struct {
	DeepLinking              bool     `json:"deepLinking"`
	DisplayOperationID       bool     `json:"displayOperationId"`
	DefaultModelsExpandDepth int      `json:"defaultModelsExpandDepth"`
	DefaultModelExpandDepth  int      `json:"defaultModelExpandDepth"`
	DefaultModelRendering    string   `json:"defaultModelRendering"`
	DisplayRequestDuration   bool     `json:"displayRequestDuration"`
	DocExpansion             string   `json:"docExpansion"`
	Filter                   bool     `json:"filter"`
	OperationsSorter         string   `json:"operationsSorter"`
	ShowExtensions           bool     `json:"showExtensions"`
	ShowCommonExtensions     bool     `json:"showCommonExtensions"`
	TagsSorter               string   `json:"tagsSorter"`
	ValidatorURL             string   `json:"validatorUrl"`
	SupportedSubmitMethods   []string `json:"supportedSubmitMethods"`
	SwaggerBaseUIURL         string   `json:"swaggerBaseUiUrl"`
}

// DefaultUIConfiguration returns the UI configuration for a base URL.
func DefaultUIConfiguration(baseURL string) UIConfiguration {
	return UIConfiguration{
		DeepLinking:              true,
		DefaultModelsExpandDepth: 1,
		DefaultModelExpandDepth:  1,
		DefaultModelRendering:    "example",
		DocExpansion:             "none",
		OperationsSorter:         "alpha",
		TagsSorter:               "alpha",
		ValidatorURL:             "",
		SupportedSubmitMethods: []string{
			"get", "put", "post", "delete", "options", "head", "patch", "trace",
		},
		SwaggerBaseUIURL: baseURL,
	}
}

// Options configure how a documentation provider is mounted on an application.
type Options struct {
	// BaseURL is springfox.documentation.swagger-ui.base-url. The
	// /swagger-resources endpoints move underneath it; the api-docs endpoints
	// do not.
	BaseURL string
	// SecurityConfiguration is served at /swagger-resources/configuration/security.
	SecurityConfiguration SecurityConfiguration
	// PublishResources controls whether /swagger-resources lists the registered
	// groups. Springfox populates the listing for @EnableSwagger2 /
	// @EnableOpenApi and for the boot starter, but leaves it empty for
	// @EnableSwagger2WebMvc and @EnableSwagger2WebFlux, whose resource provider
	// never sees the documentation cache.
	PublishResources bool
}

// The paths each specification version's api-docs endpoint is served at. Unlike
// the /swagger-resources endpoints these do not move under the configured
// swagger-ui base URL.
const (
	Swagger12Path = "/api-docs"
	Swagger2Path  = "/v2/api-docs"
	OAS30Path     = "/v3/api-docs"
)

// specPath is the api-docs path for a specification version.
func specPath(t DocumentationType) string {
	switch t {
	case Swagger12:
		return Swagger12Path
	case OAS30:
		return OAS30Path
	default:
		return Swagger2Path
	}
}

// Mount registers the documentation endpoints on the application. It must be
// called after every application route is registered, because the documents are
// built from the complete route table.
func (p *Provider) Mount(opts Options) {
	p.Build()
	base := strings.TrimSuffix(opts.BaseURL, "/")

	for _, spec := range p.mounted {
		spec := spec
		p.app.Handle(web.Route{
			Method:  http.MethodGet,
			Pattern: specPath(spec),
			Handler: func(r *web.Request) (web.ResponseEntity, error) {
				group := r.QueryOr("group", "default")
				doc, ok := p.Document(spec, group)
				if !ok {
					// The api-docs controller answers an unknown group with
					// `new ResponseEntity<>(HttpStatus.NOT_FOUND)`: a bodyless
					// 404 with no Content-Type, not the servlet error document.
					return web.Status(http.StatusNotFound), nil
				}
				return web.OK(doc), nil
			},
		})
	}

	p.app.Handle(web.Route{
		Method:  http.MethodGet,
		Pattern: base + "/swagger-resources",
		Handler: func(*web.Request) (web.ResponseEntity, error) {
			return web.OK(p.Resources(opts.PublishResources)), nil
		},
	})
	p.app.Handle(web.Route{
		Method:  http.MethodGet,
		Pattern: base + "/swagger-resources/configuration/ui",
		Handler: func(*web.Request) (web.ResponseEntity, error) {
			return web.OK(DefaultUIConfiguration(base)), nil
		},
	})
	p.app.Handle(web.Route{
		Method:  http.MethodGet,
		Pattern: base + "/swagger-resources/configuration/security",
		Handler: func(*web.Request) (web.ResponseEntity, error) {
			return web.OK(opts.SecurityConfiguration), nil
		},
	})
}

// Resources builds the /swagger-resources listing: every group, repeated once
// per enabled specification version, versions in the order they were enabled
// and groups sorted by name within each version.
func (p *Provider) Resources(publish bool) []SwaggerResource {
	out := []SwaggerResource{}
	if !publish {
		return out
	}
	names := make([]string, 0, len(p.dockets))
	for _, d := range p.dockets {
		names = append(names, d.Group())
	}
	sort.Strings(names)

	for _, spec := range p.advertised {
		path := specPath(spec)
		for _, name := range names {
			url := path
			if name != "default" {
				url = path + "?group=" + name
			}
			out = append(out, SwaggerResource{
				Name:           name,
				URL:            url,
				SwaggerVersion: spec.Version,
				Location:       url,
			})
		}
	}
	return out
}
