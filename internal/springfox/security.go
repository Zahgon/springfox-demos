package springfox

// AuthorizationScope is springfox.documentation.service.AuthorizationScope.
type AuthorizationScope struct {
	Scope       string
	Description string
}

// AuthorizationScopeBuilder is the matching builder.
type AuthorizationScopeBuilder struct{ scope AuthorizationScope }

// NewAuthorizationScopeBuilder starts a scope builder.
func NewAuthorizationScopeBuilder() *AuthorizationScopeBuilder { return &AuthorizationScopeBuilder{} }

// Scope sets the scope name.
func (b *AuthorizationScopeBuilder) Scope(v string) *AuthorizationScopeBuilder {
	b.scope.Scope = v
	return b
}

// Description sets the scope description.
func (b *AuthorizationScopeBuilder) Description(v string) *AuthorizationScopeBuilder {
	b.scope.Description = v
	return b
}

// Build returns the assembled scope.
func (b *AuthorizationScopeBuilder) Build() AuthorizationScope { return b.scope }

// SchemeType distinguishes the security-scheme flavours Springfox renders.
type SchemeType string

// The scheme types the demos declare.
const (
	SchemeAPIKey SchemeType = "apiKey"
	SchemeBasic  SchemeType = "basic"
	SchemeOAuth2 SchemeType = "oauth2"
)

// SecurityScheme is springfox.documentation.service.SecurityScheme.
type SecurityScheme struct {
	Type SchemeType
	Name string
	// KeyName and PassAs apply to apiKey schemes.
	KeyName string
	PassAs  string
	// GrantTypes and Scopes apply to oauth2 schemes.
	GrantTypes []GrantType
	Scopes     []AuthorizationScope
}

// NewAPIKey is `new ApiKey(name, keyname, passAs)`.
func NewAPIKey(name, keyName, passAs string) SecurityScheme {
	return SecurityScheme{Type: SchemeAPIKey, Name: name, KeyName: keyName, PassAs: passAs}
}

// NewBasicAuth is `new BasicAuth(name)`.
func NewBasicAuth(name string) SecurityScheme {
	return SecurityScheme{Type: SchemeBasic, Name: name}
}

// GrantType is springfox.documentation.service.GrantType.
type GrantType struct {
	Type          string
	LoginEndpoint string
}

// LoginEndpoint is springfox.documentation.service.LoginEndpoint.
type LoginEndpoint struct{ URL string }

// ImplicitGrantBuilder is springfox.documentation.builders.ImplicitGrantBuilder.
type ImplicitGrantBuilder struct{ grant GrantType }

// NewImplicitGrantBuilder starts an implicit-grant builder.
func NewImplicitGrantBuilder() *ImplicitGrantBuilder {
	return &ImplicitGrantBuilder{grant: GrantType{Type: "implicit"}}
}

// LoginEndpoint sets the authorisation URL.
func (b *ImplicitGrantBuilder) LoginEndpoint(e LoginEndpoint) *ImplicitGrantBuilder {
	b.grant.LoginEndpoint = e.URL
	return b
}

// Build returns the assembled grant type.
func (b *ImplicitGrantBuilder) Build() GrantType { return b.grant }

// OAuthBuilder is springfox.documentation.builders.OAuthBuilder.
type OAuthBuilder struct{ scheme SecurityScheme }

// NewOAuthBuilder starts an OAuth scheme builder.
func NewOAuthBuilder() *OAuthBuilder {
	return &OAuthBuilder{scheme: SecurityScheme{Type: SchemeOAuth2}}
}

// Name sets the scheme name.
func (b *OAuthBuilder) Name(v string) *OAuthBuilder { b.scheme.Name = v; return b }

// GrantTypes sets the grant types.
func (b *OAuthBuilder) GrantTypes(v []GrantType) *OAuthBuilder { b.scheme.GrantTypes = v; return b }

// Scopes sets the authorisation scopes.
func (b *OAuthBuilder) Scopes(v []AuthorizationScope) *OAuthBuilder { b.scheme.Scopes = v; return b }

// Build returns the assembled scheme.
func (b *OAuthBuilder) Build() SecurityScheme { return b.scheme }

// SecurityReference is springfox.documentation.service.SecurityReference.
type SecurityReference struct {
	Reference string
	Scopes    []AuthorizationScope
}

// SecurityReferenceBuilder is the matching builder.
type SecurityReferenceBuilder struct{ ref SecurityReference }

// NewSecurityReferenceBuilder starts a security-reference builder.
func NewSecurityReferenceBuilder() *SecurityReferenceBuilder { return &SecurityReferenceBuilder{} }

// Reference sets the referenced scheme name.
func (b *SecurityReferenceBuilder) Reference(v string) *SecurityReferenceBuilder {
	b.ref.Reference = v
	return b
}

// Scopes sets the required scopes.
func (b *SecurityReferenceBuilder) Scopes(v []AuthorizationScope) *SecurityReferenceBuilder {
	b.ref.Scopes = v
	return b
}

// Build returns the assembled reference.
func (b *SecurityReferenceBuilder) Build() SecurityReference { return b.ref }

// SecurityContext is springfox.documentation.spi.service.contexts.SecurityContext.
type SecurityContext struct {
	References []SecurityReference
	// ForPaths restricts the context; a nil selector applies it everywhere.
	ForPaths PathSelector
}

// SecurityContextBuilder is the matching builder.
type SecurityContextBuilder struct{ ctx SecurityContext }

// NewSecurityContextBuilder starts a security-context builder.
func NewSecurityContextBuilder() *SecurityContextBuilder { return &SecurityContextBuilder{} }

// SecurityReferences sets the references the context applies.
func (b *SecurityContextBuilder) SecurityReferences(v []SecurityReference) *SecurityContextBuilder {
	b.ctx.References = v
	return b
}

// ForPaths restricts the context to paths the selector admits.
func (b *SecurityContextBuilder) ForPaths(sel PathSelector) *SecurityContextBuilder {
	b.ctx.ForPaths = sel
	return b
}

// Build returns the assembled context.
func (b *SecurityContextBuilder) Build() SecurityContext { return b.ctx }

// SecurityConfiguration is springfox.documentation.swagger.web.SecurityConfiguration,
// served verbatim at /swagger-resources/configuration/security.
type SecurityConfiguration struct {
	ClientID          string `json:"clientId,omitempty"`
	ClientSecret      string `json:"clientSecret,omitempty"`
	Realm             string `json:"realm,omitempty"`
	AppName           string `json:"appName,omitempty"`
	ScopeSeparator    string `json:"scopeSeparator,omitempty"`
	EnableCsrfSupport bool   `json:"enableCsrfSupport,omitempty"`
}

// SecurityConfigurationBuilder is the matching builder.
type SecurityConfigurationBuilder struct{ cfg SecurityConfiguration }

// NewSecurityConfigurationBuilder starts a security-configuration builder.
func NewSecurityConfigurationBuilder() *SecurityConfigurationBuilder {
	return &SecurityConfigurationBuilder{}
}

// ClientID sets the OAuth client id.
func (b *SecurityConfigurationBuilder) ClientID(v string) *SecurityConfigurationBuilder {
	b.cfg.ClientID = v
	return b
}

// ClientSecret sets the OAuth client secret.
func (b *SecurityConfigurationBuilder) ClientSecret(v string) *SecurityConfigurationBuilder {
	b.cfg.ClientSecret = v
	return b
}

// Realm sets the OAuth realm.
func (b *SecurityConfigurationBuilder) Realm(v string) *SecurityConfigurationBuilder {
	b.cfg.Realm = v
	return b
}

// AppName sets the application name shown by the UI.
func (b *SecurityConfigurationBuilder) AppName(v string) *SecurityConfigurationBuilder {
	b.cfg.AppName = v
	return b
}

// ScopeSeparator sets the separator used between scopes.
func (b *SecurityConfigurationBuilder) ScopeSeparator(v string) *SecurityConfigurationBuilder {
	b.cfg.ScopeSeparator = v
	return b
}

// EnableCsrfSupport turns on the UI's CSRF handling.
func (b *SecurityConfigurationBuilder) EnableCsrfSupport(v bool) *SecurityConfigurationBuilder {
	b.cfg.EnableCsrfSupport = v
	return b
}

// Build returns the assembled configuration.
func (b *SecurityConfigurationBuilder) Build() SecurityConfiguration { return b.cfg }
