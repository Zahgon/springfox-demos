package springfox

// Docket is springfox.documentation.spring.web.plugins.Docket: one
// documentation group, its specification version, and the selection and
// decoration applied to it.
type Docket struct {
	docType DocumentationType

	groupName        string
	apiInfo          *APIInfo
	paths            PathSelector
	securitySchemes  []SecurityScheme
	securityContexts []SecurityContext
	// ignoredParameterTypes reproduces .ignoredParameterTypes(ApiIgnore.class):
	// a parameter marked with that annotation is dropped from the document.
	ignoreAnnotatedParams bool
	// urlTemplating reproduces .enableUrlTemplating(true), which renders paths
	// as RFC 6570 URI templates with their query parameters appended.
	urlTemplating bool
}

// NewDocket is `new Docket(documentationType)`.
func NewDocket(docType DocumentationType) *Docket {
	return &Docket{docType: docType, groupName: "default", paths: Any()}
}

// GroupName sets the group name, which becomes the ?group= query value.
func (d *Docket) GroupName(name string) *Docket { d.groupName = name; return d }

// APIInfo sets the info block.
func (d *Docket) APIInfo(info APIInfo) *Docket { d.apiInfo = &info; return d }

// Paths sets the path selector applied by select().paths(...).
func (d *Docket) Paths(sel PathSelector) *Docket { d.paths = sel; return d }

// SecuritySchemes sets the schemes published in the document.
func (d *Docket) SecuritySchemes(s []SecurityScheme) *Docket { d.securitySchemes = s; return d }

// SecurityContexts sets the contexts applied to matching operations.
func (d *Docket) SecurityContexts(c []SecurityContext) *Docket { d.securityContexts = c; return d }

// IgnoredAnnotatedParameters reproduces .ignoredParameterTypes(ApiIgnore.class).
func (d *Docket) IgnoredAnnotatedParameters() *Docket { d.ignoreAnnotatedParams = true; return d }

// EnableURLTemplating reproduces .enableUrlTemplating(v).
func (d *Docket) EnableURLTemplating(v bool) *Docket { d.urlTemplating = v; return d }

// Type returns the documentation type.
func (d *Docket) Type() DocumentationType { return d.docType }

// Group returns the group name.
func (d *Docket) Group() string { return d.groupName }

// Info returns the effective ApiInfo, falling back to Springfox's default.
func (d *Docket) Info() APIInfo {
	if d.apiInfo == nil {
		return DefaultAPIInfo()
	}
	return *d.apiInfo
}
