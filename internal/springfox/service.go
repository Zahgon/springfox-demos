// Package springfox reimplements the observable surface of the Springfox
// documentation library: the Docket configuration API, the path selectors, the
// API-info and security models, the Swagger 1.2 / Swagger 2.0 / OpenAPI 3.0.3
// document generators, and the /swagger-resources endpoints.
//
// Springfox recovered its input by reflecting over Spring's handler mappings.
// The Go port has no reflection to lean on, so it consumes the explicit
// api.Operation metadata each route carries. Everything downstream of that —
// grouping, path selection, name de-duplication, document layout and JSON key
// order — is reproduced here because it is observable at /v2/api-docs,
// /v3/api-docs and /swagger-resources.
package springfox

// DocumentationType selects the specification a Docket renders.
type DocumentationType struct {
	Name    string
	Version string
}

// The three documentation types the demos use.
var (
	// Swagger12 is DocumentationType.SWAGGER_12.
	Swagger12 = DocumentationType{Name: "swagger", Version: "1.2"}
	// Swagger2 is DocumentationType.SWAGGER_2.
	Swagger2 = DocumentationType{Name: "swagger", Version: "2.0"}
	// OAS30 is DocumentationType.OAS_30.
	OAS30 = DocumentationType{Name: "openApi", Version: "3.0.3"}
)

// Contact is springfox.documentation.service.Contact.
type Contact struct {
	Name  string
	URL   string
	Email string
}

// APIInfo is springfox.documentation.service.ApiInfo.
type APIInfo struct {
	Title             string
	Description       string
	Version           string
	TermsOfServiceURL string
	Contact           Contact
	License           string
	LicenseURL        string
}

// DefaultAPIInfo is the ApiInfo Springfox falls back to when a Docket declares
// none. Its values are visible in every document that does not override them.
func DefaultAPIInfo() APIInfo {
	return APIInfo{
		Title:             "Api Documentation",
		Description:       "Api Documentation",
		Version:           "1.0",
		TermsOfServiceURL: "urn:tos",
		License:           "Apache 2.0",
		LicenseURL:        "http://www.apache.org/licenses/LICENSE-2.0",
	}
}

// APIInfoBuilder is springfox.documentation.builders.ApiInfoBuilder.
type APIInfoBuilder struct{ info APIInfo }

// NewAPIInfoBuilder starts an ApiInfo builder.
func NewAPIInfoBuilder() *APIInfoBuilder { return &APIInfoBuilder{} }

// Title sets the title.
func (b *APIInfoBuilder) Title(v string) *APIInfoBuilder { b.info.Title = v; return b }

// Description sets the description.
func (b *APIInfoBuilder) Description(v string) *APIInfoBuilder { b.info.Description = v; return b }

// TermsOfServiceURL sets the terms-of-service URL.
func (b *APIInfoBuilder) TermsOfServiceURL(v string) *APIInfoBuilder {
	b.info.TermsOfServiceURL = v
	return b
}

// Contact sets the contact block.
func (b *APIInfoBuilder) Contact(v Contact) *APIInfoBuilder { b.info.Contact = v; return b }

// License sets the licence name.
func (b *APIInfoBuilder) License(v string) *APIInfoBuilder { b.info.License = v; return b }

// LicenseURL sets the licence URL.
func (b *APIInfoBuilder) LicenseURL(v string) *APIInfoBuilder { b.info.LicenseURL = v; return b }

// Version sets the API version.
func (b *APIInfoBuilder) Version(v string) *APIInfoBuilder { b.info.Version = v; return b }

// Build returns the assembled ApiInfo.
func (b *APIInfoBuilder) Build() APIInfo { return b.info }
