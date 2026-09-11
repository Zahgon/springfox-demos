# springfox-demos

Springfox demo applications, in Go.

Each demo is an independent `main` package that builds to its own binary and
listens on port 8080.

# Build
```
go build ./...
```

# Test
```
go test ./...
```

## Examples
|Example   | Description  |
|---|---|
| boot-static-docs  | demonstrates generating static docs @ build time   |
| boot-swagger  | demonstrates application with manual configuration using explicit dockets and beans   |
| boot-webflux  | demonstrates reactive support and open api 3.0.3 support with auto configuration   |
| boot-webmvc  | demonstrates servlet support and open api 3.0.3 support with auto configuration   |
| spring-java-swagger  | demonstrates manual code configuration api 3.0.3 support on a non-boot app |
| spring-xml-swagger  | demonstrates the ported xml configuration api 3.0.3 support on a non-boot app |
| spring-integration-webflux  | demonstrates message-gateway support on a reactive project |
| spring-integration-webmvc  | demonstrates message-gateway support on a servlet project|

## Shared packages

| Package | What it provides |
|---|---|
| `internal/web` | the request mapping, binding, content negotiation, CORS, CSRF and error rendering the demos ran on Spring MVC and Spring WebFlux for |
| `internal/springfox` | the Docket API, path selectors, security model and the Swagger 1.2 / Swagger 2.0 / OpenAPI 3.0.3 document generators |
| `internal/petstore` | the sample petstore API three of the demos mount |
| `internal/boot` | Spring Boot's error controller and the actuator endpoints |
| `internal/integration` | the HTTP inbound gateway two of the demos build flows from |
| `internal/restdocs` | the REST Docs response-fields snippet writer |
| `internal/staticdocs` | the Swagger-to-Asciidoc writer |
| `internal/api` | the specification-neutral description each route carries |
| `internal/javalang` | the `java.lang` formatting the ported response bodies depend on |
