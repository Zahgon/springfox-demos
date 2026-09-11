// Command boot-swagger is a Spring Boot application with Swagger 1.2, 2.0 and
// OpenAPI 3.0 documentation enabled, ported from
// springfoxdemo.boot.swagger.Application. It is the only module with
// hand-written Springfox configuration.
package main

import (
	"time"

	demoweb "github.com/springfox/springfox-demos/boot-swagger/web"
	"github.com/springfox/springfox-demos/internal/boot"
	"github.com/springfox/springfox-demos/internal/petstore"
	"github.com/springfox/springfox-demos/internal/springfox"
	"github.com/springfox/springfox-demos/internal/web"
)

// contextPath mirrors src/main/resources/application.properties:
//
//	#Comment the below property for the Swagger page to work again
//	server.servlet.contextPath=/springfox
const contextPath = "/springfox"

// apiDescription is the ApiInfo description, assembled in the original from
// five concatenated string literals.
const apiDescription = "Lorem Ipsum is simply dummy text of the printing and typesetting industry. " +
	"Lorem Ipsum has been the industry's standard dummy text ever since the 1500s, when an " +
	"unknown printer took a galley of type and scrambled it to make a type specimen book. " +
	"It has survived not only five centuries, but also the leap into electronic typesetting, " +
	"remaining essentially unchanged. It was popularised in the 1960s with the release of " +
	"Letraset sheets containing Lorem Ipsum passages, and more recently with desktop " +
	"publishing software like Aldus PageMaker including versions of Lorem Ipsum."

// NewApplication assembles the application context.
func NewApplication() (*web.App, error) {
	app := web.NewApp("boot-swagger", web.StackServlet)
	app.ContextPath = contextPath

	// The three petstore controllers are declared as beans by the Java
	// application class.
	petstore.Register(app, func() int64 { return time.Now().UnixMilli() })

	demoweb.NewCategoryController().Register(app)
	demoweb.NewFileUploadController().Register(app)
	// HomeController is a component with no request mapping; instantiating it
	// registers nothing, which is exactly what the original does.
	_ = demoweb.NewHomeController()

	boot.RegisterErrorController(app, false)

	docs := springfox.NewProvider(app)
	// @EnableSwagger, @EnableSwagger2 and @EnableOpenApi, in that order.
	docs.EnableSpec(springfox.Swagger12)
	docs.EnableSpec(springfox.Swagger2)
	docs.EnableSpec(springfox.OAS30)

	// The five Docket beans. Groups are generated OpenAPI-first, so the
	// operation ids of the shared petstore handlers land in the same groups as
	// the original's.
	docs.AddDocket(openAPIPetStore())
	docs.AddDocket(petAPI())
	docs.AddDocket(categoryAPI())
	docs.AddDocket(multipartAPI())
	docs.AddDocket(userAPI())

	docs.Mount(springfox.Options{
		SecurityConfiguration: securityInfo(),
		PublishResources:      true,
	})

	ConfigureSwaggerUI(app, swaggerUIBaseURL)
	return app, nil
}

// petstorePaths is the shared path selector of petApi and openApiPetStore.
func petstorePaths() springfox.PathSelector {
	return springfox.Regex(".*/api/pet.*").
		Or(springfox.Regex(".*/api/user.*").
			Or(springfox.Regex(".*/api/store.*")))
}

// categoryPaths is categoryApi's path selector.
func categoryPaths() springfox.PathSelector {
	return springfox.Regex(".*/category.*").
		Or(springfox.Regex(".*/category").
			Or(springfox.Regex(".*/categories")))
}

// multipartPaths is multipartApi's path selector.
func multipartPaths() springfox.PathSelector { return springfox.Regex(".*/upload.*") }

// petAPI is the petApi bean.
func petAPI() *springfox.Docket {
	return springfox.NewDocket(springfox.Swagger2).
		GroupName("full-petstore-api").
		APIInfo(apiInfo()).
		Paths(petstorePaths()).
		SecuritySchemes([]springfox.SecurityScheme{oauth()}).
		SecurityContexts([]springfox.SecurityContext{securityContext()})
}

// categoryAPI is the categoryApi bean.
func categoryAPI() *springfox.Docket {
	return springfox.NewDocket(springfox.Swagger2).
		GroupName("category-api").
		APIInfo(apiInfo()).
		Paths(categoryPaths()).
		IgnoredAnnotatedParameters().
		EnableURLTemplating(true)
}

// multipartAPI is the multipartApi bean.
func multipartAPI() *springfox.Docket {
	return springfox.NewDocket(springfox.Swagger2).
		GroupName("multipart-api").
		APIInfo(apiInfo()).
		Paths(multipartPaths())
}

// userAPI is the userApi bean.
func userAPI() *springfox.Docket {
	authScopes := []springfox.AuthorizationScope{
		springfox.NewAuthorizationScopeBuilder().
			Scope("read").
			Description("read access").
			Build(),
	}
	securityReference := springfox.NewSecurityReferenceBuilder().
		Reference("test").
		Scopes(authScopes).
		Build()
	securityContexts := []springfox.SecurityContext{
		springfox.NewSecurityContextBuilder().
			SecurityReferences([]springfox.SecurityReference{securityReference}).
			Build(),
	}
	return springfox.NewDocket(springfox.Swagger2).
		SecuritySchemes([]springfox.SecurityScheme{springfox.NewBasicAuth("test")}).
		SecurityContexts(securityContexts).
		GroupName("user-api").
		APIInfo(apiInfo()).
		Paths(springfox.Contains("user"))
}

// openAPIPetStore is the openApiPetStore bean. It declares no ApiInfo, so its
// document carries Springfox's default info block.
func openAPIPetStore() *springfox.Docket {
	return springfox.NewDocket(springfox.OAS30).
		GroupName("open-api-pet-store").
		Paths(petstorePaths())
}

// apiInfo is the shared ApiInfo of the four Swagger 2.0 dockets.
func apiInfo() springfox.APIInfo {
	return springfox.NewAPIInfoBuilder().
		Title("Springfox petstore API").
		Description(apiDescription).
		TermsOfServiceURL("http://springfox.io").
		Contact(springfox.Contact{Name: "springfox"}).
		License("Apache License Version 2.0").
		LicenseURL("https://github.com/springfox/springfox/blob/master/LICENSE").
		Version("2.0").
		Build()
}

// securityContext is the securityContext bean.
func securityContext() springfox.SecurityContext {
	readScope := springfox.AuthorizationScope{Scope: "read:pets", Description: "read your pets"}
	securityReference := springfox.NewSecurityReferenceBuilder().
		Reference("petstore_auth").
		Scopes([]springfox.AuthorizationScope{readScope}).
		Build()
	return springfox.NewSecurityContextBuilder().
		SecurityReferences([]springfox.SecurityReference{securityReference}).
		ForPaths(springfox.Ant(".*/api/pet.*")).
		Build()
}

// oauth is the oauth bean.
func oauth() springfox.SecurityScheme {
	return springfox.NewOAuthBuilder().
		Name("petstore_auth").
		GrantTypes(grantTypes()).
		Scopes(scopes()).
		Build()
}

// apiKey is the apiKey bean. No Docket publishes it, so it never reaches a
// document; the bean exists because the original declares it.
func apiKey() springfox.SecurityScheme {
	return springfox.NewAPIKey("api_key", "api_key", "header")
}

// scopes are the OAuth scopes, in declaration order.
func scopes() []springfox.AuthorizationScope {
	return []springfox.AuthorizationScope{
		{Scope: "write:pets", Description: "modify pets in your account"},
		{Scope: "read:pets", Description: "read your pets"},
	}
}

// grantTypes is the single implicit grant.
func grantTypes() []springfox.GrantType {
	grantType := springfox.NewImplicitGrantBuilder().
		LoginEndpoint(springfox.LoginEndpoint{URL: "http://petstore.swagger.io/api/oauth/dialog"}).
		Build()
	return []springfox.GrantType{grantType}
}

// securityInfo is the securityInfo bean, served at
// /swagger-resources/configuration/security.
func securityInfo() springfox.SecurityConfiguration {
	return springfox.NewSecurityConfigurationBuilder().
		ClientID("abc").
		ClientSecret("123").
		Realm("pets").
		AppName("petstore").
		ScopeSeparator(",").
		Build()
}
